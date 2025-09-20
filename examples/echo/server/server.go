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
	"github.com/jurgen-kluft/go-sensor-server/examples/echo"
)

type Callback struct {
	// Free indices used for connections

}

func (this *Callback) OnConnect(c *sensor_server.Conn) bool {
	addr := c.GetRawConn().RemoteAddr()
	c.SetIndex(-1)
	fmt.Println("OnConnect:", addr)
	return true
}

func (this *Callback) OnMessage(c *sensor_server.Conn, p sensor_server.Packet) bool {
	var packet sensor_server.Packet
	fmt.Printf("OnMessage:[%v] [%v]\n", echo.GetLength(packet), string(echo.GetBody(packet)))
	//	c.AsyncWritePacket(echo.NewEchoPacket(echoPacket.Serialize(), true), time.Second)
	return true
}

func (this *Callback) OnClose(c *sensor_server.Conn) {
	fmt.Println("OnClose:", c.GetIndex())
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	// creates a tcp listener
	tcpAddr, err := net.ResolveTCPAddr("tcp4", ":8989")
	checkError(err)
	listener, err := net.ListenTCP("tcp", tcpAddr)
	checkError(err)

	// creates a server
	config := &sensor_server.Config{
		PacketSendChanLimit:    20,
		PacketReceiveChanLimit: 20,
	}
	srv := sensor_server.NewServer(config, &Callback{}, &echo.EchoProtocol{})

	// starts service
	go srv.Start(listener, time.Second)
	fmt.Println("listening:", listener.Addr())

	// catchs system signal
	chSig := make(chan os.Signal, 1)
	signal.Notify(chSig, syscall.SIGINT, syscall.SIGTERM)
	fmt.Println("Signal: ", <-chSig)

	// stops service
	srv.Stop()
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
