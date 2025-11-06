package sensor_server

type SensorType uint8

// Note:
//
//	The values are used to write the sensor data to disk in binary format, so
//	changing the values will break compatibility with existing data files.
const (
	Unknown     SensorType = 0  // Unknown sensor type
	Temperature SensorType = 1  // (s8, °C)
	Humidity    SensorType = 2  // (s8, %)
	Pressure    SensorType = 3  // (s16, hPa)
	Light       SensorType = 4  // (s16, lux)
	UV          SensorType = 5  // (s16, index)
	CO          SensorType = 6  // (s16, ppm)
	CO2         SensorType = 7  // (s16, ppm)
	HCHO        SensorType = 8  // (s16, ppm)
	VOC         SensorType = 9  // (s16, ppm)
	NOX         SensorType = 10 // (s16, ppm)
	PM005       SensorType = 11 // (s16, µg/m3)
	PM010       SensorType = 12 // (s16, µg/m3)
	PM025       SensorType = 13 // (s16, µg/m3)
	PM040       SensorType = 14 // (s16, µg/m3)
	PM100       SensorType = 15 // (s16, µg/m3)
	Noise       SensorType = 16 // (s16, dB)
	Vibration   SensorType = 17 // (s8,  <=16=none, <=64=low, <=128=medium, <=192=high, <=255=extreme)
	State       SensorType = 18 // (s32 (u8[4]), sensor model, sensor state)
	Battery     SensorType = 19 // (u8, battery level 0-100%)
	Switch      SensorType = 20 // (u8, 0=none, 1=open, 2=close)
	Presence1   SensorType = 21 // (u8, 0=none, 1=trigger up/down, 2=presence)
	Presence2   SensorType = 22 // (u8, 0=none, 1=trigger up/down, 2=presence)
	Presence3   SensorType = 23 // (u8, 0=none, 1=trigger up/down, 2=presence)
	Distance1   SensorType = 24 // (u16, cm)
	Distance2   SensorType = 25 // (u16, cm)
	Distance3   SensorType = 26 // (u16, cm)
	X           SensorType = 27 // (s16, cm)
	Y           SensorType = 28 // (s16, cm)
	Z           SensorType = 29 // (s16, cm)
	RSSI        SensorType = 30 // (s16, dBm)
	Perf1       SensorType = 31 // Performance Metric 1
	Perf2       SensorType = 32 // Performance Metric 2
	Perf3       SensorType = 33 // Performance Metric 3
	SensorCount            = 34 // The maximum number of sensor types (highest index + 1)
)

type SensorState uint8

const (
	Off   SensorState = 0x10
	On    SensorState = 0x20
	Error SensorState = 0x30
)

type SensorFieldType uint8

const (
	FieldTypeNone SensorFieldType = 0x00
	FieldTypeU1   SensorFieldType = 0x01
	FieldTypeU2   SensorFieldType = 0x02
	FieldTypeS8   SensorFieldType = 0x08
	FieldTypeS16  SensorFieldType = 0x10
	FieldTypeS32  SensorFieldType = 0x20
	FieldTypeS64  SensorFieldType = 0x40
	FieldTypeU8   SensorFieldType = 0x88
	FieldTypeU16  SensorFieldType = 0x90
	FieldTypeU32  SensorFieldType = 0xA0
	FieldTypeU64  SensorFieldType = 0xC0
)

var SensorFieldNameToSensorFieldTypeMap map[string]SensorFieldType = map[string]SensorFieldType{
	"none": FieldTypeNone,
	"u1":   FieldTypeU1,
	"u2":   FieldTypeU2,
	"s8":   FieldTypeS8,
	"s16":  FieldTypeS16,
	"s32":  FieldTypeS32,
	"s64":  FieldTypeS64,
	"u8":   FieldTypeU8,
	"u16":  FieldTypeU16,
	"u32":  FieldTypeU32,
	"u64":  FieldTypeU64,
}

var SensorNameToSensorTypeMap map[string]SensorType = map[string]SensorType{
	"temperature": Temperature,
	"humidity":    Humidity,
	"pressure":    Pressure,
	"light":       Light,
	"co2":         CO2,
	"voc":         VOC,
	"pm005":       PM005,
	"pm010":       PM010,
	"pm025":       PM025,
	"pm100":       PM100,
	"noise":       Noise,
	"uv":          UV,
	"co":          CO,
	"vibration":   Vibration,
	"openclose":   Switch,
	"switch":      Switch,
	"presence1":   Presence1,
	"presence2":   Presence2,
	"presence3":   Presence3,
	"distance1":   Distance1,
	"distance2":   Distance2,
	"distance3":   Distance3,
	"battery":     Battery,
	"state":       State,
	"x":           X,
	"y":           Y,
	"z":           Z,
	"rssi":        RSSI,
	"perf1":       Perf1,
	"perf2":       Perf2,
	"perf3":       Perf3,
}

var SensorTypeToSensorNameMap map[SensorType]string = map[SensorType]string{
	Temperature: "temperature",
	Humidity:    "humidity",
	Pressure:    "pressure",
	Light:       "light",
	CO2:         "co2",
	VOC:         "voc",
	PM005:       "pm005",
	PM010:       "pm010",
	PM025:       "pm025",
	PM100:       "pm100",
	Noise:       "noise",
	UV:          "uv",
	CO:          "co",
	Vibration:   "vibration",
	Switch:      "switch",
	Presence1:   "presence1",
	Presence2:   "presence2",
	Presence3:   "presence3",
	Distance1:   "distance1",
	Distance2:   "distance2",
	Distance3:   "distance3",
	Battery:     "battery",
	State:       "state",
	X:           "x",
	Y:           "y",
	Z:           "z",
	RSSI:        "rssi",
	Perf1:       "perf1",
	Perf2:       "perf2",
	Perf3:       "perf3",
}

var SensorTypeToFieldTypeMap map[SensorType]SensorFieldType = map[SensorType]SensorFieldType{
	Unknown:     FieldTypeS16,
	Temperature: FieldTypeS8,
	Humidity:    FieldTypeS8,
	Pressure:    FieldTypeS16,
	Light:       FieldTypeS16,
	CO2:         FieldTypeS16,
	VOC:         FieldTypeS16,
	PM005:       FieldTypeS16,
	PM010:       FieldTypeS16,
	PM025:       FieldTypeS16,
	PM100:       FieldTypeS16,
	Noise:       FieldTypeS16,
	UV:          FieldTypeS16,
	CO:          FieldTypeS16,
	Vibration:   FieldTypeS8,
	Switch:      FieldTypeU8,
	Presence1:   FieldTypeU8,
	Presence2:   FieldTypeU8,
	Presence3:   FieldTypeU8,
	Distance1:   FieldTypeU16,
	Distance2:   FieldTypeU16,
	Distance3:   FieldTypeU16,
	X:           FieldTypeS16,
	Y:           FieldTypeS16,
	Z:           FieldTypeS16,
	RSSI:        FieldTypeS16,
	Perf1:       FieldTypeS16,
	Perf2:       FieldTypeS16,
	Perf3:       FieldTypeS16,
}

// Frequency unit is milliseconds
const (
	OneEveryHalfSecond int32 = 500
	OneEverySecond     int32 = 1000
	OneEveryHalfMinute int32 = 30 * OneEverySecond
	OneEveryMinute     int32 = 60 * OneEverySecond
	OnePerTwoMinutes   int32 = 2 * OneEveryMinute
	OneEveryTenSeconds int32 = 10 * OneEverySecond
)

var SensorTypeToFrequencyMap map[SensorType]int32 = map[SensorType]int32{
	Unknown:     OneEveryMinute,
	Temperature: OneEveryMinute,
	Humidity:    OneEveryMinute,
	Pressure:    OneEveryMinute,
	Light:       OneEveryHalfMinute,
	CO2:         OnePerTwoMinutes,
	VOC:         OnePerTwoMinutes,
	PM005:       OnePerTwoMinutes,
	PM010:       OnePerTwoMinutes,
	PM025:       OnePerTwoMinutes,
	PM100:       OnePerTwoMinutes,
	Noise:       OneEveryTenSeconds,
	UV:          OneEveryMinute,
	CO:          OneEveryTenSeconds,
	Vibration:   OneEveryTenSeconds,
	Switch:      OneEveryHalfSecond,
	Presence1:   OneEveryHalfSecond,
	Presence2:   OneEveryHalfSecond,
	Presence3:   OneEveryHalfSecond,
	Distance1:   OneEveryHalfSecond,
	Distance2:   OneEveryHalfSecond,
	Distance3:   OneEveryHalfSecond,
	X:           OneEveryHalfSecond,
	Y:           OneEveryHalfSecond,
	Z:           OneEveryHalfSecond,
	RSSI:        OneEveryTenSeconds,
	Perf1:       OneEveryMinute,
	Perf2:       OneEveryMinute,
	Perf3:       OneEveryMinute,
}
