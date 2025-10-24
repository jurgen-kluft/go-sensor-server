package sensor_server

import (
	"encoding/binary"
	"fmt"
	"time"
)

// Note: Little Endian byte order
// Packet structure
// {
//     u8  length;    // Number of words in the packet
//     u8  version;   // Packet version (currently 1)
//
//     // sensor value 1
//     u8 id;             // stream id
//     u16|s16 value;     // value
//
//     // sensor value 2
//     u8 id;             // stream id
//     u16|s16 value;     // value
//
//     ...
//
//     Padding to align packet size to 2 bytes
// };

const (
	SensorPacketLengthOffset  = 0
	SensorPacketVersionOffset = 1
	SensorPacketHeaderSize    = 1 + 1 // length, version, time-sync
)

type SensorPacket struct {
	Length  uint16
	Version uint8
	Time    time.Time
	Values  []SensorValue
}

type SensorValue struct {
	Sensor *SensorConfig
	Value  int32
}

func (v *SensorValue) IsZero() bool {
	return v.Value == 0
}

func DecodeNetworkPacket(sensorMap map[int]*SensorConfig, data []byte) (SensorPacket, error) {
	if len(data) < 6 {
		return SensorPacket{}, fmt.Errorf("sensor packet, size too small")
	}

	pkt := SensorPacket{
		Length:  uint16(data[SensorPacketLengthOffset] * 2),
		Version: uint8(data[SensorPacketVersionOffset]),
		Time:    time.Now(),
		Values:  nil,
	}

	if pkt.Version == 1 {
		if len(data) < int(pkt.Length) {
			return pkt, fmt.Errorf("sensor packet, unexpected data length, %d != %d", len(data), pkt.Length)
		}

		offset := SensorPacketHeaderSize

		// Compute the number of sensor values in the packet.
		numberOfValues := 0
		for offset <= int(pkt.Length)-2 {
			index := int(uint(data[offset])<<8 | uint(data[offset+1]))
			if _, ok := sensorMap[index]; !ok {
				return pkt, fmt.Errorf("sensor packet, unknown sensor index %d", index)
			}
			offset += 2 + 2
			numberOfValues++
		}
		pkt.Values = make([]SensorValue, 0, numberOfValues)

		// Now decode the values.
		offset = SensorPacketHeaderSize

		for offset <= int(pkt.Length)-2 {
			index := int(uint(data[offset])<<8 | uint(data[offset+1]))
			sensor, _ := sensorMap[index]
			value := SensorValue{Sensor: sensor}
			offset += 2

			// values are in little-endian format
			value.Value = int32(binary.LittleEndian.Uint16(data[offset : offset+2]))
			pkt.Values = append(pkt.Values, value)
			offset += 2
		}
		return pkt, nil
	} else {
		return pkt, fmt.Errorf("sensor packet, unknown version %d", pkt.Version)
	}
}

func EncodeNetworkPacket(pkt *SensorPacket) ([]byte, error) {
	if pkt.Version != 1 {
		return nil, fmt.Errorf("sensor packet, unknown version %d", pkt.Version)
	}

	// Compute the length of the packet.
	length := SensorPacketHeaderSize + len(pkt.Values)*(2+2)
	if length > (2 + 32*2) {
		return nil, fmt.Errorf("sensor packet, too many values")
	}

	data := make([]byte, length)

	data[SensorPacketLengthOffset] = uint8(length / 2)
	data[SensorPacketVersionOffset] = pkt.Version

	offset := SensorPacketHeaderSize
	for _, v := range pkt.Values {
		data[offset] = byte(v.Sensor.mIndex>>8) & 0xFF
		offset++
		data[offset] = byte(v.Sensor.mIndex & 0xFF)
		offset++

		binary.LittleEndian.PutUint16(data[offset:offset+2], uint16(v.Value&0xFFFF))
		offset += 2
	}

	// Padding byte if needed
	if offset < length {
		data[offset] = 0
	}

	// Print the encoded packet for debugging
	// fmt.Printf("Encoded packet (length=%d): ", length)
	// for i := 0; i < length; i++ {
	// 	fmt.Printf("%02X ", data[i])
	// }
	// fmt.Printf("\n")

	return data, nil
}
