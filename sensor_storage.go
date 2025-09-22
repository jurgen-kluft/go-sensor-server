package sensor_server

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Storage for sensor data
// One sensor store for [DeviceLocation | SensorType]
// DeviceLocation is associated with
type sensorStream struct {
	group      *sensorDevice      // The parent group (device)
	sensorName string             // List of sensor identifiers
	sensorType SensorType         // List of sensor types
	reference  time.Time          // Reference time for this stream
	today      *sensorStreamBlock // Data block for today
	yesterday  *sensorStreamBlock // Data block for yesterday
}

func newSensorStream(group *sensorDevice, sensorType SensorType) *sensorStream {
	return &sensorStream{
		group:      group,
		sensorType: sensorType,
		reference:  time.Time{},
		today:      nil,
		yesterday:  nil,
	}
}

// sensorStreamBlock format:
//   - Year (int)
//   - Month (int)
//   - Day (int)
//   - Hour (int)
//   - SampleType (int) - bit, int8, int16, int32
//   - SampleFreq (int) - samples per hour
//   - DataLength (int) - length of data in bytes
//   - Data (variable length, depends on SampleType and SampleFreq)
type sensorStreamBlock struct {
	Stream       *sensorStream   // The parent stream
	Time         time.Time       // Time (Year, Month, Day, at zero hour)
	SampleType   SensorFieldType // Type of samples in this block
	SampleFreq   int32           // Unit = samples per hour
	SamplePeriod int32           // Milliseconds between samples
	LastSample   SensorValue     // Last sample value
	Content      []byte          // Content to hold the samples
	isModified   atomic.Int32    // Indicates if the block has been modified
}

const (
	SensorDataBlockHeaderSize = 64 // Size of the header in bytes
)

func newSensorStreamBlock(stream *sensorStream, sampleType SensorFieldType, sampleFreq int32, samplePeriod int32, content []byte) *sensorStreamBlock {
	// Construct the correct time for the start of this block
	t := time.Now()
	blockTime := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())

	// TODO This datablock might already exist on disk, load it instead of creating a new one.
	//      This can happen when the server crashes and/or restarts during the day.

	blockSizeInBytes := (int(sampleFreq*24)*int(sampleType.SizeInBits()) + 7) / 8
	blockSizeInBytes = blockSizeInBytes + SensorDataBlockHeaderSize
	block := &sensorStreamBlock{
		Stream:       stream,
		Time:         blockTime,
		SampleType:   sampleType,
		SampleFreq:   sampleFreq,
		SamplePeriod: samplePeriod,
		LastSample:   SensorValue{},
		Content:      nil,
		isModified:   atomic.Int32{},
	}

	if content != nil && len(content) == blockSizeInBytes {
		block.Content = content
	}

	return block
}

func (s *sensorStreamBlock) createContentBuffer() {
	if s.Content != nil {
		return // Already created
	}

	blockSizeInBytes := (int(s.SampleFreq*24)*int(s.SampleType.SizeInBits()) + 7) / 8
	blockSizeInBytes = blockSizeInBytes + SensorDataBlockHeaderSize
	s.Content = make([]byte, blockSizeInBytes)

	// Write the header
	binary.LittleEndian.PutUint32(s.Content, uint32(s.Time.Year()))
	binary.LittleEndian.PutUint32(s.Content[4:], uint32(s.Time.Month()))
	binary.LittleEndian.PutUint32(s.Content[8:], uint32(s.Time.Day()))
	binary.LittleEndian.PutUint32(s.Content[12:], uint32(s.SampleType))
	binary.LittleEndian.PutUint32(s.Content[16:], uint32(s.SampleFreq))
	binary.LittleEndian.PutUint32(s.Content[20:], uint32(len(s.Content)-SensorDataBlockHeaderSize))
}

func (s *sensorStreamBlock) WriteSensorValue(sessionReferenceTime time.Time, packetTimeSync int32, sensorValue SensorValue) {
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

	// Write the sensor value to the buffer based on its type
	switch s.SampleType {
	case TypeBit:
		byteIndex := SensorDataBlockHeaderSize + (sampleIndex / 8)
		bitIndex := sampleIndex % 8
		s.Content[byteIndex] |= (1 << bitIndex) // Set the bit
	case TypeS8:
		byteIndex := SensorDataBlockHeaderSize + sampleIndex
		s.Content[byteIndex] = byte(sensorValue.Value)
	case TypeS16:
		byteIndex := SensorDataBlockHeaderSize + sampleIndex*2
		binary.LittleEndian.PutUint16(s.Content[byteIndex:byteIndex+2], uint16(sensorValue.Value))
	case TypeS32:
		byteIndex := SensorDataBlockHeaderSize + sampleIndex*4
		binary.LittleEndian.PutUint32(s.Content[byteIndex:byteIndex+4], uint32(sensorValue.Value))
	case TypeS64:
		byteIndex := SensorDataBlockHeaderSize + sampleIndex*8
		binary.LittleEndian.PutUint64(s.Content[byteIndex:byteIndex+8], uint64(sensorValue.Value))
	case TypeU8:
		byteIndex := SensorDataBlockHeaderSize + sampleIndex
		s.Content[byteIndex] = byte(sensorValue.Value)
	case TypeU16:
		byteIndex := SensorDataBlockHeaderSize + sampleIndex*2
		binary.LittleEndian.PutUint16(s.Content[byteIndex:byteIndex+2], uint16(sensorValue.Value))
	case TypeU32:
		byteIndex := SensorDataBlockHeaderSize + sampleIndex*4
		binary.LittleEndian.PutUint32(s.Content[byteIndex:byteIndex+4], uint32(sensorValue.Value))
	case TypeU64:
		byteIndex := SensorDataBlockHeaderSize + sampleIndex*8
		binary.LittleEndian.PutUint64(s.Content[byteIndex:byteIndex+8], uint64(sensorValue.Value))
	}

	// Indicate that this block has been modified
	s.isModified.Add(1)
}

// sensorDevice represents a group of sensors, typically associated with a device
type sensorDevice struct {
	config        *SensorDeviceConfig // Configuration for this sensor group
	sensorStreams []*sensorStream     // Active data streams
}

func newSensorGroup(config *SensorDeviceConfig) *sensorDevice {
	return &sensorDevice{
		config:        config,
		sensorStreams: make([]*sensorStream, len(SensorTypes)),
	}
}

type SensorStorage struct {
	config            *SensorServerConfig     // Configuration for the sensor storage
	deviceConfigList  []*SensorDeviceConfig   // List of sensor group configurations
	macToDeviceIndex  map[string]int          // Map from sensor name to ID
	devices           []*sensorDevice         // Active groups (devices)
	writeChannel      chan *sensorStreamBlock // Channel for writing data blocks
	blockHeaderBuffer bytes.Buffer            //
	blockHeaderWriter io.Writer               //

	// Flush go-routine data
	flushTicker  *time.Ticker            // Ticker for periodic flushing
	flushLock    *sync.Mutex             // Mutex for synchronizing queue access
	flushChannel chan *sensorStreamBlock // Channel for flushing data blocks
	flushQueue   []*sensorStreamBlock    // Queue of blocks to flush
}

func NewSensorStorage(config *SensorServerConfig) *SensorStorage {
	storage := &SensorStorage{
		config:            config,
		macToDeviceIndex:  make(map[string]int),
		devices:           nil,
		writeChannel:      make(chan *sensorStreamBlock, 64),
		blockHeaderBuffer: bytes.Buffer{},
		blockHeaderWriter: nil,
		flushTicker:       nil,
		flushChannel:      make(chan *sensorStreamBlock, 128),
		flushQueue:        make([]*sensorStreamBlock, 0, 256),
	}
	storage.blockHeaderWriter = &storage.blockHeaderBuffer

	storage.macToDeviceIndex = make(map[string]int, len(config.Devices))
	storage.devices = make([]*sensorDevice, len(config.Devices))
	for i, groupConfig := range config.Devices {
		storage.devices[i] = newSensorGroup(groupConfig)
		storage.macToDeviceIndex[groupConfig.Mac] = i
	}

	dirpath := config.StoragePath
	// resolve any environment variables in the path
	dirpath = os.ExpandEnv(dirpath)
	// Create the storage directory if it does not exist
	if _, err := os.Stat(dirpath); os.IsNotExist(err) {
		err := os.MkdirAll(dirpath, 0755)
		if err != nil {
			fmt.Println("Failed to create storage directory:", err)
			return nil
		}
	}

	// Create subdirectories for each device and sensor type
	for _, group := range storage.devices {
		groupPath := path.Join(dirpath, group.config.Name)
		if _, err := os.Stat(groupPath); os.IsNotExist(err) {
			err := os.MkdirAll(groupPath, 0755)
			if err != nil {
				fmt.Println("Failed to create group directory:", err)
				return nil
			}
		}
		for _, sensorType := range SensorTypes {
			if sensorType == Unknown || sensorType == MacAddress {
				continue
			}
			sensorPath := path.Join(groupPath, sensorType.String())
			if _, err := os.Stat(sensorPath); os.IsNotExist(err) {
				err := os.MkdirAll(sensorPath, 0755)
				if err != nil {
					fmt.Println("Failed to create sensor directory:", err)
					return nil
				}
			}
		}
	}

	// Initialize the sensor streams for each group
	for _, group := range storage.devices {
		for _, sensorType := range SensorTypes {
			if sensorType == Unknown {
				continue
			}

			sensorStream := newSensorStream(group, sensorType)
			group.sensorStreams[sensorType.Index()] = sensorStream
			storage.setupSensorStream(sensorStream)

			storage.flushChannel <- sensorStream.today
			if sensorStream.yesterday != nil {
				storage.flushChannel <- sensorStream.yesterday
			}
		}
	}

	// The go-routine that writes data blocks to the file system
	go processWrite(storage)

	fmt.Printf("Periodic flush interval: %d seconds\n", config.FlushPeriodInSeconds)
	schedulePeriodicFlush(storage, time.Duration(config.FlushPeriodInSeconds)*time.Second)

	fmt.Printf("Sensor storage initialized, storage path: %s\n", dirpath)
	return storage
}

func (s *SensorStorage) Shutdown() {
	// Terminate the flush go-routine
	s.flushChannel <- nil
	if s.flushTicker != nil {
		s.flushTicker.Stop()
	}

	time.Sleep(1 * time.Second) // Give some time for the flush go-routine to finish

	// Flush all blocks to disk
	for _, group := range s.devices {
		for _, sensorStream := range group.sensorStreams {
			if sensorStream.today != nil {
				count := sensorStream.today.isModified.Load()
				if count > 0 {
					sensorStream.today.isModified.Add(-count)
					s.writeChannel <- sensorStream.today
				}
			}
			if sensorStream.yesterday != nil {
				count := sensorStream.yesterday.isModified.Load()
				if count > 0 {
					sensorStream.yesterday.isModified.Add(-count)
					s.writeChannel <- sensorStream.yesterday
				}
			}
		}
	}

	// End the flush go-routine
	s.writeChannel <- nil

	time.Sleep(1 * time.Second) // Give some time for the write go-routine to finish

	// Close the write channel and wait for it to finish
	close(s.writeChannel)
}

func processWrite(s *SensorStorage) {
	for block := range s.writeChannel {
		if block == nil {
			fmt.Printf("go-routine for writing stream data blocks has exited\n")
			return // Exit the go-routine
		}
		if err := writeStreamBlock(s, block); err != nil {
			fmt.Printf("error writing data block: %v\n", err)
		}
	}
}

func processFlush(s *SensorStorage) {
	now := time.Now()

	fmt.Println("Processing flush queue, length:", len(s.flushQueue))

	for i, sensorStream := range s.flushQueue {
		count := sensorStream.isModified.Load()
		if count > 0 {
			sensorStream.isModified.Add(-count)
			s.writeChannel <- sensorStream
		}

		// Remove from queue if older than 2 days
		if now.Sub(sensorStream.Time) >= 48*time.Hour {
			s.flushQueue[i] = nil
		}
	}

	// Sort queue (nil entries will be at the end)
	sort.SliceStable(s.flushQueue, func(i, j int) bool {
		if s.flushQueue[i] == nil {
			return false
		}
		if s.flushQueue[j] == nil {
			return true
		}
		return s.flushQueue[i].Time.Before(s.flushQueue[j].Time)
	})

	// Figure out the index of the last non-nil entry
	lastNonNilIndex := -1
	for i := len(s.flushQueue) - 1; i >= 0; i-- {
		if s.flushQueue[i] != nil {
			lastNonNilIndex = i
			break
		}
	}

	// Trim the queue to remove nil entries
	s.flushQueue = s.flushQueue[:lastNonNilIndex+1]

}

func schedulePeriodicFlush(s *SensorStorage, interval time.Duration) {
	// Although we are saving the data blocks when they are full, we also want to
	// periodically flush the content of all 'modified' blocks to disk to minimize
	// data loss in case of a crash.

	// Everytime a stream block is created it is pushed on the flush channel, where
	// we here are taking it and pushing it on the flush queue.
	// Every 'tick' we iterate the queue and flush all blocks that are marked as modified.
	// Also any block that is older than 2 days is removed from the queue.

	s.flushTicker = time.NewTicker(interval)
	s.flushLock = &sync.Mutex{}

	go func() {
		for {
			select {
			case block := <-s.flushChannel:
				if block == nil {
					fmt.Printf("go-routine for flushing stream data blocks has exited\n")
					return // Exit the go-routine
				}
				s.flushLock.Lock()
				s.flushQueue = append(s.flushQueue, block)
				s.flushLock.Unlock()
			case <-s.flushTicker.C:
				s.flushLock.Lock()
				fmt.Printf("Periodic flush: flushing %d blocks\n", len(s.flushQueue))
				processFlush(s)
				s.flushLock.Unlock()
			}
		}
	}()
}

// RegisterSensor registers a sensor and returns its ID.
// Note: If this sensor type was not configured as part of the sensor group, -1 is returned.
func (s *SensorStorage) RegisterSensor(groupIndex int32, sensorType SensorType) int32 {
	sensorGroup := s.devices[groupIndex]
	sensorIndex := sensorType.Index()
	if sensorGroup.sensorStreams[sensorIndex] != nil {
		return int32(sensorIndex)
	}
	return -1
}

func (s *SensorStorage) WriteSensorValue(groupIndex int32, sensorIndex int32, isPacketTimeSync bool, packetTimeSync int32, sensorValue SensorValue) error {
	// Create or get the data block for the given location and sensor type
	group := s.devices[groupIndex]
	if group == nil {
		return fmt.Errorf("sensor group is nil")
	}

	sensorStream := group.sensorStreams[sensorIndex]
	if sensorStream == nil {
		return fmt.Errorf("sensor sensorStream for %s is nil, not registered in group %s", SensorType(sensorIndex).String(), group.config.Name)
	}

	// Current time
	now := time.Now()

	// Update the sensor stream (check if day has changed)
	s.updateSensorStream(sensorStream, now)

	// Figure out the sensorStreamBlock:
	//   We should know the reference time of this session, with (packetTimeSync / 40) seconds accuracy.
	//   With that information we can figure out the sensorStreamBlock and the SampleIndex within that block.
	//   We only need to keep two SensorDataBlocks in memory, today and yesterday.
	packetTime := now
	if isPacketTimeSync {
		// When this packet was marked as a time sync packet, we can update the reference time.
		// Also the packet time is the reference time, however the sender may have a guesstimate
		// on the transfer time of the packet, so we subtract that from the today time.
		sensorStream.reference = packetTime.Add(-time.Duration(packetTimeSync*25) * time.Millisecond)
	} else {
		// If we do not have a reference time yet then we are unable to figure out the time for this packet.
		// So we can do nothing else then dropping this packet.
		if sensorStream.reference.IsZero() {
			return fmt.Errorf("sensor stream reference time is zero, cannot handle non-immediate packet")
		}
		packetTime = sensorStream.reference.Add(time.Duration(packetTimeSync*25) * time.Millisecond)
	}

	if packetTime.Before(sensorStream.today.Time) {
		// If packet is really old (which should not happen) then we drop it
		if packetTime.Before(sensorStream.yesterday.Time) {
			return fmt.Errorf("sensor stream packet time is before yesterday block, dropping packet")
		}
		// Write to the yesterday block if it exists and the time fits
		if packetTime.After(sensorStream.yesterday.Time) {
			if sensorStream.yesterday.Content == nil {
				sensorStream.yesterday.createContentBuffer()
				s.flushChannel <- sensorStream.yesterday
			}
			sensorStream.yesterday.WriteSensorValue(sensorStream.reference, packetTimeSync, sensorValue)
		}
	} else {
		// Write the sensor value to the today block's buffer
		if sensorStream.today.Content == nil {
			sensorStream.today.createContentBuffer()
			s.flushChannel <- sensorStream.today
		}
		sensorStream.today.WriteSensorValue(sensorStream.reference, packetTimeSync, sensorValue)
	}
	return nil
}

// The go-routine for writing data blocks to the file system
func (s *SensorStorage) getStreamBlockFilepath(stream *sensorStream, time time.Time) string {
	storePathFilename := fmt.Sprintf("%04d_%02d_%02d.data", time.Year(), time.Month(), time.Day())
	storeFullFilepath := path.Join(s.config.StoragePath, stream.group.config.Name, stream.sensorName, storePathFilename)
	return storeFullFilepath
}

func writeStreamBlock(s *SensorStorage, stream *sensorStreamBlock) error {
	// Check if the stream is valid and has data
	if stream == nil || len(stream.Content) == 0 {
		return nil // Nothing to write
	}

	// Open the file for writing, fully overwrite if it exists
	storeFullFilepath := s.getStreamBlockFilepath(stream.Stream, stream.Time)
	file, err := os.OpenFile(storeFullFilepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the header and data to the file
	_, err = file.Write(stream.Content)

	return err
}

func (s *SensorStorage) loadSensorDataBlockContent(stream *sensorStream, time time.Time) ([]byte, error) {
	// First try to load today's file if it exists
	storeFullFilepath := s.getStreamBlockFilepath(stream, time)

	// Check if the file exists
	var fileSize int64
	if stat, err := os.Stat(storeFullFilepath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", storeFullFilepath)
	} else {
		fileSize = stat.Size()
		if fileSize < SensorDataBlockHeaderSize {
			return nil, fmt.Errorf("file too small to be a valid data block: %s", storeFullFilepath)
		}
	}

	// Open the file for reading
	file, err := os.Open(storeFullFilepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read the content
	content := make([]byte, fileSize)
	_, err = io.ReadFull(file, content)
	if err != nil {
		return nil, err
	}

	// Do some basic validation of the content
	dataLength := int32(binary.LittleEndian.Uint32(content[20:24]))
	if int(dataLength) != len(content)-SensorDataBlockHeaderSize {
		return nil, fmt.Errorf("data length mismatch in file: %s", storeFullFilepath)
	}

	return content, nil
}

func (s *SensorStorage) setupSensorStream(sensorStream *sensorStream) error {

	sampleFreq := GetSampleFrequencyFromSensorType(sensorStream.sensorType)
	samplePeriod := GetSamplePeriodInMsFromSensorType(sensorStream.sensorType)

	today := time.Now()
	if currentContent, err := s.loadSensorDataBlockContent(sensorStream, today); err == nil {
		sensorStream.today = newSensorStreamBlock(sensorStream, SensorFieldType(sensorStream.sensorType), sampleFreq, samplePeriod, currentContent)
	} else {
		sensorStream.today = newSensorStreamBlock(sensorStream, SensorFieldType(sensorStream.sensorType), sampleFreq, samplePeriod, nil)
	}

	if sensorStream.today.Content != nil {
		s.flushChannel <- sensorStream.today
	}

	yesterday := today.AddDate(0, 0, -1)
	if yesterdayContent, err := s.loadSensorDataBlockContent(sensorStream, yesterday); err == nil {
		sensorStream.yesterday = newSensorStreamBlock(sensorStream, SensorFieldType(sensorStream.sensorType), sampleFreq, samplePeriod, yesterdayContent)
	} else {
		sensorStream.yesterday = newSensorStreamBlock(sensorStream, SensorFieldType(sensorStream.sensorType), sampleFreq, samplePeriod, nil)
	}

	if sensorStream.yesterday.Content != nil {
		s.flushChannel <- sensorStream.yesterday
	}

	return nil
}

func (s *SensorStorage) updateSensorStream(sensorStream *sensorStream, today time.Time) {

	// If the day has changed, finalize the today block and start a new one
	if !sensorStream.today.Time.Equal(time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())) {
		if sensorStream.today != nil {
			count := sensorStream.today.isModified.Load()
			if count > 0 {
				sensorStream.today.isModified.Add(-count)
				s.writeChannel <- sensorStream.today
			}
		}
		sensorStream.yesterday = sensorStream.today
		sampleFreq := GetSampleFrequencyFromSensorType(sensorStream.sensorType)
		samplePeriod := GetSamplePeriodInMsFromSensorType(sensorStream.sensorType)
		sensorStream.today = newSensorStreamBlock(sensorStream, SensorFieldType(sensorStream.sensorType), sampleFreq, samplePeriod, nil)
		s.flushChannel <- sensorStream.today
	}
}
