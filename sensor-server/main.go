package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	sensor_server "github.com/jurgen-kluft/go-sensor-server"
	"github.com/jurgen-kluft/go-sensor-server/xtcp"
	"github.com/jurgen-kluft/go-sensor-server/xudp"
)

type SensorHandler struct {
	config  *sensor_server.SensorServerConfig
	storage *sensor_server.SensorStorage
}

func newSensorHandler(config *sensor_server.SensorServerConfig) *SensorHandler {
	storage := sensor_server.NewSensorStorage(config)
	return &SensorHandler{config: config, storage: storage}
}

func onShutdown(h *SensorHandler) {
	h.storage.Shutdown()
}

func (h *SensorHandler) OnTcpAccept(c *xtcp.Conn) {
	fmt.Println("OnAccept:", c.String())
}

func (h *SensorHandler) OnTcpConnect(c *xtcp.Conn) {
	fmt.Println("OnConnect:", c.String())
}

func (h *SensorHandler) OnTcpRecv(c *xtcp.Conn, p xtcp.Packet) {
	fmt.Println("OnRecv:", c.String(), "len:", len(p.Body))

	// The first packet for this connection should be a sensor packet that contains the MacAddress of the device.
	sensorPacket, err := sensor_server.DecodeNetworkPacket(h.config.SensorMap, p.Body)
	if err != nil {
		fmt.Println("Failed to decode sensor packet:", err)
		c.Stop(xtcp.StopGracefullyAndWait)
		return
	}

	if c.UserData >= 0 {
		for _, v := range sensorPacket.Values {
			sensorIndex := h.storage.RegisterSensor(v.Sensor)
			h.storage.WriteSensorValue(sensorIndex, sensorPacket.Time, v)
		}
	}
}

// OnRecv Udp
func (h *SensorHandler) OnUdpRecv(p []byte) {
}

func (h *SensorHandler) OnTcpClose(c *xtcp.Conn) {
	fmt.Println("OnClose:", c.String())
}

const (
	exitCodeErr       = 1
	exitCodeInterrupt = 2
)

func run(ctx context.Context) error {
	var tcpServer *xtcp.Server
	var udpServer *xudp.Server

	config, err := sensor_server.LoadSensorServerConfig("sensor-server.config.json")
	if err != nil {
		return err
	}

	handler := newSensorHandler(config)
	options := xtcp.NewOpts(handler).SetRecvBufSize(1024)

	var (
		logger = log.New(os.Stdout, "logger: ", log.Lshortfile)
	)
	tcpServer = xtcp.NewServer(options, logger)
	go tcpServer.ListenAndServe(fmt.Sprintf(":%d", config.TcpPort))

	udpServer = xudp.NewServer(xudp.NewOpts(handler))
	go udpServer.ListenAndServe(fmt.Sprintf(":%d", config.UdpPort))

	for range ctx.Done() {
		tcpServer.Stop(xtcp.StopGracefullyAndWait)
		udpServer.Stop(xudp.StopGracefullyAndWait)
		onShutdown(handler)
		break
	}

	return nil
}

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()
	go func() {
		select {
		case <-signalChan: // first signal, cancel context
			cancel()
		case <-ctx.Done():
		}
		<-signalChan // second signal, hard exit
		os.Exit(exitCodeInterrupt)
	}()
	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(exitCodeErr)
	}
}
