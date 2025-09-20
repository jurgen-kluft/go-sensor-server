package sensor_server

import (
	"encoding/binary"
	"fmt"
)

// Packet structure
// {
//     u8  length;   // Number of words in the packet
//     u24 timesync; // TimeSync value of the packet
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
	Length    uint16
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
	if len(data) < 6 {
		return SensorPacket{}, fmt.Errorf("sensor packet, size too small")
	}

	pkt := SensorPacket{
		Length:   uint16(data[0] * 2),
		TimeSync: int32(data[1]) | (int32(data[2]) << 8) | (int32(data[3]) << 16),
		Values:   nil,
	}
	pkt.Immediate = (pkt.TimeSync & 0x800000) != 0
	pkt.TimeSync = pkt.TimeSync & 0x7FFFFF

	if len(data) < int(pkt.Length) {
		return pkt, fmt.Errorf("data length mismatch, %d < %d", len(data), pkt.Length)
	}

	offset := 4

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
	offset = 4

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
