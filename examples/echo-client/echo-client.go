package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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

	timeSync := 1234567890

	// Send first packet which is nothing more than a mac address
	packet1 := make([]byte, 16)
	packet1[0] = 16 / 2
	packet1[1] = 1
	packet1[2] = byte((timeSync >> 24) & 0xFF)
	packet1[3] = byte((timeSync >> 16) & 0xFF)
	packet1[4] = byte((timeSync >> 8) & 0xFF)
	packet1[5] = byte((timeSync) & 0xFF)

	// "mac": "00:1A:2B:3C:4D:5E", u64
	packet1[6] = byte(sensor_server.MacAddress)
	packet1[7] = 0x00
	packet1[8] = 0x1A
	packet1[9] = 0x2B
	packet1[10] = 0x3C
	packet1[11] = 0x4D
	packet1[12] = 0x5E
	packet1[13] = 0
	packet1[14] = 0
	packet1[15] = 0 // padding byte to make length even

	// send message
	n, err := client.Send(packet1)
	if err != nil {
		fmt.Println("Send error:", err)
		return
	}
	fmt.Printf("Sent %d bytes: %s\n", n, message)

	// wait for signal to exit.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-sigCh
}
