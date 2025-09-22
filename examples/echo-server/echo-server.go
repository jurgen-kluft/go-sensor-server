package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	sensor_server "github.com/jurgen-kluft/go-sensor-server"
	"github.com/jurgen-kluft/go-sensor-server/xtcp"
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

	sensor_server.LoadSensorServerConfig("echo-server.config.json")

	handler := &EchoHandler{}
	options := xtcp.NewOpts(handler).SetRecvBufSize(1024).SetAsyncWrite(true).SetSendBufListLen(1024)

	srv := xtcp.NewServer(options)
	go srv.ListenAndServe(":31339")

	// wait for signal to exit.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-sigCh

	srv.Stop(xtcp.StopGracefullyAndWait)
}
