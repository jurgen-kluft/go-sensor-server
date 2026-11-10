package sensorserver

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
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
  "devices": [{"mac":"02:00:00:ab:cd:ef","floor":"second","room":"living"}],
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
	data, err := os.ReadFile(filepath.Join(directory, "data",
		"020000abcdef",
		fmt.Sprintf("%04x", SENSOR_ID_TEMPERATURE),
		"00000001.dat"))
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
  "devices": [{"mac":"02:00:00:ab:cd:ef","floor":"second","room":"living"}],
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

func TestServerRestoresCurrentBeforeListenersAndQuarantineReplay(t *testing.T) {
	for _, replay := range []bool{false, true} {
		t.Run(fmt.Sprintf("replay=%v", replay), func(t *testing.T) {
			configPath, config := writeStorageTestConfig(t)
			mac := MACAddress{2, 0, 0, 0xab, 0xcd, 0xef}
			directory := dataStreamDirectory(config.DataRoot, mac, SENSOR_ID_TEMPERATURE)
			if err := os.MkdirAll(directory, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(directory, segmentName(1)), append(encodeTestRecord(100), 1, 2, 3), 0o644); err != nil {
				t.Fatal(err)
			}
			want := SensorData{Timestamp: 100, Value: 100}
			if replay {
				source, err := OpenQuarantineWriter(OSFileSystem{}, DefaultQuarantineOptions(config.QuarantineRoot))
				if err != nil {
					t.Fatal(err)
				}
				message := quarantineTestMessage(t, mac, 10, TransportUDP)
				message.Timestamp = 7
				if err := source.WriteUnknownMessage(context.Background(), message); err != nil {
					t.Fatal(err)
				}
				if err := source.Close(); err != nil {
					t.Fatal(err)
				}
				want = SensorData{Timestamp: 7, Value: 10}
			}
			fileSystem := &recoveryTestFileSystem{}
			var warnings []error
			network := &recoveryTestNetwork{beforeListen: func() {
				if fileSystem.reads != 1 || len(warnings) != 1 {
					t.Errorf("listener started before recovery: reads=%d, warnings=%v", fileSystem.reads, warnings)
				}
			}}
			var observations atomic.Int64
			server, err := OpenServer(context.Background(), configPath, ServerOptions{
				FileSystem: fileSystem, Network: network, Clock: SystemClock{},
				OnWarning:           func(err error) { warnings = append(warnings, err) },
				OnSensorObservation: func(SensorObservation) { observations.Add(1) },
			})
			if err != nil {
				t.Fatal(err)
			}
			defer server.Close()
			if current, ok := server.engine.Current(mac, SENSOR_ID_TEMPERATURE); !ok || current != want {
				t.Fatalf("Current() = %+v, %v, want %+v", current, ok, want)
			}
			if observations.Load() != 0 || server.Counters().Router.RecordsAccepted != 0 {
				t.Fatal("restoration or replay emitted a live observation")
			}
			wantAccepted := uint64(0)
			if replay {
				wantAccepted = 1
			}
			if streams := server.Counters().DataStreams; len(streams) != 1 || streams[0].Counters.Accepted != wantAccepted {
				t.Fatalf("stream counters = %+v", streams)
			}
			if err := server.Close(); err != nil {
				t.Fatal(err)
			}
			for _, file := range fileSystem.files {
				if !file.closed {
					t.Fatal("shutdown leaked a recovered storage file")
				}
			}
		})
	}
}

func TestServerRecoveryFailureClosesPreviouslyRestoredStreams(t *testing.T) {
	configPath, config := writeStorageTestConfig(t, DeviceConfig{MAC: "03:00:00:00:00:00", Floor: "second", Room: "living"})
	for _, mac := range []MACAddress{{2, 0, 0, 0xab, 0xcd, 0xef}, {3}} {
		directory := dataStreamDirectory(config.DataRoot, mac, SENSOR_ID_TEMPERATURE)
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, segmentName(1)), encodeTestRecord(1), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	failure := errors.New("injected final-record read failure")
	fileSystem := &recoveryTestFileSystem{operation: "read", failReadAt: 2, failure: failure}
	network := &recoveryTestNetwork{beforeListen: func() { t.Error("listener started despite recovery failure") }}
	_, err := OpenServer(context.Background(), configPath, ServerOptions{FileSystem: fileSystem, Network: network, Clock: SystemClock{}})
	if !errors.Is(err, failure) {
		t.Fatalf("OpenServer() = %v, want recovery failure", err)
	}
	if fileSystem.reads != 2 {
		t.Fatalf("recovery read %d records, want 2", fileSystem.reads)
	}
	for _, file := range fileSystem.files {
		if !file.closed {
			t.Fatal("failed startup leaked an open storage file")
		}
	}
}

func writeStorageTestConfig(t *testing.T, extraDevices ...DeviceConfig) (string, Config) {
	t.Helper()
	directory := t.TempDir()
	config := loadTestConfig(t, validConfigJSON).Config()
	config.TCPAddress = "127.0.0.1:9000"
	config.UDPAddress = "127.0.0.1:9001"
	config.DataRoot = filepath.Join(directory, "data")
	config.QuarantineRoot = filepath.Join(directory, "quarantine")
	config.Devices = append(config.Devices, extraDevices...)
	encoded, err := json.Marshal(map[string]any{
		"tcp_address": config.TCPAddress, "udp_address": config.UDPAddress,
		"data_root": config.DataRoot, "quarantine_root": config.QuarantineRoot,
		"devices": config.Devices, "sensors": config.Sensors,
	})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "config.json")
	if err := os.WriteFile(path, encoded, 0o644); err != nil {
		t.Fatal(err)
	}
	return path, config
}

type recoveryTestNetwork struct {
	StandardNetwork
	beforeListen func()
}

func (network *recoveryTestNetwork) Listen(transport, _ string) (net.Listener, error) {
	network.beforeListen()
	return network.StandardNetwork.Listen(transport, "127.0.0.1:0")
}

func (network *recoveryTestNetwork) ListenPacket(transport, _ string) (net.PacketConn, error) {
	return network.StandardNetwork.ListenPacket(transport, "127.0.0.1:0")
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
