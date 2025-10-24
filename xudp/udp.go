package xudp

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
	OnUdpRecv([]byte) // OnRecv mean conn recv a packet.
}

// Options is the options used for net conn.
type Options struct {
	Handler Handler
}

// NewOpts create a new options and set some default value.
// will panic if handler or protocol is nil.
// eg: opts := NewOpts().SetSendListLen(len).SetRecvBufInitSize(len)...
func NewOpts(h Handler) *Options {
	if h == nil {
		panic("XUDP.NewOpts: nil handler")
	}
	return &Options{
		Handler: h,
	}
}
