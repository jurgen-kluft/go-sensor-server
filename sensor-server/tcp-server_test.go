package sensorserver

import (
	"context"
	"encoding/binary"
	"errors"
	"net"
	"sync"
	"testing"
	"time"
)

func TestTCPServerRoutesLongLivedConnection(t *testing.T) {
	server, handler := newPipeTCPServer(t)
	client, serverConnection := net.Pipe()
	server.startConnection(serverConnection)

	first := encodedSensorMessage(t, MACAddress{1, 2, 3, 4, 5, 6}, 10)
	second := encodedSensorMessage(t, MACAddress{1, 2, 3, 4, 5, 6}, 20)
	go func() {
		_, _ = client.Write(append(first, second...))
		_ = client.Close()
	}()

	handler.waitFor(t, 2)
	if err := server.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if handler.messages[0].message.Sensors[0].Value != 10 || handler.messages[1].message.Sensors[0].Value != 20 {
		t.Fatalf("messages = %+v", handler.messages)
	}
}

func TestTCPServerDropsMACMismatchAndKeepsConnection(t *testing.T) {
	server, handler := newPipeTCPServer(t)
	client, serverConnection := net.Pipe()
	server.startConnection(serverConnection)

	mac := MACAddress{1, 2, 3, 4, 5, 6}
	other := MACAddress{6, 5, 4, 3, 2, 1}
	stream := append(encodedSensorMessage(t, mac, 1), encodedSensorMessage(t, other, 2)...)
	stream = append(stream, encodedSensorMessage(t, mac, 3)...)
	go func() {
		_, _ = client.Write(stream)
		_ = client.Close()
	}()

	handler.waitFor(t, 2)
	if err := server.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if handler.messages[0].message.Sensors[0].Value != 1 || handler.messages[1].message.Sensors[0].Value != 3 {
		t.Fatalf("messages = %+v", handler.messages)
	}
	if server.Counters().MACMismatches != 1 {
		t.Fatalf("MACMismatches = %d, want 1", server.Counters().MACMismatches)
	}
}

func TestTCPServerReplacesOlderConnectionForMAC(t *testing.T) {
	server, handler := newPipeTCPServer(t)
	firstClient, firstServer := net.Pipe()
	secondClient, secondServer := net.Pipe()
	server.startConnection(firstServer)
	server.startConnection(secondServer)
	mac := MACAddress{1, 2, 3, 4, 5, 6}

	go func() { _, _ = firstClient.Write(encodedSensorMessage(t, mac, 1)) }()
	handler.waitFor(t, 1)
	go func() { _, _ = secondClient.Write(encodedSensorMessage(t, mac, 2)) }()
	handler.waitFor(t, 2)

	_ = firstClient.SetWriteDeadline(time.Now().Add(100 * time.Millisecond))
	_, err := firstClient.Write(encodedSensorMessage(t, mac, 3))
	if err == nil {
		t.Fatal("older connection remained writable")
	}
	_ = firstClient.Close()
	_ = secondClient.Close()
	if err := server.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestTCPServerReportsMACBoundDeviceLifecycle(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	handler := &recordingMessageHandler{notify: make(chan struct{}, 16)}
	connected := make(chan MACAddress, 1)
	disconnected := make(chan MACAddress, 1)
	server, err := NewTCPServer(listener, handler, SystemClock{}, TCPServerOptions{
		MaximumConnections:   8,
		PayloadDeadline:      time.Second,
		OnDeviceConnected:    func(_ uint64, mac MACAddress) { connected <- mac },
		OnDeviceDisconnected: func(_ uint64, mac MACAddress) { disconnected <- mac },
	})
	if err != nil {
		t.Fatalf("NewTCPServer() error = %v", err)
	}
	client, serverConnection := net.Pipe()
	server.startConnection(serverConnection)
	mac := MACAddress{1, 2, 3, 4, 5, 6}
	go func() {
		_, _ = client.Write(encodedSensorMessage(t, mac, 10))
		_ = client.Close()
	}()

	handler.waitFor(t, 1)
	if got := <-connected; got != mac {
		t.Fatalf("connected MAC = %x, want %x", got, mac)
	}
	if err := server.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if got := <-disconnected; got != mac {
		t.Fatalf("disconnected MAC = %x, want %x", got, mac)
	}
}

func TestTCPServerContinuesAfterBadChecksum(t *testing.T) {
	server, handler := newPipeTCPServer(t)
	client, serverConnection := net.Pipe()
	server.startConnection(serverConnection)
	mac := MACAddress{1, 2, 3, 4, 5, 6}
	bad := encodedSensorMessage(t, mac, 1)
	bad[MessageHeaderSize] ^= 0xff
	stream := append(bad, encodedSensorMessage(t, mac, 2)...)
	go func() {
		_, _ = client.Write(stream)
		_ = client.Close()
	}()

	handler.waitFor(t, 1)
	if err := server.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if handler.messages[0].message.Sensors[0].Value != 2 || server.Counters().Malformed != 1 {
		t.Fatalf("messages = %+v, counters = %+v", handler.messages, server.Counters())
	}
}

func newPipeTCPServer(t *testing.T) (*TCPServer, *recordingMessageHandler) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	handler := &recordingMessageHandler{notify: make(chan struct{}, 16)}
	server, err := NewTCPServer(listener, handler, SystemClock{}, TCPServerOptions{
		MaximumConnections: 8,
		PayloadDeadline:    time.Second,
	})
	if err != nil {
		t.Fatalf("NewTCPServer() error = %v", err)
	}
	return server, handler
}

func encodedSensorMessage(t *testing.T, mac MACAddress, value int16) []byte {
	t.Helper()
	payload := make([]byte, SensorRecordSize)
	binary.LittleEndian.PutUint16(payload[0:2], 1)
	binary.LittleEndian.PutUint16(payload[2:4], uint16(value))
	encoded, err := EncodeMessage(MessageTypeSensorData, mac, payload)
	if err != nil {
		t.Fatalf("EncodeMessage() error = %v", err)
	}
	return encoded
}

type handledMessage struct {
	transport Transport
	timestamp int64
	message   Message
}

type recordingMessageHandler struct {
	mu       sync.Mutex
	messages []handledMessage
	notify   chan struct{}
}

func (handler *recordingMessageHandler) Route(_ context.Context, transport Transport, timestamp int64, message Message) error {
	handler.mu.Lock()
	handler.messages = append(handler.messages, handledMessage{transport: transport, timestamp: timestamp, message: message})
	handler.mu.Unlock()
	handler.notify <- struct{}{}
	return nil
}

func (handler *recordingMessageHandler) waitFor(t *testing.T, count int) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		handler.mu.Lock()
		current := len(handler.messages)
		handler.mu.Unlock()
		if current >= count {
			return
		}
		select {
		case <-handler.notify:
		case <-deadline:
			t.Fatalf("received %d messages, want %d", current, count)
		}
	}
}

var _ = errors.Is
