package sensorserver

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
