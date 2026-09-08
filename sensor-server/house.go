package sensorserver

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

type FloorType uint8

const (
	FLOOR_BASEMENT FloorType = 0
	FLOOR_FIRST    FloorType = 1
	FLOOR_SECOND   FloorType = 2
	FLOOR_THIRD    FloorType = 3
	FLOOR_COUNT    FloorType = 4
	FLOOR_INVALID  FloorType = 255
)

var floorTypeNames = map[FloorType]string{
	FLOOR_BASEMENT: "basement",
	FLOOR_FIRST:    "first",
	FLOOR_SECOND:   "second",
	FLOOR_THIRD:    "third",
}

var floorNameToType = map[string]FloorType{
	"basement": FLOOR_BASEMENT,
	"first":    FLOOR_FIRST,
	"second":   FLOOR_SECOND,
	"third":    FLOOR_THIRD,
}

func (f FloorType) String() string {
	if name, ok := floorTypeNames[f]; ok {
		return name
	}
	return "unknown"
}

func FloorTypeFromString(s string) FloorType {
	if floorType, ok := floorNameToType[s]; ok {
		return floorType
	}
	return FLOOR_INVALID // Unknown floor type
}

type ContactType uint8

const (
	CONTACT_1     ContactType = 0
	CONTACT_2     ContactType = 1
	CONTACT_3     ContactType = 2
	CONTACT_4     ContactType = 3
	CONTACT_5     ContactType = 4
	CONTACT_6     ContactType = 5
	CONTACT_7     ContactType = 6
	CONTACT_8     ContactType = 7
	CONTACT_9     ContactType = 8
	CONTACT_10    ContactType = 9
	CONTACT_11    ContactType = 10
	CONTACT_12    ContactType = 11
	CONTACT_13    ContactType = 12
	CONTACT_14    ContactType = 13
	CONTACT_15    ContactType = 14
	CONTACT_16    ContactType = 15
	CONTACT_17    ContactType = 16
	CONTACT_18    ContactType = 17
	CONTACT_19    ContactType = 18
	CONTACT_20    ContactType = 19
	CONTACT_21    ContactType = 20
	CONTACT_22    ContactType = 21
	CONTACT_23    ContactType = 22
	CONTACT_24    ContactType = 23
	CONTACT_25    ContactType = 24
	CONTACT_26    ContactType = 25
	CONTACT_27    ContactType = 26
	CONTACT_28    ContactType = 27
	CONTACT_29    ContactType = 28
	CONTACT_30    ContactType = 29
	CONTACT_31    ContactType = 30
	CONTACT_32    ContactType = 31
	CONTACT_COUNT ContactType = 32
)

type Contact struct {
	ContactAndBatteryLevel uint8  // 0-1 & 0-127
	RSSI                   uint8  // -100 to 0 dBm
	BootTime               uint16 // ms, 0-60000
}

type Room struct {
	SensorCount        uint8
	SensorOccupancy    uint32
	SensorValueOffsets []uint16
}

type Global struct {
	Year                  uint16                 // Year (e.g., 2026)
	Month                 uint8                  // Month (1-12)
	Day                   uint8                  // Day of the month (1-31)
	DayOfWeek             uint8                  // Day of the week (0-6, where 0 = Sunday, 1 = Monday, ..., 6 = Saturday)
	Hour                  uint8                  // Hour of the day (0-23)
	Minute                uint8                  // Minute of the hour (0-59)
	Second                uint8                  // Second of the minute (0-59)
	OutsideHumidity       uint8                  // Percentage (0-100%)
	OutsideTemperature    int8                   // Celsius
	OutsidePressure       uint16                 // hPa
	OutsideLux            uint16                 // Lux (Illuminance)
	OutsideSunPercentage  uint8                  // 0-100%
	OutsideCloudCover     uint8                  // 0-100%
	OutsideWindSpeed      uint8                  // m/s
	OutsideWindDirection  uint8                  // 0-7 for N, NE, E, SE, S, SW, W, NW
	OutsideRainPercentage uint8                  // 0-100%
	OutsideSnowPercentage uint8                  // 0-100%
	ObjectsActive         uint32                 // a bit per object, 1 = active, 0 = inactive
	PresenceState         [FLOOR_COUNT]uint16    // m_presence_state[floor_id] |= (1 << room_id) if presence detected in that room
	ContactState          [CONTACT_COUNT]Contact // m_contact_state[CONTACT_COUNT]
}

type House struct {
	Global *Global
	Rooms  []Room
}
