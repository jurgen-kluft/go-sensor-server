package sensorserver

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
)

var ErrDataEngineClosed = errors.New("data engine is closed")

type DataStreamFactory func(mac MACAddress, sensorType SensorType) (*DataStream, error)

type DataEngine struct {
	factory DataStreamFactory
	mu      sync.RWMutex
	streams map[dataStreamKey]*DataStream
	closed  atomic.Bool
}

type dataStreamKey struct {
	mac        MACAddress
	sensorType SensorType
}

type DataStreamSnapshot struct {
	MAC        MACAddress
	SensorType SensorType
	Counters   DataStreamCounters
}

func NewDataEngine(factory DataStreamFactory) (*DataEngine, error) {
	if factory == nil {
		return nil, errors.New("new data engine: nil stream factory")
	}
	return &DataEngine{factory: factory, streams: make(map[dataStreamKey]*DataStream)}, nil
}

func NewFileDataStreamFactory(fileSystem FileSystem, clock Clock, dataRoot string, config DataStreamConfig, onError func(error)) DataStreamFactory {
	return func(mac MACAddress, sensorType SensorType) (*DataStream, error) {
		fileOptions := DefaultDataFileOptions(dataStreamDirectory(dataRoot, mac, sensorType))
		fileOptions.BufferSize = config.WriteBufferSize
		fileOptions.RotationSize = config.RotationSize
		fileOptions.OnWarning = onError
		writer, err := OpenDataFile(fileSystem, fileOptions)
		if err != nil {
			return nil, err
		}
		stream, err := NewDataStream(writer, clock, DataStreamOptions{
			InitialValue:   writer.initialValue,
			QueueCapacity:  config.QueueCapacity,
			EnqueueWait:    config.EnqueueWait.Value(),
			FlushInterval:  config.FlushInterval.Value(),
			RetryCount:     config.RetryCount,
			InitialBackoff: config.InitialBackoff.Value(),
			MaximumBackoff: config.MaximumBackoff.Value(),
			OnError: func(err error) {
				if onError != nil {
					onError(fmt.Errorf("data stream %x/%s: %w", mac, sensorType, err))
				}
			},
		})
		if err != nil {
			return nil, errors.Join(err, writer.Close())
		}
		return stream, nil
	}
}

func dataStreamDirectory(dataRoot string, mac MACAddress, sensorType SensorType) string {
	return filepath.Join(dataRoot, fmt.Sprintf("%x", mac), fmt.Sprintf("%04x", sensorType))
}

func (engine *DataEngine) WriteSensorData(ctx context.Context, mac MACAddress, sensorType SensorType, timestamp int64, value int32) error {
	if engine.closed.Load() {
		return ErrDataEngineClosed
	}
	stream, err := engine.stream(mac, sensorType)
	if err != nil {
		return err
	}
	return stream.Write(ctx, SensorData{Timestamp: timestamp, Value: value})
}

func (engine *DataEngine) Current(mac MACAddress, sensorType SensorType) (SensorData, bool) {
	engine.mu.RLock()
	stream := engine.streams[dataStreamKey{mac: mac, sensorType: sensorType}]
	engine.mu.RUnlock()
	if stream == nil {
		return SensorData{}, false
	}
	return stream.Current()
}

// RestoreStreams opens only existing streams for the configured devices and sensors.
func (engine *DataEngine) RestoreStreams(fileSystem FileSystem, dataRoot string, snapshot *ConfigSnapshot) error {
	if fileSystem == nil || snapshot == nil {
		return errors.New("restore data streams: nil filesystem or configuration")
	}
	for _, device := range snapshot.devices {
		for sensorType := range snapshot.sensors {
			directory := dataStreamDirectory(dataRoot, device.MAC, sensorType)
			entries, err := fileSystem.ReadDir(directory)
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			if err != nil {
				return fmt.Errorf("discover data stream %q: %w", directory, err)
			}
			for _, entry := range entries {
				if _, valid := parseSegmentName(entry.Name()); !valid || entry.IsDir() {
					continue
				}
				if _, err := engine.stream(device.MAC, sensorType); err != nil {
					return fmt.Errorf("restore data streams: %w", err)
				}
				break
			}
		}
	}
	return nil
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
			MAC: key.mac, SensorType: key.sensorType, Counters: stream.Counters(),
		})
	}
	sort.Slice(snapshots, func(left, right int) bool {
		if order := bytes.Compare(snapshots[left].MAC[:], snapshots[right].MAC[:]); order != 0 {
			return order < 0
		}
		return snapshots[left].SensorType < snapshots[right].SensorType
	})
	return snapshots
}

func (engine *DataEngine) stream(mac MACAddress, sensorType SensorType) (*DataStream, error) {
	key := dataStreamKey{mac: mac, sensorType: sensorType}
	engine.mu.Lock()
	defer engine.mu.Unlock()
	if engine.closed.Load() {
		return nil, ErrDataEngineClosed
	}
	if stream := engine.streams[key]; stream != nil {
		return stream, nil
	}
	stream, err := engine.factory(mac, sensorType)
	if err != nil {
		return nil, fmt.Errorf("create data stream for %x/%s: %w", mac, sensorType, err)
	}
	engine.streams[key] = stream
	return stream, nil
}
