package xtcp

import (
	"net"
	"sync"
	"time"

	"github.com/jurgen-kluft/go-sensor-server/logging"
)

// Server used for running a tcp server.
type Server struct {
	Opts    *Options
	logger  logging.Logger
	stopped chan struct{}
	wg      sync.WaitGroup
	mu      sync.Mutex
	once    sync.Once
	lis     net.Listener
	conns   map[*Conn]bool
}

// ListenAndServe listens on the TCP network address addr and then
// calls Serve to handle requests on incoming connections.
func (s *Server) ListenAndServe(addr string) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	s.Serve(l)

	return nil
}

// Serve start the tcp server to accept.
func (s *Server) Serve(l net.Listener) {
	defer s.wg.Done()

	s.wg.Add(1)

	s.mu.Lock()
	s.lis = l
	s.mu.Unlock()

	s.logger.LogInfof("XTCP - Server listen on: %s", l.Addr().String())

	var tempDelay time.Duration // how long to sleep on accept failure
	maxDelay := 1 * time.Second

	for {
		conn, err := l.Accept()
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if tempDelay > maxDelay {
					tempDelay = maxDelay
				}
				s.logger.LogErrorf(err, "XTCP - Server Accept, retrying in %v", tempDelay)
				select {
				case <-time.After(tempDelay):
					continue
				case <-s.stopped:
					return
				}
			}

			if !s.IsStopped() {
				s.logger.LogError(err, "XTCP - Server Accept, server closed!")
				s.Stop(StopImmediately)
			}

			return
		}

		tempDelay = 0
		go s.handleRawConn(conn)
	}
}

// IsStopped check if server is stopped.
func (s *Server) IsStopped() bool {
	select {
	case <-s.stopped:
		return true
	default:
		return false
	}
}

// Stop stops the tcp server.
// StopImmediately: immediately closes all open connections and listener.
// StopGracefullyButNotWait: stops the server and stop all connections gracefully.
// StopGracefullyAndWait: stops the server and blocks until all connections are stopped gracefully.
func (s *Server) Stop(mode StopMode) {
	s.once.Do(func() {
		close(s.stopped)

		s.mu.Lock()
		lis := s.lis
		s.lis = nil
		conns := s.conns
		s.conns = nil
		s.mu.Unlock()

		if lis != nil {
			lis.Close()
		}

		m := mode
		if m == StopGracefullyAndWait {
			// don't wait each conn stop.
			m = StopGracefullyButNotWait
		}
		for c := range conns {
			c.Stop(m)
		}

		if mode == StopGracefullyAndWait {
			s.wg.Wait()
		}

		s.logger.LogInfo("XTCP - Server stopped.")
	})
}

func (s *Server) handleRawConn(conn net.Conn) {
	s.mu.Lock()
	if s.conns == nil { // s.conns == nil mean server stopped
		s.mu.Unlock()
		conn.Close()
		return
	}
	s.mu.Unlock()

	tcpConn := NewConn(s.Opts, s.logger)
	tcpConn.RawConn = conn

	if !s.addConn(tcpConn) {
		tcpConn.Stop(StopImmediately)
		return
	}

	s.wg.Add(1)
	defer func() {
		s.removeConn(tcpConn)
		s.wg.Done()
	}()

	s.Opts.Handler.OnTcpAccept(tcpConn)

	tcpConn.serve()
}

func (s *Server) addConn(conn *Conn) bool {
	s.mu.Lock()
	if s.conns == nil {
		s.mu.Unlock()
		return false
	}
	s.conns[conn] = true
	s.mu.Unlock()
	return true
}

func (s *Server) removeConn(conn *Conn) {
	s.mu.Lock()
	if s.conns != nil {
		delete(s.conns, conn)
	}
	s.mu.Unlock()
}

// CurClientCount return current client count.
func (s *Server) CurClientCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.conns)
}

// NewServer create a tcp server but not start to accept.
// The opts will set to all accept conns.
func NewServer(opts *Options, logger logging.Logger) *Server {
	if opts.RecvBufSize == 0 {
		opts.RecvBufSize = DefaultRecvBufSize
	}
	if opts.SendBufListLen == 0 {
		opts.SendBufListLen = DefaultSendBufListLen
	}
	s := &Server{
		Opts:    opts,
		logger:  logger,
		stopped: make(chan struct{}),
		conns:   make(map[*Conn]bool),
	}
	return s
}
