package sensor_server

import "strings"

type SensorType uint8

// Note: The values must match the SensorType enum in the firmware.
//
//	So it is not a good idea to change the value of a SensorType.
//	Add new types at the end of the list.
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
	Presence    SensorType = 0x0B // (s8, 0-1-2-3, 0=unknown, 1=none, 2=detection, 3=presence)
	Distance    SensorType = 0x0C // (s16, cm)
	UV          SensorType = 0x0D // (s16, index)
	CO          SensorType = 0x0E // (s16, ppm)
	Vibration   SensorType = 0x0F // (s8,  <=16=none, <=64=low, <=128=medium, <=192=high, <=255=extreme)
	State       SensorType = 0x10 // (s32 (u8[4]), sensor model, sensor state)
	MacAddress  SensorType = 0x11 // (u64, MAC address)
	SensorCount            = 0x12 // The maximum number of sensor types (highest index + 1)
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
	TypeS64  SensorFieldType = 0x40
	TypeU8   SensorFieldType = 0x88
	TypeU16  SensorFieldType = 0x90
	TypeU32  SensorFieldType = 0xA0
	TypeU64  SensorFieldType = 0xC0
)

var SensorTypeToSensorMap map[SensorType]Sensor = map[SensorType]Sensor{
	Temperature: {mName: "temperature", mType: Temperature, mField: TypeS8, mFreq: 12},
	Humidity:    {mName: "humidity", mType: Humidity, mField: TypeS8, mFreq: 12},
	Pressure:    {mName: "pressure", mType: Pressure, mField: TypeS16, mFreq: 12},
	Light:       {mName: "light", mType: Light, mField: TypeS16, mFreq: 120},
	CO2:         {mName: "co2", mType: CO2, mField: TypeS16, mFreq: 60},
	VOC:         {mName: "voc", mType: VOC, mField: TypeS16, mFreq: 30},
	PM1_0:       {mName: "pm1_0", mType: PM1_0, mField: TypeS16, mFreq: 30},
	PM2_5:       {mName: "pm2_5", mType: PM2_5, mField: TypeS16, mFreq: 30},
	PM10:        {mName: "pm10", mType: PM10, mField: TypeS16, mFreq: 30},
	Noise:       {mName: "noise", mType: Noise, mField: TypeS8, mFreq: 60},
	Presence:    {mName: "presence", mType: Presence, mField: TypeS8, mFreq: 7200},
	Distance:    {mName: "distance", mType: Distance, mField: TypeS16, mFreq: 7200},
	UV:          {mName: "uv", mType: UV, mField: TypeS8, mFreq: 30},
	CO:          {mName: "co", mType: CO, mField: TypeS8, mFreq: 30},
	Vibration:   {mName: "vibration", mType: Vibration, mField: TypeS8, mFreq: 3600},
	State:       {mName: "state", mType: State, mField: TypeS16, mFreq: 12},
	MacAddress:  {mName: "macaddress", mType: MacAddress, mField: TypeU64, mFreq: 1},
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
	"presence":    Presence,
	"distance":    Distance,
	"uv":          UV,
	"co":          CO,
	"vibration":   Vibration,
	"state":       State,
	"macaddress":  MacAddress,
}

type Sensor struct {
	mName  string          // sensor name, e.g. "temperature"
	mType  SensorType      // sensor type
	mField SensorFieldType // data field type
	mFreq  int32           // samples per hour
}

func NewSensorByName(name string) Sensor {
	name = strings.ToLower(name)
	if st, ok := SensorNameToSensorTypeMap[name]; ok {
		if sensor, ok := SensorTypeToSensorMap[st]; ok {
			return sensor
		}
	}
	return Sensor{mName: "unknown", mType: Unknown, mField: TypeNone, mFreq: 0}
}

func (t Sensor) Name() string {
	return t.mName
}

func (t Sensor) FullIdentifier() uint64 {
	return (uint64(t.mField)<<8)&0xFF | uint64(t.mType)&0xFF | ((uint64(t.mFreq) << 32) & 0xFFFFFFFF00000000)
}

func (t Sensor) SizeInBits() int {
	return int(t.mField) & 0x7F
}

func (st Sensor) Index() int {
	return int(st.mType)
}

func (st Sensor) IsValid() bool {
	return st.mType != Unknown
}

func (st Sensor) IsMac() bool {
	return st.mType == MacAddress
}

func (st Sensor) String() string {
	return st.mName
}

func (st Sensor) SampleFrequency() int32 {
	return st.mFreq
}

func (st Sensor) SamplePeriodInMs() int32 {
	return 60 * 60 * 1000 / st.mFreq
}

func (st Sensor) Type() SensorType {
	return st.mType
}

func (st Sensor) FieldType() SensorFieldType {
	return st.mField
}
