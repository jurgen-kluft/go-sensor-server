package main

import (
	"fmt"
	"net"
	"time"

	sensor_server "github.com/jurgen-kluft/go-sensor-server"
	"github.com/jurgen-kluft/go-sensor-server/xtcp"
)

type myHandler struct {
}

func (h *myHandler) OnAccept(c *xtcp.Conn) {
	fmt.Println("accept : ", c.String())
	c.UserData = 0
}

func (h *myHandler) OnConnect(c *xtcp.Conn) {
	fmt.Println("connected : ", c.String())
	c.UserData = 1
}

func (h *myHandler) OnRecv(c *xtcp.Conn, p xtcp.Packet) {
	fmt.Println("recv : ", c.String(), " len=", len(p.Body), " userdata=", c.UserData)
}

func (h *myHandler) OnClose(c *xtcp.Conn) {
	fmt.Println("close : ", c.String(), " userdata=", c.UserData)
}

func main() {
	h := &myHandler{}
	opts := xtcp.NewOpts(h)
	l, err := net.Listen("tcp", ":")
	if err != nil {
		fmt.Println("listen err : ", err)
		return
	}
	server := xtcp.NewServer(opts)
	go func() {
		server.Serve(l)
	}()

	client := xtcp.NewConn(opts)
	clientClosed := make(chan struct{})
	go func() {
		err := client.DialAndServe(l.Addr().String())
		if err != nil {
			fmt.Println("client dial err : ", err)
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

	client.SendPacket(xtcp.Packet{Body: packet1})

	// Delay of 5 seconds before sending next packet
	time.Sleep(5 * time.Second)

	packet2 := make([]byte, 10)
	packet2[0] = 10 / 2
	packet2[1] = 1
	packet2[2] = byte((timeSync >> 24) & 0xFF)
	packet2[3] = byte((timeSync >> 16) & 0xFF)
	packet2[4] = byte((timeSync >> 8) & 0xFF)
	packet2[5] = byte((timeSync) & 0xFF)

	// 2 sensor values
	packet2[6] = byte(sensor_server.Temperature)
	packet2[7] = byte(25)
	packet2[8] = byte(sensor_server.Humidity)
	packet2[9] = byte(50)

	client.SendPacket(xtcp.Packet{Body: packet2})

	<-clientClosed
	server.Stop(xtcp.StopGracefullyAndWait)

	fmt.Println("server and client stopped")
}
