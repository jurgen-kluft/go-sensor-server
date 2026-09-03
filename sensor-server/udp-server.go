package sensorserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
)

type UDPServerOptions struct {
	WorkerCount   int
	QueueCapacity int
	OnWarning     func(error)
}

type UDPServerCounters struct {
	Datagrams uint64
	Messages  uint64
	Malformed uint64
	Dropped   uint64
}

type udpDatagram struct {
	data   []byte
	remote net.Addr
}

type UDPServer struct {
	connection net.PacketConn
	handler    MessageHandler
	clock      Clock
	options    UDPServerOptions
	ctx        context.Context
	cancel     context.CancelFunc
	queue      chan *UdpPacket
	queueMu    sync.RWMutex
	waitGroup  sync.WaitGroup
	closeOnce  sync.Once
	closeErr   error
	packetPool *sync.Pool
	datagrams  atomic.Uint64
	messages   atomic.Uint64
	malformed  atomic.Uint64
	dropped    atomic.Uint64
}

func NewUDPServer(connection net.PacketConn, handler MessageHandler, clock Clock, options UDPServerOptions) (*UDPServer, error) {
	if connection == nil || handler == nil || clock == nil {
		return nil, errors.New("new UDP server: nil dependency")
	}
	if options.WorkerCount < 1 || options.QueueCapacity < 1 {
		return nil, errors.New("new UDP server: worker count and queue capacity must be positive")
	}
	if options.OnWarning == nil {
		options.OnWarning = func(error) {}
	}
	ctx, cancel := context.WithCancel(context.Background())
	server := &UDPServer{
		connection: connection,
		handler:    handler,
		clock:      clock,
		options:    options,
		ctx:        ctx,
		cancel:     cancel,
		queue:      make(chan *UdpPacket, options.QueueCapacity),
	}
	server.waitGroup.Add(options.WorkerCount)
	for index := 0; index < options.WorkerCount; index++ {
		go server.worker()
	}
	return server, nil
}

// Define your MTU size limit. 1500 bytes is standard for Ethernet.
const UdpMaxPacketSize = 1500

type UdpPacket struct {
	Buf    []byte
	N      int // Keeps track of the actual bytes read
	Remote net.Addr
}

// Serve receives datagrams until Close is called or the packet connection fails.
func (server *UDPServer) Serve() error {
	// Initialize the pool with a New function for when the pool is empty
	server.packetPool = &sync.Pool{
		New: func() any {
			// Allocate exactly what you need for one MTU-sized packet
			return &UdpPacket{Buf: make([]byte, UdpMaxPacketSize)}
		},
	}

	for {
		p := server.packetPool.Get().(*UdpPacket)
		count, remote, err := server.connection.ReadFrom(p.Buf)
		if err != nil {
			server.packetPool.Put(p) // Return the packet to the pool even on error
			if server.ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("read UDP datagram: %w", err)
		}
		p.N = count // Store the actual number of bytes read
		p.Remote = remote
		server.datagrams.Add(1)
		server.queueMu.RLock()
		select {
		case server.queue <- p:
		case <-server.ctx.Done():
			server.queueMu.RUnlock()
			return nil
		default:
			server.dropped.Add(1)
			server.options.OnWarning(fmt.Errorf("drop UDP datagram from %s: processing queue full", remote))
		}
		server.queueMu.RUnlock()
	}
}

func (server *UDPServer) Close() error {
	server.closeOnce.Do(func() {
		server.cancel()
		server.closeErr = server.connection.Close()
		server.queueMu.Lock()
		close(server.queue)
		server.queueMu.Unlock()
		server.waitGroup.Wait()
	})
	if errors.Is(server.closeErr, net.ErrClosed) {
		return nil
	}
	return server.closeErr
}

func (server *UDPServer) Counters() UDPServerCounters {
	return UDPServerCounters{
		Datagrams: server.datagrams.Load(),
		Messages:  server.messages.Load(),
		Malformed: server.malformed.Load(),
		Dropped:   server.dropped.Load(),
	}
}

func (server *UDPServer) worker() {
	defer server.waitGroup.Done()
	for datagram := range server.queue {
		message, err := DecodeDatagram(datagram.Buf[:datagram.N])
		if err != nil {
			server.malformed.Add(1)
			server.options.OnWarning(fmt.Errorf("UDP datagram from %s: %w", datagram.Remote.String(), err))
			server.packetPool.Put(datagram) // Return the packet to the pool on error
			continue
		}
		server.messages.Add(1)
		timestamp := server.clock.Now().UnixMicro()
		if err := server.handler.Route(server.ctx, TransportUDP, timestamp, message); err != nil && !errors.Is(err, ErrUnknownDevice) && !errors.Is(err, ErrUnknownSensor) {
			server.options.OnWarning(fmt.Errorf("route UDP datagram from %s: %w", datagram.Remote.String(), err))
		}
		server.packetPool.Put(datagram) // Return the packet to the pool after processing
	}
}
