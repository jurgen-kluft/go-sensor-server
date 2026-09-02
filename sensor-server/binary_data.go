package sensorserver

import (
	"encoding/binary"
)

type BinaryData struct {
	buf []byte // contents are the bytes buf[off : len(buf)]
	off int    // read at &buf[off], write at &buf[len(buf)]
}

func (data *BinaryData) SkipBytes(length int) {
	data.off += length
}
func (data *BinaryData) ReadMAC() MACAddress {
	mac := MACAddress{}
	copy(mac[:], data.buf[data.off:data.off+6])
	data.off += 6
	return mac
}
func (data *BinaryData) ReadUint8() uint8 {
	value := data.buf[data.off]
	data.off++
	return value
}
func (data *BinaryData) ReadUint16() uint16 {
	value := binary.LittleEndian.Uint16(data.buf[data.off : data.off+2])
	data.off += 2
	return value
}
func (data *BinaryData) ReadInt24() int32 {
	value := int32((uint32(data.buf[data.off]) << 8) | (uint32(data.buf[data.off+1]) << 16) | (uint32(data.buf[data.off+2]) << 24))
	data.off += 3
	value = value >> 8 // Sign extend to 32 bits
	return value
}
func (data *BinaryData) ReadUint32() uint32 {
	value := binary.LittleEndian.Uint32(data.buf[data.off : data.off+4])
	data.off += 4
	return value
}
func (data *BinaryData) ReadUint64() uint64 {
	value := binary.LittleEndian.Uint64(data.buf[data.off : data.off+8])
	data.off += 8
	return value
}
func (data *BinaryData) ReadBytes(length int) []byte {
	bytes := make([]byte, length)
	copy(bytes, data.buf[data.off:data.off+length])
	data.off += length
	return bytes
}

func (data *BinaryData) WriteUint8(value byte) {
	data.buf[data.off] = value
	data.off++
}
func (data *BinaryData) WriteUint16(value uint16) {
	binary.LittleEndian.PutUint16(data.buf[data.off:data.off+2], value)
	data.off += 2
}
func (data *BinaryData) WriteUint32(value uint32) {
	binary.LittleEndian.PutUint32(data.buf[data.off:data.off+4], value)
	data.off += 4
}
func (data *BinaryData) WriteUint32At(value uint32, offset int) {
	binary.LittleEndian.PutUint32(data.buf[offset:offset+4], value)
}
func (data *BinaryData) WriteUint64(value uint64) {
	binary.LittleEndian.PutUint64(data.buf[data.off:data.off+8], value)
	data.off += 8
}
func (data *BinaryData) WriteBytes(bytes []byte) {
	copy(data.buf[data.off:data.off+len(bytes)], bytes)
	data.off += len(bytes)
}
