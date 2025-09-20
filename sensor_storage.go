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
	group      *SensorGroup // The parent group (device)
	sensorName string       // List of sensor identifiers
	sensorType SensorType   // List of sensor types
	reference  time.Time    // Reference time for this stream
	current    *SensorStreamBlock
	previous   *SensorStreamBlock
}

func NewSensorStream(group *SensorGroup, sensorType SensorType) *SensorStream {
	return &SensorStream{
		group:      group,
		sensorType: sensorType,
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
	SampleType   SensorFieldType // Type of samples in this block
	SampleFreq   int32           // Unit = samples per hour
	SamplePeriod int32           // Milliseconds between samples
	SampleCount  int32           // Number of samples in this block
	LastSample   SensorValue     // Last sample value
	isModified   bool            // Indicates if the block has been modified/updated
	Buffer       []byte          // Buffer to hold the samples
}

const (
	SensorDataBlockHeaderSize = 64 // Size of the header in bytes
)

func NewSensorStreamBlock(stream *SensorStream, sampleType SensorFieldType, sampleFreq int32, samplePeriod int32) *SensorStreamBlock {
	// Construct the correct time for the start of this block
	t := time.Now()
	blockTime := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())

	// TODO This datablock might already exist on disk, load it instead of creating a new one.
	//      This can happen when the server crashes and/or restarts during the day.

	blockSizeInBytes := (int(sampleFreq*24)*int(sampleType.SizeInBits()) + 7) / 8
	blockSizeInBytes = blockSizeInBytes + SensorDataBlockHeaderSize
	block := &SensorStreamBlock{
		Stream:       stream,
		Time:         blockTime,
		SampleType:   sampleType,
		SampleFreq:   sampleFreq,
		SamplePeriod: samplePeriod,
		SampleCount:  0,
		LastSample:   SensorValue{},
		isModified:   false,
		Buffer:       make([]byte, blockSizeInBytes),
	}

	binary.LittleEndian.PutUint32(block.Buffer, uint32(blockTime.Year()))
	binary.LittleEndian.PutUint32(block.Buffer[4:], uint32(blockTime.Month()))
	binary.LittleEndian.PutUint32(block.Buffer[8:], uint32(blockTime.Day()))
	binary.LittleEndian.PutUint32(block.Buffer[12:], uint32(block.SampleType))
	binary.LittleEndian.PutUint32(block.Buffer[16:], uint32(block.SampleFreq))
	binary.LittleEndian.PutUint32(block.Buffer[20:], uint32(len(block.Buffer)-SensorDataBlockHeaderSize))

	return block
}

func (s *SensorStreamBlock) WriteSensorValue(sessionReferenceTime time.Time, packetTimeSync int32, sensorValue SensorValue) {
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
		s.Buffer[byteIndex] = byte(sensorValue.Value)
	case TypeS16:
		byteIndex := SensorDataBlockHeaderSize + sampleIndex*2
		binary.LittleEndian.PutUint16(s.Buffer[byteIndex:byteIndex+2], uint16(sensorValue.Value))
	case TypeS32:
		byteIndex := SensorDataBlockHeaderSize + sampleIndex*4
		binary.LittleEndian.PutUint32(s.Buffer[byteIndex:byteIndex+4], uint32(sensorValue.Value))
	}
}

func (s *SensorStreamBlock) IsDone() bool {
	// Determine if the block is full based on SampleFreq and time
	return s.SampleCount >= s.SampleFreq
}

// SensorGroup represents a group of sensors, typically associated with a device
type SensorGroup struct {
	config        *SensorGroupConfig // Configuration for this sensor group
	sensorStreams []*SensorStream    // Active data streams
}

func NewSensorGroup(config *SensorGroupConfig) *SensorGroup {
	return &SensorGroup{
		config:        config,
		sensorStreams: make([]*SensorStream, NumberOfSensorTypes()),
	}
}

type SensorStorage struct {
	config            *SensorServerConfig     // Configuration for the sensor storage
	groupConfigList   []*SensorGroupConfig    // List of sensor group configurations
	macToGroupIndex   map[string]int          // Map from sensor name to ID
	sensorGroups      []*SensorGroup          // Active groups (devices)
	writeChannel      chan *SensorStreamBlock // Channel for writing data blocks
	blockHeaderBuffer bytes.Buffer
	blockHeaderWriter io.Writer
}

func NewSensorStorage(config *SensorServerConfig, writeChannel chan *SensorStreamBlock) *SensorStorage {
	storage := &SensorStorage{
		config:          config,
		macToGroupIndex: make(map[string]int),
		sensorGroups:    nil,
		writeChannel:    writeChannel,
	}
	storage.blockHeaderWriter = &storage.blockHeaderBuffer

	storage.macToGroupIndex = make(map[string]int, len(config.Devices))
	storage.sensorGroups = make([]*SensorGroup, len(config.Devices))
	for i, groupConfig := range config.Devices {
		storage.sensorGroups[i] = NewSensorGroup(groupConfig)
		storage.macToGroupIndex[groupConfig.Mac] = i
	}

	// Initialize the sensor streams for each group
	for _, group := range storage.sensorGroups {
		group.sensorStreams = make([]*SensorStream, NumberOfSensorTypes())
		for _, sensorType := range SensorTypeArray() {
			sensorStream := NewSensorStream(group, sensorType)
			group.sensorStreams[sensorType.Index()] = sensorStream

			// TODO Check here if the file for today already exists, if so load it.
			//      Also check if there is a previous file for yesterday, if so load it as well.
		}
	}

	// The go-routine that writes data blocks to the file system
	go func() {
		for block := range writeChannel {
			if err := storage.WriteStreamBlock(block); err != nil {
				fmt.Printf("Error writing data block: %v\n", err)
			}
		}
	}()

	return storage
}

// RegisterSensor registers a sensor and returns its ID.
// Note: If this sensor type was not configured as part of the sensor group, -1, -1 is returned.
func (s *SensorStorage) RegisterSensor(groupIndex int32, sensorType SensorType) (int32, int32) {
	sensorGroup := s.sensorGroups[groupIndex]
	sensorIndex := sensorType.Index()
	if sensorGroup.sensorStreams[sensorIndex] != nil {
		return groupIndex, int32(sensorIndex)
	}
	return -1, -1
}

func (s *SensorStorage) WriteSensorValue(groupIndex int32, sensorIndex int32, isPacketTimeSync bool, packetTimeSync int32, sensorValue SensorValue) error {
	// Create or get the data block for the given location and sensor type
	group := s.sensorGroups[groupIndex]
	if group == nil {
		return fmt.Errorf("sensor group is nil")
	}

	sensorStream := group.sensorStreams[sensorIndex]
	if sensorStream == nil {
		return fmt.Errorf("sensor sensorStream for %s is nil, not registered in group %s", SensorType(sensorIndex).String(), group.config.Name)
	}

	if sensorStream.current == nil {
		sampleFreq := GetSampleFrequencyFromSensorType(sensorStream.sensorType)
		samplePeriod := GetSamplePeriodInMsFromSensorType(sensorStream.sensorType)
		sensorStream.current = NewSensorStreamBlock(sensorStream, SensorFieldType(sensorStream.sensorType), sampleFreq, samplePeriod)
	}

	// If the day has changed, finalize the current block and start a new one
	now := time.Now()
	if !sensorStream.current.Time.Equal(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())) {
		// Day has changed
		if sensorStream.current != nil {
			s.writeChannel <- sensorStream.current
		}
		sensorStream.previous = sensorStream.current
		sampleFreq := GetSampleFrequencyFromSensorType(sensorStream.sensorType)
		samplePeriod := GetSamplePeriodInMsFromSensorType(sensorStream.sensorType)
		sensorStream.current = NewSensorStreamBlock(sensorStream, SensorFieldType(sensorStream.sensorType), sampleFreq, samplePeriod)
	}

	// Figure out the SensorStreamBlock:
	//   We should know the reference time of this session, with (packetTimeSync / 40) seconds accuracy.
	//   With that information we can figure out the SensorStreamBlock and the SampleIndex within that block.
	//   We only need to keep two SensorDataBlocks in memory, current and previous.
	packetTime := time.Now()
	if isPacketTimeSync {
		// When this packet was marked as a time sync packet, we can update the reference time.
		// Also the packet time is the reference time, however the sender may have a guesstimate
		// on the transfer time of the packet, so we subtract that from the current time.
		sensorStream.reference = packetTime.Add(-time.Duration(packetTimeSync*25) * time.Millisecond)
	} else {
		// If we do not have a reference time yet then we are unable to figure out the time for this packet.
		// So we can do nothing else then dropping this packet.
		if sensorStream.reference.IsZero() {
			return fmt.Errorf("sensor stream reference time is zero, cannot handle non-immediate packet")
		}
		packetTime = sensorStream.reference.Add(time.Duration(packetTimeSync*25) * time.Millisecond)
	}

	if packetTime.Before(sensorStream.current.Time) {
		// Write to the previous block if it exists and the time fits
		if sensorStream.previous != nil && packetTime.After(sensorStream.previous.Time) {
			sensorStream.previous.WriteSensorValue(sensorStream.reference, packetTimeSync, sensorValue)
		}
	} else {
		// Write the sensor value to the current block's buffer
		sensorStream.current.WriteSensorValue(sensorStream.reference, packetTimeSync, sensorValue)
		if sensorStream.current.IsDone() {
			s.writeChannel <- sensorStream.current       // Send the full block to the write channel
			sensorStream.previous = sensorStream.current // Move current to previous
			sensorStream.current = nil                   // Clear the current block to create a new one next time
		}
	}
	return nil
}

// The go-routine for writing data blocks to the file system
func (s *SensorStorage) WriteStreamBlock(stream *SensorStreamBlock) error {
	// Check if the stream is valid and has data
	if stream == nil || len(stream.Buffer) == 0 {
		return nil // Nothing to write
	}

	group := stream.Stream.group

	// Open the file for writing, fully overwrite if it exists
	storePathFilename := fmt.Sprintf("%04d_%02d_%02d.data", stream.Time.Year(), stream.Time.Month(), stream.Time.Day())
	storeFullFilepath := path.Join(s.config.StoragePath, group.config.Name, stream.Stream.sensorName, storePathFilename)
	file, err := os.OpenFile(storeFullFilepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the header and data to the file
	_, err = file.Write(stream.Buffer)

	return err
}
