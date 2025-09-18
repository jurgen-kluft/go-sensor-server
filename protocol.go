package sensor_server

import (
	"net"
)

type Packet struct {
	Body []byte
}

type Protocol interface {
	ReadPacket(conn *net.TCPConn) (Packet, error)
}
