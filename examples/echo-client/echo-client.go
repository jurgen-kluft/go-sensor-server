package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	sensor_server "github.com/jurgen-kluft/go-sensor-server"
	"github.com/jurgen-kluft/go-sensor-server/xtcp"
)

// echo client, simply connects to the echo server, sends a message and exits.
var (
	serverAddr string
	message    string
)

type EchoHandler struct{}

func (h *EchoHandler) OnAccept(c *xtcp.Conn) {
	fmt.Println("OnAccept:", c.String())
}

func (h *EchoHandler) OnConnect(c *xtcp.Conn) {
	fmt.Println("OnConnect:", c.String())
}

func (h *EchoHandler) OnRecv(c *xtcp.Conn, p xtcp.Packet) {
	fmt.Println("OnRecv:", c.String(), "len:", len(p.Body))
}

func (h *EchoHandler) OnClose(c *xtcp.Conn) {
	fmt.Println("OnClose:", c.String())
}

func main() {
	flag.StringVar(&serverAddr, "server", ":31339", "server address")
	flag.StringVar(&message, "message", "hello", "message to send")
	flag.Parse()

	handler := &EchoHandler{}
	opts := xtcp.NewOpts(handler)

	client := xtcp.NewConn(opts)
	clientClosed := make(chan struct{})
	go func() {
		err := client.DialAndServe(serverAddr)
		if err != nil {
			fmt.Println("Client dial error:", err)
		}
		close(clientClosed)
	}()

	packet := createMacAddressPacket()
	sendPacket(client, packet)

	// delay for 5 seconds to simulate time between packets
	n := 5
	for i := 1; i <= n; i++ {
		time.Sleep(16 * time.Second)
		packet = createSensorValuePacket(uint32(i * 10000))
		sendPacket(client, packet)
	}

	// wait for signal to exit.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-sigCh
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

func createMacAddressPacket() []byte {

	// Send first packet which is nothing more than a mac address
	packet := make([]byte, 0, 16)
	packet = writePacketHeader(packet, 0)

	// "mac": "00:1A:2B:3C:4D:5E", u64, highest byte should be at lowest byte in u64
	mac := uint64(0x005E4D3C2B1A00)
	packet = writeSensorValue(packet, sensor_server.MacAddress, mac)

	packet = finalizePacket(packet)
	return packet
}

func createSensorValuePacket(time uint32) []byte {
	packet := make([]byte, 0, 32)
	packet = writePacketHeader(packet, time)

	// "temperature": 24
	packet = writeSensorValue(packet, sensor_server.Temperature, 24)

	// "humidity": 50
	packet = writeSensorValue(packet, sensor_server.Humidity, 50)

	packet = finalizePacket(packet)
	return packet
}

func writePacketHeader(packet []byte, timeSync uint32) []byte {
	id := sensor_server.SensorPacketId

	packet = append(packet, 0)                         // length
	packet = append(packet, 1)                         // version
	packet = append(packet, byte((id>>0)&0xFF))        // packet id
	packet = append(packet, byte((id>>8)&0xFF))        // packet id
	packet = append(packet, byte((timeSync)&0xFF))     // time sync
	packet = append(packet, byte((timeSync>>8)&0xFF))  // time sync
	packet = append(packet, byte((timeSync>>16)&0xFF)) // time sync
	packet = append(packet, byte((timeSync>>24)&0xFF)) // time sync

	return packet
}

func writeSensorValue(packet []byte, sensorType sensor_server.SensorType, value uint64) []byte {
	// sensor type
	packet = append(packet, byte(sensorType))

	n := (sensorType.SizeInBits() + 7) / 8
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
