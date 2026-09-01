package sensorserver

import (
	"net"
	"testing"
	"time"
)

func TestUDPServerRoutesDatagram(t *testing.T) {
	server, handler, sender := newLoopbackUDPServer(t)
	defer sender.Close()
	serveDone := serveUDP(t, server)

	if _, err := sender.Write(encodedSensorMessage(t, MACAddress{1, 2, 3, 4, 5, 6}, 42)); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	handler.waitFor(t, 1)
	if err := server.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := <-serveDone; err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	if handler.messages[0].transport != TransportUDP || handler.messages[0].message.Sensors[0].Value != 42 {
		t.Fatalf("message = %+v", handler.messages[0])
	}
}

func TestUDPServerContinuesAfterMalformedDatagram(t *testing.T) {
	server, handler, sender := newLoopbackUDPServer(t)
	defer sender.Close()
	serveDone := serveUDP(t, server)

	if _, err := sender.Write([]byte{1, 2, 3}); err != nil {
		t.Fatalf("Write(malformed) error = %v", err)
	}
	if _, err := sender.Write(encodedSensorMessage(t, MACAddress{1}, 7)); err != nil {
		t.Fatalf("Write(valid) error = %v", err)
	}
	handler.waitFor(t, 1)
	if err := server.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := <-serveDone; err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	if server.Counters().Malformed != 1 || server.Counters().Messages != 1 {
		t.Fatalf("Counters() = %+v", server.Counters())
	}
}

func TestUDPServerRejectsTrailingBytes(t *testing.T) {
	server, handler, sender := newLoopbackUDPServer(t)
	defer sender.Close()
	serveDone := serveUDP(t, server)
	message := append(encodedSensorMessage(t, MACAddress{1}, 7), 0)

	if _, err := sender.Write(message); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	time.Sleep(20 * time.Millisecond)
	if err := server.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := <-serveDone; err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	if len(handler.messages) != 0 || server.Counters().Malformed != 1 {
		t.Fatalf("messages = %d, counters = %+v", len(handler.messages), server.Counters())
	}
}

func newLoopbackUDPServer(t *testing.T) (*UDPServer, *recordingMessageHandler, net.Conn) {
	t.Helper()
	connection, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ListenPacket() error = %v", err)
	}
	handler := &recordingMessageHandler{notify: make(chan struct{}, 16)}
	server, err := NewUDPServer(connection, handler, SystemClock{}, UDPServerOptions{WorkerCount: 2, QueueCapacity: 8})
	if err != nil {
		t.Fatalf("NewUDPServer() error = %v", err)
	}
	sender, err := net.Dial("udp", connection.LocalAddr().String())
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	return server, handler, sender
}

func serveUDP(t *testing.T, server *UDPServer) <-chan error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- server.Serve() }()
	return done
}
