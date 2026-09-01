package sensorserver

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestDataFileEncodesRecordsLittleEndian(t *testing.T) {
	directory := t.TempDir()
	options := DefaultDataFileOptions(directory)
	dataFile, err := OpenDataFile(OSFileSystem{}, options)
	if err != nil {
		t.Fatalf("OpenDataFile() error = %v", err)
	}
	if err := dataFile.WriteRecord(0x0102030405060708, -2); err != nil {
		t.Fatalf("WriteRecord() error = %v", err)
	}
	if err := dataFile.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	encoded, err := os.ReadFile(filepath.Join(directory, "00000001.dat"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	want := []byte{0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, 0xfe, 0xff}
	if string(encoded) != string(want) {
		t.Fatalf("encoded record = %x, want %x", encoded, want)
	}
}

func TestDataFileRotatesBeforeExceedingLimit(t *testing.T) {
	directory := t.TempDir()
	options := DefaultDataFileOptions(directory)
	options.RotationSize = 2 * SensorDataRecordSize
	dataFile, err := OpenDataFile(OSFileSystem{}, options)
	if err != nil {
		t.Fatalf("OpenDataFile() error = %v", err)
	}
	for index := int64(1); index <= 3; index++ {
		if err := dataFile.WriteRecord(index, int16(index)); err != nil {
			t.Fatalf("WriteRecord(%d) error = %v", index, err)
		}
	}
	counters := dataFile.Counters()
	if counters.BytesWritten != 3*uint64(SensorDataRecordSize) || counters.Rotations != 1 {
		t.Fatalf("Counters() = %+v", counters)
	}
	if err := dataFile.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	assertSegment(t, directory, "00000001.dat", []int64{1, 2})
	assertSegment(t, directory, "00000002.dat", []int64{3})
}

func TestDataFileRepairsTornTailAndContinuesHighestSegment(t *testing.T) {
	directory := t.TempDir()
	firstPath := filepath.Join(directory, "00000001.dat")
	secondPath := filepath.Join(directory, "00000002.dat")
	if err := os.WriteFile(firstPath, make([]byte, SensorDataRecordSize+3), 0o644); err != nil {
		t.Fatalf("WriteFile(first) error = %v", err)
	}
	if err := os.WriteFile(secondPath, encodeTestRecord(2), 0o644); err != nil {
		t.Fatalf("WriteFile(second) error = %v", err)
	}

	options := DefaultDataFileOptions(directory)
	dataFile, err := OpenDataFile(OSFileSystem{}, options)
	if err != nil {
		t.Fatalf("OpenDataFile() error = %v", err)
	}
	if err := dataFile.WriteRecord(3, 3); err != nil {
		t.Fatalf("WriteRecord() error = %v", err)
	}
	if err := dataFile.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	firstInfo, err := os.Stat(firstPath)
	if err != nil {
		t.Fatalf("Stat(first) error = %v", err)
	}
	if firstInfo.Size() != SensorDataRecordSize {
		t.Fatalf("first segment size = %d, want %d", firstInfo.Size(), SensorDataRecordSize)
	}
	assertSegment(t, directory, "00000002.dat", []int64{2, 3})
}

func TestDataFileFlushMakesBufferedRecordVisible(t *testing.T) {
	directory := t.TempDir()
	dataFile, err := OpenDataFile(OSFileSystem{}, DefaultDataFileOptions(directory))
	if err != nil {
		t.Fatalf("OpenDataFile() error = %v", err)
	}
	defer dataFile.Close()
	if err := dataFile.WriteRecord(1, 1); err != nil {
		t.Fatalf("WriteRecord() error = %v", err)
	}
	if err := dataFile.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}
	info, err := os.Stat(filepath.Join(directory, "00000001.dat"))
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Size() != SensorDataRecordSize {
		t.Fatalf("segment size = %d, want %d", info.Size(), SensorDataRecordSize)
	}
}

func assertSegment(t *testing.T, directory, name string, timestamps []int64) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(directory, name))
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", name, err)
	}
	if len(data) != len(timestamps)*int(SensorDataRecordSize) {
		t.Fatalf("%s size = %d, want %d", name, len(data), len(timestamps)*int(SensorDataRecordSize))
	}
	for index, timestamp := range timestamps {
		offset := index * int(SensorDataRecordSize)
		got := int64(binary.LittleEndian.Uint64(data[offset : offset+8]))
		if got != timestamp {
			t.Fatalf("%s record %d timestamp = %d, want %d", name, index, got, timestamp)
		}
	}
}

func encodeTestRecord(timestamp int64) []byte {
	data := make([]byte, SensorDataRecordSize)
	binary.LittleEndian.PutUint64(data[:8], uint64(timestamp))
	return data
}
