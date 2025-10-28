package xipc

import (
	"time"
)

var ()

// StopMode define the stop mode of server and conn.
type StopMode uint8

const (
	StopImmediately          StopMode = iota // StopImmediately; stop directly, the cached data maybe will not send.
	StopGracefullyButNotWait                 // StopGracefullyButNotWait; stop and flush cached data.
	StopGracefullyAndWait                    // StopGracefullyAndWait; stop and block until cached data sended.
)

// Handler is the event callback.
// Note : don't block in event handler.
type Handler interface {
	OnIpcAccept(*Conn)                  // OnAccept; server accepted a new connect.
	OnIpcConnect(*Conn)                 // OnConnect; a new client connected to a server.
	OnIpcRecv(*Conn, []byte, time.Time) // OnRecv; a connection received data
	OnIpcSend(*Conn, []byte)            // OnSend; data was sent
	OnIpcClose(*Conn)                   // OnClose; a connection closed
}

// Packet is the unit of data.
type Packet struct {
	Body []byte
}

// Options is the options used for net conn.
type Options struct {
	Mode    SendReceiveMode
	Handler Handler
}

type SendReceiveMode int

const (
	ModeSendOnly SendReceiveMode = iota
	ModeReceiveOnly
	ModeSendAndReceive
)

// NewOpts create a new options and set some default value.
// will panic if handler or protocol is nil.
// eg: opts := NewOpts().SetSendListLen(len).SetRecvBufInitSize(len)...
func NewOptions(h Handler, mode SendReceiveMode) *Options {
	if h == nil {
		panic("xtcp.NewOpts: nil handler")
	}
	return &Options{
		Mode:    mode,
		Handler: h,
	}
}
