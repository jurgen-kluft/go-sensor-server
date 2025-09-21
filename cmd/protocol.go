package main

import (
	"encoding/binary"
	"errors"
	"io"
	"net"

	sensor_server "github.com/jurgen-kluft/go-sensor-server"
)

func GetLength(p sensor_server.Packet) uint32 {
	return binary.BigEndian.Uint32(p.Body[0:1]) << 1
}

func GetVersion(p sensor_server.Packet) uint32 {
	return binary.BigEndian.Uint32(p.Body[1:2])
}

func GetTimeSync(p sensor_server.Packet) uint32 {
	return binary.BigEndian.Uint32(p.Body[2:6])
}

func GetBody(this sensor_server.Packet) []byte {
	return this.Body[6:]
}

func NewSensorPacket(buff []byte) sensor_server.Packet {
	return sensor_server.Packet{Body: buff}
}

type SensorPacketProtocol struct {
}

func (this *SensorPacketProtocol) ReadPacket(conn *net.TCPConn) (sensor_server.Packet, error) {
	var (
		header []byte = make([]byte, 6)
		length int32
	)

	if _, err := io.ReadFull(conn, header); err != nil {
		return sensor_server.Packet{}, err
	}
	if length = int32(binary.BigEndian.Uint32(header[:1])) * 2; length > 500 {
		return sensor_server.Packet{}, errors.New("the size of packet is larger than the limit")
	}

	buff := make([]byte, int32(len(header))+length)
	copy(buff[0:6], header)

	if _, err := io.ReadFull(conn, buff[4:]); err != nil {
		return sensor_server.Packet{}, err
	}

	return NewSensorPacket(buff), nil
}
