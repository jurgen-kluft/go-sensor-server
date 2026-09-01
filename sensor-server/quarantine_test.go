package sensorserver

import (
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestQuarantineEntryRoundTrip(t *testing.T) {
	original := quarantineTestMessage(t, MACAddress{1, 2, 3, 4, 5, 6}, 42, TransportUDP)
	encoded, err := encodeQuarantineEntry(original)
	if err != nil {
		t.Fatalf("encodeQuarantineEntry() error = %v", err)
	}
	decoded, err := decodeQuarantineEntry(encoded)
	if err != nil {
		t.Fatalf("decodeQuarantineEntry() error = %v", err)
	}
	if decoded.Timestamp != original.Timestamp || decoded.Transport != original.Transport || decoded.Message.Header.MAC != original.Message.Header.MAC {
		t.Fatalf("decoded entry = %+v", decoded)
	}
	if len(decoded.Message.Sensors) != 1 || decoded.Message.Sensors[0].Value != 42 {
		t.Fatalf("decoded sensors = %+v", decoded.Message.Sensors)
	}
}

func TestQuarantineWriterRepairsTornTail(t *testing.T) {
	directory := t.TempDir()
	options := DefaultQuarantineOptions(directory)
	writer, err := OpenQuarantineWriter(OSFileSystem{}, options)
	if err != nil {
		t.Fatalf("OpenQuarantineWriter() error = %v", err)
	}
	if err := writer.WriteUnknownMessage(context.Background(), quarantineTestMessage(t, MACAddress{1}, 1, TransportTCP)); err != nil {
		t.Fatalf("WriteUnknownMessage() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	path := filepath.Join(directory, quarantineSegmentName(1))
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	validSize := info.Size()
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	if _, err := file.Write([]byte{0x47, 0x53, 0x4d}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close(torn file) error = %v", err)
	}

	reopened, err := OpenQuarantineWriter(OSFileSystem{}, options)
	if err != nil {
		t.Fatalf("OpenQuarantineWriter(repair) error = %v", err)
	}
	defer reopened.Close()
	info, err = os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(repaired) error = %v", err)
	}
	if info.Size() != validSize {
		t.Fatalf("repaired size = %d, want %d", info.Size(), validSize)
	}
}

func TestReplayQuarantineRoutesKnownAndCarriesUnknown(t *testing.T) {
	directory := t.TempDir()
	options := DefaultQuarantineOptions(directory)
	source, err := OpenQuarantineWriter(OSFileSystem{}, options)
	if err != nil {
		t.Fatalf("OpenQuarantineWriter(source) error = %v", err)
	}
	knownMAC := MACAddress{0x02, 0, 0, 0xab, 0xcd, 0xef}
	unknownMAC := MACAddress{1, 2, 3, 4, 5, 6}
	for _, message := range []UnknownMessage{
		quarantineTestMessage(t, knownMAC, 10, TransportTCP),
		quarantineTestMessage(t, unknownMAC, 20, TransportUDP),
	} {
		if err := source.WriteUnknownMessage(context.Background(), message); err != nil {
			t.Fatalf("WriteUnknownMessage() error = %v", err)
		}
	}
	if err := source.Close(); err != nil {
		t.Fatalf("Close(source) error = %v", err)
	}

	writer, err := OpenQuarantineWriter(OSFileSystem{}, options)
	if err != nil {
		t.Fatalf("OpenQuarantineWriter(replay) error = %v", err)
	}
	defer writer.Close()
	registry, err := NewConfigRegistry(loadTestConfig(t, validConfigJSON))
	if err != nil {
		t.Fatalf("NewConfigRegistry() error = %v", err)
	}
	dataWriter := &recordingDataWriter{}
	counters, err := ReplayQuarantine(context.Background(), writer, registry, dataWriter)
	if err != nil {
		t.Fatalf("ReplayQuarantine() error = %v", err)
	}
	if counters.SegmentsCompleted != 1 || counters.MessagesReplayed != 1 || counters.MessagesCarried != 1 || counters.RecordsWritten != 1 {
		t.Fatalf("replay counters = %+v", counters)
	}
	if len(dataWriter.writes) != 1 || dataWriter.writes[0].value != 10 {
		t.Fatalf("data writes = %+v", dataWriter.writes)
	}
	if _, err := os.Stat(filepath.Join(directory, quarantineSegmentName(1)+".done")); err != nil {
		t.Fatalf("completion marker: %v", err)
	}

	secondCounters, err := ReplayQuarantine(context.Background(), writer, registry, dataWriter)
	if err != nil {
		t.Fatalf("ReplayQuarantine(second) error = %v", err)
	}
	if secondCounters != (ReplayCounters{}) || len(dataWriter.writes) != 1 {
		t.Fatalf("second replay counters = %+v, writes = %d", secondCounters, len(dataWriter.writes))
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Close(writer) error = %v", err)
	}
	carriedEntries, err := readQuarantineEntries(OSFileSystem{}, filepath.Join(directory, quarantineSegmentName(2)))
	if err != nil {
		t.Fatalf("readQuarantineEntries(carried) error = %v", err)
	}
	if len(carriedEntries) != 1 || carriedEntries[0].Message.Header.MAC != unknownMAC {
		t.Fatalf("carried entries = %+v", carriedEntries)
	}
}

func TestQuarantineWriterRotates(t *testing.T) {
	directory := t.TempDir()
	message := quarantineTestMessage(t, MACAddress{1}, 1, TransportTCP)
	encoded, err := encodeQuarantineEntry(message)
	if err != nil {
		t.Fatalf("encodeQuarantineEntry() error = %v", err)
	}
	options := DefaultQuarantineOptions(directory)
	options.RotationSize = int64(len(encoded))
	writer, err := OpenQuarantineWriter(OSFileSystem{}, options)
	if err != nil {
		t.Fatalf("OpenQuarantineWriter() error = %v", err)
	}
	if err := writer.WriteUnknownMessage(context.Background(), message); err != nil {
		t.Fatalf("WriteUnknownMessage(first) error = %v", err)
	}
	if err := writer.WriteUnknownMessage(context.Background(), message); err != nil {
		t.Fatalf("WriteUnknownMessage(second) error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	for sequence := uint64(1); sequence <= 2; sequence++ {
		if _, err := os.Stat(filepath.Join(directory, quarantineSegmentName(sequence))); err != nil {
			t.Fatalf("segment %d: %v", sequence, err)
		}
	}
}

func quarantineTestMessage(t *testing.T, mac MACAddress, value int16, transport Transport) UnknownMessage {
	t.Helper()
	payload := make([]byte, SensorRecordSize)
	binary.LittleEndian.PutUint16(payload[0:2], 1)
	binary.LittleEndian.PutUint16(payload[2:4], uint16(value))
	encoded, err := EncodeMessage(MessageTypeSensorData, mac, payload)
	if err != nil {
		t.Fatalf("EncodeMessage() error = %v", err)
	}
	message, err := DecodeDatagram(encoded)
	if err != nil {
		t.Fatalf("DecodeDatagram() error = %v", err)
	}
	return UnknownMessage{Timestamp: 123456, Transport: transport, Message: message}
}
