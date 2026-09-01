package sensorserver

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

var goldenSensorMessage = []byte{
	0x0d, 0xf0,
	0x01, 0x00,
	0x08, 0x00,
	0xba, 0x42, 0xad, 0x8b,
	0x02, 0x00, 0x00, 0xab, 0xcd, 0xef,
	0x01, 0x00, 0xd7, 0x00,
	0x02, 0x00, 0xd3, 0xff,
}

func TestDecodeDatagramGoldenMessage(t *testing.T) {
	message, err := DecodeDatagram(goldenSensorMessage)
	if err != nil {
		t.Fatalf("DecodeDatagram() error = %v", err)
	}

	if message.Header.Magic != MessageMagic || message.Header.Type != MessageTypeSensorData {
		t.Fatalf("header = %+v", message.Header)
	}
	if message.Header.MAC != (MACAddress{0x02, 0x00, 0x00, 0xab, 0xcd, 0xef}) {
		t.Fatalf("MAC = %x", message.Header.MAC)
	}
	want := []SensorRecord{{ID: 1, Value: 215}, {ID: 2, Value: -45}}
	if len(message.Sensors) != len(want) {
		t.Fatalf("Sensors length = %d, want %d", len(message.Sensors), len(want))
	}
	for index := range want {
		if message.Sensors[index] != want[index] {
			t.Fatalf("Sensors[%d] = %+v, want %+v", index, message.Sensors[index], want[index])
		}
	}
}

func TestEncodeMessageGoldenMessage(t *testing.T) {
	payload := goldenSensorMessage[MessageHeaderSize:]
	encoded, err := EncodeMessage(MessageTypeSensorData, MACAddress{0x02, 0x00, 0x00, 0xab, 0xcd, 0xef}, payload)
	if err != nil {
		t.Fatalf("EncodeMessage() error = %v", err)
	}
	if !bytes.Equal(encoded, goldenSensorMessage) {
		t.Fatalf("EncodeMessage() = %x, want %x", encoded, goldenSensorMessage)
	}
}

func TestReadMessageHandlesSplitReadsAndConsecutiveMessages(t *testing.T) {
	stream := append(append([]byte(nil), goldenSensorMessage...), goldenSensorMessage...)
	reader := &limitedReader{reader: bytes.NewReader(stream), maximum: 3}

	for index := 0; index < 2; index++ {
		message, err := ReadMessage(reader)
		if err != nil {
			t.Fatalf("ReadMessage(%d) error = %v", index, err)
		}
		if len(message.Sensors) != 2 {
			t.Fatalf("ReadMessage(%d) sensor count = %d, want 2", index, len(message.Sensors))
		}
	}
}

func TestReadMessageConsumesUnsupportedMessagePayload(t *testing.T) {
	unsupported := mutateByte(goldenSensorMessage, 2, 2)
	stream := append(unsupported, goldenSensorMessage...)
	reader := bytes.NewReader(stream)

	if _, err := ReadMessage(reader); !errors.Is(err, ErrUnsupportedType) {
		t.Fatalf("first ReadMessage() error = %v, want ErrUnsupportedType", err)
	}
	message, err := ReadMessage(reader)
	if err != nil {
		t.Fatalf("second ReadMessage() error = %v", err)
	}
	if len(message.Sensors) != 2 {
		t.Fatalf("second ReadMessage() sensor count = %d, want 2", len(message.Sensors))
	}
}

func TestSensorKeepalive(t *testing.T) {
	encoded, err := EncodeMessage(MessageTypeSensorData, MACAddress{1, 2, 3, 4, 5, 6}, nil)
	if err != nil {
		t.Fatalf("EncodeMessage() error = %v", err)
	}
	message, err := DecodeDatagram(encoded)
	if err != nil {
		t.Fatalf("DecodeDatagram() error = %v", err)
	}
	if len(message.Sensors) != 0 {
		t.Fatalf("Sensors length = %d, want 0", len(message.Sensors))
	}
}

func TestDecodeDatagramRejectsMalformedMessages(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		err  error
	}{
		{name: "short", data: goldenSensorMessage[:15], err: ErrDatagramSize},
		{name: "trailing", data: append(append([]byte(nil), goldenSensorMessage...), 0), err: ErrDatagramSize},
		{name: "bad magic", data: mutateByte(goldenSensorMessage, 0, 0), err: ErrInvalidMagic},
		{name: "unsupported type", data: mutateByte(goldenSensorMessage, 2, 2), err: ErrUnsupportedType},
		{name: "bad checksum", data: mutateByte(goldenSensorMessage, MessageHeaderSize, 0xff), err: ErrChecksumMismatch},
		{name: "invalid sensor length", data: withPayloadLength(goldenSensorMessage, 7), err: ErrInvalidPayload},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := DecodeDatagram(test.data)
			if !errors.Is(err, test.err) {
				t.Fatalf("DecodeDatagram() error = %v, want %v", err, test.err)
			}
		})
	}
}

func TestReadMessageReportsTruncatedPayload(t *testing.T) {
	_, err := ReadMessage(bytes.NewReader(goldenSensorMessage[:len(goldenSensorMessage)-1]))
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("ReadMessage() error = %v, want io.ErrUnexpectedEOF", err)
	}
}

func FuzzDecodeDatagram(f *testing.F) {
	f.Add([]byte{})
	f.Add(append([]byte(nil), goldenSensorMessage...))
	f.Add(append(append([]byte(nil), goldenSensorMessage...), 0))
	f.Fuzz(func(t *testing.T, datagram []byte) {
		message, err := DecodeDatagram(datagram)
		if err == nil {
			if len(datagram) != MessageHeaderSize+len(message.Payload) {
				t.Fatalf("accepted datagram size %d for payload %d", len(datagram), len(message.Payload))
			}
			if len(message.Payload)%SensorRecordSize != 0 {
				t.Fatalf("accepted payload size %d", len(message.Payload))
			}
		}
	})
}

type limitedReader struct {
	reader  io.Reader
	maximum int
}

func (reader *limitedReader) Read(buffer []byte) (int, error) {
	if len(buffer) > reader.maximum {
		buffer = buffer[:reader.maximum]
	}
	return reader.reader.Read(buffer)
}

func mutateByte(source []byte, index int, value byte) []byte {
	result := append([]byte(nil), source...)
	result[index] = value
	return result
}

func withPayloadLength(source []byte, length byte) []byte {
	result := append([]byte(nil), source[:MessageHeaderSize+int(length)]...)
	result[4] = length
	return result
}
