package sensorserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

var ErrTCPServerClosed = errors.New("TCP server is closed")

type MessageHandler interface {
	Route(ctx context.Context, transport Transport, timestamp int64, message Message) error
}

type TCPServerOptions struct {
	MaximumConnections int
	PayloadDeadline    time.Duration
	OnWarning          func(error)
	OnConnection       func(connectionID uint64, remoteAddress string)
	OnDisconnection    func(connectionID uint64, mac MACAddress, remoteAddress string)
}

type TCPServerCounters struct {
	ActiveConnections uint64
	TotalConnections  uint64
	Rejected          uint64
	Messages          uint64
	Malformed         uint64
	MACMismatches     uint64
}

type TCPServer struct {
	listener net.Listener
	handler  MessageHandler
	clock    Clock
	options  TCPServerOptions
	ctx      context.Context
	cancel   context.CancelFunc
	limit    chan struct{}

	mu          sync.Mutex
	connections map[uint64]*tcpConnection
	byMAC       map[MACAddress]*tcpConnection
	nextID      atomic.Uint64
	waitGroup   sync.WaitGroup
	closeOnce   sync.Once
	closeErr    error

	active      atomic.Uint64
	total       atomic.Uint64
	rejected    atomic.Uint64
	messages    atomic.Uint64
	malformed   atomic.Uint64
	macMismatch atomic.Uint64
}

type tcpConnection struct {
	id     uint64
	conn   net.Conn
	remote string
	mac    MACAddress
	bound  bool
}

func NewTCPServer(listener net.Listener, handler MessageHandler, clock Clock, options TCPServerOptions) (*TCPServer, error) {
	if listener == nil || handler == nil || clock == nil {
		return nil, errors.New("new TCP server: nil dependency")
	}
	if options.MaximumConnections < 1 {
		return nil, errors.New("new TCP server: maximum connections must be positive")
	}
	if options.PayloadDeadline < time.Second || options.PayloadDeadline > time.Minute {
		return nil, errors.New("new TCP server: payload deadline must be between 1s and 1m")
	}
	if options.OnWarning == nil {
		options.OnWarning = func(error) {}
	}
	if options.OnConnection == nil {
		options.OnConnection = func(uint64, string) {}
	}
	if options.OnDisconnection == nil {
		options.OnDisconnection = func(uint64, MACAddress, string) {}
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &TCPServer{
		listener:    listener,
		handler:     handler,
		clock:       clock,
		options:     options,
		ctx:         ctx,
		cancel:      cancel,
		limit:       make(chan struct{}, options.MaximumConnections),
		connections: make(map[uint64]*tcpConnection),
		byMAC:       make(map[MACAddress]*tcpConnection),
	}, nil
}

// Serve accepts connections until Close is called or the listener fails.
func (server *TCPServer) Serve() error {
	for {
		connection, err := server.listener.Accept()
		if err != nil {
			if server.ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("accept TCP connection: %w", err)
		}
		if !server.startConnection(connection) {
			server.rejected.Add(1)
			server.options.OnWarning(fmt.Errorf("reject TCP connection from %s: connection limit reached", connection.RemoteAddr()))
			_ = connection.Close()
		}
	}
}

func (server *TCPServer) Close() error {
	server.closeOnce.Do(func() {
		server.cancel()
		server.closeErr = server.listener.Close()
		server.mu.Lock()
		connections := make([]net.Conn, 0, len(server.connections))
		for _, connection := range server.connections {
			connections = append(connections, connection.conn)
		}
		server.mu.Unlock()
		for _, connection := range connections {
			_ = connection.Close()
		}
		server.waitGroup.Wait()
	})
	if errors.Is(server.closeErr, net.ErrClosed) {
		return nil
	}
	return server.closeErr
}

func (server *TCPServer) Counters() TCPServerCounters {
	return TCPServerCounters{
		ActiveConnections: server.active.Load(),
		TotalConnections:  server.total.Load(),
		Rejected:          server.rejected.Load(),
		Messages:          server.messages.Load(),
		Malformed:         server.malformed.Load(),
		MACMismatches:     server.macMismatch.Load(),
	}
}

func (server *TCPServer) startConnection(connection net.Conn) bool {
	select {
	case server.limit <- struct{}{}:
	default:
		return false
	}
	id := server.nextID.Add(1)
	tracked := &tcpConnection{id: id, conn: connection, remote: connection.RemoteAddr().String()}
	server.mu.Lock()
	server.connections[id] = tracked
	server.mu.Unlock()
	server.active.Add(1)
	server.total.Add(1)
	server.options.OnConnection(id, tracked.remote)
	server.waitGroup.Add(1)
	go server.serveConnection(tracked)
	return true
}

func (server *TCPServer) serveConnection(connection *tcpConnection) {
	defer server.waitGroup.Done()
	defer func() {
		_ = connection.conn.Close()
		server.removeConnection(connection)
		<-server.limit
		server.active.Add(^uint64(0))
		server.options.OnDisconnection(connection.id, connection.mac, connection.remote)
	}()

	for {
		message, structural, err := server.readMessage(connection.conn)
		if err != nil {
			if server.ctx.Err() != nil || errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return
			}
			server.malformed.Add(1)
			server.options.OnWarning(fmt.Errorf("TCP connection %d from %s: %w", connection.id, connection.remote, err))
			if structural {
				return
			}
			continue
		}

		if !connection.bound {
			server.bindConnection(connection, message.Header.MAC)
		} else if connection.mac != message.Header.MAC {
			server.macMismatch.Add(1)
			server.options.OnWarning(fmt.Errorf("TCP connection %d MAC mismatch: bound %x, got %x", connection.id, connection.mac, message.Header.MAC))
			continue
		}

		server.messages.Add(1)
		timestamp := server.clock.Now().UnixMicro()
		if err := server.handler.Route(server.ctx, TransportTCP, timestamp, message); err != nil && !errors.Is(err, ErrUnknownDevice) && !errors.Is(err, ErrUnknownSensor) {
			server.options.OnWarning(fmt.Errorf("route TCP connection %d: %w", connection.id, err))
		}
	}
}

// readMessage returns structural=true when the next frame boundary is unknown.
func (server *TCPServer) readMessage(connection net.Conn) (Message, bool, error) {
	headerBytes := make([]byte, MessageHeaderSize)
	if _, err := io.ReadFull(connection, headerBytes); err != nil {
		return Message{}, true, fmt.Errorf("read header: %w", err)
	}
	header, err := DecodeHeader(headerBytes)
	if err != nil {
		return Message{}, true, err
	}
	if header.PayloadLength > 0 {
		if err := connection.SetReadDeadline(server.clock.Now().Add(server.options.PayloadDeadline)); err != nil {
			return Message{}, true, fmt.Errorf("set payload deadline: %w", err)
		}
	}
	payload := make([]byte, header.PayloadLength)
	_, readErr := io.ReadFull(connection, payload)
	_ = connection.SetReadDeadline(time.Time{})
	if readErr != nil {
		return Message{}, true, fmt.Errorf("read payload: %w", readErr)
	}
	message, err := DecodeMessage(header, payload)
	if err != nil {
		return Message{}, false, err
	}
	return message, false, nil
}

func (server *TCPServer) bindConnection(connection *tcpConnection, mac MACAddress) {
	server.mu.Lock()
	previous := server.byMAC[mac]
	connection.mac = mac
	connection.bound = true
	server.byMAC[mac] = connection
	server.mu.Unlock()
	if previous != nil && previous != connection {
		_ = previous.conn.Close()
	}
}

func (server *TCPServer) removeConnection(connection *tcpConnection) {
	server.mu.Lock()
	delete(server.connections, connection.id)
	if server.byMAC[connection.mac] == connection {
		delete(server.byMAC, connection.mac)
	}
	server.mu.Unlock()
}
