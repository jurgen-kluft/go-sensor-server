package sensor_server

import (
	"encoding/binary"
	"fmt"
)

// Packet structure
// {
//     u8  length;   // Number of words in the packet
//     u24 sequence; // Sequence number of the packet
//
//     // sensor value 1
//     u8 type;
//     union
//     {
//         s8  s8_value;
//         u8  u8_value;
//         s16 s16_value;
//         u16 u16_value;
//         s32 s32_value;
//         u32 u32_value;
//     } value;

//     // sensor value 2

// };

type SensorPacket struct {
	length   uint16
	sequence uint32
	values   []SensorValue
}

type SensorValue struct {
	sensorType SensorType
	fieldType  SensorFieldType
	value      int
}

func (v *SensorValue) IsZero() bool {
	return v.value == 0
}

func DecodeNetworkPacket(data []byte) (SensorPacket, error) {
	if len(data) < 5 {
		return SensorPacket{}, fmt.Errorf("data too short")
	}

	pkt := SensorPacket{
		length:   uint16(data[0] * 2),
		sequence: uint32(data[1] | (data[2] << 8) | (data[3] << 16)),
		values:   nil,
	}

	if len(data) < int(pkt.length) {
		return pkt, fmt.Errorf("data length mismatch, %d < %d", len(data), pkt.length)
	}

	offset := 3

	// Compute the number of sensor values in the packet.
	numberOfValues := 0
	for offset <= int(pkt.length)-2 {
		sensorType := SensorType(data[offset])
		fieldType := GetFieldTypeFromType(sensorType)
		offset += 1 + ((fieldType.SizeInBits() + 7) / 8)
		numberOfValues++
	}
	pkt.values = make([]SensorValue, 0, numberOfValues)

	// Now decode the values.
	offset = 3

	for offset <= int(pkt.length)-2 {
		value := SensorValue{sensorType: SensorType(data[offset])}
		value.fieldType = GetFieldTypeFromType(value.sensorType)

		offset += 1

		// depending on fieldType, read the appropriate value.
		// the written values are in little-endian format
		switch value.fieldType {
		case TypeS8:
			value.value = int(data[offset])
			pkt.values = append(pkt.values, value)
			offset += 1
		case TypeS16:
			value.value = int(binary.LittleEndian.Uint16(data[offset : offset+2]))
			pkt.values = append(pkt.values, value)
			offset += 2
		case TypeS32:
			value.value = int(int32(binary.LittleEndian.Uint32(data[offset : offset+4])))
			pkt.values = append(pkt.values, value)
			offset += 4
		}
	}
	return pkt, nil
}
