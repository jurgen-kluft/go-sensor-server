package sensorserver

import (
	"bytes"
	"testing"
)

func TestBinaryDataSequentialReads(t *testing.T) {
	data := BinaryData{buf: []byte{
		0x11,
		0x33, 0x22,
		0x05, 0x00, 0x00,
		0xaa, 0xbb,
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06,
		0xdd, 0xcc, 0xbb, 0xaa,
		0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11,
		0x99, 0x98, 0x97,
	}, off: 0}

	if got := data.ReadUint8(); got != 0x11 {
		t.Fatalf("ReadUint8() = 0x%02x, want 0x11", got)
	}
	if got := data.ReadUint16(); got != 0x2233 {
		t.Fatalf("ReadUint16() = 0x%04x, want 0x2233", got)
	}
	if got := data.ReadInt24(); got != 5 {
		t.Fatalf("ReadInt24() = %d, want 5", got)
	}
	data.SkipBytes(2)
	if got := data.ReadMAC(); got != (MACAddress{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}) {
		t.Fatalf("ReadMAC() = %x, want 010203040506", got)
	}
	if got := data.ReadUint32(); got != 0xaabbccdd {
		t.Fatalf("ReadUint32() = 0x%08x, want 0xaabbccdd", got)
	}
	if got := data.ReadUint64(); got != 0x1122334455667788 {
		t.Fatalf("ReadUint64() = 0x%016x, want 0x1122334455667788", got)
	}
	if got := data.ReadBytes(3); !bytes.Equal(got, []byte{0x99, 0x98, 0x97}) {
		t.Fatalf("ReadBytes() = %x, want 999897", got)
	}
}

func TestBinaryDataSequentialWrites(t *testing.T) {
	buffer := make([]byte, 1+2+4+8+3)
	data := BinaryData{buf: buffer, off: 0}

	data.WriteUint8(0x11)
	data.WriteUint16(0x2233)
	data.WriteUint32(0xaabbccdd)
	data.WriteUint64(0x1122334455667788)
	data.WriteBytes([]byte{0x99, 0x98, 0x97})

	want := []byte{
		0x11,
		0x33, 0x22,
		0xdd, 0xcc, 0xbb, 0xaa,
		0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11,
		0x99, 0x98, 0x97,
	}
	if !bytes.Equal(buffer, want) {
		t.Fatalf("buffer = %x, want %x", buffer, want)
	}
}

func TestBinaryDataReadInt24(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want int32
	}{
		{name: "zero", data: []byte{0x00, 0x00, 0x00}, want: 0},
		{name: "positive", data: []byte{0x34, 0x12, 0x00}, want: 0x1234},
		{name: "max positive", data: []byte{0xff, 0xff, 0x7f}, want: 8388607},
		{name: "minus one", data: []byte{0xff, 0xff, 0xff}, want: -1},
		{name: "min negative", data: []byte{0x00, 0x00, 0x80}, want: -8388608},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := BinaryData{buf: test.data, off: 0}
			if got := data.ReadInt24(); got != test.want {
				t.Fatalf("ReadInt24() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestBinaryDataReadBytesReturnsCopy(t *testing.T) {
	buffer := []byte{0x01, 0x02, 0x03}
	data := BinaryData{buf: buffer, off: 0}
	read := data.ReadBytes(3)

	read[0] = 0xff
	if buffer[0] != 0x01 {
		t.Fatalf("buffer[0] = 0x%02x, want 0x01", buffer[0])
	}
}

func TestBinaryDataReadMACReturnsCopy(t *testing.T) {
	buffer := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}
	data := BinaryData{buf: buffer, off: 0}
	mac := data.ReadMAC()

	mac[0] = 0xff
	if buffer[0] != 0x01 {
		t.Fatalf("buffer[0] = 0x%02x, want 0x01", buffer[0])
	}
}

func TestBinaryDataWriteUint32At(t *testing.T) {
	buffer := make([]byte, 8)
	data := BinaryData{buf: buffer, off: 0}

	data.WriteUint32At(0xaabbccdd, 2)
	data.WriteUint16(0x1122)

	want := []byte{0x22, 0x11, 0xdd, 0xcc, 0xbb, 0xaa, 0x00, 0x00}
	if !bytes.Equal(buffer, want) {
		t.Fatalf("buffer = %x, want %x", buffer, want)
	}
}
