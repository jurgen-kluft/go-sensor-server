package sensor_server

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path"
	"time"
)

// Storage for sensor data
// One sensor store for [DeviceLocation | SensorType]
// DeviceLocation is associated with
type SensorStream struct {
	index      int        // Unique index for this sensor
	sensorName string     // List of sensor identifiers
	sensorType SensorType // List of sensor types
	sensorFreq int32      // Sample frequency per sensor type
	reference  time.Time  // Reference time for this stream
	current    *SensorStreamBlock
	previous   *SensorStreamBlock
}

func NewSensorStream(index int, sensorName string, sensorType SensorType, sensorFreq int32) *SensorStream {
	return &SensorStream{
		index:      index,
		sensorName: sensorName,
		sensorType: sensorType,
		sensorFreq: sensorFreq,
		reference:  time.Time{},
		current:    nil,
		previous:   nil,
	}
}

// SensorStreamBlock format:
//   - Year (int)
//   - Month (int)
//   - Day (int)
//   - Hour (int)
//   - SampleType (int) - bit, int8, int16, int32
//   - SampleFreq (int) - samples per hour
//   - DataLength (int) - length of data in bytes
//   - Data (variable length, depends on SampleType and SampleFreq)
type SensorStreamBlock struct {
	Stream       *SensorStream   // The parent stream
	Time         time.Time       // Time (Year, Month, Day, at zero hour)
	SensorType   SensorType      // Type of sensor (Temperature, Humidity, etc)
	SampleType   SensorFieldType // Type of samples in this block
	SampleFreq   int32           // Unit = samples per hour
	SamplePeriod int32           // Milliseconds between samples
	SampleCount  int32           // Number of samples in this block
	LastSample   SensorValue     // Last sample value
	Buffer       []byte          // Buffer to hold the samples
}

const (
	SensorDataBlockHeaderSize = 64 // Size of the header in bytes
)

func NewSensorStreamBlock(stream *SensorStream, sampleType SensorFieldType, sampleFreq int32) *SensorStreamBlock {
	// Construct the correct time for the start of this block
	t := time.Now()
	blockTime := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())

	// TODO This datablock might already exist on disk, load it instead of creating a new one.
	//      This can happen when the server crashes and/or restarts during the day.

	blockSizeInBytes := (int(sampleFreq*24)*int(sampleType.SizeInBits()) + 7) / 8
	blockSizeInBytes = blockSizeInBytes + SensorDataBlockHeaderSize
	block := &SensorStreamBlock{
		Stream:     stream,
		Time:       blockTime,
		SampleType: sampleType,
		SampleFreq: sampleFreq,
		Buffer:     make([]byte, blockSizeInBytes),
	}

	binary.LittleEndian.PutUint32(block.Buffer, uint32(blockTime.Year()))
	binary.LittleEndian.PutUint32(block.Buffer[4:], uint32(blockTime.Month()))
	binary.LittleEndian.PutUint32(block.Buffer[8:], uint32(blockTime.Day()))
	binary.LittleEndian.PutUint32(block.Buffer[12:], uint32(block.SampleType))
	binary.LittleEndian.PutUint32(block.Buffer[16:], uint32(block.SampleFreq))
	binary.LittleEndian.PutUint32(block.Buffer[20:], uint32(len(block.Buffer)-SensorDataBlockHeaderSize))

	return block
}

func (s *SensorStreamBlock) WriteSensorValue(sessionReferenceTime time.Time, packetTimeSync int, sensorValue SensorValue) {
	s.LastSample = sensorValue
	if sensorValue.IsZero() {
		return // Default value
	}

	sampleTime := time.Now()
	sampleIndex := sampleTime.Sub(s.Time).Milliseconds() / (60 * 60 * 1000 / int64(s.SampleFreq))
	if sampleIndex < 0 || sampleIndex >= int64(s.SampleFreq*24) {
		// Sample is out of range for this block, ignore it
		return
	}

	s.SampleCount = int32(sampleIndex) + 1

	// Write the sensor value to the buffer based on its type
	switch s.SampleType {
	case TypeBit:
		byteIndex := SensorDataBlockHeaderSize + (sampleIndex / 8)
		bitIndex := sampleIndex % 8
		s.Buffer[byteIndex] |= (1 << bitIndex) // Set the bit
	case TypeS8:
		byteIndex := SensorDataBlockHeaderSize + sampleIndex
		s.Buffer[byteIndex] = byte(sensorValue.value)
	case TypeS16:
		byteIndex := SensorDataBlockHeaderSize + sampleIndex*2
		binary.LittleEndian.PutUint16(s.Buffer[byteIndex:byteIndex+2], uint16(sensorValue.value))
	case TypeS32:
		byteIndex := SensorDataBlockHeaderSize + sampleIndex*4
		binary.LittleEndian.PutUint32(s.Buffer[byteIndex:byteIndex+4], uint32(sensorValue.value))
	}
}

func (s *SensorStreamBlock) IsDone() bool {
	// Determine if the block is full based on SampleFreq and time
	return s.SampleCount >= s.SampleFreq
}

// SensorGroup represents a group of sensors, typically associated with a device
type SensorGroup struct {
	config  *SensorGroupConfig // Configuration for this sensor group
	streams []*SensorStream    // Active data streams
}

func NewSensorGroup(config *SensorGroupConfig) *SensorGroup {
	return &SensorGroup{
		config:  config,
		streams: make([]*SensorStream, len(config.Sensors), len(config.Sensors)),
	}
}

type SensorStorage struct {
	config                *SensorServerConfig     // Configuration for the sensor storage
	groupConfigList       []*SensorGroupConfig    // List of sensor group configurations
	groupConfigMap        map[string]int          // map from mac address to store index
	macToSensorGroupIndex map[string]int          // Map from sensor name to ID
	sensorGroups          []*SensorGroup          // Active groups (devices)
	writeChannel          chan *SensorStreamBlock // Channel for writing data blocks
	blockHeaderBuffer     bytes.Buffer
	blockHeaderWriter     io.Writer
}

func NewSensorStorage(config *SensorServerConfig, writeChannel chan *SensorStreamBlock) *SensorStorage {
	storage := &SensorStorage{
		config:                config,
		macToSensorGroupIndex: make(map[string]int),
		sensorGroups:          make([]*SensorGroup, config.MaximumStores, config.MaximumStores),
		writeChannel:          writeChannel,
	}
	storage.blockHeaderWriter = &storage.blockHeaderBuffer

	sensorListMaxSize := 0
	for i, _ := range storage.sensorGroups {
		group := NewSensorGroup(config.Groups[i])
		storage.sensorGroups[i] = group
		if (group.config.Index + 1) > sensorListMaxSize {
			sensorListMaxSize = group.config.Index + 1
		}
	}

	for _, group := range storage.sensorGroups {
		storage.groupConfigMap[group.config.Mac] = group.config.Index
		group.streams = make([]*SensorStream, sensorListMaxSize, sensorListMaxSize)
		for _, sensor := range group.config.Sensors {
			sensorType := NewSensorType(sensor.Type)
			group.streams[sensor.Index] = NewSensorStream(sensor.Index, sensor.Type, sensorType, GetSamplePeriodInMsFromSensorType(sensorType))
		}
	}

	// The go-routine that writes data blocks to the file system
	go func() {
		for block := range writeChannel {
			err := storage.WriteStreamBlock(block)
			if err != nil {
				fmt.Printf("Error writing data block: %v\n", err)
			}
		}
	}()

	return storage
}

// RegisterSensor registers a new sensor and returns its ID.
// Note: 'sensorName' should be unique.
func (s *SensorStorage) RegisterSensor(mac string, sensorName string, sensorType SensorType) (int, int) {
	if groupIndex, exists := s.macToSensorGroupIndex[mac]; exists {
		sensorGroup := s.sensorGroups[groupIndex]
		streamIndex := sensorGroup.config.NameToIndex[sensorName]
		return groupIndex, streamIndex
	}

	groupIndex := s.groupConfigMap[mac]
	if groupIndex < 0 || groupIndex >= len(s.groupConfigList) {
		return -1, -1 // Invalid store index
	}

	// Sensor group index
	groupConfig := s.groupConfigList[groupIndex]
	s.macToSensorGroupIndex[sensorName] = groupIndex

	// Sensor stream index
	streamIndex := groupConfig.NameToIndex[sensorName]

	// Frequency is derived from the sensor type
	sensorFreq := GetSamplePeriodInMsFromSensorType(sensorType)

	sensorGroup := s.sensorGroups[groupIndex]
	if sensorGroup == nil {
		sensorGroup = &SensorGroup{
			config:  groupConfig,
			streams: make([]*SensorStream, len(groupConfig.Sensors), len(groupConfig.Sensors)),
		}
		s.sensorGroups[groupIndex] = sensorGroup
	}

	sensorStream := sensorGroup.streams[streamIndex]
	if sensorStream == nil {
		sensorStream = &SensorStream{
			index:      streamIndex,
			sensorName: sensorName,
			sensorType: sensorType,
			sensorFreq: sensorFreq,
			reference:  time.Time{},
			current:    nil,
			previous:   nil,
		}
		sensorGroup.streams[streamIndex] = sensorStream
	}

	return groupIndex, streamIndex
}

func (s *SensorStorage) WriteSensorValue(groupIndex int, sensorIndex int, packetImmediate bool, packetTimeSync int, sensorValue SensorValue) error {
	// Create or get the data block for the given location and sensor type
	group := s.sensorGroups[groupIndex]
	if group == nil {
		return fmt.Errorf("sensor group %d not registered", groupIndex)
	}

	stream := group.streams[sensorIndex]
	if stream == nil {
		return fmt.Errorf("sensor stream %d not registered in group %d", sensorIndex, groupIndex)
	}

	if stream.current == nil {
		stream.current = NewSensorStreamBlock(stream, SensorFieldType(stream.sensorType), stream.sensorFreq)
	}

	// If the day has changed, finalize the current block and start a new one
	now := time.Now()
	if !stream.current.Time.Equal(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())) {
		// Day has changed
		if stream.current != nil {
			s.writeChannel <- stream.current
		}
		stream.previous = stream.current
		stream.current = NewSensorStreamBlock(stream, SensorFieldType(stream.sensorType), stream.sensorFreq)
	}

	// Figure out the SensorStreamBlock:
	//   We should know the reference time of this session, with (packetTimeSync / 40) seconds accuracy.
	//   With that information we can figure out the SensorStreamBlock and the SampleIndex within that block.
	//   We only need to keep two SensorDataBlocks in memory, current and previous.
	if packetImmediate {
		// Immediate packets always go to the current block but they also set the reference time when
		// the reference time was not set yet.
		if stream.reference.IsZero() {
			now := time.Now()
			stream.reference = now.Add(-time.Duration(packetTimeSync*25) * time.Millisecond)
		}

	}

	// Write the sensor value to the block's buffer
	stream.current.WriteSensorValue(stream.reference, packetTimeSync, sensorValue)
	if stream.current.IsDone() {
		s.writeChannel <- stream.current // Send the full block to the write channel
		stream.previous = stream.current // Move current to previous
		stream.current = nil             // Clear the current block to create a new one next time
	}

	return nil
}

// The go-routine for writing data blocks to the file system
func (s *SensorStorage) WriteStreamBlock(stream *SensorStreamBlock) error {
	// Check if the stream is valid and has data
	if stream == nil || len(stream.Buffer) == 0 {
		return nil // Nothing to write
	}

	// Open the file for writing, append-only
	sensorStorePath := path.Join(s.config.StoragePath, stream.Stream.sensorName, fmt.Sprintf("sensor_data_%04d_%02d_%02d.dat", stream.Time.Year(), stream.Time.Month(), stream.Time.Day()))
	file, err := os.OpenFile(sensorStorePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the header and data to the file
	_, err = file.Write(stream.Buffer)

	return err
}
