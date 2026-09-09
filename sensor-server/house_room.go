package sensorserver

type Room struct {
	SensorCount        uint8
	SensorOccupancy    uint32
	SensorValueOffsets []uint16
}

type RoomType uint8

const (
	ROOM_LIVING         RoomType = 0
	ROOM_DINING         RoomType = 1
	ROOM_KITCHEN        RoomType = 2
	ROOM_BATHROOM       RoomType = 3
	ROOM_BEDROOM1       RoomType = 4
	ROOM_BEDROOM2       RoomType = 5
	ROOM_STUDYROOM      RoomType = 6
	ROOM_MASTER_BEDROOM RoomType = 7
	ROOM_STAIRCASE      RoomType = 8
	ROOM_ENTRANCE       RoomType = 9
	ROOM_COUNT          RoomType = 10
	ROOM_INVALID        RoomType = 255
)

var roomTypeNames = map[RoomType]string{
	ROOM_LIVING:         "living",
	ROOM_DINING:         "dining",
	ROOM_KITCHEN:        "kitchen",
	ROOM_BATHROOM:       "bathroom",
	ROOM_BEDROOM1:       "bedroom1",
	ROOM_BEDROOM2:       "bedroom2",
	ROOM_STUDYROOM:      "studyroom",
	ROOM_MASTER_BEDROOM: "master_bedroom",
	ROOM_STAIRCASE:      "staircase",
	ROOM_ENTRANCE:       "entrance",
}

var roomNameToType = map[string]RoomType{
	"living":         ROOM_LIVING,
	"dining":         ROOM_DINING,
	"kitchen":        ROOM_KITCHEN,
	"bathroom":       ROOM_BATHROOM,
	"bedroom1":       ROOM_BEDROOM1,
	"bedroom2":       ROOM_BEDROOM2,
	"studyroom":      ROOM_STUDYROOM,
	"master_bedroom": ROOM_MASTER_BEDROOM,
	"staircase":      ROOM_STAIRCASE,
	"entrance":       ROOM_ENTRANCE,
	// aliases
	"storage":  ROOM_BEDROOM1, // Assuming storage is an alias for bedroom1
	"backroom": ROOM_BEDROOM2, // Assuming backroom is an alias for bedroom2
}

func (r RoomType) String() string {
	if name, ok := roomTypeNames[r]; ok {
		return name
	}
	return "unknown"
}

func RoomTypeFromString(s string) RoomType {
	if roomType, ok := roomNameToType[s]; ok {
		return roomType
	}
	return ROOM_INVALID // Unknown room type
}
