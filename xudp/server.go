package xudp

import (
	"fmt"
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
	conn    *net.UDPConn
}

// ListenAndServe listens on the TCP network address addr and then
// calls Serve to handle requests on incoming connections.
func (s *Server) ListenAndServe(_udpAddr string) error {
	udpAddr, err := net.ResolveUDPAddr("udp", _udpAddr)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}

	s.Serve(conn)
	return nil
}

// Serve start the tcp server to accept.
func (s *Server) Serve(conn *net.UDPConn) {
	defer s.wg.Done()

	s.wg.Add(1)

	s.mu.Lock()
	s.conn = conn
	s.mu.Unlock()

	s.logger.LogInfo("XUDP - Server listen on: ", conn.LocalAddr().String())

	var buf [1472]byte
	for {
		n, _, err := conn.ReadFromUDP(buf[0:])
		now := time.Now()
		if err != nil {
			fmt.Println(err)
			return
		}

		s.Opts.Handler.OnUdpRecv(buf[0:n], now)

		if !s.IsStopped() {
			s.logger.LogErrorf(err, "XUDP - Server stop error: %v; server closed!")
			s.Stop(StopImmediately)
		}
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
		conn := s.conn
		s.conn = nil
		s.mu.Unlock()

		if conn != nil {
			conn.Close()
		}

		m := mode
		if m == StopGracefullyAndWait {
			// don't wait each conn stop.
			m = StopGracefullyButNotWait
		}

		if mode == StopGracefullyAndWait {
			s.wg.Wait()
		}

		s.logger.LogInfo("XUDP - Server stopped.")
	})
}

// NewServer create a tcp server but not start to accept.
// The opts will set to all accept conns.
func NewServer(opts *Options, logger logging.Logger) *Server {
	s := &Server{
		Opts:    opts,
		logger:  logger,
		stopped: make(chan struct{}),
	}
	return s
}
