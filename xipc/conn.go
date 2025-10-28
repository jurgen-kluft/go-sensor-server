package xipc

import (
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jurgen-kluft/go-sensor-server/logging"
)

const (
	connStateNormal int32 = iota
	connStateStopping
	connStateStopped
)

// A Conn represents the server side of an tcp connection.
type Conn struct {
	sync.Mutex
	Opts         *Options
	UnixSockConn net.Conn
	logger       logging.Logger
	sendChannel  chan []byte // channel for send data
	closed       chan struct{}
	state        int32
	wg           sync.WaitGroup
	once         sync.Once
}

// NewConn returns a new Conn instance.
func NewConn(opts *Options, logger logging.Logger) *Conn {
	c := &Conn{
		Opts:   opts,
		logger: logger,
		closed: make(chan struct{}),
		state:  connStateNormal,
	}

	return c
}

func (c *Conn) String() string {
	return c.UnixSockConn.LocalAddr().String() + " -> " + c.UnixSockConn.RemoteAddr().String()
}

// Stop stops the conn.
func (c *Conn) Stop(mode StopMode) {
	c.once.Do(func() {
		if mode == StopImmediately {
			atomic.StoreInt32(&c.state, connStateStopped)
			c.UnixSockConn.Close()
			close(c.closed)
		} else {
			atomic.StoreInt32(&c.state, connStateStopping)
			close(c.closed)
			if mode == StopGracefullyAndWait {
				c.wg.Wait()
			}
		}
	})
}

// IsStoped return true if Conn is stopped, otherwise return false.
func (c *Conn) IsStopped() bool {
	return atomic.LoadInt32(&c.state) != connStateNormal
}

func (c *Conn) Serve(mode SendReceiveMode) {
	switch mode {
	case ModeReceiveOnly:
		c.recvLoop()
		c.wg.Add(1)
	case ModeSendOnly:
		c.sendLoop()
		c.wg.Add(1)
	case ModeSendAndReceive:
		c.wg.Add(2)
		go c.sendLoop()
		c.recvLoop()
	}
	c.Opts.Handler.OnIpcClose(c)
}

func (c *Conn) sendLoop() {
	defer func() {
		c.logger.LogInfo("XIPC - Conn send-loop exit: ", c.UnixSockConn.RemoteAddr())
		c.wg.Done()
	}()

	for {
		select {
		case data, ok := <-c.sendChannel:
			if !ok {
				return
			}
			_, err := c.UnixSockConn.Write(data)
			c.Opts.Handler.OnIpcSend(c, data)
			if err != nil {
				if !c.IsStopped() {
					c.logger.LogErrorf(err, "XIPC - Conn[%v] send", c.UnixSockConn.RemoteAddr())
					c.Stop(StopImmediately)
				}
				return
			}
		case <-c.closed:
			return
		}
	}
}

func (c *Conn) recvLoop() {
	defer func() {
		c.logger.LogInfo("XIPC - Conn recv-loop exit: ", c.UnixSockConn.RemoteAddr())
		c.wg.Done()
	}()

	buf := make([]byte, 4096)
	for {
		count, err := c.UnixSockConn.Read(buf)
		if err != nil {
			if !c.IsStopped() {
				if err != io.EOF {
					c.logger.LogErrorf(err, "XIPC - Conn[%v] recv", c.UnixSockConn.RemoteAddr())
				}
				c.Stop(StopImmediately)
			}
			return
		}

		c.Opts.Handler.OnIpcRecv(c, buf[0:count], time.Now())
	}
}

// DialAndServe connects to the addr and start serving.
func (c *Conn) DialAndServe(addr string, send, recv bool) error {
	unixSockConn, err := net.Dial("unix", addr)
	if err != nil {
		return err
	}

	c.UnixSockConn = unixSockConn
	c.Opts.Handler.OnIpcConnect(c)

	c.Serve(c.Opts.Mode)
	return nil
}

func (c *Conn) Send(data []byte) error {
	if c.IsStopped() || c.Opts.Mode == ModeReceiveOnly {
		return io.EOF
	}
	c.sendChannel <- data
	return nil
}
