package sensor_server

import (
	"encoding/binary"
	"fmt"
)

// Note: Little Endian byte order
// Packet structure
// {
//     u8  length;    // Number of words in the packet
//     u8  version;   // Packet version (currently 1)
//     u32 timesync;  // TimeSync value of the packet (bit 31 indicates if this packet is a time-sync packet)
//
//     // sensor value 1
//     u8 type;
//     s8|s16|s32 value;  // depending on type
//
//     // sensor value 2
//     u8 type;
//     s8|s16|s32 value;  // depending on type
//
//     ...
//
//     Padding to align packet size to 2 bytes
// };

const (
	SensorPacketLengthOffset  = 0
	SensorPacketVersionOffset = 1
	SensorPacketTimeOffset    = 2
	SensorPacketHeaderSize    = 1 + 1 + 4 // length, version, time-sync
)

type SensorPacket struct {
	Length    uint16
	Version   uint8
	TimeSync  int32
	Immediate bool
	Values    []SensorValue
}

type SensorValue struct {
	SensorType SensorType
	FieldType  SensorFieldType
	Value      int32
}

func (v *SensorValue) IsZero() bool {
	return v.Value == 0
}

func DecodeNetworkPacket(data []byte) (SensorPacket, error) {
	if len(data) < 8 {
		return SensorPacket{}, fmt.Errorf("sensor packet, size too small")
	}

	pkt := SensorPacket{
		Length:   uint16(data[SensorPacketLengthOffset] * 2),
		Version:  uint8(data[SensorPacketVersionOffset]),
		TimeSync: int32(data[SensorPacketTimeOffset]) | (int32(data[SensorPacketTimeOffset+1]) << 8) | (int32(data[SensorPacketTimeOffset+2]) << 16) | (int32(data[SensorPacketTimeOffset+3]) << 24),
		Values:   nil,
	}

	if pkt.Version == 1 {

		pkt.Immediate = (pkt.TimeSync & 0x800000) != 0
		pkt.TimeSync = pkt.TimeSync & 0x7FFFFF

		if len(data) < int(pkt.Length) {
			return pkt, fmt.Errorf("sensor packet, unexpected data length, %d != %d", len(data), pkt.Length)
		}

		offset := SensorPacketHeaderSize

		// Compute the number of sensor values in the packet.
		numberOfValues := 0
		for offset <= int(pkt.Length)-2 {
			sensorType := SensorType(data[offset])
			fieldType := GetFieldTypeFromType(sensorType)
			offset += 1 + ((fieldType.SizeInBits() + 7) / 8)
			numberOfValues++
		}
		pkt.Values = make([]SensorValue, 0, numberOfValues)

		// Now decode the values.
		offset = SensorPacketHeaderSize

		for offset <= int(pkt.Length)-2 {
			value := SensorValue{SensorType: SensorType(data[offset])}
			value.FieldType = GetFieldTypeFromType(value.SensorType)

			offset += 1

			// depending on FieldType, read the appropriate value.
			// the written values are in little-endian format
			switch value.FieldType {
			case TypeS8:
				value.Value = int32(data[offset])
				pkt.Values = append(pkt.Values, value)
				offset += 1
			case TypeS16:
				value.Value = int32(binary.LittleEndian.Uint16(data[offset : offset+2]))
				pkt.Values = append(pkt.Values, value)
				offset += 2
			case TypeS32:
				value.Value = int32(binary.LittleEndian.Uint32(data[offset : offset+4]))
				pkt.Values = append(pkt.Values, value)
				offset += 4
			}
		}
		return pkt, nil
	}

	return pkt, fmt.Errorf("sensor packet, unknown version %d", pkt.Version)
}

func EncodeNetworkPacket(pkt *SensorPacket) ([]byte, error) {
	if pkt.Version != 1 {
		return nil, fmt.Errorf("sensor packet, unknown version %d", pkt.Version)
	}

	// Compute the length of the packet.
	length := SensorPacketHeaderSize
	for _, v := range pkt.Values {
		if v.FieldType.SizeInBits() == 0 {
			return nil, fmt.Errorf("sensor packet, unknown field type for sensor type %d", v.SensorType)
		}
		length += 1 + ((v.FieldType.SizeInBits() + 7) / 8)
	}
	if length > 255*2 {
		return nil, fmt.Errorf("sensor packet, too many values")
	}
	length = (length + 1) & 0xFFFE // align to 2 bytes

	data := make([]byte, length)

	data[SensorPacketLengthOffset] = uint8(length / 2)
	data[SensorPacketVersionOffset] = pkt.Version
	timeSync := pkt.TimeSync & 0x7FFFFF
	if pkt.Immediate {
		timeSync = timeSync | 0x800000
	}
	data[SensorPacketTimeOffset] = uint8(timeSync & 0xFF)
	data[SensorPacketTimeOffset+1] = uint8((timeSync >> 8) & 0xFF)
	data[SensorPacketTimeOffset+2] = uint8((timeSync >> 16) & 0xFF)
	data[SensorPacketTimeOffset+3] = uint8((timeSync >> 24) & 0xFF)

	offset := SensorPacketHeaderSize
	for _, v := range pkt.Values {
		data[offset] = byte(v.SensorType)
		offset += 1
		switch v.FieldType {
		case TypeS8:
			data[offset] = byte(v.Value & 0xFF)
			offset += 1
		case TypeS16:
			binary.LittleEndian.PutUint16(data[offset:offset+2], uint16(v.Value&0xFFFF))
			offset += 2
		case TypeS32:
			binary.LittleEndian.PutUint32(data[offset:offset+4], uint32(v.Value))
			offset += 4
		}
	}

	// Padding byte if needed
	if offset < length {
		data[offset] = 0
	}

	// Print the encoded packet for debugging
	fmt.Printf("Encoded packet (length=%d): ", length)
	for i := 0; i < length; i++ {
		fmt.Printf("%02X ", data[i])
	}
	fmt.Printf("\n")

	return data, nil
}
