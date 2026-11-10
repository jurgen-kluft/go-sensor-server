package sensorserver

import (
	"encoding/binary"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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
	want := []byte{0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, 0xfe, 0xff, 0xff, 0xff}
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
		if err := dataFile.WriteRecord(index, int32(index)); err != nil {
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

func TestDataFileRecoversLastCompleteRecord(t *testing.T) {
	tests := []struct {
		name     string
		segments [][]byte
		want     *SensorData
		warnings int
	}{
		{name: "new stream"},
		{name: "empty segment", segments: [][]byte{nil}},
		{
			name:     "append order not timestamp order",
			segments: [][]byte{encodeTestRecord(100), append(encodeTestRecord(200), encodeTestRecord(-5)...)},
			want:     &SensorData{Timestamp: -5, Value: -5},
		},
		{
			name:     "partial newest record",
			segments: [][]byte{encodeTestRecord(1), append(encodeTestRecord(2), 1, 2, 3)},
			want:     &SensorData{Timestamp: 2, Value: 2}, warnings: 1,
		},
		{
			name:     "empty newest segment",
			segments: [][]byte{encodeTestRecord(1), nil},
			want:     &SensorData{Timestamp: 1, Value: 1},
		},
		{
			name:     "torn-only newest segment",
			segments: [][]byte{encodeTestRecord(1), {1, 2, 3}},
			want:     &SensorData{Timestamp: 1, Value: 1}, warnings: 1,
		},
		{name: "no complete record", segments: [][]byte{{1, 2, 3}}, warnings: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			for index, data := range test.segments {
				if err := os.WriteFile(filepath.Join(directory, segmentName(uint64(index+1))), data, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			var warnings []error
			options := DefaultDataFileOptions(directory)
			options.OnWarning = func(err error) { warnings = append(warnings, err) }
			dataFile, err := OpenDataFile(OSFileSystem{}, options)
			if err != nil {
				t.Fatal(err)
			}
			defer dataFile.Close()
			if (dataFile.initialValue == nil) != (test.want == nil) || (test.want != nil && *dataFile.initialValue != *test.want) {
				t.Fatalf("recovered = %+v, want %+v", dataFile.initialValue, test.want)
			}
			if len(warnings) != test.warnings {
				t.Fatalf("warnings = %v, want %d", warnings, test.warnings)
			}
			for _, warning := range warnings {
				if !strings.Contains(warning.Error(), directory) || !strings.Contains(warning.Error(), "truncated 3") {
					t.Fatalf("warning lacks repair context: %v", warning)
				}
			}
			if counters := dataFile.Counters(); counters.BytesWritten != 0 || counters.Rotations != 0 {
				t.Fatalf("recovery counted as ingestion: %+v", counters)
			}
			for index, original := range test.segments {
				info, err := os.Stat(filepath.Join(directory, segmentName(uint64(index+1))))
				if err != nil {
					t.Fatal(err)
				}
				if wantSize := int64(len(original)) / SensorDataRecordSize * SensorDataRecordSize; info.Size() != wantSize {
					t.Fatalf("repaired segment size = %d, want %d", info.Size(), wantSize)
				}
			}
			if err := dataFile.WriteRecord(3, 3); err != nil {
				t.Fatal(err)
			}
			if err := dataFile.Close(); err != nil {
				t.Fatal(err)
			}
			sequence := len(test.segments)
			if sequence == 0 {
				sequence = 1
			}
			path := filepath.Join(directory, segmentName(uint64(sequence)))
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if got := int64(binary.LittleEndian.Uint64(data[len(data)-int(SensorDataRecordSize):])); got != 3 {
				t.Fatalf("new record did not append to highest segment: %d", got)
			}
		})
	}
}

func TestDataFileRecoveryFailuresAreExplicitAndCloseFiles(t *testing.T) {
	for _, operation := range []string{"read", "truncate", "sync"} {
		t.Run(operation, func(t *testing.T) {
			directory := t.TempDir()
			path := filepath.Join(directory, segmentName(1))
			data := encodeTestRecord(1)
			if operation != "read" {
				data = append(data, 1)
			}
			if err := os.WriteFile(path, data, 0o644); err != nil {
				t.Fatal(err)
			}
			failure := errors.New("injected recovery failure")
			fileSystem := &recoveryTestFileSystem{operation: operation, failure: failure}
			_, err := OpenDataFile(fileSystem, DefaultDataFileOptions(directory))
			if !errors.Is(err, failure) || !strings.Contains(err.Error(), path) {
				t.Fatalf("OpenDataFile() = %v, want contextual recovery error", err)
			}
			for _, file := range fileSystem.files {
				if !file.closed {
					t.Fatal("recovery leaked an open file")
				}
			}
		})
	}
}

type recoveryTestFileSystem struct {
	OSFileSystem
	operation  string
	failure    error
	failReadAt int
	reads      int
	files      []*recoveryTestFile
}

func (fileSystem *recoveryTestFileSystem) OpenFile(name string, flag int, perm fs.FileMode) (File, error) {
	file, err := fileSystem.OSFileSystem.OpenFile(name, flag, perm)
	if err != nil {
		return nil, err
	}
	wrapped := &recoveryTestFile{File: file, fileSystem: fileSystem}
	fileSystem.files = append(fileSystem.files, wrapped)
	return wrapped, nil
}

type recoveryTestFile struct {
	File
	fileSystem *recoveryTestFileSystem
	closed     bool
}

func (file *recoveryTestFile) ReadAt(data []byte, offset int64) (int, error) {
	file.fileSystem.reads++
	if file.fileSystem.operation == "read" && (file.fileSystem.failReadAt == 0 || file.fileSystem.reads == file.fileSystem.failReadAt) {
		return 0, file.fileSystem.failure
	}
	return file.File.ReadAt(data, offset)
}

func (file *recoveryTestFile) Truncate(size int64) error {
	if file.fileSystem.operation == "truncate" {
		return file.fileSystem.failure
	}
	return file.File.Truncate(size)
}

func (file *recoveryTestFile) Sync() error {
	if file.fileSystem.operation == "sync" {
		return file.fileSystem.failure
	}
	return file.File.Sync()
}

func (file *recoveryTestFile) Close() error {
	file.closed = true
	return file.File.Close()
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
	binary.LittleEndian.PutUint32(data[8:12], uint32(timestamp))
	return data
}
