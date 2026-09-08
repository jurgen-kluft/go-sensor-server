package sensorserver

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const (
	cQuarantineMagic        uint32 = 0x514D5347
	cQuarantineVersion      uint16 = 1
	cQuarantineHeaderSize          = 30
	cQuarantineTrailerSize         = 4
	cQuarantineEntryMinimum        = cQuarantineHeaderSize + cQuarantineTrailerSize
)

var (
	ErrInvalidQuarantineEntry = errors.New("invalid quarantine entry")
	ErrQuarantineClosed       = errors.New("quarantine writer is closed")
)

type QuarantineOptions struct {
	Directory     string
	RotationSize  int64
	DirectoryMode fs.FileMode
	FileMode      fs.FileMode
}

func DefaultQuarantineOptions(directory string) QuarantineOptions {
	return QuarantineOptions{
		Directory:     directory,
		RotationSize:  DefaultRotationSize,
		DirectoryMode: 0o755,
		FileMode:      0o644,
	}
}

type QuarantineWriter struct {
	fileSystem FileSystem
	options    QuarantineOptions
	mu         sync.Mutex
	file       File
	sequence   uint64
	size       int64
	closed     bool
}

func OpenQuarantineWriter(fileSystem FileSystem, options QuarantineOptions) (*QuarantineWriter, error) {
	if fileSystem == nil {
		return nil, errors.New("open quarantine: nil filesystem")
	}
	if strings.TrimSpace(options.Directory) == "" {
		return nil, errors.New("open quarantine: empty directory")
	}
	if options.RotationSize < cQuarantineEntryMinimum {
		return nil, errors.New("open quarantine: rotation size is too small")
	}
	if options.DirectoryMode == 0 {
		options.DirectoryMode = 0o755
	}
	if options.FileMode == 0 {
		options.FileMode = 0o644
	}
	if err := fileSystem.MkdirAll(options.Directory, options.DirectoryMode); err != nil {
		return nil, fmt.Errorf("create quarantine directory %q: %w", options.Directory, err)
	}

	segments, err := discoverQuarantineSegments(fileSystem, options.Directory)
	if err != nil {
		return nil, err
	}
	for _, segment := range segments {
		if err := repairQuarantineTail(fileSystem, segment.path, options.FileMode); err != nil {
			return nil, err
		}
	}
	sequence := firstSegmentSequence
	if len(segments) > 0 {
		sequence = segments[len(segments)-1].sequence + 1
	}
	writer := &QuarantineWriter{fileSystem: fileSystem, options: options, sequence: sequence}
	if err := writer.openSegment(); err != nil {
		return nil, err
	}
	return writer, nil
}

func (writer *QuarantineWriter) WriteUnknownMessage(ctx context.Context, message UnknownMessage) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	encoded, err := encodeQuarantineEntry(message)
	if err != nil {
		return err
	}

	writer.mu.Lock()
	defer writer.mu.Unlock()
	if writer.closed {
		return ErrQuarantineClosed
	}
	if writer.size > 0 && writer.size+int64(len(encoded)) > writer.options.RotationSize {
		if err := writer.rotate(); err != nil {
			return err
		}
	}
	if _, err := writer.file.Write(encoded); err != nil {
		return fmt.Errorf("write quarantine segment %q: %w", writer.segmentPath(), err)
	}
	writer.size += int64(len(encoded))
	return nil
}

func (writer *QuarantineWriter) Close() error {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	if writer.closed {
		return nil
	}
	writer.closed = true
	return writer.closeSegment()
}

func (writer *QuarantineWriter) rotate() error {
	if err := writer.closeSegment(); err != nil {
		return err
	}
	writer.sequence++
	writer.size = 0
	return writer.openSegment()
}

func (writer *QuarantineWriter) openSegment() error {
	file, err := writer.fileSystem.OpenFile(writer.segmentPath(), createAppendReadWriteFlags, writer.options.FileMode)
	if err != nil {
		return fmt.Errorf("open quarantine segment %q: %w", writer.segmentPath(), err)
	}
	writer.file = file
	return nil
}

func (writer *QuarantineWriter) closeSegment() error {
	if writer.file == nil {
		return nil
	}
	result := errors.Join(writer.file.Sync(), writer.file.Close())
	writer.file = nil
	if result != nil {
		return fmt.Errorf("close quarantine segment %q: %w", writer.segmentPath(), result)
	}
	return nil
}

func (writer *QuarantineWriter) segmentPath() string {
	return filepath.Join(writer.options.Directory, quarantineSegmentName(writer.sequence))
}

type ReplayCounters struct {
	SegmentsCompleted uint64
	MessagesReplayed  uint64
	MessagesCarried   uint64
	RecordsWritten    uint64
}

// ReplayQuarantine processes immutable segments before network listeners start.
// The supplied writer receives messages whose MAC remains unknown.
func ReplayQuarantine(ctx context.Context, writer *QuarantineWriter, registry *ConfigRegistry, dataWriter SensorDataWriter) (ReplayCounters, error) {
	if writer == nil || registry == nil || dataWriter == nil {
		return ReplayCounters{}, errors.New("replay quarantine: nil dependency")
	}
	segments, err := discoverQuarantineSegments(writer.fileSystem, writer.options.Directory)
	if err != nil {
		return ReplayCounters{}, err
	}
	var counters ReplayCounters
	for _, segment := range segments {
		if segment.sequence >= writer.sequence || quarantineSegmentComplete(writer.fileSystem, segment.path) {
			continue
		}
		entries, err := readQuarantineEntries(writer.fileSystem, segment.path)
		if err != nil {
			return counters, err
		}
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return counters, err
			}
			snapshot := registry.Snapshot()
			device, known := snapshot.Device(entry.Message.Header.MAC)
			if !known {
				if err := writer.WriteUnknownMessage(ctx, entry); err != nil {
					return counters, err
				}
				counters.MessagesCarried++
				continue
			}
			for _, record := range entry.Message.Sensors {
				sensor, ok := snapshot.Sensor(record.SensorType)
				if !ok {
					continue
				}
				if err := dataWriter.WriteSensorData(ctx, device.Floor, device.Room, sensor.Type, entry.Timestamp, record.Value); err != nil {
					return counters, fmt.Errorf("replay %q: %w", segment.path, err)
				}
				counters.RecordsWritten++
			}
			counters.MessagesReplayed++
		}
		if err := markQuarantineSegmentComplete(writer.fileSystem, segment.path, writer.options.FileMode); err != nil {
			return counters, err
		}
		counters.SegmentsCompleted++
	}
	return counters, nil
}

func encodeQuarantineEntry(message UnknownMessage) ([]byte, error) {
	if message.Transport != TransportTCP && message.Transport != TransportUDP {
		return nil, fmt.Errorf("encode quarantine: transport %d: %w", message.Transport, ErrInvalidQuarantineEntry)
	}
	payload := message.Message.Payload
	if len(payload) != int(message.Message.Header.PayloadLength) || len(payload) > MaximumPayloadSize {
		return nil, fmt.Errorf("encode quarantine: payload length: %w", ErrInvalidQuarantineEntry)
	}
	totalLength := cQuarantineEntryMinimum + len(payload)

	encoded := make([]byte, totalLength)

	binaryData := &BinaryData{buf: encoded, off: 0}
	binaryData.WriteUint32(cQuarantineMagic)
	binaryData.WriteUint16(cQuarantineVersion)
	binaryData.WriteUint32(0) // placeholder for CRC
	binaryData.WriteUint32(uint32(totalLength))
	binaryData.WriteUint64(uint64(message.Timestamp))
	binaryData.WriteUint8(byte(message.Transport))
	binaryData.WriteUint8(0)
	binaryData.WriteUint16(uint16(message.Message.Header.Type))
	binaryData.WriteBytes(message.Message.Header.MAC[:])
	binaryData.WriteUint16(uint16(len(payload)))
	binaryData.WriteBytes(payload)

	// Compute CRC and write it into the placeholder
	crc := crc32.ChecksumIEEE(encoded)
	binaryData.WriteUint32At(crc, 6)

	return encoded, nil
}

func decodeQuarantineEntry(encoded []byte) (UnknownMessage, error) {
	if len(encoded) < cQuarantineEntryMinimum {
		return UnknownMessage{}, ErrInvalidQuarantineEntry
	}

	binaryData := &BinaryData{buf: encoded, off: 0}
	quarantineMagic := binaryData.ReadUint32()
	quarantineVersion := binaryData.ReadUint16()
	wantedCRC := binaryData.ReadUint32()
	encodedLength := binaryData.ReadUint32()
	timeStamp := binaryData.ReadUint64()
	transport := Transport(binaryData.ReadUint8())
	_ = binaryData.ReadUint8() // reserved
	headerType := MessageType(binaryData.ReadUint16())
	mac := binaryData.ReadMAC() // mac
	payloadLength := binaryData.ReadUint16()

	if quarantineMagic != cQuarantineMagic || quarantineVersion != cQuarantineVersion {
		return UnknownMessage{}, ErrInvalidQuarantineEntry
	}
	if encodedLength != uint32(len(encoded)) || int(payloadLength) > MaximumPayloadSize {
		return UnknownMessage{}, ErrInvalidQuarantineEntry
	}
	if transport != TransportTCP && transport != TransportUDP {
		return UnknownMessage{}, ErrInvalidQuarantineEntry
	}

	// Validate CRC, for this the CRC field must be set to zero before computing the checksum
	binaryData.WriteUint32At(0, 6)
	if crc32.ChecksumIEEE(encoded) != wantedCRC {
		return UnknownMessage{}, fmt.Errorf("quarantine checksum: %w", ErrInvalidQuarantineEntry)
	}

	payload := binaryData.ReadBytes(int(payloadLength))

	header := MessageHeader{
		Magic:         MessageMagic,
		Type:          headerType,
		PayloadLength: uint16(payloadLength),
		MAC:           mac,
		Checksum:      crc32.ChecksumIEEE(payload),
	}
	message, err := DecodeMessage(header, payload)
	if err != nil {
		return UnknownMessage{}, fmt.Errorf("decode quarantined message: %w", err)
	}
	return UnknownMessage{Timestamp: int64(timeStamp), Transport: transport, Message: message}, nil
}

type quarantineSegment struct {
	sequence uint64
	path     string
}

func discoverQuarantineSegments(fileSystem FileSystem, directory string) ([]quarantineSegment, error) {
	entries, err := fileSystem.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("read quarantine directory %q: %w", directory, err)
	}
	segments := make([]quarantineSegment, 0)
	for _, entry := range entries {
		sequence, ok := parseQuarantineSegmentName(entry.Name())
		if ok && !entry.IsDir() {
			segments = append(segments, quarantineSegment{sequence: sequence, path: filepath.Join(directory, entry.Name())})
		}
	}
	sort.Slice(segments, func(left, right int) bool { return segments[left].sequence < segments[right].sequence })
	return segments, nil
}

func repairQuarantineTail(fileSystem FileSystem, path string, mode fs.FileMode) error {
	data, err := readAllFile(fileSystem, path)
	if err != nil {
		return err
	}
	validLength, err := scanQuarantineEntries(data)
	if err != nil {
		return fmt.Errorf("scan quarantine segment %q: %w", path, err)
	}
	if validLength == len(data) {
		return nil
	}
	file, err := fileSystem.OpenFile(path, readWriteFlags, mode)
	if err != nil {
		return err
	}
	return errors.Join(file.Truncate(int64(validLength)), file.Sync(), file.Close())
}

func readQuarantineEntries(fileSystem FileSystem, path string) ([]UnknownMessage, error) {
	data, err := readAllFile(fileSystem, path)
	if err != nil {
		return nil, err
	}
	entries := make([]UnknownMessage, 0)
	for offset := 0; offset < len(data); {
		length := int(binary.LittleEndian.Uint32(data[offset+10 : offset+14]))
		entry, err := decodeQuarantineEntry(data[offset : offset+length])
		if err != nil {
			return nil, fmt.Errorf("decode quarantine entry at %d: %w", offset, err)
		}
		entries = append(entries, entry)
		offset += length
	}
	return entries, nil
}

func scanQuarantineEntries(data []byte) (int, error) {
	for offset := 0; offset < len(data); {
		remaining := len(data) - offset
		if remaining < cQuarantineHeaderSize {
			return offset, nil
		}
		if binary.LittleEndian.Uint32(data[offset:offset+4]) != cQuarantineMagic {
			return 0, ErrInvalidQuarantineEntry
		}
		length := int(binary.LittleEndian.Uint32(data[offset+10 : offset+14]))
		if length < cQuarantineEntryMinimum || length > cQuarantineEntryMinimum+MaximumPayloadSize {
			return 0, ErrInvalidQuarantineEntry
		}
		if length > remaining {
			return offset, nil
		}
		if _, err := decodeQuarantineEntry(data[offset : offset+length]); err != nil {
			return 0, err
		}
		offset += length
	}
	return len(data), nil
}

func readAllFile(fileSystem FileSystem, path string) ([]byte, error) {
	file, err := fileSystem.OpenFile(path, readWriteFlags, 0)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", path, err)
	}
	data, readErr := io.ReadAll(file)
	return data, errors.Join(readErr, file.Close())
}

func markQuarantineSegmentComplete(fileSystem FileSystem, segmentPath string, mode fs.FileMode) error {
	temporary := segmentPath + ".done.tmp"
	marker := segmentPath + ".done"
	file, err := fileSystem.OpenFile(temporary, createTruncateReadWriteFlags, mode)
	if err != nil {
		return err
	}
	if _, err := file.Write([]byte("complete\n")); err != nil {
		_ = file.Close()
		return err
	}
	if err := errors.Join(file.Sync(), file.Close()); err != nil {
		return err
	}
	return fileSystem.Rename(temporary, marker)
}

func quarantineSegmentComplete(fileSystem FileSystem, segmentPath string) bool {
	_, err := fileSystem.Stat(segmentPath + ".done")
	return err == nil
}

func quarantineSegmentName(sequence uint64) string {
	return fmt.Sprintf("quarantine-%08d.bin", sequence)
}

func parseQuarantineSegmentName(name string) (uint64, bool) {
	const prefix = "quarantine-"
	const suffix = ".bin"
	if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) || len(name) != len(prefix)+8+len(suffix) {
		return 0, false
	}
	sequence, err := strconv.ParseUint(name[len(prefix):len(prefix)+8], 10, 64)
	return sequence, err == nil && sequence >= firstSegmentSequence
}
