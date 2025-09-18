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

func GetBody(this sensor_server.Packet) []byte {
	return this.Body[4:]
}

func NewEchoPacket(buff []byte, hasLengthField bool) sensor_server.Packet {
	p := sensor_server.Packet{}
	if hasLengthField {
		p.Body = buff
	} else {
		p.Body = make([]byte, 4+len(buff))
		binary.BigEndian.PutUint32(p.Body[0:1], uint32(len(buff)/2))
		copy(p.Body[4:], buff)
	}
	return p
}

type SensorPacketProtocol struct {
}

func (this *SensorPacketProtocol) ReadPacket(conn *net.TCPConn) (sensor_server.Packet, error) {
	var (
		headerBytes []byte = make([]byte, 4)
		length      uint32
	)

	// read length
	if _, err := io.ReadFull(conn, headerBytes); err != nil {
		return sensor_server.Packet{}, err
	}
	if length = binary.BigEndian.Uint32(headerBytes[:1]) * 2; length > 500 {
		return sensor_server.Packet{}, errors.New("the size of packet is larger than the limit")
	}

	buff := make([]byte, 4+length)
	copy(buff[0:4], headerBytes)

	// read body ( buff = headerBytes + body )
	if _, err := io.ReadFull(conn, buff[4:]); err != nil {
		return sensor_server.Packet{}, err
	}

	return NewEchoPacket(buff, true), nil
}
