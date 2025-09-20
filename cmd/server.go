package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	sensor_server "github.com/jurgen-kluft/go-sensor-server"
)

type SensorServerCallback struct {
	config *sensor_server.SensorServerConfig
	store  *sensor_server.SensorStorage
}

func newSensorServerCallback(config *sensor_server.SensorServerConfig) *SensorServerCallback {
	store := sensor_server.NewSensorStorage(config)
	return &SensorServerCallback{config: config, store: store}
}

func (this *SensorServerCallback) OnConnect(c *sensor_server.Conn) bool {
	addr := c.GetRawConn().RemoteAddr()
	fmt.Println("OnConnect:", addr)
	c.SetIndex(-1)
	return true
}

func (this *SensorServerCallback) OnMessage(c *sensor_server.Conn, p sensor_server.Packet) bool {
	if c.GetIndex() == -1 {
		// The first message is the registration message, it contains the 'mac' address
		// of the sensor device. This mac address is used to identify the sensor store
		// that is listed in the JSON configuration file.
		fmt.Printf("OnMessage (register):[%v] [%v]\n", GetLength(p), string(GetBody(p)))
		mac := string(GetBody(p))
		if groupIndex, exists := this.config.DevicesMap[mac]; exists {
			c.SetIndex(groupIndex)
		} else {
			return false
		}
	} else {
		fmt.Printf("OnMessage (packet):[%v] [%v]\n", GetLength(p), string(GetBody(p)))

		groupIndex := int32(c.GetIndex())

		// Loop over the sensor values, since every sensor value has its own sensor data stream
		if sensorPacket, error := sensor_server.DecodeNetworkPacket(p.Body); error == nil {
			for _, sensorValue := range sensorPacket.Values {
				groupIndex, streamIndex := this.store.RegisterSensor(groupIndex, sensorValue.SensorType)
				if groupIndex >= 0 && streamIndex >= 0 {
					if err := this.store.WriteSensorValue(groupIndex, streamIndex, sensorPacket.Immediate, sensorPacket.TimeSync, sensorValue); err != nil {
						fmt.Printf("Error writing sensor value: %v\n", err)
					}
				} else {
					fmt.Printf("Error registering sensor value: %v\n", sensorValue)
				}
			}
		}
	}
	return true
}

func (this *SensorServerCallback) OnClose(c *sensor_server.Conn) {
	fmt.Println("OnClose:", c.GetIndex())
}

func main() {
	// Note: Is this not the default?
	runtime.GOMAXPROCS(runtime.NumCPU())

	sensorServerConfig, err := sensor_server.LoadSensorServerConfig("sensor-server.config.json")
	checkError(err)

	sensorPacketProtocol := &SensorPacketProtocol{}
	sensorServerCallback := newSensorServerCallback(sensorServerConfig)

	tcpAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%d", sensorServerConfig.TcpPort))
	checkError(err)
	listener, err := net.ListenTCP("tcp", tcpAddr)
	checkError(err)

	config := &sensor_server.Config{PacketSendChanLimit: 20, PacketReceiveChanLimit: 20}
	srv := sensor_server.NewServer(config, sensorServerCallback, sensorPacketProtocol)

	// starts service
	go srv.Start(listener, time.Second)
	fmt.Println("listening:", listener.Addr())

	// catchs system signal
	chSig := make(chan os.Signal, 1)
	signal.Notify(chSig, syscall.SIGINT, syscall.SIGTERM)
	fmt.Println("signal: ", <-chSig)

	// stops service
	srv.Stop()
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
