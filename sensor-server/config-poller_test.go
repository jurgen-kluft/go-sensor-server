package sensorserver

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestConfigPollerPublishesDeviceChange(t *testing.T) {
	path, registry := writePollerConfig(t, validConfigJSON)
	poller, err := NewConfigPoller(OSFileSystem{}, SystemClock{}, registry, path, ConfigPollerOptions{Interval: 15 * time.Second})
	if err != nil {
		t.Fatalf("NewConfigPoller() error = %v", err)
	}

	updated := strings.Replace(validConfigJSON, "LivingRoom", "Kitchen", 1) + "\n"
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	reloaded, err := poller.Poll()
	if err != nil || !reloaded {
		t.Fatalf("Poll() = %v, %v", reloaded, err)
	}
	mac, _ := ParseMACAddress("02:00:00:ab:cd:ef")
	device, _ := registry.Snapshot().Device(mac)
	if device.Area != "Kitchen" {
		t.Fatalf("device area = %q, want Kitchen", device.Area)
	}
}

func TestConfigPollerKeepsSnapshotAfterInvalidReload(t *testing.T) {
	path, registry := writePollerConfig(t, validConfigJSON)
	poller, err := NewConfigPoller(OSFileSystem{}, SystemClock{}, registry, path, ConfigPollerOptions{Interval: 15 * time.Second})
	if err != nil {
		t.Fatalf("NewConfigPoller() error = %v", err)
	}
	invalid := strings.Replace(validConfigJSON, ":9000", ":0", 1) + "\n"
	if err := os.WriteFile(path, []byte(invalid), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if reloaded, err := poller.Poll(); err == nil || reloaded {
		t.Fatalf("Poll() = %v, %v, want rejected reload", reloaded, err)
	}
	mac, _ := ParseMACAddress("02:00:00:ab:cd:ef")
	device, _ := registry.Snapshot().Device(mac)
	if device.Area != "LivingRoom" {
		t.Fatalf("device area = %q, want LivingRoom", device.Area)
	}
	if reloaded, err := poller.Poll(); err != nil || reloaded {
		t.Fatalf("unchanged Poll() = %v, %v", reloaded, err)
	}
}

func TestConfigPollerRejectsRestartRequiredChange(t *testing.T) {
	path, registry := writePollerConfig(t, validConfigJSON)
	poller, err := NewConfigPoller(OSFileSystem{}, SystemClock{}, registry, path, ConfigPollerOptions{Interval: 15 * time.Second})
	if err != nil {
		t.Fatalf("NewConfigPoller() error = %v", err)
	}
	updated := strings.Replace(validConfigJSON, ":9000", ":9100", 1) + "\n"
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := poller.Poll(); !errors.Is(err, ErrRestartRequired) {
		t.Fatalf("Poll() error = %v, want ErrRestartRequired", err)
	}
}

func writePollerConfig(t *testing.T, contents string) (string, *ConfigRegistry) {
	t.Helper()
	directory := t.TempDir()
	path := filepath.Join(directory, "config.json")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	snapshot, err := LoadConfig(strings.NewReader(contents))
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	registry, err := NewConfigRegistry(snapshot)
	if err != nil {
		t.Fatalf("NewConfigRegistry() error = %v", err)
	}
	return path, registry
}
