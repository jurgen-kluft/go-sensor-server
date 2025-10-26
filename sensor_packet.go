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
//     u8  mac[6];   // MAC address of the sensor device
//
//     // sensor value 1
//     u8 id;             // sensor id
//     u16|s16 value;     // value
//
//     // sensor value 2
//     u8 id;             // sensor id
//     u16|s16 value;     // value
//
//     ...
//
//     Padding to align packet size to 2 bytes
// };

const (
	SensorPacketLengthOffset  = 0
	SensorPacketVersionOffset = 1
	SensorPacketMacOffset     = 2
	SensorPacketHeaderSize    = 1 + 1 + 6
	SensorPacketDataOffset    = SensorPacketHeaderSize
)

type SensorPacket struct {
	Length  uint16
	Version uint8
	Time    time.Time
	Mac     uint64
	Values  []SensorValue
}

type SensorValue struct {
	Sensor *SensorConfig
	Value  int32
}

func (v *SensorValue) IsZero() bool {
	return v.Value == 0
}

func FindSensorConfig(id byte, mac uint64, sensorMap map[uint64]*SensorConfig) *SensorConfig {
	key := uint64(id) | (mac << 8)
	if sensor, ok := sensorMap[key]; ok {
		return sensor
	}
	return &SensorConfig{mIndex: int(id), mMac: mac, mName: fmt.Sprintf("unknown_%02X", id)}
}

func DecodeNetworkPacket(sensorMap map[uint64]*SensorConfig, data []byte) (SensorPacket, error) {
	// header(8) + one sensor value(1,2) = 11 bytes minimum
	if len(data) < 11 {
		return SensorPacket{}, fmt.Errorf("sensor packet, size too small")
	}

	pkt := SensorPacket{
		Length:  uint16(data[SensorPacketLengthOffset] * 2),
		Version: uint8(data[SensorPacketVersionOffset]),
		Time:    time.Now(),
		Mac:     uint64(binary.LittleEndian.Uint64(data[SensorPacketMacOffset : SensorPacketMacOffset+6])),
		Values:  nil,
	}

	if pkt.Version == 1 {
		if len(data) != int(pkt.Length) {
			return pkt, fmt.Errorf("sensor packet, unexpected data length, %d != %d", len(data), pkt.Length)
		}

		offset := SensorPacketDataOffset

		// Compute the number of sensor values in the packet.
		numberOfValues := (int(pkt.Length) - SensorPacketHeaderSize) / 3
		pkt.Values = make([]SensorValue, 0, numberOfValues)

		// Now decode the values.
		offset = SensorPacketDataOffset

		for offset <= int(pkt.Length)-3 {
			id := data[offset]
			sensor := FindSensorConfig(id, pkt.Mac, sensorMap)
			value := SensorValue{Sensor: sensor}
			value.Value = int32(uint32(data[offset+1])<<8 | uint32(data[offset+2]))
			pkt.Values = append(pkt.Values, value)
			offset += 3
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
	length := SensorPacketHeaderSize + len(pkt.Values)*(1+2)
	length += length & 1 // Padding byte if needed
	if length > (SensorPacketHeaderSize + 32*3) {
		return nil, fmt.Errorf("sensor packet, too many values")
	}

	data := make([]byte, length)

	data[SensorPacketLengthOffset] = uint8(length / 2)
	data[SensorPacketVersionOffset] = pkt.Version

	offset := SensorPacketHeaderSize
	for _, v := range pkt.Values {
		data[offset] = v.Sensor.Id()
		offset++

		binary.LittleEndian.PutUint16(data[offset:offset+2], uint16(v.Value&0xFFFF))
		offset += 2
	}

	return data, nil
}
