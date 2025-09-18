package sensor_server

import "strings"

type SensorModel uint8

const (
	GPIO   SensorModel = 0x00
	BH1750 SensorModel = 0x10
	BME280 SensorModel = 0x20
	SCD4X  SensorModel = 0x30
)

type SensorType uint8

const (
	Temperature SensorType = 0x00 // (s8, °C)
	Humidity    SensorType = 0x01 // (s8, %)
	Pressure    SensorType = 0x02 // (s16, hPa)
	Light       SensorType = 0x03 // (s16, lux)
	CO2         SensorType = 0x04 // (s16, ppm)
	VOC         SensorType = 0x05 // (s16, ppm)
	PM1_0       SensorType = 0x06 // (s16, µg/m3)
	PM2_5       SensorType = 0x07 // (s16, µg/m3)
	PM10        SensorType = 0x08 // (s16, µg/m3)
	Noise       SensorType = 0x09 // (s16, dB)
	Presence    SensorType = 0x0A // (s8, 0-1)
	Distance    SensorType = 0x0B // (s16, cm)
	UV          SensorType = 0x0C // (s16, index)
	CO          SensorType = 0x0D // (s16, ppm)
	Vibration   SensorType = 0x0E // (s8,  <=16=none, <=64=low, <=128=medium, <=192=high, <=255=extreme)
	State       SensorType = 0xA0 // (s32 (u8[4]), sensor model, sensor state)
	Unknown     SensorType = 0xFF // Unknown sensor type
)

type SensorState uint8

const (
	Off   SensorState = 0x10
	On    SensorState = 0x20
	Error SensorState = 0x30
)

type SensorFieldType uint8

const (
	TypeNone SensorFieldType = 0x00
	TypeBit  SensorFieldType = 0x01
	TypeS8   SensorFieldType = 0x08
	TypeS16  SensorFieldType = 0x10
	TypeS32  SensorFieldType = 0x20
)

func (t SensorFieldType) SizeInBits() int {
	return int(t)
}

// ToSensorFrequency returns the default frequency (samples per hour) for the given SensorType.

var SensorTypeToSampleFrequencyMap []int32 = []int32{
	60,
	60,
	60,
	120,
	60,
	60,
	60,
	60,
	60,
	60,
	3600,
	7200,
	60,
	60,
	3600,
	12,
}

func GetSamplePeriodInMsFromSensorType(st SensorType) int32 {
	return 60 * 60 * 1000 / SensorTypeToSampleFrequencyMap[int(st)]
}

var SensorFieldTypeMap []SensorFieldType = []SensorFieldType{
	TypeS8,
	TypeS8,
	TypeS16,
	TypeS16,
	TypeS16,
	TypeS16,
	TypeS16,
	TypeS16,
	TypeS16,
	TypeS8,
	TypeS8,
	TypeS16,
	TypeS8,
	TypeS8,
	TypeS8,
	TypeS16,
}

func GetFieldTypeFromType(st SensorType) SensorFieldType {
	return SensorFieldTypeMap[int(st)]
}

// String returns the string representation of the SensorType.
var SensorTypeName []string = []string{
	"Temperature",
	"Humidity",
	"Pressure",
	"Light",
	"CO2",
	"VOC",
	"PM1.0",
	"PM2.5",
	"PM10",
	"Noise",
	"Presence",
	"Distance",
	"UV",
	"CO",
	"Vibration",
	"State",
	"Unknown",
}

func (st SensorType) String() string {
	if int(st) < len(SensorTypeName) {
		return SensorTypeName[int(st)]
	}
	return "Unknown"
}

var StringToSensorTypeMap map[string]SensorType = map[string]SensorType{
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
	"presence":    Presence,
	"distance":    Distance,
	"uv":          UV,
	"co":          CO,
	"vibration":   Vibration,
	"state":       State,
	"unknown":     Unknown,
}

func NewSensorType(name string) SensorType {
	name = strings.ToLower(name)
	if st, ok := StringToSensorTypeMap[name]; ok {
		return st
	}
	return Unknown
}

func (st SensorType) IsValid() bool {
	return st != Unknown
}

func (st SensorType) FromString(name string) SensorType {
	name = strings.ToLower(name)
	if st, ok := StringToSensorTypeMap[name]; ok {
		return st
	}
	return State
}
