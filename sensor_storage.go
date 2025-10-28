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

	"github.com/jurgen-kluft/go-sensor-server/logging"
)

// Storage for sensor data
// One sensor store for [DeviceLocation | SensorType]
// DeviceLocation is associated with
type sensorStream struct {
	sensor    *SensorConfig      // Sensor for this stream
	reference time.Time          // Reference time for this stream
	today     *sensorStreamBlock // Data block for today
	yesterday *sensorStreamBlock // Data block for yesterday
}

func newSensorStream(sensor *SensorConfig) *sensorStream {
	return &sensorStream{
		sensor:    sensor,
		reference: time.Time{},
		today:     nil,
		yesterday: nil,
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
	Stream     *sensorStream // The parent stream
	Time       time.Time     // Time (Year, Month, Day, at zero hour)
	LastSample SensorValue   // Last sample value
	Content    []byte        // Content to hold the samples
	isModified atomic.Int32  // Indicates if the block has been modified
}

const (
	SensorDataBlockHeaderSize = 64 // Size of the header in bytes
)

func newSensorStreamBlock(stream *sensorStream, content []byte) *sensorStreamBlock {
	// Construct the correct time for the start of this block
	t := time.Now()
	blockTime := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())

	// TODO This datablock might already exist on disk, load it instead of creating a new one.
	//      This can happen when the server crashes and/or restarts during the day.
	sensor := stream.sensor
	blockSizeInBytes := sensor.NumberOfSamplesPerHour() * int32(sensor.SizeInBits())
	blockSizeInBytes = blockSizeInBytes + SensorDataBlockHeaderSize
	block := &sensorStreamBlock{
		Stream:     stream,
		Time:       blockTime,
		LastSample: SensorValue{},
		Content:    nil,
		isModified: atomic.Int32{},
	}

	if content != nil && len(content) == int(blockSizeInBytes) {
		block.Content = content
	}

	return block
}

func (s *sensorStreamBlock) createContentBuffer() {
	if s.Content != nil {
		return // Already created
	}

	sensor := s.Stream.sensor

	blockSizeInBytes := sensor.NumberOfSamplesPerHour() * int32(sensor.SizeInBits())
	blockSizeInBytes = (blockSizeInBytes + 7) / 8 // Round up to full bytes
	blockSizeInBytes = blockSizeInBytes + SensorDataBlockHeaderSize
	s.Content = make([]byte, blockSizeInBytes)

	// Write the header
	binary.LittleEndian.PutUint16(s.Content, uint16(s.Time.Year()))                                 // year
	s.Content[2] = uint8(s.Time.Month())                                                            // month
	s.Content[3] = uint8(s.Time.Day())                                                              // day
	binary.LittleEndian.PutUint64(s.Content[4:], uint64(sensor.FullIdentifier()))                   // full sensor identifier
	binary.LittleEndian.PutUint64(s.Content[12:], uint64(0))                                        // flags (compressed or not)
	binary.LittleEndian.PutUint32(s.Content[20:], uint32(len(s.Content)-SensorDataBlockHeaderSize)) // data length
	// Note: The rest of the header is reserved (zeroed)                                            // reserved (36 bytes)
}

func (s *sensorStreamBlock) WriteSensorValue(sessionReferenceTime time.Time, sensorValue SensorValue) error {
	s.LastSample = sensorValue
	if sensorValue.IsZero() {
		return nil // Default value
	}

	sampleTime := time.Now()

	sensor := s.Stream.sensor

	var sampleIndex int32
	if idx, ok := sensor.ToSampleIndex(sessionReferenceTime, sampleTime); ok {
		sampleIndex = idx
	} else {
		// Sample is out of range for this block, ignore it
		return fmt.Errorf("sample index out of range for this block")
	}

	// Write the sensor value to the buffer based on its type
	switch sensor.FieldType() {
	case FieldTypeU1:
		byteIndex := SensorDataBlockHeaderSize + (sampleIndex / 8)
		bitIndex := sampleIndex % 8
		s.Content[byteIndex] |= (1 << bitIndex) // Set the bit
	case FieldTypeU2:
		byteIndex := SensorDataBlockHeaderSize + (sampleIndex / 4)
		bitIndex := sampleIndex % 4
		s.Content[byteIndex] |= byte((sensorValue.Value & 3) << bitIndex) // Set the bit
	case FieldTypeS8:
		byteIndex := SensorDataBlockHeaderSize + sampleIndex
		s.Content[byteIndex] = byte(sensorValue.Value)
	case FieldTypeS16:
		byteIndex := SensorDataBlockHeaderSize + sampleIndex*2
		binary.LittleEndian.PutUint16(s.Content[byteIndex:byteIndex+2], uint16(sensorValue.Value))
	case FieldTypeS32:
		byteIndex := SensorDataBlockHeaderSize + sampleIndex*4
		binary.LittleEndian.PutUint32(s.Content[byteIndex:byteIndex+4], uint32(sensorValue.Value))
	case FieldTypeS64:
		byteIndex := SensorDataBlockHeaderSize + sampleIndex*8
		binary.LittleEndian.PutUint64(s.Content[byteIndex:byteIndex+8], uint64(sensorValue.Value))
	case FieldTypeU8:
		byteIndex := SensorDataBlockHeaderSize + sampleIndex
		s.Content[byteIndex] = byte(sensorValue.Value)
	case FieldTypeU16:
		byteIndex := SensorDataBlockHeaderSize + sampleIndex*2
		binary.LittleEndian.PutUint16(s.Content[byteIndex:byteIndex+2], uint16(sensorValue.Value))
	case FieldTypeU32:
		byteIndex := SensorDataBlockHeaderSize + sampleIndex*4
		binary.LittleEndian.PutUint32(s.Content[byteIndex:byteIndex+4], uint32(sensorValue.Value))
	case FieldTypeU64:
		byteIndex := SensorDataBlockHeaderSize + sampleIndex*8
		binary.LittleEndian.PutUint64(s.Content[byteIndex:byteIndex+8], uint64(sensorValue.Value))
	}

	// Indicate that this block has been modified
	s.isModified.Add(1)

	return nil
}

type SensorStorage struct {
	config            *SensorServerConfig     // Configuration for the sensor storage
	logger            logging.Logger          // Logger for logging messages
	sensorList        []*SensorConfig         // List of sensors
	sensorStreams     []*sensorStream         // List of sensor streams indexed by sensor index
	writeChannel      chan *sensorStreamBlock // Channel for writing data blocks
	blockHeaderBuffer bytes.Buffer            //
	blockHeaderWriter io.Writer               //
	flushTicker       *time.Ticker            // Ticker for periodic flushing
	flushLock         *sync.Mutex             // Mutex for synchronizing queue access
	flushChannel      chan *sensorStreamBlock // Channel for flushing data blocks
	flushQueue        []*sensorStreamBlock    // Queue of blocks to flush
}

func NewSensorStorage(config *SensorServerConfig, logger logging.Logger) *SensorStorage {
	storage := &SensorStorage{
		config:            config,
		logger:            logger,
		sensorList:        config.Sensors,
		sensorStreams:     make([]*sensorStream, 65536),
		writeChannel:      make(chan *sensorStreamBlock, 512),
		blockHeaderBuffer: bytes.Buffer{},
		blockHeaderWriter: nil,
		flushTicker:       nil,
		flushChannel:      make(chan *sensorStreamBlock, 128),
		flushQueue:        make([]*sensorStreamBlock, 0, 256),
	}
	storage.blockHeaderWriter = &storage.blockHeaderBuffer

	dirpath := config.StoragePath
	// resolve any environment variables in the path
	dirpath = os.ExpandEnv(dirpath)
	// Create the storage directory if it does not exist
	if _, err := os.Stat(dirpath); os.IsNotExist(err) {
		err := os.MkdirAll(dirpath, 0755)
		if err != nil {
			storage.logger.LogError(err, "failed to create storage directory")
			return nil
		}
	}

	// Create subdirectories for each device and sensor type
	for _, sensor := range storage.config.Sensors {
		sensorStreamDirPath := path.Join(dirpath, sensor.Name())
		if _, err := os.Stat(sensorStreamDirPath); os.IsNotExist(err) {
			err := os.MkdirAll(sensorStreamDirPath, 0755)
			if err != nil {
				storage.logger.LogError(err, "failed to create sensor stream directory")
				return nil
			}
		}
	}

	// Initialize the sensor stream for each sensor
	for _, sensor := range storage.config.Sensors {
		if !sensor.IsValid() {
			continue
		}

		sensorStream := newSensorStream(sensor)
		storage.sensorStreams[sensor.Index()] = sensorStream
		storage.setupSensorStream(sensorStream)
	}

	// The go-routine that writes data blocks to the file system, receives blocks on the writeChannel
	go processWrite(storage)

	storage.logger.LogInfof("Periodic flush interval: %d seconds", config.FlushPeriodInSeconds)
	schedulePeriodicFlush(storage, time.Duration(config.FlushPeriodInSeconds)*time.Second)

	storage.logger.LogInfof("Sensor storage initialized, storage path: %s", dirpath)
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
	for _, sensorStream := range s.sensorStreams {
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

	s.writeChannel <- nil       // End the flush go-routine
	time.Sleep(5 * time.Second) // Give some time for the write go-routine to finish
	close(s.writeChannel)       // Close the write channel
}

func processWrite(s *SensorStorage) {
	for block := range s.writeChannel {
		if block == nil {
			s.logger.LogInfo("go-routine for writing stream data blocks has exited")
			return // Exit the go-routine
		}
		if err := writeStreamBlock(s, block); err != nil {
			s.logger.LogError(err, "writing data block")
		}
	}
}

func processFlush(s *SensorStorage) {
	now := time.Now()

	toWrite := 0
	toNil := 0
	for i, sensorStream := range s.flushQueue {
		count := sensorStream.isModified.Load()
		if count > 0 {
			sensorStream.isModified.Add(-count)
			if toWrite == 0 {
				s.logger.LogInfof("Processing flush queue")
			}
			toWrite += 1
			s.writeChannel <- sensorStream
		}

		// Remove from queue if older than 2 days
		if now.Sub(sensorStream.Time) >= 48*time.Hour {
			toNil += 1
			s.flushQueue[i] = nil
		}
	}

	if toWrite > 0 {
		s.logger.LogInfof("Processed flush queue, writing %d streams to disk", toWrite)
	}

	if toNil > 0 {
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
					s.logger.LogInfo("go-routine for flushing stream data blocks has exited")
					return // Exit the go-routine
				}
				s.flushLock.Lock()
				s.flushQueue = append(s.flushQueue, block)
				s.flushLock.Unlock()
			case <-s.flushTicker.C:
				s.flushLock.Lock()
				processFlush(s)
				s.flushLock.Unlock()
			}
		}
	}()
}

func (s *SensorStorage) WriteSensorValue(sensorConfig *SensorConfig, sensorValue SensorValue, sensorTime time.Time) error {

	// Create or get the data block for the given location and sensor type
	sensorIndex := sensorConfig.Index()
	if sensorIndex < 0 || sensorIndex >= len(s.sensorStreams) || s.sensorStreams[sensorIndex] == nil {
		return fmt.Errorf("invalid sensor %s, index %d", sensorConfig.Name(), sensorIndex)
	}
	sensorStream := s.sensorStreams[sensorIndex]

	s.logger.LogInfof("WriteSensorValue: sensor=%s, value=%d", sensorConfig.Name(), sensorValue.Value)

	// Update the sensor stream (check if day has changed, etc..)
	s.activeSensorStream(sensorStream, time.Now())

	if sensorTime.Before(sensorStream.today.Time) {
		// If packet is really old (which should not happen) then we drop it
		if sensorTime.Before(sensorStream.yesterday.Time) {
			return fmt.Errorf("sensor data time is before yesterday block, dropping packet for sensor %s with index %d", sensorConfig.Name(), sensorConfig.Index())
		}
		// Write to the yesterday block if it exists and the time fits
		if sensorTime.After(sensorStream.yesterday.Time) {
			if sensorStream.yesterday.Content == nil {
				sensorStream.yesterday.createContentBuffer()
				s.flushChannel <- sensorStream.yesterday
			}
			sensorStream.yesterday.WriteSensorValue(sensorStream.reference, sensorValue)
		}
	} else {
		// Write the sensor value to the today block's buffer
		if sensorStream.today.Content == nil {
			sensorStream.today.createContentBuffer()
			s.flushChannel <- sensorStream.today
		}
		sensorStream.today.WriteSensorValue(sensorStream.reference, sensorValue)
	}

	return nil
}

// The go-routine for writing data blocks to the file system
func (s *SensorStorage) getStreamBlockFilepath(stream *sensorStream, time time.Time) string {
	storePathFilename := fmt.Sprintf("%04d_%02d_%02d.data", time.Year(), time.Month(), time.Day())
	storeFullFilepath := path.Join(s.config.StoragePath, stream.sensor.Name(), storePathFilename)
	storeFullFilepath = os.ExpandEnv(storeFullFilepath)
	return storeFullFilepath
}

func writeStreamBlock(s *SensorStorage, stream *sensorStreamBlock) error {
	// Check if the stream is valid and has data
	if stream == nil || len(stream.Content) == 0 {
		return nil // Nothing to write
	}

	storeFullFilepath := s.getStreamBlockFilepath(stream.Stream, stream.Time)
	storeFullFilepathCurrent := storeFullFilepath + ".current"

	// If 'storeFullFilepath' exists, rename it to 'storeFullFilepath'.current
	if stat, err := os.Stat(storeFullFilepath); err == nil && !stat.IsDir() {
		if stat, err := os.Stat(storeFullFilepathCurrent); err == nil && !stat.IsDir() {
			os.Remove(storeFullFilepathCurrent) // Remove current file if it exists
		}
		os.Rename(storeFullFilepath, storeFullFilepathCurrent)
	}

	// Write the data block to 'storeFullFilepath'
	file, err := os.OpenFile(storeFullFilepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// If 'storeFullFilepath'.backup exists, remove it and rename 'storeFullFilepath'.current to 'storeFullFilepath'.backup after successful write
	{
		if stat, err := os.Stat(storeFullFilepathCurrent); err == nil && !stat.IsDir() {
			storeFullFilepathBackup := storeFullFilepath + ".backup"
			if stat, err := os.Stat(storeFullFilepathBackup); err == nil && !stat.IsDir() {
				os.Remove(storeFullFilepathBackup) // Remove backup
			}
			os.Rename(storeFullFilepathCurrent, storeFullFilepathBackup)
		}
	}

	s.logger.LogInfof("Writing data block to file: %s", storeFullFilepath)

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
	today := time.Now()
	if currentContent, err := s.loadSensorDataBlockContent(sensorStream, today); err == nil {
		sensorStream.today = newSensorStreamBlock(sensorStream, currentContent)
	} else {
		sensorStream.today = newSensorStreamBlock(sensorStream, nil)
	}

	if sensorStream.today.Content != nil {
		s.flushChannel <- sensorStream.today
	}

	yesterday := today.AddDate(0, 0, -1)
	if yesterdayContent, err := s.loadSensorDataBlockContent(sensorStream, yesterday); err == nil {
		sensorStream.yesterday = newSensorStreamBlock(sensorStream, yesterdayContent)
	} else {
		sensorStream.yesterday = newSensorStreamBlock(sensorStream, nil)
	}

	if sensorStream.yesterday.Content != nil {
		s.flushChannel <- sensorStream.yesterday
	}

	return nil
}

func (s *SensorStorage) activeSensorStream(sensorStream *sensorStream, today time.Time) {

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
		sensorStream.today = newSensorStreamBlock(sensorStream, nil)
		s.flushChannel <- sensorStream.today
	} else {
		if sensorStream.today.Content == nil {
			sensorStream.today.createContentBuffer()
			s.flushChannel <- sensorStream.today
		}
	}
}
