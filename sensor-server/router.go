package sensorserver

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
)

var (
	ErrUnknownDevice = errors.New("unknown device")
	ErrUnknownSensor = errors.New("unknown sensor")
)

type Transport uint8

const (
	TransportUnknown Transport = iota
	TransportTCP
	TransportUDP
)

type ConfigRegistry struct {
	snapshot atomic.Pointer[ConfigSnapshot]
}

func NewConfigRegistry(snapshot *ConfigSnapshot) (*ConfigRegistry, error) {
	if snapshot == nil {
		return nil, errors.New("new config registry: nil snapshot")
	}
	registry := &ConfigRegistry{}
	registry.snapshot.Store(snapshot)
	return registry, nil
}

func (registry *ConfigRegistry) Snapshot() *ConfigSnapshot {
	return registry.snapshot.Load()
}

func (registry *ConfigRegistry) Reload(candidate *ConfigSnapshot) error {
	current := registry.snapshot.Load()
	if err := current.ValidateReload(candidate); err != nil {
		return err
	}
	registry.snapshot.Store(candidate)
	return nil
}

type SensorDataWriter interface {
	WriteSensorData(ctx context.Context, area AreaType, sensorType SensorType, timestamp int64, value int32) error
}

type UnknownMessage struct {
	Timestamp int64
	Transport Transport
	Message   Message
}

type UnknownMessageWriter interface {
	WriteUnknownMessage(ctx context.Context, message UnknownMessage) error
}

type SensorObservation struct {
	MAC        MACAddress
	Area       AreaType
	Transport  Transport
	SensorType SensorType
	UnitType   UnitType
	Timestamp  int64
	Value      int32
}

type RouterCounters struct {
	Messages           uint64
	Keepalives         uint64
	RecordsAccepted    uint64
	RecordsRejected    uint64
	UnknownDevices     uint64
	UnknownSensors     uint64
	QuarantineFailures uint64
	DataEngineFailures uint64
}

type MessageRouter struct {
	registry            *ConfigRegistry
	dataWriter          SensorDataWriter
	unknown             UnknownMessageWriter
	onSensorObservation func(SensorObservation)

	messages           atomic.Uint64
	keepalives         atomic.Uint64
	recordsAccepted    atomic.Uint64
	recordsRejected    atomic.Uint64
	unknownDevices     atomic.Uint64
	unknownSensors     atomic.Uint64
	quarantineFailures atomic.Uint64
	dataEngineFailures atomic.Uint64
}

func NewMessageRouter(registry *ConfigRegistry, dataWriter SensorDataWriter, unknown UnknownMessageWriter) (*MessageRouter, error) {
	if registry == nil || registry.Snapshot() == nil {
		return nil, errors.New("new message router: nil config registry")
	}
	if dataWriter == nil {
		return nil, errors.New("new message router: nil data writer")
	}
	if unknown == nil {
		return nil, errors.New("new message router: nil unknown message writer")
	}
	return &MessageRouter{registry: registry, dataWriter: dataWriter, unknown: unknown}, nil
}

func (router *MessageRouter) Route(ctx context.Context, transport Transport, timestamp int64, message Message) error {
	router.messages.Add(1)
	if len(message.Sensors) == 0 {
		router.keepalives.Add(1)
	}

	snapshot := router.registry.Snapshot()
	device, known := snapshot.Device(message.Header.MAC)
	if !known {
		router.unknownDevices.Add(1)
		if err := router.unknown.WriteUnknownMessage(ctx, UnknownMessage{
			Timestamp: timestamp,
			Transport: transport,
			Message:   cloneMessage(message),
		}); err != nil {
			router.quarantineFailures.Add(1)
			return fmt.Errorf("quarantine message from %x: %w", message.Header.MAC, err)
		}
		return ErrUnknownDevice
	}

	var result error
	for _, record := range message.Sensors {
		sensor, exists := snapshot.Sensor(record.SensorType)
		if !exists {
			router.unknownSensors.Add(1)
			router.recordsRejected.Add(1)
			result = errors.Join(result, fmt.Errorf("sensor ID %d: %w", record.SensorType, ErrUnknownSensor))
			continue
		}
		if router.onSensorObservation != nil {
			router.onSensorObservation(SensorObservation{
				MAC: message.Header.MAC, Area: device.Area, Transport: transport,
				SensorType: record.SensorType, UnitType: sensor.Unit,
				Timestamp: timestamp, Value: record.Value,
			})
		}
		if err := router.dataWriter.WriteSensorData(ctx, device.Area, sensor.Type, timestamp, record.Value); err != nil {
			router.dataEngineFailures.Add(1)
			router.recordsRejected.Add(1)
			result = errors.Join(result, fmt.Errorf("write %s/%s sensor %d: %w", device.Area, sensor.Type, record.SensorType, err))
			continue
		}
		router.recordsAccepted.Add(1)
	}
	return result
}

func (router *MessageRouter) Counters() RouterCounters {
	return RouterCounters{
		Messages:           router.messages.Load(),
		Keepalives:         router.keepalives.Load(),
		RecordsAccepted:    router.recordsAccepted.Load(),
		RecordsRejected:    router.recordsRejected.Load(),
		UnknownDevices:     router.unknownDevices.Load(),
		UnknownSensors:     router.unknownSensors.Load(),
		QuarantineFailures: router.quarantineFailures.Load(),
		DataEngineFailures: router.dataEngineFailures.Load(),
	}
}

func cloneMessage(message Message) Message {
	message.Payload = append([]byte(nil), message.Payload...)
	message.Sensors = append([]SensorRecord(nil), message.Sensors...)
	return message
}
