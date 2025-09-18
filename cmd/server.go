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

type Callback struct {
	// Free indices used for connections
	config *sensor_server.SensorServerConfig
	store  *sensor_server.SensorStorage
}

func (this *Callback) OnConnect(c *sensor_server.Conn) bool {
	addr := c.GetRawConn().RemoteAddr()
	c.SetIndex(-1)
	fmt.Println("OnConnect:", addr)
	return true
}

func (this *Callback) OnMessage(c *sensor_server.Conn, p sensor_server.Packet) bool {
	if c.GetIndex() == -1 {
		// The first message is a registration message, it contains the 'mac' address
		// of the sensor device. This mac address is used to identify the sensor store
		// that is listed in the JSON configuration file.
		fmt.Printf("OnMessage (register):[%v] [%v]\n", GetLength(p), string(GetBody(p)))
		mac := string(GetBody(p))
		groupIndex := this.config.GroupMap[mac]
		c.SetIndex(groupIndex)
	} else {
		fmt.Printf("OnMessage:[%v] [%v]\n", GetLength(p), string(GetBody(p)))

		groupIndex := int32(c.GetIndex())

		// Loop over the sensor values, since every sensor value has its own sensor data stream
		if sensorPacket, error := sensor_server.DecodeNetworkPacket(p.Body); error == nil {
			for _, sensorValue := range sensorPacket.Values {
				// Get sensor type from the packet
				groupIndex, streamIndex := this.store.RegisterSensor(groupIndex, "sensor1", sensorValue.SensorType)
				if groupIndex >= 0 && streamIndex >= 0 {
					// WriteSensorValue(groupIndex int, sensorIndex int, packetImmediate bool, packetTimeSync int, sensorValue SensorValue) error {
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

func (this *Callback) OnClose(c *sensor_server.Conn) {
	fmt.Println("OnClose:", c.GetIndex())
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	tcpAddr, err := net.ResolveTCPAddr("tcp4", ":31338")
	checkError(err)
	listener, err := net.ListenTCP("tcp", tcpAddr)
	checkError(err)

	// creates a server
	config := &sensor_server.Config{
		PacketSendChanLimit:    20,
		PacketReceiveChanLimit: 20,
	}
	srv := sensor_server.NewServer(config, &Callback{}, &SensorPacketProtocol{})

	// starts service
	go srv.Start(listener, time.Second)
	fmt.Println("listening:", listener.Addr())

	// catchs system signal
	chSig := make(chan os.Signal)
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
