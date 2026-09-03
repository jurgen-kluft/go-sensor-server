package sensorserver

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
)

const (
	MessageMagic       uint16 = 0xF00D
	MessageHeaderSize         = 16
	SensorRecordSize          = 4
	MaximumPayloadSize        = 8 * 1024
)

type MessageType uint16

const (
	MessageTypeInvalid MessageType = iota
	MessageTypeSensorData
)

var (
	ErrInvalidMagic     = errors.New("invalid message magic")
	ErrUnsupportedType  = errors.New("unsupported message type")
	ErrPayloadTooLarge  = errors.New("payload exceeds maximum size")
	ErrInvalidPayload   = errors.New("invalid payload")
	ErrChecksumMismatch = errors.New("payload checksum mismatch")
	ErrDatagramSize     = errors.New("datagram size does not match header")
)

type MACAddress [6]byte

type MessageHeader struct {
	Magic         uint16
	Type          MessageType
	PayloadLength uint16
	MAC           MACAddress
	Checksum      uint32
}

type SensorRecord struct {
	SensorType SensorType // byte
	Value      int32      // [3]byte
}

type Message struct {
	Header  MessageHeader
	Payload []byte
	Sensors []SensorRecord
}

// DecodeHeader validates only the fields needed to frame a message safely.
func DecodeHeader(data []byte) (MessageHeader, error) {
	if len(data) != MessageHeaderSize {
		return MessageHeader{}, fmt.Errorf("decode header: got %d bytes, want %d: %w", len(data), MessageHeaderSize, io.ErrUnexpectedEOF)
	}

	binaryData := &BinaryData{buf: data, off: 0}

	hdrMagic := binaryData.ReadUint16()
	hdrType := MessageType(binaryData.ReadUint16())
	payloadLength := binaryData.ReadUint16()
	mac := binaryData.ReadMAC()
	checksum := binaryData.ReadUint32()

	header := MessageHeader{
		Magic:         hdrMagic,
		Type:          hdrType,
		PayloadLength: payloadLength,
		MAC:           mac,
		Checksum:      checksum,
	}

	if header.Magic != MessageMagic {
		return MessageHeader{}, fmt.Errorf("decode header: got 0x%04X: %w", header.Magic, ErrInvalidMagic)
	}
	if int(header.PayloadLength) > MaximumPayloadSize {
		return MessageHeader{}, fmt.Errorf("decode header: got %d bytes: %w", header.PayloadLength, ErrPayloadTooLarge)
	}
	return header, nil
}

// DecodeMessage validates and decodes a payload using an already framed header.
func DecodeMessage(header MessageHeader, payload []byte) (Message, error) {
	if len(payload) != int(header.PayloadLength) {
		return Message{}, fmt.Errorf("decode payload: got %d bytes, want %d: %w", len(payload), header.PayloadLength, ErrInvalidPayload)
	}
	if header.Type != MessageTypeSensorData {
		return Message{}, fmt.Errorf("decode payload: got type %d: %w", header.Type, ErrUnsupportedType)
	}
	if header.PayloadLength%SensorRecordSize != 0 {
		return Message{}, fmt.Errorf("decode payload: sensor payload length %d: %w", header.PayloadLength, ErrInvalidPayload)
	}
	if crc32.ChecksumIEEE(payload) != header.Checksum {
		return Message{}, fmt.Errorf("decode payload: %w", ErrChecksumMismatch)
	}

	message := Message{
		Header:  header,
		Payload: append([]byte(nil), payload...),
		Sensors: make([]SensorRecord, 0, len(payload)/SensorRecordSize),
	}

	binaryData := &BinaryData{buf: payload, off: 0}
	for i := 0; i < cap(message.Sensors); i += 1 {
		sensorTypeValue := uint16(binaryData.ReadUint8())
		message.Sensors = append(message.Sensors, SensorRecord{
			SensorType: ToSensorType(sensorTypeValue),
			Value:      binaryData.ReadInt24(),
		})
	}

	return message, nil
}

// ReadMessage reads exactly one message from a TCP byte stream.
func ReadMessage(reader io.Reader) (Message, error) {
	headerBytes := make([]byte, MessageHeaderSize)
	if _, err := io.ReadFull(reader, headerBytes); err != nil {
		return Message{}, fmt.Errorf("read header: %w", err)
	}
	header, err := DecodeHeader(headerBytes)
	if err != nil {
		return Message{}, err
	}

	payload := make([]byte, header.PayloadLength)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return Message{}, fmt.Errorf("read payload: %w", err)
	}
	return DecodeMessage(header, payload)
}

// DecodeDatagram decodes exactly one complete UDP datagram.
func DecodeDatagram(datagram []byte) (Message, error) {
	if len(datagram) < MessageHeaderSize {
		return Message{}, fmt.Errorf("decode datagram: got %d bytes: %w", len(datagram), ErrDatagramSize)
	}
	header, err := DecodeHeader(datagram[:MessageHeaderSize])
	if err != nil {
		return Message{}, err
	}
	expectedSize := MessageHeaderSize + int(header.PayloadLength)
	if len(datagram) != expectedSize {
		return Message{}, fmt.Errorf("decode datagram: got %d bytes, want %d: %w", len(datagram), expectedSize, ErrDatagramSize)
	}
	return DecodeMessage(header, datagram[MessageHeaderSize:])
}

// EncodeMessage creates a canonical wire message and is shared by tests and
// future server-side protocol tooling.
func EncodeMessage(messageType MessageType, mac MACAddress, payload []byte) ([]byte, error) {
	if len(payload) > MaximumPayloadSize {
		return nil, ErrPayloadTooLarge
	}
	if messageType != MessageTypeSensorData {
		return nil, ErrUnsupportedType
	}
	if len(payload)%SensorRecordSize != 0 {
		return nil, ErrInvalidPayload
	}
	headerBytes := make([]byte, MessageHeaderSize)
	binary.LittleEndian.PutUint16(headerBytes[0:2], MessageMagic)
	binary.LittleEndian.PutUint16(headerBytes[2:4], uint16(messageType))
	binary.LittleEndian.PutUint16(headerBytes[4:6], uint16(len(payload)))
	copy(headerBytes[6:12], mac[:])
	binary.LittleEndian.PutUint32(headerBytes[12:16], crc32.ChecksumIEEE(payload))

	if _, err := DecodeHeader(headerBytes); err != nil {
		return nil, err
	}
	encoded := make([]byte, 0, MessageHeaderSize+len(payload))
	encoded = append(encoded, headerBytes...)
	encoded = append(encoded, payload...)
	return encoded, nil
}
