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
	if write.area != Area1stLivingRoom || write.sensorType != SENSOR_ID_TEMPERATURE || write.timestamp != 1234 || write.value != 215 {
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
		MAC: mac, Area: Area1stLivingRoom, Transport: TransportUDP,
		SensorType: SENSOR_ID_TEMPERATURE, UnitType: UCelcius,
		Timestamp: 1234, Value: 215,
	}
	if observations[0] != want {
		t.Fatalf("observation = %+v, want %+v", observations[0], want)
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
			if !ok || (device.Area != Area1stLivingRoom && device.Area != Area1stKitchen) {
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
	area       AreaType
	sensorType SensorType
	timestamp  int64
	value      int16
}

type recordingDataWriter struct {
	writes []dataWrite
}

func (writer *recordingDataWriter) WriteSensorData(_ context.Context, area AreaType, sensorType SensorType, timestamp int64, value int16) error {
	writer.writes = append(writer.writes, dataWrite{area: area, sensorType: sensorType, timestamp: timestamp, value: value})
	return nil
}

type recordingUnknownWriter struct {
	messages []UnknownMessage
}

func (writer *recordingUnknownWriter) WriteUnknownMessage(_ context.Context, message UnknownMessage) error {
	writer.messages = append(writer.messages, message)
	return nil
}
