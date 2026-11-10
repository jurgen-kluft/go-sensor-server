package sensorserver

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

func TestMessageRouterRoutesKnownDeviceRecords(t *testing.T) {
	router, dataWriter, _ := newTestRouter(t)
	message := testRoutedMessage(MACAddress{0x02, 0, 0, 0xab, 0xcd, 0xef}, []SensorRecord{{SensorType: SENSOR_ID_TEMPERATURE, Value: 215}})

	if err := router.Route(context.Background(), TransportTCP, 1234, message); err != nil {
		t.Fatalf("Route() error = %v", err)
	}
	if len(dataWriter.writes) != 1 {
		t.Fatalf("writes = %d, want 1", len(dataWriter.writes))
	}
	write := dataWriter.writes[0]
	if write.mac != message.Header.MAC || write.sensorType != SENSOR_ID_TEMPERATURE || write.timestamp != 1234 || write.value != 215 {
		t.Fatalf("write = %+v", write)
	}
}

func TestMessageRouterObservesOnlyKnownDeviceAndSensorRecords(t *testing.T) {
	router, _, _ := newTestRouter(t)
	var observations []SensorObservation
	router.onSensorObservation = func(observation SensorObservation) {
		observations = append(observations, observation)
	}
	mac := MACAddress{0x02, 0, 0, 0xab, 0xcd, 0xef}
	message := testRoutedMessage(mac, []SensorRecord{{SensorType: SENSOR_ID_UNKNOWN, Value: 1}, {SensorType: SENSOR_ID_TEMPERATURE, Value: 215}})

	_ = router.Route(context.Background(), TransportUDP, 1234, message)

	if len(observations) != 1 {
		t.Fatalf("observations = %d, want 1", len(observations))
	}
	want := SensorObservation{
		MAC: mac, Floor: FLOOR_SECOND, Room: ROOM_LIVING, Transport: TransportUDP,
		SensorType: SENSOR_ID_TEMPERATURE, UnitType: UCelcius,
		Timestamp: 1234, Value: 215,
	}
	if observations[0] != want {
		t.Fatalf("observation = %+v, want %+v", observations[0], want)
	}
}

func TestMessageRouterSeparatesDevicesAndPreservesStreamAcrossLocationReload(t *testing.T) {
	snapshot := loadTestConfig(t, validConfigJSON)
	config := snapshot.Config()
	config.Devices = append(config.Devices, DeviceConfig{MAC: "02:00:00:ab:cd:ee", Floor: config.Devices[0].Floor, Room: config.Devices[0].Room})
	snapshot, err := NewConfigSnapshot(config)
	if err != nil {
		t.Fatal(err)
	}
	registry, err := NewConfigRegistry(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	engine, err := NewDataEngine(func(MACAddress, SensorType) (*DataStream, error) {
		return newTestDataStream(t, &recordingWriter{}, 4)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	router, err := NewMessageRouter(registry, engine, &recordingUnknownWriter{})
	if err != nil {
		t.Fatal(err)
	}
	first := MACAddress{2, 0, 0, 0xab, 0xcd, 0xef}
	second := MACAddress{2, 0, 0, 0xab, 0xcd, 0xee}
	for index, mac := range []MACAddress{first, second} {
		if err := router.Route(context.Background(), TransportTCP, 1, testRoutedMessage(mac, []SensorRecord{{SensorType: SENSOR_ID_TEMPERATURE, Value: int32(index)}})); err != nil {
			t.Fatal(err)
		}
	}
	config.Devices[0].Room = "kitchen"
	reloaded, err := NewConfigSnapshot(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Reload(reloaded); err != nil {
		t.Fatal(err)
	}
	if err := router.Route(context.Background(), TransportUDP, 2, testRoutedMessage(first, []SensorRecord{{SensorType: SENSOR_ID_TEMPERATURE, Value: 3}})); err != nil {
		t.Fatal(err)
	}
	if engine.StreamCount() != 2 {
		t.Fatalf("stream count after location reload = %d, want 2", engine.StreamCount())
	}
	if data, ok := engine.Current(first, SENSOR_ID_TEMPERATURE); !ok || data.Value != 3 {
		t.Fatalf("first device current = %+v, %v", data, ok)
	}
	if data, ok := engine.Current(second, SENSOR_ID_TEMPERATURE); !ok || data.Value != 1 {
		t.Fatalf("second device current = %+v, %v", data, ok)
	}
}

func TestMessageRouterQuarantinesUnknownDevice(t *testing.T) {
	router, dataWriter, unknownWriter := newTestRouter(t)
	message := testRoutedMessage(MACAddress{1, 2, 3, 4, 5, 6}, []SensorRecord{{SensorType: 1, Value: 2}})

	err := router.Route(context.Background(), TransportUDP, 99, message)
	if !errors.Is(err, ErrUnknownDevice) {
		t.Fatalf("Route() error = %v, want ErrUnknownDevice", err)
	}
	if len(dataWriter.writes) != 0 || len(unknownWriter.messages) != 1 {
		t.Fatalf("data writes = %d, quarantined = %d", len(dataWriter.writes), len(unknownWriter.messages))
	}
	unknown := unknownWriter.messages[0]
	if unknown.Timestamp != 99 || unknown.Transport != TransportUDP || unknown.Message.Header.MAC != message.Header.MAC {
		t.Fatalf("unknown message = %+v", unknown)
	}
}

func TestMessageRouterRejectsOnlyUnknownSensorRecord(t *testing.T) {
	router, dataWriter, _ := newTestRouter(t)
	message := testRoutedMessage(MACAddress{0x02, 0, 0, 0xab, 0xcd, 0xef}, []SensorRecord{
		{SensorType: SENSOR_ID_ENERGY, Value: 1},
		{SensorType: SENSOR_ID_TEMPERATURE, Value: 2},
	})

	err := router.Route(context.Background(), TransportTCP, 1, message)
	if !errors.Is(err, ErrUnknownSensor) {
		t.Fatalf("Route() error = %v, want ErrUnknownSensor", err)
	}
	if len(dataWriter.writes) != 1 || dataWriter.writes[0].value != 2 {
		t.Fatalf("writes = %+v", dataWriter.writes)
	}
	counters := router.Counters()
	if counters.RecordsAccepted != 1 || counters.RecordsRejected != 1 || counters.UnknownSensors != 1 {
		t.Fatalf("counters = %+v", counters)
	}
}

func TestConfigRegistryReloadIsAtomicForReaders(t *testing.T) {
	current := loadTestConfig(t, validConfigJSON)
	registry, err := NewConfigRegistry(current)
	if err != nil {
		t.Fatalf("NewConfigRegistry() error = %v", err)
	}
	nextJSON := strings.Replace(validConfigJSON, "LivingRoom", "Kitchen", 1)
	next := loadTestConfig(t, nextJSON)

	mac, _ := ParseMACAddress("02:00:00:ab:cd:ef")
	var waitGroup sync.WaitGroup
	for index := 0; index < 100; index++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			device, ok := registry.Snapshot().Device(mac)
			if !ok || (device.Floor != FLOOR_SECOND && device.Room != ROOM_LIVING && device.Room != ROOM_KITCHEN) {
				t.Errorf("Device() = %+v, %v", device, ok)
			}
		}()
	}
	if err := registry.Reload(next); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}
	waitGroup.Wait()
}

func newTestRouter(t *testing.T) (*MessageRouter, *recordingDataWriter, *recordingUnknownWriter) {
	t.Helper()
	snapshot := loadTestConfig(t, validConfigJSON)
	registry, err := NewConfigRegistry(snapshot)
	if err != nil {
		t.Fatalf("NewConfigRegistry() error = %v", err)
	}
	dataWriter := &recordingDataWriter{}
	unknownWriter := &recordingUnknownWriter{}
	router, err := NewMessageRouter(registry, dataWriter, unknownWriter)
	if err != nil {
		t.Fatalf("NewMessageRouter() error = %v", err)
	}
	return router, dataWriter, unknownWriter
}

func loadTestConfig(t *testing.T, value string) *ConfigSnapshot {
	t.Helper()
	snapshot, err := LoadConfig(strings.NewReader(value))
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	return snapshot
}

func testRoutedMessage(mac MACAddress, sensors []SensorRecord) Message {
	return Message{Header: MessageHeader{Magic: MessageMagic, Type: MessageTypeSensorData, MAC: mac}, Sensors: sensors}
}

type dataWrite struct {
	mac        MACAddress
	sensorType SensorType
	timestamp  int64
	value      int32
}

type recordingDataWriter struct {
	writes []dataWrite
}

func (writer *recordingDataWriter) WriteSensorData(_ context.Context, mac MACAddress, sensorType SensorType, timestamp int64, value int32) error {
	writer.writes = append(writer.writes, dataWrite{mac: mac, sensorType: sensorType, timestamp: timestamp, value: value})
	return nil
}

type recordingUnknownWriter struct {
	messages []UnknownMessage
}

func (writer *recordingUnknownWriter) WriteUnknownMessage(_ context.Context, message UnknownMessage) error {
	writer.messages = append(writer.messages, message)
	return nil
}
