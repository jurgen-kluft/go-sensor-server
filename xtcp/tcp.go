package xtcp

import (
	"time"
)

var (
	DefaultRecvBufSize    = 4 << 10 // DefaultRecvBufSize is 4KB, the default size of recv buf.
	DefaultSendBufListLen = 1 << 10 // DefaultSendBufListLen is 1KB, the default length of send buf list.
	DefaultAsyncWrite     = true    // DefaultAsyncWrite is enable async write or not.
)

// StopMode define the stop mode of server and conn.
type StopMode uint8

const (
	StopImmediately          StopMode = iota // StopImmediately mean stop directly, the cached data maybe will not send.
	StopGracefullyButNotWait                 // StopGracefullyButNotWait stop and flush cached data.
	StopGracefullyAndWait                    // StopGracefullyAndWait stop and block until cached data sended.
)

// LogLevel used to filter log message by the Logger.
type LogLevel uint8

const (
	Panic LogLevel = iota
	Fatal
	Error
	Warn
	Info
	Debug
)

// Logger is the log interface
type Logger interface {
	Log(l LogLevel, v ...interface{})
	Logf(l LogLevel, format string, v ...interface{})
}
type emptyLogger struct{}

func (*emptyLogger) Log(l LogLevel, v ...interface{})                 {}
func (*emptyLogger) Logf(l LogLevel, format string, v ...interface{}) {}

var logger Logger = &emptyLogger{}

// SetLogger set the logger
func SetLogger(l Logger) {
	logger = l
}

// Handler is the event callback.
// Note : don't block in event handler.
type Handler interface {
	OnAccept(*Conn)       // OnAccept mean server accept a new connect.
	OnConnect(*Conn)      // OnConnect mean client connected to a server.
	OnRecv(*Conn, Packet) // OnRecv mean conn recv a packet.
	OnClose(*Conn)        // OnClose mean conn is closed.
}

// Packet is the unit of data.
type Packet struct {
	Body []byte
}

// Options is the options used for net conn.
type Options struct {
	Handler         Handler
	RecvBufSize     int           // default is DefaultRecvBufSize if you don't set.
	SendBufListLen  int           // default is DefaultSendBufListLen if you don't set.
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

func (o *Options) SetRecvBufSize(size int) *Options {
	if size > 0 {
		o.RecvBufSize = size
	}
	return o
}

func (o *Options) SetSendBufListLen(length int) *Options {
	if length > 0 {
		o.SendBufListLen = length
	}
	return o
}

func (o *Options) SetAsyncWrite(async bool) *Options {
	o.AsyncWrite = async
	return o
}
