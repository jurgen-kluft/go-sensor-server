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
	CO2         SensorType = 5  // (s16, ppm)
	VOC         SensorType = 6  // (s16, ppm)
	PM1_0       SensorType = 7  // (s16, µg/m3)
	PM2_5       SensorType = 8  // (s16, µg/m3)
	PM10        SensorType = 9  // (s16, µg/m3)
	Noise       SensorType = 10 // (s16, dB)
	UV          SensorType = 11 // (s16, index)
	CO          SensorType = 12 // (s16, ppm)
	Vibration   SensorType = 13 // (s8,  <=16=none, <=64=low, <=128=medium, <=192=high, <=255=extreme)
	State       SensorType = 14 // (s32 (u8[4]), sensor model, sensor state)
	Battery     SensorType = 15 // (u8, battery level 0-100%)
	Switch      SensorType = 16 // (u8, 0=none, 1=open, 2=close)
	Presence1   SensorType = 17 // (u8, 0=none, 1=trigger up/down, 2=presence)
	Presence2   SensorType = 18 // (u8, 0=none, 1=trigger up/down, 2=presence)
	Presence3   SensorType = 19 // (u8, 0=none, 1=trigger up/down, 2=presence)
	Distance1   SensorType = 20 // (u16, cm)
	Distance2   SensorType = 21 // (u16, cm)
	Distance3   SensorType = 22 // (u16, cm)
	X           SensorType = 23 // (s16, cm)
	Y           SensorType = 24 // (s16, cm)
	Z           SensorType = 25 // (s16, cm)
	RSSI        SensorType = 26 // (s16, dBm)
	SensorCount            = 27 // The maximum number of sensor types (highest index + 1)
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
	"pm1_0":       PM1_0,
	"pm2_5":       PM2_5,
	"pm10":        PM10,
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
}

var SensorTypeToSensorNameMap map[SensorType]string = map[SensorType]string{
	Temperature: "temperature",
	Humidity:    "humidity",
	Pressure:    "pressure",
	Light:       "light",
	CO2:         "co2",
	VOC:         "voc",
	PM1_0:       "pm1_0",
	PM2_5:       "pm2_5",
	PM10:        "pm10",
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
}

var SensorTypeToFieldTypeMap map[SensorType]SensorFieldType = map[SensorType]SensorFieldType{
	Unknown:     FieldTypeS16,
	Temperature: FieldTypeS8,
	Humidity:    FieldTypeS8,
	Pressure:    FieldTypeS16,
	Light:       FieldTypeS16,
	CO2:         FieldTypeS16,
	VOC:         FieldTypeS16,
	PM1_0:       FieldTypeS16,
	PM2_5:       FieldTypeS16,
	PM10:        FieldTypeS16,
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
	PM1_0:       OnePerTwoMinutes,
	PM2_5:       OnePerTwoMinutes,
	PM10:        OnePerTwoMinutes,
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
}
