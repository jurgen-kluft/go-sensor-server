package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	sensor_server "github.com/jurgen-kluft/go-sensor-server"
	"github.com/jurgen-kluft/go-sensor-server/xtcp"
)

type SensorHandler struct {
	config  *sensor_server.SensorServerConfig
	storage *sensor_server.SensorStorage
}

func newSensorHandler(config *sensor_server.SensorServerConfig) *SensorHandler {
	storage := sensor_server.NewSensorStorage(config)
	return &SensorHandler{config: config, storage: storage}
}

func (h *SensorHandler) OnAccept(c *xtcp.Conn) {
	fmt.Println("OnAccept:", c.String())
}

func (h *SensorHandler) OnConnect(c *xtcp.Conn) {
	fmt.Println("OnConnect:", c.String())
}

func (h *SensorHandler) OnRecv(c *xtcp.Conn, p xtcp.Packet) {
	fmt.Println("OnRecv:", c.String(), "len:", len(p.Body))

	// The first packet for this connection should be a sensor packet that contains the MacAddress of the device.
	sensor, err := sensor_server.DecodeNetworkPacket(p.Body)
	if err != nil {
		fmt.Println("Failed to decode sensor packet:", err)
		c.Stop(xtcp.StopGracefullyAndWait)
		return
	}

	if len(sensor.Values) == 1 && sensor.Values[0].SensorType == sensor_server.MacAddress {
		value := uint64(sensor.Values[0].Value)
		valueBytes := []byte{byte(value >> 0), byte(value >> 8), byte(value >> 16), byte(value >> 24), byte(value >> 32), byte(value >> 40), byte(value >> 48), byte(value >> 56)}
		macAddress := fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X", valueBytes[0], valueBytes[1], valueBytes[2], valueBytes[3], valueBytes[4], valueBytes[5])

		// Find the device with this macAddress
		if index, ok := h.config.DevicesMap[macAddress]; ok {
			c.UserData = int32(index) // Mark that we have received the sensor packet.
			fmt.Println("Registered device:", h.config.Devices[index].Name)
		} else {
			// This device is not registered, close this connection
			fmt.Println("Unknown device with MacAddress:", macAddress)
			c.Stop(xtcp.StopGracefullyAndWait)
		}
	} else {
		if c.UserData >= 0 {
			for _, v := range sensor.Values {
				sensorIndex := h.storage.RegisterSensor(c.UserData, v.SensorType)
				h.storage.WriteSensorValue(c.UserData, sensorIndex, sensor.IsTimeSync, sensor.TimeSync, v)
			}
		}
	}
}

func (h *SensorHandler) OnClose(c *xtcp.Conn) {
	fmt.Println("OnClose:", c.String())
}

const (
	exitCodeErr       = 1
	exitCodeInterrupt = 2
)

func run(ctx context.Context, args []string) error {
	var server *xtcp.Server

	config, err := sensor_server.LoadSensorServerConfig("sensor-server.config.json")
	if err != nil {
		return err
	}

	handler := newSensorHandler(config)
	options := xtcp.NewOpts(handler).SetRecvBufSize(1024)

	server = xtcp.NewServer(options)
	go server.ListenAndServe(fmt.Sprintf(":%d", config.TcpPort))

	for range ctx.Done() {
		server.Stop(xtcp.StopGracefullyAndWait)
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
	if err := run(ctx, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(exitCodeErr)
	}
}
