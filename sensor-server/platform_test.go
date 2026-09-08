package sensorserver

import (
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOSFileSystem(t *testing.T) {
	fileSystem := OSFileSystem{}
	directory := filepath.Join(t.TempDir(), "floor", "sensor")
	if err := fileSystem.MkdirAll(directory, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	path := filepath.Join(directory, "00000001.dat")
	file, err := fileSystem.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}

	if _, err := file.Write([]byte("sensor-data")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := file.Sync(); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if err := file.Truncate(6); err != nil {
		t.Fatalf("Truncate() error = %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	info, err := fileSystem.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Size() != 6 {
		t.Fatalf("Stat().Size() = %d, want 6", info.Size())
	}

	entries, err := fileSystem.ReadDir(directory)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "00000001.dat" {
		t.Fatalf("ReadDir() = %v, want one data segment", entries)
	}

	renamedPath := filepath.Join(directory, "00000002.dat")
	if err := fileSystem.Rename(path, renamedPath); err != nil {
		t.Fatalf("Rename() error = %v", err)
	}
	if err := fileSystem.Remove(renamedPath); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
}

func TestStandardNetworkTCP(t *testing.T) {
	network := StandardNetwork{}
	listener, err := network.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	defer listener.Close()

	accepted := make(chan net.Conn, 1)
	go func() {
		connection, acceptErr := listener.Accept()
		if acceptErr == nil {
			accepted <- connection
		}
	}()

	client, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer client.Close()

	server := <-accepted
	defer server.Close()

	if _, err := client.Write([]byte("ping")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	buffer := make([]byte, 4)
	if _, err := io.ReadFull(server, buffer); err != nil {
		t.Fatalf("ReadFull() error = %v", err)
	}
	if string(buffer) != "ping" {
		t.Fatalf("received %q, want ping", buffer)
	}
}

func TestStandardNetworkUDP(t *testing.T) {
	network := StandardNetwork{}
	packetConnection, err := network.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ListenPacket() error = %v", err)
	}
	defer packetConnection.Close()

	sender, err := net.Dial("udp", packetConnection.LocalAddr().String())
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer sender.Close()

	if _, err := sender.Write([]byte("ping")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	buffer := make([]byte, 4)
	if _, _, err := packetConnection.ReadFrom(buffer); err != nil {
		t.Fatalf("ReadFrom() error = %v", err)
	}
	if string(buffer) != "ping" {
		t.Fatalf("received %q, want ping", buffer)
	}
}

func TestSystemClock(t *testing.T) {
	clock := SystemClock{}
	if clock.Now().IsZero() {
		t.Fatal("Now() returned the zero time")
	}

	timer := clock.NewTimer(time.Hour)
	if !timer.Stop() {
		t.Fatal("Stop() = false for a new timer")
	}

	ticker := clock.NewTicker(time.Hour)
	ticker.Stop()
}
