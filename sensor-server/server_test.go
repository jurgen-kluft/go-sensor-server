package sensorserver

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"
)

func TestServerRoutesTCPAndUDPAndDrainsOnShutdown(t *testing.T) {
	tcpListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	udpConnection, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		_ = tcpListener.Close()
		t.Fatalf("ListenPacket() error = %v", err)
	}
	directory := t.TempDir()
	configPath := filepath.Join(directory, "config.json")
	config := fmt.Sprintf(`{
  "tcp_address": %q,
  "udp_address": %q,
  "data_root": %q,
  "quarantine_root": %q,
  "devices": [{"mac":"02:00:00:ab:cd:ef","area":"LivingRoom"}],
  "sensors": [{"id":1,"type":"Temperature","unit":"Celsius"}],
	"data_stream": {"flush_interval":"1m"},
  "network": {},
  "logging": {}
}`, tcpListener.Addr().String(), udpConnection.LocalAddr().String(), filepath.Join(directory, "data"), filepath.Join(directory, "quarantine"))
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	network := &preparedNetwork{tcp: tcpListener, udp: udpConnection}
	server, err := OpenServer(context.Background(), configPath, ServerOptions{Network: network, FileSystem: OSFileSystem{}, Clock: SystemClock{}})
	if err != nil {
		t.Fatalf("OpenServer() error = %v", err)
	}

	tcpClient, err := net.Dial("tcp", tcpListener.Addr().String())
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	if _, err := tcpClient.Write(serverTestMessage(t, 21)); err != nil {
		t.Fatalf("TCP Write() error = %v", err)
	}
	_ = tcpClient.Close()
	udpClient, err := net.Dial("udp", udpConnection.LocalAddr().String())
	if err != nil {
		t.Fatalf("UDP Dial() error = %v", err)
	}
	if _, err := udpClient.Write(serverTestMessage(t, 22)); err != nil {
		t.Fatalf("UDP Write() error = %v", err)
	}
	_ = udpClient.Close()
	waitForAcceptedRecords(t, server, 2)

	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if err := server.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	counters := server.Counters()
	if counters.TCP.Messages != 1 || counters.UDP.Messages != 1 || counters.Router.RecordsAccepted != 2 {
		t.Fatalf("Counters() = %+v", counters)
	}
	if len(counters.DataStreams) != 1 || counters.DataStreams[0].Counters.Written != 2 || counters.DataStreams[0].Counters.BytesWritten != 2*uint64(SensorDataRecordSize) {
		t.Fatalf("data stream counters = %+v", counters.DataStreams)
	}
	data, err := os.ReadFile(filepath.Join(directory, "data", "317374204c6976696e6720526f6f6d", "54656d7065726174757265", "00000001.dat"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if int64(len(data)) != 2*SensorDataRecordSize {
		t.Fatalf("data size = %d, want %d", len(data), 2*SensorDataRecordSize)
	}
	values := []int32{
		int32(binary.LittleEndian.Uint32(data[8:12])),
		int32(binary.LittleEndian.Uint32(data[20:24])),
	}
	sort.Slice(values, func(left, right int) bool { return values[left] < values[right] })
	if values[0] != 21 || values[1] != 22 {
		t.Fatalf("stored values = %v, want [21 22]", values)
	}
}

func TestServerHandles100MixedSenders(t *testing.T) {
	tcpListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	udpConnection, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		_ = tcpListener.Close()
		t.Fatalf("ListenPacket() error = %v", err)
	}
	directory := t.TempDir()
	configPath := filepath.Join(directory, "config.json")
	config := fmt.Sprintf(`{
  "tcp_address": %q, "udp_address": %q,
  "data_root": %q, "quarantine_root": %q,
  "devices": [{"mac":"02:00:00:ab:cd:ef","area":"LivingRoom"}],
  "sensors": [{"id":1,"type":"Temperature","unit":"Celsius"}],
  "data_stream": {"queue_capacity":512}, "network": {}, "logging": {}
}`, tcpListener.Addr().String(), udpConnection.LocalAddr().String(), filepath.Join(directory, "data"), filepath.Join(directory, "quarantine"))
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	server, err := OpenServer(context.Background(), configPath, ServerOptions{
		Network: &preparedNetwork{tcp: tcpListener, udp: udpConnection}, FileSystem: OSFileSystem{}, Clock: SystemClock{},
	})
	if err != nil {
		t.Fatalf("OpenServer() error = %v", err)
	}
	defer server.Close()

	var waitGroup sync.WaitGroup
	senderErrors := make(chan error, 100)
	for index := 0; index < 100; index++ {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			transport, address := "udp", udpConnection.LocalAddr().String()
			if index%2 == 0 {
				transport, address = "tcp", tcpListener.Addr().String()
			}
			connection, err := net.Dial(transport, address)
			if err != nil {
				senderErrors <- err
				return
			}
			defer connection.Close()
			message, err := encodeServerTestMessage(int16(index))
			if err == nil {
				_, err = connection.Write(message)
			}
			if err != nil {
				senderErrors <- err
			}
		}(index)
	}
	waitGroup.Wait()
	close(senderErrors)
	for err := range senderErrors {
		t.Errorf("sender error = %v", err)
	}
	waitForAcceptedRecords(t, server, 100)
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	counters := server.Counters()
	if counters.TCP.Messages != 50 || counters.UDP.Messages != 50 || counters.Router.RecordsAccepted != 100 {
		t.Fatalf("Counters() = %+v", counters)
	}
}

func TestServerPropagatesListenerFailure(t *testing.T) {
	listenerError := errors.New("listener failed")
	udpConnection, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ListenPacket() error = %v", err)
	}
	directory := t.TempDir()
	configPath := filepath.Join(directory, "config.json")
	config := fmt.Sprintf(`{
  "tcp_address": ":9000", "udp_address": %q,
  "data_root": %q, "quarantine_root": %q,
  "devices": [], "sensors": [], "data_stream": {}, "network": {}, "logging": {}
}`, udpConnection.LocalAddr().String(), filepath.Join(directory, "data"), filepath.Join(directory, "quarantine"))
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	server, err := OpenServer(context.Background(), configPath, ServerOptions{
		Network: &preparedNetwork{tcp: failingListener{err: listenerError}, udp: udpConnection}, FileSystem: OSFileSystem{}, Clock: SystemClock{},
	})
	if err != nil {
		t.Fatalf("OpenServer() error = %v", err)
	}
	select {
	case <-server.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("server did not stop after listener failure")
	}
	if err := server.Close(); !errors.Is(err, listenerError) {
		t.Fatalf("Close() error = %v, want listener error", err)
	}
}

type preparedNetwork struct {
	tcp net.Listener
	udp net.PacketConn
}

type failingListener struct{ err error }

func (listener failingListener) Accept() (net.Conn, error) { return nil, listener.err }
func (listener failingListener) Close() error              { return nil }
func (listener failingListener) Addr() net.Addr            { return testAddress("tcp") }

type testAddress string

func (address testAddress) Network() string { return string(address) }
func (address testAddress) String() string  { return string(address) }

func (network *preparedNetwork) Listen(string, string) (net.Listener, error) {
	return network.tcp, nil
}

func (network *preparedNetwork) ListenPacket(string, string) (net.PacketConn, error) {
	return network.udp, nil
}

func serverTestMessage(t *testing.T, value int16) []byte {
	t.Helper()
	message, err := encodeServerTestMessage(value)
	if err != nil {
		t.Fatalf("EncodeMessage() error = %v", err)
	}
	return message
}

func encodeServerTestMessage(value int16) ([]byte, error) {
	payload := make([]byte, SensorRecordSize)
	encodedValue := int32(value)
	payload[0] = 1
	payload[1] = byte(encodedValue)
	payload[2] = byte(encodedValue >> 8)
	payload[3] = byte(encodedValue >> 16)
	return EncodeMessage(MessageTypeSensorData, MACAddress{0x02, 0x00, 0x00, 0xab, 0xcd, 0xef}, payload)
}

func waitForAcceptedRecords(t *testing.T, server *Server, count uint64) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if server.Router().Counters().RecordsAccepted >= count {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("accepted records = %d, want %d", server.Router().Counters().RecordsAccepted, count)
}
