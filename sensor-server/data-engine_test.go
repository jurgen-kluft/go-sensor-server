package sensorserver

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDataEngineCreatesOneStreamForConcurrentWrites(t *testing.T) {
	var creations atomic.Int64
	writers := make(chan *recordingWriter, 1)
	engine, err := NewDataEngine(func(area AreaType, sensorType SensorType) (*DataStream, error) {
		creations.Add(1)
		writer := &recordingWriter{}
		writers <- writer
		return newTestDataStream(t, writer, 128)
	})
	if err != nil {
		t.Fatalf("NewDataEngine() error = %v", err)
	}

	const writes = 100
	var waitGroup sync.WaitGroup
	for index := 0; index < writes; index++ {
		waitGroup.Add(1)
		go func(value int) {
			defer waitGroup.Done()
			if err := engine.WriteSensorData(context.Background(), AreaLivingRoom, SENSOR_ID_TEMPERATURE, int64(value), int32(value)); err != nil {
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
}

func TestDataEngineSeparatesStreamKeys(t *testing.T) {
	engine, err := NewDataEngine(func(area AreaType, sensorType SensorType) (*DataStream, error) {
		return newTestDataStream(t, &recordingWriter{}, 4)
	})
	if err != nil {
		t.Fatalf("NewDataEngine() error = %v", err)
	}
	defer engine.Close()

	areas := []AreaType{AreaKitchen, AreaLivingRoom, AreaBedroom}
	sensors := []SensorType{SENSOR_ID_TEMPERATURE, SENSOR_ID_HUMIDITY, SENSOR_ID_PRESSURE}

	for i, area := range areas {
		sensor := sensors[i%len(sensors)]
		if err := engine.WriteSensorData(context.Background(), area, sensor, 1, 1); err != nil {
			t.Fatalf("WriteSensorData(%v, %v) error = %v", area, sensor, err)
		}
	}
	if engine.StreamCount() != 3 {
		t.Fatalf("StreamCount() = %d, want 3", engine.StreamCount())
	}
}

func TestDataEngineRejectsWritesAfterClose(t *testing.T) {
	engine, err := NewDataEngine(func(area AreaType, sensorType SensorType) (*DataStream, error) {
		return newTestDataStream(t, &recordingWriter{}, 1)
	})
	if err != nil {
		t.Fatalf("NewDataEngine() error = %v", err)
	}
	if err := engine.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := engine.WriteSensorData(context.Background(), AreaLivingRoom, SENSOR_ID_TEMPERATURE, 1, 1); !errors.Is(err, ErrDataEngineClosed) {
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
