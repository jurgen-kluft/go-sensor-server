package sensorserver

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
)

const firstSegmentSequence uint64 = 1

var ErrDataFileClosed = errors.New("data file is closed")

type DataFileOptions struct {
	Directory     string
	BufferSize    int
	RotationSize  int64
	DirectoryMode fs.FileMode
	FileMode      fs.FileMode
}

type DataFileCounters struct {
	BytesWritten uint64
	Rotations    uint64
}

func DefaultDataFileOptions(directory string) DataFileOptions {
	return DataFileOptions{
		Directory:     directory,
		BufferSize:    DefaultWriteBufferSize,
		RotationSize:  DefaultRotationSize,
		DirectoryMode: 0o755,
		FileMode:      0o644,
	}
}

// DataFile appends fixed-size sensor records to numbered segments. It is not
// safe for concurrent use; a Data Stream must provide single-owner access.
type DataFile struct {
	fileSystem   FileSystem
	options      DataFileOptions
	file         File
	buffer       *bufio.Writer
	sequence     uint64
	size         int64
	closed       bool
	bytesWritten atomic.Uint64
	rotations    atomic.Uint64
}

func OpenDataFile(fileSystem FileSystem, options DataFileOptions) (*DataFile, error) {
	if fileSystem == nil {
		return nil, errors.New("open data file: nil filesystem")
	}
	if strings.TrimSpace(options.Directory) == "" {
		return nil, errors.New("open data file: empty directory")
	}
	if options.BufferSize < 1 {
		return nil, errors.New("open data file: buffer size must be positive")
	}
	if options.RotationSize < SensorDataRecordSize {
		return nil, errors.New("open data file: rotation size must fit one record")
	}
	if options.DirectoryMode == 0 {
		options.DirectoryMode = 0o755
	}
	if options.FileMode == 0 {
		options.FileMode = 0o644
	}
	if err := fileSystem.MkdirAll(options.Directory, options.DirectoryMode); err != nil {
		return nil, fmt.Errorf("create data directory %q: %w", options.Directory, err)
	}

	sequence, size, err := discoverAndRepairSegments(fileSystem, options)
	if err != nil {
		return nil, err
	}
	dataFile := &DataFile{fileSystem: fileSystem, options: options, sequence: sequence, size: size}
	if err := dataFile.openCurrentSegment(); err != nil {
		return nil, err
	}
	return dataFile, nil
}

func (dataFile *DataFile) WriteRecord(timestamp int64, value int16) error {
	if dataFile.closed {
		return ErrDataFileClosed
	}
	if dataFile.size+SensorDataRecordSize > dataFile.options.RotationSize {
		if err := dataFile.rotate(); err != nil {
			return err
		}
	}

	var encoded [SensorDataRecordSize]byte
	binary.LittleEndian.PutUint64(encoded[0:8], uint64(timestamp))
	binary.LittleEndian.PutUint16(encoded[8:10], uint16(value))
	if _, err := dataFile.buffer.Write(encoded[:]); err != nil {
		return fmt.Errorf("write data segment %q: %w", dataFile.segmentPath(), err)
	}
	dataFile.size += SensorDataRecordSize
	dataFile.bytesWritten.Add(uint64(SensorDataRecordSize))
	return nil
}

func (dataFile *DataFile) Counters() DataFileCounters {
	return DataFileCounters{BytesWritten: dataFile.bytesWritten.Load(), Rotations: dataFile.rotations.Load()}
}

func (dataFile *DataFile) Flush() error {
	if dataFile.closed {
		return ErrDataFileClosed
	}
	if err := dataFile.buffer.Flush(); err != nil {
		return fmt.Errorf("flush data segment %q: %w", dataFile.segmentPath(), err)
	}
	return nil
}

func (dataFile *DataFile) Close() error {
	if dataFile.closed {
		return nil
	}
	dataFile.closed = true
	return dataFile.closeCurrentSegment()
}

func (dataFile *DataFile) rotate() error {
	if err := dataFile.closeCurrentSegment(); err != nil {
		return err
	}
	dataFile.sequence++
	dataFile.rotations.Add(1)
	dataFile.size = 0
	if err := dataFile.openCurrentSegment(); err != nil {
		return fmt.Errorf("rotate data segment: %w", err)
	}
	return nil
}

func (dataFile *DataFile) openCurrentSegment() error {
	file, err := dataFile.fileSystem.OpenFile(dataFile.segmentPath(), createAppendReadWriteFlags, dataFile.options.FileMode)
	if err != nil {
		return fmt.Errorf("open data segment %q: %w", dataFile.segmentPath(), err)
	}
	dataFile.file = file
	dataFile.buffer = bufio.NewWriterSize(file, dataFile.options.BufferSize)
	return nil
}

func (dataFile *DataFile) closeCurrentSegment() error {
	if dataFile.file == nil {
		return nil
	}
	var result error
	if dataFile.buffer != nil {
		result = errors.Join(result, dataFile.buffer.Flush())
	}
	result = errors.Join(result, dataFile.file.Sync())
	result = errors.Join(result, dataFile.file.Close())
	dataFile.file = nil
	dataFile.buffer = nil
	if result != nil {
		return fmt.Errorf("close data segment %q: %w", dataFile.segmentPath(), result)
	}
	return nil
}

func (dataFile *DataFile) segmentPath() string {
	return filepath.Join(dataFile.options.Directory, segmentName(dataFile.sequence))
}

func discoverAndRepairSegments(fileSystem FileSystem, options DataFileOptions) (uint64, int64, error) {
	entries, err := fileSystem.ReadDir(options.Directory)
	if err != nil {
		return 0, 0, fmt.Errorf("read data directory %q: %w", options.Directory, err)
	}
	sequence := firstSegmentSequence
	found := false
	var activeSize int64
	for _, entry := range entries {
		entrySequence, ok := parseSegmentName(entry.Name())
		if !ok || entry.IsDir() {
			continue
		}
		path := filepath.Join(options.Directory, entry.Name())
		info, err := fileSystem.Stat(path)
		if err != nil {
			return 0, 0, fmt.Errorf("stat data segment %q: %w", path, err)
		}
		size := info.Size()
		if remainder := size % SensorDataRecordSize; remainder != 0 {
			repairedSize := size - remainder
			file, err := fileSystem.OpenFile(path, readWriteFlags, options.FileMode)
			if err != nil {
				return 0, 0, fmt.Errorf("open data segment %q for repair: %w", path, err)
			}
			if err := file.Truncate(repairedSize); err != nil {
				_ = file.Close()
				return 0, 0, fmt.Errorf("truncate data segment %q: %w", path, err)
			}
			if err := errors.Join(file.Sync(), file.Close()); err != nil {
				return 0, 0, fmt.Errorf("close repaired data segment %q: %w", path, err)
			}
			size = repairedSize
		}
		if !found || entrySequence > sequence {
			found = true
			sequence = entrySequence
			activeSize = size
		}
	}
	if !found {
		return firstSegmentSequence, 0, nil
	}
	return sequence, activeSize, nil
}

func segmentName(sequence uint64) string {
	return fmt.Sprintf("%08d.dat", sequence)
}

func parseSegmentName(name string) (uint64, bool) {
	if len(name) != len("00000001.dat") || !strings.HasSuffix(name, ".dat") {
		return 0, false
	}
	sequence, err := strconv.ParseUint(name[:8], 10, 64)
	return sequence, err == nil && sequence >= firstSegmentSequence
}
