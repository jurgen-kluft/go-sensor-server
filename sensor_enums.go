package sensor_server

type SensorType uint8

// Note:
//
//	The values are used to write the sensor data to disk in binary format, so
//	changing the values will break compatibility with existing data files.
const (
	Unknown     SensorType = 0    // Unknown sensor type
	Temperature SensorType = 0x01 // (s8, °C)
	Humidity    SensorType = 0x02 // (s8, %)
	Pressure    SensorType = 0x03 // (s16, hPa)
	Light       SensorType = 0x04 // (s16, lux)
	CO2         SensorType = 0x05 // (s16, ppm)
	VOC         SensorType = 0x06 // (s16, ppm)
	PM1_0       SensorType = 0x07 // (s16, µg/m3)
	PM2_5       SensorType = 0x08 // (s16, µg/m3)
	PM10        SensorType = 0x09 // (s16, µg/m3)
	Noise       SensorType = 0x0A // (s16, dB)
	UV          SensorType = 0x0B // (s16, index)
	CO          SensorType = 0x0C // (s16, ppm)
	Vibration   SensorType = 0x0D // (s8,  <=16=none, <=64=low, <=128=medium, <=192=high, <=255=extreme)
	OpenClose   SensorType = 0x0E // (u8, 0=none, 1=open, 2=close)
	Presence1   SensorType = 0x10 // (u8, 0=none, 1=trigger up/down, 2=presence)
	Presence2   SensorType = 0x11 // (u8, 0=none, 1=trigger up/down, 2=presence)
	Presence3   SensorType = 0x12 // (u8, 0=none, 1=trigger up/down, 2=presence)
	Distance1   SensorType = 0x13 // (u16, cm)
	Distance2   SensorType = 0x14 // (u16, cm)
	Distance3   SensorType = 0x15 // (u16, cm)
	Battery     SensorType = 0x16 // (u8, battery level 0-100%)
	State       SensorType = 0x17 // (s32 (u8[4]), sensor model, sensor state)
	RSSI        SensorType = 0x18 // (s16, dBm)
	SensorCount            = 0x19 // The maximum number of sensor types (highest index + 1)
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
	TypeU1   SensorFieldType = 0x01
	TypeU2   SensorFieldType = 0x02
	TypeS8   SensorFieldType = 0x08
	TypeS16  SensorFieldType = 0x10
	TypeS32  SensorFieldType = 0x20
	TypeS64  SensorFieldType = 0x40
	TypeU8   SensorFieldType = 0x88
	TypeU16  SensorFieldType = 0x90
	TypeU32  SensorFieldType = 0xA0
	TypeU64  SensorFieldType = 0xC0
)

var SensorFieldNameToSensorFieldTypeMap map[string]SensorFieldType = map[string]SensorFieldType{
	"none": TypeNone,
	"u1":   TypeU1,
	"u2":   TypeU2,
	"s8":   TypeS8,
	"s16":  TypeS16,
	"s32":  TypeS32,
	"s64":  TypeS64,
	"u8":   TypeU8,
	"u16":  TypeU16,
	"u32":  TypeU32,
	"u64":  TypeU64,
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
	"openclose":   OpenClose,
	"presence1":   Presence1,
	"presence2":   Presence2,
	"presence3":   Presence3,
	"distance1":   Distance1,
	"distance2":   Distance2,
	"distance3":   Distance3,
	"battery":     Battery,
	"state":       State,
}
