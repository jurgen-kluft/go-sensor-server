package sensorserver

type RoomType uint8

const (
	ROOM_LIVING     RoomType = 0
	ROOM_DINING     RoomType = 1
	ROOM_KITCHEN    RoomType = 2
	ROOM_BATHROOM   RoomType = 3
	ROOM_BEDROOM    RoomType = 4
	ROOM_STUDY      RoomType = 5
	ROOM_MASTER_BED RoomType = 6
	ROOM_STAIRCASE  RoomType = 7
	ROOM_ENTRANCE   RoomType = 8
	ROOM_COUNT      RoomType = 9
)

type FloorType uint8

const (
	FLOOR_BASEMENT FloorType = 0
	FLOOR_GROUND   FloorType = 1
	FLOOR_FIRST    FloorType = 2
	FLOOR_SECOND   FloorType = 3
	FLOOR_COUNT    FloorType = 4
)

type RoomID uint8

type Room struct {
	SensorDataOffset  uint16
	SensorOccupancy   uint32
	SensorByteOffsets []uint8
}

type House struct {
	Rooms []Room
}
