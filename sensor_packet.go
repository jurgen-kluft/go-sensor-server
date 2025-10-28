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
	SensorPacketValueSize     = 3 // id(1) + value(2)
	SensorPacketMinSize       = SensorPacketHeaderSize + SensorPacketValueSize
	SensorPacketMaxValues     = 16
	SensorPacketMaxSize       = SensorPacketHeaderSize + SensorPacketMaxValues*SensorPacketValueSize
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

func DecodeNetworkPacket(sensorStorage *SensorStorage, sensorMap map[uint64]*SensorConfig, packetData []byte, packetTime time.Time) error {
	// header(8) + one sensor value(1,2) = 11 bytes minimum
	if len(packetData) < SensorPacketMinSize {
		return fmt.Errorf("sensor packet, size too small")
	}

	now := time.Now()
	Length := int(packetData[SensorPacketLengthOffset] * 2)
	Version := uint8(packetData[SensorPacketVersionOffset])
	Mac := uint64(binary.LittleEndian.Uint64(packetData[SensorPacketMacOffset : SensorPacketMacOffset+6]))

	if Version == 1 {
		if len(packetData) != Length {
			return fmt.Errorf("sensor packet, unexpected length, %d != %d", len(packetData), Length)
		}
		// Now decode the values.
		offset := SensorPacketDataOffset
		for offset <= Length-SensorPacketValueSize {
			id := packetData[offset]
			sensor := FindSensorConfig(id, Mac, sensorMap)
			value := SensorValue{Sensor: sensor}
			value.Value = int32(uint32(packetData[offset+1])<<8 | uint32(packetData[offset+2]))
			if err := sensorStorage.WriteSensorValue(sensor, value, now); err != nil {
				sensorStorage.logger.LogError(err)
			}
			offset += SensorPacketValueSize
		}
		return nil
	} else {
		return fmt.Errorf("sensor packet, unknown version %d", Version)
	}
}

func EncodeNetworkPacket(pkt *SensorPacket) ([]byte, error) {
	if pkt.Version != 1 {
		return nil, fmt.Errorf("sensor packet, unknown version %d", pkt.Version)
	}

	// Compute the length of the packet.
	length := SensorPacketHeaderSize + len(pkt.Values)*SensorPacketValueSize
	length += length & 1 // Padding byte if needed
	if length > SensorPacketMaxSize {
		return nil, fmt.Errorf("sensor packet, too many values")
	}

	data := make([]byte, length)

	data[SensorPacketLengthOffset] = uint8(length / 2)
	data[SensorPacketVersionOffset] = pkt.Version

	offset := SensorPacketHeaderSize
	for _, v := range pkt.Values {
		data[offset] = v.Sensor.Id()
		binary.LittleEndian.PutUint16(data[offset+1:offset+3], uint16(v.Value&0xFFFF))
		offset += SensorPacketValueSize
	}

	return data, nil
}
