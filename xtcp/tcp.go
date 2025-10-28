package xtcp

import (
	"time"
)

var (
	DefaultRecvBufSize    = uint(8) << 10 // DefaultRecvBufSize is 8KB, the default size of recv buf.
	DefaultSendBufListLen = uint(1) << 10 // DefaultSendBufListLen is 1KB, the default length of send buf list.
	DefaultAsyncWrite     = false         // DefaultAsyncWrite is enable async write or not.
)

// StopMode define the stop mode of server and conn.
type StopMode uint8

const (
	StopImmediately          StopMode = iota // StopImmediately mean stop directly, the cached data maybe will not send.
	StopGracefullyButNotWait                 // StopGracefullyButNotWait stop and flush cached data.
	StopGracefullyAndWait                    // StopGracefullyAndWait stop and block until cached data sended.
)

// Handler is the event callback.
// Note : don't block in event handler.
type Handler interface {
	OnTcpAccept(*Conn)                  // OnAccept mean server accept a new connect.
	OnTcpConnect(*Conn)                 // OnConnect mean client connected to a server.
	OnTcpRecv(*Conn, []byte, time.Time) // OnRecv mean conn recv a packet.
	OnTcpClose(*Conn)                   // OnClose mean conn is closed.
}

// Packet is the unit of data.
type Packet struct {
	Body []byte
}

// Options is the options used for net conn.
type Options struct {
	Handler         Handler
	RecvBufSize     uint          // default is DefaultRecvBufSize if you don't set.
	SendBufListLen  uint          // default is DefaultSendBufListLen if you don't set.
	AsyncWrite      bool          // default is DefaultAsyncWrite  if you don't set.
	NoDelay         bool          // default is true
	KeepAlive       bool          // default is false
	KeepAlivePeriod time.Duration // default is 0, mean use system setting.
	ReadDeadline    time.Duration // default is 0, means Read will not time out.
	WriteDeadline   time.Duration // default is 0, means Write will not time out.
}

// NewOpts create a new options and set some default value.
// will panic if handler or protocol is nil.
// eg: opts := NewOpts().SetSendListLen(len).SetRecvBufInitSize(len)...
func NewOpts(h Handler) *Options {
	if h == nil {
		panic("xtcp.NewOpts: nil handler")
	}
	return &Options{
		Handler:        h,
		RecvBufSize:    DefaultRecvBufSize,
		SendBufListLen: DefaultSendBufListLen,
		AsyncWrite:     DefaultAsyncWrite,
		NoDelay:        true,
		KeepAlive:      false,
	}
}

func (o *Options) SetRecvBufSize(size uint) *Options {
	o.RecvBufSize = size
	return o
}

func (o *Options) SetSendBufListLen(length uint) *Options {
	o.SendBufListLen = length
	return o
}

func (o *Options) SetAsyncWrite(async bool) *Options {
	o.AsyncWrite = async
	return o
}
