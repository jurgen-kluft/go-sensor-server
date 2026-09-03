package sensorserver

import (
	"errors"
	"strings"
	"testing"
	"time"
)

const validConfigJSON = `{
  "tcp_address": ":9000",
  "udp_address": ":9001",
  "data_root": "/tmp/sensors",
  "quarantine_root": "/tmp/quarantine",
  "devices": [{"mac":"02:00:00:ab:cd:ef","area":"1st Living Room"}],
	"sensors": [{"id":1,"type":"Temperature","unit":"Celsius"}],
  "data_stream": {},
  "network": {},
  "logging": {}
}`

func TestLoadConfigAppliesDefaultsAndBuildsLookups(t *testing.T) {
	snapshot, err := LoadConfig(strings.NewReader(validConfigJSON))
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	config := snapshot.Config()
	if config.DataStream.QueueCapacity != 512 || config.DataStream.FlushInterval.Value() != time.Second {
		t.Fatalf("data stream defaults = %+v", config.DataStream)
	}
	if config.Network.MaximumTCPConnections != 100 || config.Network.PayloadDeadline.Value() != 5*time.Second {
		t.Fatalf("network defaults = %+v", config.Network)
	}
	if config.ShutdownDeadline.Value() != 30*time.Second {
		t.Fatalf("shutdown deadline = %v", config.ShutdownDeadline.Value())
	}

	mac, _ := ParseMACAddress("02:00:00:ab:cd:ef")
	device, ok := snapshot.Device(mac)
	if !ok || device.Area != AreaLivingRoom {
		t.Fatalf("Device() = %+v, %v", device, ok)
	}
	sensor, ok := snapshot.Sensor(1)
	if !ok || sensor.Type != SENSOR_ID_TEMPERATURE {
		t.Fatalf("Sensor() = %+v, %v", sensor, ok)
	}
}

func TestConfigSnapshotDoesNotExposeMutableSlices(t *testing.T) {
	snapshot, err := LoadConfig(strings.NewReader(validConfigJSON))
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	config := snapshot.Config()
	config.Devices[0].Area = "Changed"
	mac, _ := ParseMACAddress("02:00:00:ab:cd:ef")
	device, _ := snapshot.Device(mac)
	if device.Area != AreaLivingRoom {
		t.Fatalf("snapshot device area = %q, want 1st Living Room", device.Area)
	}
}

func TestLoadConfigRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		old  string
		new  string
	}{
		{name: "unknown field", old: `"logging": {}`, new: `"unknown": true, "logging": {}`},
		{name: "duplicate MAC", old: `"devices": [`, new: `"devices": [{"mac":"02:00:00:ab:cd:ef","area":"1st Kitchen"},`},
		{name: "unknown area", old: `"area":"1st Living Room"`, new: `"area":"Unknown Room"`},
		{name: "invalid port", old: `":9000"`, new: `":0"`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := strings.Replace(validConfigJSON, test.old, test.new, 1)
			if _, err := LoadConfig(strings.NewReader(input)); err == nil {
				t.Fatal("LoadConfig() error = nil")
			}
		})
	}
}

func TestValidateReloadAllowsOnlyDeviceChanges(t *testing.T) {
	current, err := LoadConfig(strings.NewReader(validConfigJSON))
	if err != nil {
		t.Fatalf("LoadConfig(current) error = %v", err)
	}
	changedDeviceJSON := strings.Replace(validConfigJSON, "1st Living Room", "1st Kitchen", 1)
	changedDevice, err := LoadConfig(strings.NewReader(changedDeviceJSON))
	if err != nil {
		t.Fatalf("LoadConfig(device change) error = %v", err)
	}
	if err := current.ValidateReload(changedDevice); err != nil {
		t.Fatalf("ValidateReload(device change) error = %v", err)
	}

	changedPortJSON := strings.Replace(validConfigJSON, ":9000", ":9100", 1)
	changedPort, err := LoadConfig(strings.NewReader(changedPortJSON))
	if err != nil {
		t.Fatalf("LoadConfig(port change) error = %v", err)
	}
	if err := current.ValidateReload(changedPort); !errors.Is(err, ErrRestartRequired) {
		t.Fatalf("ValidateReload(port change) error = %v, want ErrRestartRequired", err)
	}
}

func TestParseMACAddress(t *testing.T) {
	mac, err := ParseMACAddress("02:00:00:ab:CD:ef")
	if err != nil {
		t.Fatalf("ParseMACAddress() error = %v", err)
	}
	if mac != (MACAddress{0x02, 0, 0, 0xab, 0xcd, 0xef}) {
		t.Fatalf("ParseMACAddress() = %x", mac)
	}
	if _, err := ParseMACAddress("not-a-mac"); err == nil {
		t.Fatal("ParseMACAddress(invalid) error = nil")
	}
}

func FuzzLoadConfig(f *testing.F) {
	f.Add([]byte(validConfigJSON))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"tcp_address":`))
	f.Fuzz(func(t *testing.T, data []byte) {
		snapshot, err := LoadConfig(strings.NewReader(string(data)))
		if err == nil {
			if snapshot == nil {
				t.Fatal("successful LoadConfig returned nil snapshot")
			}
			if _, err := NewConfigSnapshot(snapshot.Config()); err != nil {
				t.Fatalf("successful configuration did not round trip: %v", err)
			}
		}
	})
}
