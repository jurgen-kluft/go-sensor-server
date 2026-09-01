package sensorserver

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
)

var ErrDataEngineClosed = errors.New("data engine is closed")

type DataStreamFactory func(area, sensorType string) (*DataStream, error)

type DataEngine struct {
	factory DataStreamFactory
	mu      sync.Mutex
	streams map[dataStreamKey]*DataStream
	closed  atomic.Bool
}

type dataStreamKey struct {
	area       string
	sensorType string
}

type DataStreamSnapshot struct {
	Area       string
	SensorType string
	Counters   DataStreamCounters
}

func NewDataEngine(factory DataStreamFactory) (*DataEngine, error) {
	if factory == nil {
		return nil, errors.New("new data engine: nil stream factory")
	}
	return &DataEngine{factory: factory, streams: make(map[dataStreamKey]*DataStream)}, nil
}

func NewFileDataStreamFactory(fileSystem FileSystem, clock Clock, dataRoot string, config DataStreamConfig, onError func(error)) DataStreamFactory {
	return func(area, sensorType string) (*DataStream, error) {
		fileOptions := DefaultDataFileOptions(filepath.Join(dataRoot, area, sensorType))
		fileOptions.BufferSize = config.WriteBufferSize
		fileOptions.RotationSize = config.RotationSize
		writer, err := OpenDataFile(fileSystem, fileOptions)
		if err != nil {
			return nil, err
		}
		stream, err := NewDataStream(writer, clock, DataStreamOptions{
			QueueCapacity:  config.QueueCapacity,
			EnqueueWait:    config.EnqueueWait.Value(),
			FlushInterval:  config.FlushInterval.Value(),
			RetryCount:     config.RetryCount,
			InitialBackoff: config.InitialBackoff.Value(),
			MaximumBackoff: config.MaximumBackoff.Value(),
			OnError:        onError,
		})
		if err != nil {
			_ = writer.Close()
			return nil, err
		}
		return stream, nil
	}
}

func (engine *DataEngine) WriteSensorData(ctx context.Context, area, sensorType string, timestamp int64, value int16) error {
	if engine.closed.Load() {
		return ErrDataEngineClosed
	}
	stream, err := engine.stream(area, sensorType)
	if err != nil {
		return err
	}
	return stream.Write(ctx, SensorData{Timestamp: timestamp, Value: value})
}

func (engine *DataEngine) Close() error {
	if !engine.closed.CompareAndSwap(false, true) {
		return nil
	}
	engine.mu.Lock()
	streams := make([]*DataStream, 0, len(engine.streams))
	for _, stream := range engine.streams {
		streams = append(streams, stream)
	}
	engine.mu.Unlock()

	var result error
	for _, stream := range streams {
		result = errors.Join(result, stream.Close())
	}
	return result
}

func (engine *DataEngine) StreamCount() int {
	engine.mu.Lock()
	defer engine.mu.Unlock()
	return len(engine.streams)
}

func (engine *DataEngine) Counters() []DataStreamSnapshot {
	engine.mu.Lock()
	defer engine.mu.Unlock()
	snapshots := make([]DataStreamSnapshot, 0, len(engine.streams))
	for key, stream := range engine.streams {
		snapshots = append(snapshots, DataStreamSnapshot{
			Area: key.area, SensorType: key.sensorType, Counters: stream.Counters(),
		})
	}
	sort.Slice(snapshots, func(left, right int) bool {
		if snapshots[left].Area != snapshots[right].Area {
			return snapshots[left].Area < snapshots[right].Area
		}
		return snapshots[left].SensorType < snapshots[right].SensorType
	})
	return snapshots
}

func (engine *DataEngine) stream(area, sensorType string) (*DataStream, error) {
	key := dataStreamKey{area: area, sensorType: sensorType}
	engine.mu.Lock()
	defer engine.mu.Unlock()
	if engine.closed.Load() {
		return nil, ErrDataEngineClosed
	}
	if stream := engine.streams[key]; stream != nil {
		return stream, nil
	}
	stream, err := engine.factory(area, sensorType)
	if err != nil {
		return nil, fmt.Errorf("create data stream for %s/%s: %w", area, sensorType, err)
	}
	engine.streams[key] = stream
	return stream, nil
}
