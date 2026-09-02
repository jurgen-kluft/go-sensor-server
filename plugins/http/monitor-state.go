package httpplugin

import (
	"fmt"
	"sort"
	"sync"

	sensorserver "github.com/jurgen-kluft/go-sensor-server/sensor-server"
)

type SensorSnapshot struct {
	ID          uint16   `json:"id"`
	Type        string   `json:"type"`
	Unit        string   `json:"unit"`
	Seen        bool     `json:"seen"`
	Current     *Sample  `json:"current,omitempty"`
	Warning     string   `json:"warning,omitempty"`
	SampleCount int      `json:"sample_count"`
	History     []Sample `json:"history,omitempty"`
}

type DeviceSnapshot struct {
	MAC             string           `json:"mac"`
	Area            string           `json:"area"`
	Transport       string           `json:"transport"`
	Connected       bool             `json:"connected"`
	LastSeenAt      int64            `json:"last_seen_at_us,omitempty"`
	ObservedSensors int              `json:"observed_sensor_count"`
	Warnings        []string         `json:"warnings"`
	Sensors         []SensorSnapshot `json:"sensors,omitempty"`
}

type RoomSnapshot struct {
	Name             string `json:"name"`
	DeviceCount      int    `json:"device_count"`
	ConnectedDevices int    `json:"connected_device_count"`
	WarningCount     int    `json:"warning_count"`
}

type SensorObservationEvent struct {
	MAC        string `json:"mac"`
	Area       string `json:"area"`
	Transport  string `json:"transport"`
	SensorType string `json:"sensor_type"`
	Unit       string `json:"unit_type"`
	Sample     Sample `json:"sample"`
	Warning    string `json:"warning,omitempty"`
}

type DeviceConnectionEvent struct {
	MAC          string `json:"mac"`
	ConnectionID uint64 `json:"connection_id"`
	Connected    bool   `json:"connected"`
}

type sensorState struct {
	definition sensorserver.SensorDefinition
	seen       bool
	current    Sample
	warning    string
	history    *sampleRing
}

type deviceState struct {
	mac          sensorserver.MACAddress
	area         string
	transport    sensorserver.Transport
	connectionID uint64
	connected    bool
	lastSeenAt   int64
	sensors      map[sensorserver.SensorType]*sensorState
}

type MonitoringState struct {
	mu              sync.RWMutex
	historyCapacity int
	thresholds      map[sensorserver.SensorType]Threshold
	devices         map[sensorserver.MACAddress]*deviceState
	events          *eventHub
}

func NewMonitoringState(config Config) (*MonitoringState, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	thresholds := make(map[sensorserver.SensorType]Threshold, len(config.Thresholds))
	for _, threshold := range config.Thresholds {
		thresholds[threshold.SensorType] = threshold
	}
	return &MonitoringState{
		historyCapacity: config.HistoryCapacity,
		thresholds:      thresholds,
		devices:         make(map[sensorserver.MACAddress]*deviceState),
		events:          newEventHub(),
	}, nil
}

func (state *MonitoringState) OnSensorObservation(observation sensorserver.SensorObservation) {
	state.mu.Lock()
	device := state.device(observation.MAC, observation.Area.String())
	device.transport = observation.Transport
	device.lastSeenAt = observation.Timestamp
	sensor := device.sensors[observation.SensorType]
	if sensor == nil {
		sensor = &sensorState{
			definition: sensorserver.SensorDefinition{Type: observation.SensorType, Unit: observation.UnitType},
			history:    newSampleRing(state.historyCapacity),
		}
		device.sensors[observation.SensorType] = sensor
	}
	sensor.seen = true
	sensor.current = Sample{Timestamp: observation.Timestamp, Value: observation.Value}
	sensor.history.Append(sensor.current)
	sensor.warning = state.thresholdWarning(sensorserver.SensorType(observation.SensorType), observation.Value)
	event := SensorObservationEvent{
		MAC: formatMAC(observation.MAC), Area: observation.Area.String(),
		Transport: transportName(observation.Transport), SensorType: observation.SensorType.String(),
		Unit:   observation.UnitType.String(),
		Sample: sensor.current, Warning: sensor.warning,
	}
	state.mu.Unlock()
	state.events.Publish("sensor_observation", event)
}

func (state *MonitoringState) OnDeviceConnected(connectionID uint64, mac sensorserver.MACAddress) {
	state.mu.Lock()
	device := state.device(mac, "")
	changed := !device.connected
	device.connectionID = connectionID
	device.connected = true
	device.transport = sensorserver.TransportTCP
	state.mu.Unlock()
	if changed {
		state.events.Publish("device_connected", DeviceConnectionEvent{
			MAC: formatMAC(mac), ConnectionID: connectionID, Connected: true,
		})
	}
}

func (state *MonitoringState) OnDeviceDisconnected(connectionID uint64, mac sensorserver.MACAddress) {
	state.mu.Lock()
	device := state.devices[mac]
	changed := device != nil && device.connected && device.connectionID == connectionID
	if changed {
		device.connected = false
		device.connectionID = 0
	}
	state.mu.Unlock()
	if changed {
		state.events.Publish("device_disconnected", DeviceConnectionEvent{
			MAC: formatMAC(mac), ConnectionID: connectionID, Connected: false,
		})
	}
}

func (state *MonitoringState) Subscribe() (<-chan Event, func()) {
	return state.events.Subscribe()
}

func (state *MonitoringState) Rooms(snapshot *sensorserver.ConfigSnapshot) []RoomSnapshot {
	devices := state.Devices(snapshot)
	rooms := make(map[string]*RoomSnapshot)
	for _, device := range devices {
		room := rooms[device.Area]
		if room == nil {
			room = &RoomSnapshot{Name: device.Area}
			rooms[device.Area] = room
		}
		room.DeviceCount++
		if device.Connected {
			room.ConnectedDevices++
		}
		room.WarningCount += len(device.Warnings)
	}
	result := make([]RoomSnapshot, 0, len(rooms))
	for _, room := range rooms {
		result = append(result, *room)
	}
	sort.Slice(result, func(left, right int) bool { return result[left].Name < result[right].Name })
	return result
}

func (state *MonitoringState) Devices(snapshot *sensorserver.ConfigSnapshot) []DeviceSnapshot {
	state.mu.RLock()
	defer state.mu.RUnlock()
	configured := snapshot.Config()
	result := make([]DeviceSnapshot, 0, len(configured.Devices))
	for _, configuredDevice := range configured.Devices {
		mac, _ := sensorserver.ParseMACAddress(configuredDevice.MAC)
		result = append(result, state.snapshotDevice(mac, configuredDevice.Area, false))
	}
	sort.Slice(result, func(left, right int) bool { return result[left].MAC < result[right].MAC })
	return result
}

func (state *MonitoringState) Device(snapshot *sensorserver.ConfigSnapshot, mac sensorserver.MACAddress) (DeviceSnapshot, bool) {
	configuredDevice, exists := snapshot.Device(mac)
	if !exists {
		return DeviceSnapshot{}, false
	}
	state.mu.RLock()
	defer state.mu.RUnlock()
	areaName := configuredDevice.Area.String()
	return state.snapshotDevice(mac, areaName, true), true
}

func (state *MonitoringState) Readings(mac sensorserver.MACAddress, sensorID sensorserver.SensorType) ([]Sample, bool) {
	state.mu.RLock()
	defer state.mu.RUnlock()
	device := state.devices[mac]
	if device == nil {
		return nil, false
	}
	sensor := device.sensors[sensorID]
	if sensor == nil {
		return nil, false
	}
	return sensor.history.Snapshot(), true
}

func (state *MonitoringState) device(mac sensorserver.MACAddress, area string) *deviceState {
	device := state.devices[mac]
	if device == nil {
		device = &deviceState{mac: mac, sensors: make(map[sensorserver.SensorType]*sensorState)}
		state.devices[mac] = device
	}
	if area != "" {
		device.area = area
	}
	return device
}

func (state *MonitoringState) thresholdWarning(sensorType sensorserver.SensorType, value int16) string {
	threshold, exists := state.thresholds[sensorType]
	if !exists {
		return ""
	}
	if threshold.Minimum != nil && value < *threshold.Minimum {
		return fmt.Sprintf("%s value %d is below minimum %d", sensorType, value, *threshold.Minimum)
	}
	if threshold.Maximum != nil && value > *threshold.Maximum {
		return fmt.Sprintf("%s value %d is above maximum %d", sensorType, value, *threshold.Maximum)
	}
	return ""
}

func (state *MonitoringState) snapshotDevice(mac sensorserver.MACAddress, area string, includeSensors bool) DeviceSnapshot {
	result := DeviceSnapshot{MAC: formatMAC(mac), Area: area, Warnings: []string{}}
	device := state.devices[mac]
	if device != nil {
		result.Transport = transportName(device.transport)
		result.Connected = device.connected
		result.LastSeenAt = device.lastSeenAt
		for _, sensor := range device.sensors {
			if sensor.seen {
				result.ObservedSensors++
			}
			if sensor.warning != "" {
				result.Warnings = append(result.Warnings, sensor.warning)
			}
		}
	}
	sort.Strings(result.Warnings)
	if !includeSensors {
		return result
	}
	for sensorType, sensorTypeName := range sensorserver.SensorTypeNames {
		unitString := sensorserver.UnitForSensorType(sensorType).String()
		sensorSnapshot := SensorSnapshot{ID: uint16(sensorType), Type: sensorTypeName, Unit: unitString}
		if device != nil {
			if sensor := device.sensors[sensorType]; sensor != nil {
				sensorSnapshot.Seen = sensor.seen
				sensorSnapshot.Warning = sensor.warning
				sensorSnapshot.SampleCount = sensor.history.count
				if sensor.seen {
					current := sensor.current
					sensorSnapshot.Current = &current
				}
			}
		}
		result.Sensors = append(result.Sensors, sensorSnapshot)
	}
	sort.Slice(result.Sensors, func(left, right int) bool { return result.Sensors[left].ID < result.Sensors[right].ID })
	return result
}

func formatMAC(mac sensorserver.MACAddress) string {
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", mac[0], mac[1], mac[2], mac[3], mac[4], mac[5])
}

func transportName(transport sensorserver.Transport) string {
	switch transport {
	case sensorserver.TransportTCP:
		return "tcp"
	case sensorserver.TransportUDP:
		return "udp"
	default:
		return "unknown"
	}
}
