package httpplugin

import (
	"strings"
	"testing"

	sensorserver "github.com/jurgen-kluft/go-sensor-server/sensor-server"
)

func TestMonitoringStateKeepsBoundedHistoryAndClearsThreshold(t *testing.T) {
	state, err := NewMonitoringState(Config{Address: ":8080", HistoryCapacity: 2})
	if err != nil {
		t.Fatalf("NewMonitoringState() error = %v", err)
	}
	mac := sensorserver.MACAddress{2, 0, 0, 0xab, 0xcd, 0xef}
	for index, value := range []int32{90, 110, 95} {
		state.OnSensorObservation(sensorserver.SensorObservation{
			MAC: mac, Floor: sensorserver.FLOOR_SECOND, Room: sensorserver.ROOM_LIVING, Transport: sensorserver.TransportUDP,
			SensorType: sensorserver.SENSOR_ID_TEMPERATURE, UnitType: sensorserver.UCelcius,
			Timestamp: int64(index + 1), Value: value,
		})
	}
	readings, ok := state.Readings(mac, 1)
	if !ok || len(readings) != 2 || readings[0].Value != 110 || readings[1].Value != 95 {
		t.Fatalf("Readings() = %+v, %v", readings, ok)
	}
	device, ok := state.Device(testSnapshot(t), mac)
	if !ok || len(device.Warnings) != 0 || device.Transport != "udp" || device.Connected {
		t.Fatalf("Device() = %+v, %v", device, ok)
	}
}

func TestMonitoringStateIgnoresReplacedTCPDisconnect(t *testing.T) {
	state, err := NewMonitoringState(Config{Address: ":8080", HistoryCapacity: 1})
	if err != nil {
		t.Fatalf("NewMonitoringState() error = %v", err)
	}
	mac := sensorserver.MACAddress{2, 0, 0, 0xab, 0xcd, 0xef}
	state.OnDeviceConnected(1, mac)
	state.OnDeviceConnected(2, mac)
	state.OnDeviceDisconnected(1, mac)
	device, _ := state.Device(testSnapshot(t), mac)
	if !device.Connected {
		t.Fatal("replacement device reported disconnected")
	}
	state.OnDeviceDisconnected(2, mac)
	device, _ = state.Device(testSnapshot(t), mac)
	if device.Connected {
		t.Fatal("current device reported connected after disconnect")
	}
}

func TestMonitoringStatePublishesSensorAndEffectiveConnectionEvents(t *testing.T) {
	state, err := NewMonitoringState(Config{Address: ":8080", HistoryCapacity: 1})
	if err != nil {
		t.Fatalf("NewMonitoringState() error = %v", err)
	}
	events, unsubscribe := state.Subscribe()
	defer unsubscribe()
	mac := sensorserver.MACAddress{2, 0, 0, 0xab, 0xcd, 0xef}

	state.OnDeviceConnected(1, mac)
	state.OnDeviceConnected(2, mac)
	state.OnDeviceDisconnected(1, mac)
	state.OnSensorObservation(sensorserver.SensorObservation{
		MAC: mac, Floor: sensorserver.FLOOR_SECOND, Room: sensorserver.ROOM_LIVING, Transport: sensorserver.TransportTCP,
		SensorType: sensorserver.SENSOR_ID_TEMPERATURE, UnitType: sensorserver.UCelcius, Timestamp: 10, Value: 20,
	})
	state.OnDeviceDisconnected(2, mac)

	want := []string{"device_connected", "sensor_observation", "device_disconnected"}
	for index, name := range want {
		select {
		case event := <-events:
			if event.Name != name {
				t.Fatalf("event %d name = %q, want %q", index, event.Name, name)
			}
		default:
			t.Fatalf("event %d missing", index)
		}
	}
	select {
	case event := <-events:
		t.Fatalf("unexpected event = %+v", event)
	default:
	}
}

func testSnapshot(t *testing.T) *sensorserver.ConfigSnapshot {
	t.Helper()
	const config = `{
		"tcp_address": ":9000", "udp_address": ":9001",
		"data_root": "data", "quarantine_root": "quarantine",
		"devices": [{"mac":"02:00:00:ab:cd:ef", "area":"1st Living Room"}],
		"sensors": [{"id":1, "type":"Temperature", "unit":"Celsius"}],
		"data_stream": {}, "network": {}, "logging": {}, "shutdown_deadline":"30s"
	}`
	snapshot, err := sensorserver.LoadConfig(strings.NewReader(config))
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	return snapshot
}
