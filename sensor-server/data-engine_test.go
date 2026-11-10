package sensorserver

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDataEngineCreatesOneStreamForConcurrentWrites(t *testing.T) {
	var creations atomic.Int64
	writers := make(chan *recordingWriter, 1)
	engine, err := NewDataEngine(func(mac MACAddress, sensorType SensorType) (*DataStream, error) {
		creations.Add(1)
		writer := &recordingWriter{}
		writers <- writer
		return newTestDataStream(t, writer, 128)
	})
	if err != nil {
		t.Fatalf("NewDataEngine() error = %v", err)
	}

	const writes = 100
	stopReader := make(chan struct{})
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		for {
			select {
			case <-stopReader:
				return
			default:
				if current, ok := engine.Current(MACAddress{2}, SENSOR_ID_TEMPERATURE); ok && current.Timestamp != int64(current.Value) {
					t.Errorf("incoherent Current() = %+v", current)
					return
				}
			}
		}
	}()
	defer func() {
		close(stopReader)
		<-readerDone
	}()
	var waitGroup sync.WaitGroup
	for index := 0; index < writes; index++ {
		waitGroup.Add(1)
		go func(value int) {
			defer waitGroup.Done()
			if err := engine.WriteSensorData(context.Background(), MACAddress{2}, SENSOR_ID_TEMPERATURE, int64(value), int32(value)); err != nil {
				t.Errorf("WriteSensorData() error = %v", err)
			}
		}(index)
	}
	waitGroup.Wait()
	if err := engine.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	writer := <-writers
	if creations.Load() != 1 || engine.StreamCount() != 1 {
		t.Fatalf("creations = %d, streams = %d", creations.Load(), engine.StreamCount())
	}
	if len(writer.records) != writes {
		t.Fatalf("written records = %d, want %d", len(writer.records), writes)
	}
	if current, ok := engine.Current(MACAddress{2}, SENSOR_ID_TEMPERATURE); !ok || current != writer.records[writes-1] {
		t.Fatalf("Current() = %+v, %v, want %+v", current, ok, writer.records[writes-1])
	}
}

func TestDataEngineSeparatesStreamKeys(t *testing.T) {
	engine, err := NewDataEngine(func(mac MACAddress, sensorType SensorType) (*DataStream, error) {
		return newTestDataStream(t, &recordingWriter{}, 4)
	})
	if err != nil {
		t.Fatalf("NewDataEngine() error = %v", err)
	}
	defer engine.Close()

	if _, ok := engine.Current(MACAddress{2}, SENSOR_ID_TEMPERATURE); ok || engine.StreamCount() != 0 {
		t.Fatal("Current() created an absent stream")
	}
	for _, mac := range []MACAddress{{3}, {2}} {
		for _, sensor := range []SensorType{SENSOR_ID_HUMIDITY, SENSOR_ID_TEMPERATURE} {
			if err := engine.WriteSensorData(context.Background(), mac, sensor, int64(mac[0]), int32(sensor)); err != nil {
				t.Fatalf("WriteSensorData(%x, %v) error = %v", mac, sensor, err)
			}
			if current, ok := engine.Current(mac, sensor); !ok || current != (SensorData{Timestamp: int64(mac[0]), Value: int32(sensor)}) {
				t.Fatalf("Current(%x, %v) = %+v, %v", mac, sensor, current, ok)
			}
		}
	}
	if engine.StreamCount() != 4 {
		t.Fatalf("StreamCount() = %d, want 4", engine.StreamCount())
	}
	snapshots := engine.Counters()
	for index, snapshot := range snapshots {
		if snapshot.MAC != (MACAddress{byte(2 + index/2)}) || snapshot.SensorType != SensorType(1+index%2) {
			t.Fatalf("unsorted snapshots: %+v", snapshots)
		}
	}
}

func TestNewFileDataStreamFactoryUsesEncodedStreamPath(t *testing.T) {
	directory := t.TempDir()
	factory := NewFileDataStreamFactory(OSFileSystem{}, SystemClock{}, directory, DataStreamConfig{
		QueueCapacity:   1,
		EnqueueWait:     Duration(time.Second),
		RetryCount:      1,
		InitialBackoff:  Duration(time.Millisecond),
		MaximumBackoff:  Duration(time.Millisecond),
		WriteBufferSize: DefaultWriteBufferSize,
		FlushInterval:   Duration(time.Minute),
		RotationSize:    DefaultRotationSize,
	}, nil)
	stream, err := factory(MACAddress{2, 0, 0, 0xab, 0xcd, 0xef}, SENSOR_ID_TEMPERATURE)
	if err != nil {
		t.Fatalf("factory() error = %v", err)
	}
	if err := stream.Write(context.Background(), SensorData{Timestamp: 1, Value: 2}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(directory,
		"020000abcdef",
		"54656d7065726174757265",
		"00000001.dat"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if int64(len(data)) != SensorDataRecordSize {
		t.Fatalf("data size = %d, want %d", len(data), SensorDataRecordSize)
	}
	restored, err := factory(MACAddress{2, 0, 0, 0xab, 0xcd, 0xef}, SENSOR_ID_TEMPERATURE)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	if current, ok := restored.Current(); !ok || current != (SensorData{Timestamp: 1, Value: 2}) {
		t.Fatalf("restored Current() = %+v, %v", current, ok)
	}
	if counters := restored.Counters(); counters.Accepted != 0 || counters.Written != 0 || counters.BytesWritten != 0 {
		t.Fatalf("restoration counted as ingestion: %+v", counters)
	}
}

func TestDataEngineRestoresOnlyExistingConfiguredStreams(t *testing.T) {
	directory := t.TempDir()
	snapshot := loadTestConfig(t, validConfigJSON)
	config := snapshot.Config()
	engine, err := NewDataEngine(NewFileDataStreamFactory(OSFileSystem{}, SystemClock{}, directory, config.DataStream, nil))
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	if err := engine.RestoreStreams(OSFileSystem{}, directory, snapshot); err != nil {
		t.Fatal(err)
	}
	if engine.StreamCount() != 0 {
		t.Fatal("restoration created an inactive stream")
	}
	if entries, err := os.ReadDir(directory); err != nil || len(entries) != 0 {
		t.Fatalf("restoration created files: %v, %v", entries, err)
	}
	mac := MACAddress{2, 0, 0, 0xab, 0xcd, 0xef}
	other := MACAddress{3}
	for _, key := range []dataStreamKey{
		{mac: mac, sensorType: SENSOR_ID_TEMPERATURE},
		{mac: mac, sensorType: SENSOR_ID_HUMIDITY},
		{mac: other, sensorType: SENSOR_ID_TEMPERATURE},
	} {
		path := dataStreamDirectory(directory, key.mac, key.sensorType)
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, segmentName(1)), encodeTestRecord(42), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := engine.RestoreStreams(OSFileSystem{}, directory, snapshot); err != nil {
		t.Fatal(err)
	}
	if engine.StreamCount() != 1 {
		t.Fatalf("restored %d streams, want 1", engine.StreamCount())
	}
	if current, ok := engine.Current(mac, SENSOR_ID_TEMPERATURE); !ok || current != (SensorData{Timestamp: 42, Value: 42}) {
		t.Fatalf("Current() = %+v, %v", current, ok)
	}
	for _, key := range []dataStreamKey{{mac: mac, sensorType: SENSOR_ID_HUMIDITY}, {mac: other, sensorType: SENSOR_ID_TEMPERATURE}} {
		if _, ok := engine.Current(key.mac, key.sensorType); ok {
			t.Fatalf("unconfigured stream %v was restored", key)
		}
		data, err := os.ReadFile(filepath.Join(dataStreamDirectory(directory, key.mac, key.sensorType), segmentName(1)))
		if err != nil || string(data) != string(encodeTestRecord(42)) {
			t.Fatalf("unconfigured history changed: %x, %v", data, err)
		}
	}
}

func TestDataEngineRejectsWritesAfterClose(t *testing.T) {
	engine, err := NewDataEngine(func(mac MACAddress, sensorType SensorType) (*DataStream, error) {
		return newTestDataStream(t, &recordingWriter{}, 1)
	})
	if err != nil {
		t.Fatalf("NewDataEngine() error = %v", err)
	}
	if err := engine.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := engine.WriteSensorData(context.Background(), MACAddress{2}, SENSOR_ID_TEMPERATURE, 1, 1); !errors.Is(err, ErrDataEngineClosed) {
		t.Fatalf("WriteSensorData() error = %v, want ErrDataEngineClosed", err)
	}
}

func TestDataStreamConcurrentWriteAndClose(t *testing.T) {
	stream, err := newTestDataStream(t, &recordingWriter{}, 32)
	if err != nil {
		t.Fatalf("newTestDataStream() error = %v", err)
	}

	const writers = 50
	start := make(chan struct{})
	var waitGroup sync.WaitGroup
	for index := 0; index < writers; index++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			<-start
			err := stream.Write(context.Background(), SensorData{Timestamp: 1, Value: 1})
			if err != nil && !errors.Is(err, ErrDataStreamClosed) && !errors.Is(err, ErrDataStreamFull) {
				t.Errorf("Write() error = %v", err)
			}
		}()
	}
	close(start)
	if err := stream.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	waitGroup.Wait()
}

func TestDataStreamRetriesAndDrainsOnClose(t *testing.T) {
	writer := &recordingWriter{failures: 2}
	stream, err := NewDataStream(writer, SystemClock{}, DataStreamOptions{
		QueueCapacity:  4,
		EnqueueWait:    time.Second,
		FlushInterval:  time.Hour,
		RetryCount:     2,
		InitialBackoff: time.Millisecond,
		MaximumBackoff: time.Millisecond,
		Retryable:      func(error) bool { return true },
	})
	if err != nil {
		t.Fatalf("NewDataStream() error = %v", err)
	}
	if err := stream.Write(context.Background(), SensorData{Timestamp: 1, Value: 2}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	counters := stream.Counters()
	if counters.Written != 1 || counters.Retries != 2 || len(writer.records) != 1 {
		t.Fatalf("counters = %+v, records = %d", counters, len(writer.records))
	}
	if writer.flushes != 1 || writer.closes != 1 {
		t.Fatalf("flushes = %d, closes = %d", writer.flushes, writer.closes)
	}
}

func TestDataStreamCurrentTracksAcceptedQueueOrder(t *testing.T) {
	writer := &blockingRecordWriter{started: make(chan struct{}), release: make(chan struct{})}
	stream, err := NewDataStream(writer, SystemClock{}, DataStreamOptions{
		QueueCapacity: 1, EnqueueWait: 20 * time.Millisecond, FlushInterval: time.Hour,
		InitialBackoff: time.Millisecond, MaximumBackoff: time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	defer close(writer.release)
	if current, ok := stream.Current(); ok {
		t.Fatalf("empty Current() = %+v, true", current)
	}
	if err := stream.Write(context.Background(), SensorData{}); err != nil {
		t.Fatal(err)
	}
	<-writer.started
	if current, ok := stream.Current(); !ok || current != (SensorData{}) {
		t.Fatalf("zero Current() = %+v, %v", current, ok)
	}
	want := SensorData{Timestamp: -1, Value: -5}
	if err := stream.Write(context.Background(), want); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := stream.Write(ctx, SensorData{Timestamp: 10, Value: 10}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled Write() = %v", err)
	}
	if err := stream.Write(context.Background(), SensorData{Timestamp: 20, Value: 20}); !errors.Is(err, ErrDataStreamFull) {
		t.Fatalf("full Write() = %v", err)
	}
	if current, ok := stream.Current(); !ok || current != want {
		t.Fatalf("Current() = %+v, %v, want %+v before disk write", current, ok, want)
	}
}

func TestDataStreamWriteFailureDoesNotRollBackCurrent(t *testing.T) {
	failures := make(chan error, 1)
	stream, err := NewDataStream(&recordingWriter{failures: 1}, SystemClock{}, DataStreamOptions{
		QueueCapacity: 1, EnqueueWait: time.Second, FlushInterval: time.Hour,
		InitialBackoff: time.Millisecond, MaximumBackoff: time.Millisecond,
		OnError: func(err error) { failures <- err },
	})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	want := SensorData{Timestamp: 1, Value: -5}
	if err := stream.Write(context.Background(), want); err != nil {
		t.Fatal(err)
	}
	select {
	case <-failures:
	case <-time.After(time.Second):
		t.Fatal("writer did not report failure")
	}
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
	if err := stream.Write(context.Background(), SensorData{Timestamp: 2, Value: 3}); !errors.Is(err, ErrDataStreamClosed) {
		t.Fatalf("closed Write() = %v", err)
	}
	if current, ok := stream.Current(); !ok || current != want {
		t.Fatalf("Current() after failure and close = %+v, %v", current, ok)
	}
}

type blockingRecordWriter struct {
	recordingWriter
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (writer *blockingRecordWriter) WriteRecord(timestamp int64, value int32) error {
	writer.once.Do(func() {
		close(writer.started)
		<-writer.release
	})
	return writer.recordingWriter.WriteRecord(timestamp, value)
}

func newTestDataStream(t *testing.T, writer RecordWriter, capacity int) (*DataStream, error) {
	t.Helper()
	return NewDataStream(writer, SystemClock{}, DataStreamOptions{
		QueueCapacity:  capacity,
		EnqueueWait:    time.Second,
		FlushInterval:  time.Hour,
		RetryCount:     1,
		InitialBackoff: time.Millisecond,
		MaximumBackoff: time.Millisecond,
	})
}

type recordingWriter struct {
	mu       sync.Mutex
	records  []SensorData
	failures int
	flushes  int
	closes   int
}

func (writer *recordingWriter) WriteRecord(timestamp int64, value int32) error {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	if writer.failures > 0 {
		writer.failures--
		return errors.New("transient write failure")
	}
	writer.records = append(writer.records, SensorData{Timestamp: timestamp, Value: value})
	return nil
}

func (writer *recordingWriter) Flush() error {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	writer.flushes++
	return nil
}

func (writer *recordingWriter) Close() error {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	writer.closes++
	return nil
}
