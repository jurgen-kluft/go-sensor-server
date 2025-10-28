package main

import (
	"flag"
	"fmt"
	"time"

	sensor_server "github.com/jurgen-kluft/go-sensor-server"
	"github.com/jurgen-kluft/go-sensor-server/logging"
	"github.com/jurgen-kluft/go-sensor-server/xtcp"
)

// echo client, simply connects to the echo server, sends a message and exits.
var (
	serverAddr string
	message    string
)

type EchoHandler struct{}

func (h *EchoHandler) OnTcpAccept(c *xtcp.Conn) {
	fmt.Println("OnAccept:", c.String())
}

func (h *EchoHandler) OnTcpConnect(c *xtcp.Conn) {
	fmt.Println("OnConnect:", c.String())
}

func (h *EchoHandler) OnTcpRecv(c *xtcp.Conn, p []byte, t time.Time) {
	fmt.Println("OnRecv:", c.String(), "len:", len(p))
}

func (h *EchoHandler) OnTcpClose(c *xtcp.Conn) {
	fmt.Println("OnClose:", c.String())
}

func main() {
	flag.StringVar(&serverAddr, "server", ":31339", "server address")
	flag.StringVar(&message, "message", "hello", "message to send")
	flag.Parse()

	handler := &EchoHandler{}
	opts := xtcp.NewOpts(handler)
	logger := logging.NewDefault()

	client := xtcp.NewConn(opts, logger)
	clientClosed := make(chan struct{})
	go func() {
		err := client.DialAndServe(serverAddr)
		if err != nil {
			fmt.Println("Client dial error:", err)
		}
		close(clientClosed)
	}()

	// delay for 5 seconds to simulate time between packets
	n := 5
	for i := 1; i <= n; i++ {
		time.Sleep(10 * time.Second)
		packet := createSensorValuePacket(uint32(i * 10000))
		sendPacket(client, packet)
	}

	time.Sleep(2 * time.Second)
	fmt.Println("Stopping client")
}

func sendPacket(c *xtcp.Conn, packet []byte) {
	// send message
	n, err := c.Send(packet)
	if err != nil {
		fmt.Println("Send error:", err)
		return
	}
	fmt.Printf("Sent %d bytes: %s\n", n, message)
}

func createSensorValuePacket(time uint32) []byte {
	packet := make([]byte, 0, 32)
	packet = writePacketHeader(packet, 1)

	// "temperature": 24
	// {"index": 22,"name": "floor1_livingarea_temperature", "type": "temperature"},
	temperatureSensor := sensor_server.NewSensorConfig(22, "floor1_livingarea_temperature", 0xAABBCCDDEEFF, sensor_server.Temperature)
	packet = writeSensorValue(packet, temperatureSensor, 24)

	// "humidity": 50
	// {"index": 23,"name": "floor1_livingarea_humidity", "type": "humidity"},
	humiditySensor := sensor_server.NewSensorConfig(23, "floor1_livingarea_humidity", 0xAABBCCDDEEFF, sensor_server.Humidity)
	packet = writeSensorValue(packet, humiditySensor, 50)

	packet = finalizePacket(packet)
	return packet
}

func writePacketHeader(packet []byte, id uint16) []byte {

	packet = append(packet, 0)                  // length
	packet = append(packet, 1)                  // version
	packet = append(packet, byte((id>>0)&0xFF)) // packet id
	packet = append(packet, byte((id>>8)&0xFF)) // packet id

	return packet
}

func writeSensorValue(packet []byte, sensor *sensor_server.SensorConfig, value uint64) []byte {
	// sensor type
	packet = append(packet, byte(sensor.Type()))

	n := (sensor.SizeInBits() + 7) / 8
	if n == 0 {
		return packet
	}

	packet = append(packet, byte(value))
	for i := 1; i < n; i++ {
		value = value >> 8
		packet = append(packet, byte(value&0xFF))
	}
	return packet
}

func finalizePacket(packet []byte) []byte {
	if len(packet)&1 == 1 {
		packet = append(packet, 0) // padding byte
	}
	packet[0] = (byte(len(packet)) + 1) / 2
	return packet
}
