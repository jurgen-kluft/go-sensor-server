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
//     u16 id;        // Fixed ID 0xA5C3
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
	SensorPacketIdOffset      = 2
	SensorPacketTimeOffset    = 4
	SensorPacketHeaderSize    = 1 + 1 + 2 + 4 // length, version, time-sync
	SensorPacketId            = 0xA5C3
)

type SensorPacket struct {
	Length     uint16
	Version    uint8
	Id         uint16
	TimeSync   uint32
	IsTimeSync bool
	Values     []SensorValue
}

type SensorValue struct {
	SensorType SensorType
	Value      int64
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
		Id:       uint16(data[SensorPacketIdOffset]) | (uint16(data[SensorPacketIdOffset+1]) << 8),
		TimeSync: uint32(data[SensorPacketTimeOffset]) | (uint32(data[SensorPacketTimeOffset+1]) << 8) | (uint32(data[SensorPacketTimeOffset+2]) << 16) | (uint32(data[SensorPacketTimeOffset+3]) << 24),
		Values:   nil,
	}

	if pkt.Version == 1 {
		pkt.IsTimeSync = (pkt.TimeSync & 0x80000000) != 0
		pkt.TimeSync = pkt.TimeSync & 0x7FFFFFFF

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
			fieldType := GetFieldTypeFromType(value.SensorType)

			offset += 1

			// depending on FieldType, read the appropriate value.
			// the written values are in little-endian format
			switch fieldType {
			case TypeS8, TypeU8:
				value.Value = int64(data[offset])
				pkt.Values = append(pkt.Values, value)
				offset += 1
			case TypeS16, TypeU16:
				value.Value = int64(binary.LittleEndian.Uint16(data[offset : offset+2]))
				pkt.Values = append(pkt.Values, value)
				offset += 2
			case TypeS32, TypeU32:
				value.Value = int64(binary.LittleEndian.Uint32(data[offset : offset+4]))
				pkt.Values = append(pkt.Values, value)
				offset += 4
			case TypeS64, TypeU64:
				value.Value = int64(binary.LittleEndian.Uint64(data[offset : offset+8]))
				pkt.Values = append(pkt.Values, value)
				offset += 8
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
		fieldType := GetFieldTypeFromType(v.SensorType)
		if fieldType.SizeInBits() == 0 {
			return nil, fmt.Errorf("sensor packet, unknown field type for sensor type %d", v.SensorType)
		}
		// type(byte) + sizeof(value)
		length += 1 + ((fieldType.SizeInBits() + 7) / 8)
	}
	if length > 255*2 {
		return nil, fmt.Errorf("sensor packet, too many values")
	}
	length = (length + 1) & 0xFFFE // align to 2 bytes

	data := make([]byte, length)

	data[SensorPacketLengthOffset] = uint8(length / 2)
	data[SensorPacketVersionOffset] = pkt.Version
	data[SensorPacketIdOffset] = uint8(pkt.Id & 0xFF)
	data[SensorPacketIdOffset+1] = uint8((pkt.Id >> 8) & 0xFF)
	// Write the time-sync value
	timeSync := pkt.TimeSync & 0x7FFFFF
	if pkt.IsTimeSync {
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
		fieldType := GetFieldTypeFromType(v.SensorType)
		switch fieldType {
		case TypeS8:
			data[offset] = byte(v.Value & 0xFF)
			offset += 1
		case TypeS16:
			binary.LittleEndian.PutUint16(data[offset:offset+2], uint16(v.Value&0xFFFF))
			offset += 2
		case TypeS32:
			binary.LittleEndian.PutUint32(data[offset:offset+4], uint32(v.Value))
			offset += 4
		case TypeS64:
			binary.LittleEndian.PutUint64(data[offset:offset+8], uint64(v.Value))
			offset += 8
		case TypeU8:
			data[offset] = byte(v.Value & 0xFF)
			offset += 1
		case TypeU16:
			binary.LittleEndian.PutUint16(data[offset:offset+2], uint16(v.Value&0xFFFF))
			offset += 2
		case TypeU32:
			binary.LittleEndian.PutUint32(data[offset:offset+4], uint32(v.Value))
			offset += 4
		case TypeU64:
			binary.LittleEndian.PutUint64(data[offset:offset+8], uint64(v.Value))
			offset += 8
		default:
			return nil, fmt.Errorf("sensor packet, unknown field type for sensor type %d", v.SensorType)
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
