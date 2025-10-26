package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	sensor_server "github.com/jurgen-kluft/go-sensor-server"
	"github.com/jurgen-kluft/go-sensor-server/logging"
	"github.com/jurgen-kluft/go-sensor-server/xtcp"
	"github.com/jurgen-kluft/go-sensor-server/xudp"
)

type SensorHandler struct {
	logger  logging.Logger
	config  *sensor_server.SensorServerConfig
	storage *sensor_server.SensorStorage
}

func newSensorHandler(config *sensor_server.SensorServerConfig, logger logging.Logger) *SensorHandler {
	storage := sensor_server.NewSensorStorage(config, logger)
	return &SensorHandler{logger: logger, config: config, storage: storage}
}

func onShutdown(h *SensorHandler) {
	h.storage.Shutdown()
}

func (h *SensorHandler) OnTcpAccept(c *xtcp.Conn) {
	h.logger.LogInfof("OnAccept: %s", c.String())
}

func (h *SensorHandler) OnTcpConnect(c *xtcp.Conn) {
	h.logger.LogInfof("OnConnect: %s", c.String())
}

// OnTcpRecv will handle Tcp packets
func (h *SensorHandler) OnTcpRecv(c *xtcp.Conn, p []byte) {
	h.logger.LogInfof("OnRecv: %s len: %d", c.String(), len(p))

	sensorPacket, err := sensor_server.DecodeNetworkPacket(h.config.SensorMap, p)
	if err != nil {
		h.logger.LogError(err, "failed to decode sensor packet")
		c.Stop(xtcp.StopGracefullyAndWait)
		return
	}

	for _, v := range sensorPacket.Values {
		if err := h.storage.WriteSensorValue(v.Sensor, sensorPacket.Time, v); err != nil {
			h.logger.LogError(err)
		}
	}
}

// OnUdpRecv will handle Udp packets
func (h *SensorHandler) OnUdpRecv(p []byte) {
	sensorPacket, err := sensor_server.DecodeNetworkPacket(h.config.SensorMap, p)
	if err != nil {
		h.logger.LogError(err, "failed to decode sensor packet")
		return
	}

	for _, v := range sensorPacket.Values {
		if err := h.storage.WriteSensorValue(v.Sensor, sensorPacket.Time, v); err != nil {
			h.logger.LogError(err)
		}
	}
}

func (h *SensorHandler) OnTcpClose(c *xtcp.Conn) {
	h.logger.LogInfof("OnClose: %s", c.String())
}

const (
	exitCodeErr       = 1
	exitCodeInterrupt = 2
)

func run(ctx context.Context) error {
	var tcpServer *xtcp.Server
	var udpServer *xudp.Server

	logger := logging.NewDefault()

	configFilepath := "sensor-server.config.json"
	if len(os.Args) > 1 {
		configFilepath = os.Args[1]
	}
	logger.LogInfof("Using config file: %s", configFilepath)

	config, err := sensor_server.LoadSensorServerConfig(configFilepath)
	if err != nil {
		return err
	}

	handler := newSensorHandler(config, logger)
	options := xtcp.NewOpts(handler).SetRecvBufSize(1024)

	tcpServer = xtcp.NewServer(options, logger)
	go tcpServer.ListenAndServe(fmt.Sprintf(":%d", config.TcpPort))

	udpServer = xudp.NewServer(xudp.NewOpts(handler), logger)
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
