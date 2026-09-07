package sensorserver

import "fmt"

type SensorType uint8

const (
	SENSOR_ID_UNKNOWN     SensorType = 0  // Unknown
	SENSOR_ID_TEMPERATURE SensorType = 1  // Temperature
	SENSOR_ID_HUMIDITY    SensorType = 2  // Humidity
	SENSOR_ID_PRESSURE    SensorType = 3  // Pressure
	SENSOR_ID_LIGHT       SensorType = 4  // Light
	SENSOR_ID_UV          SensorType = 5  // UV
	SENSOR_ID_CO          SensorType = 6  // Carbon Monoxide
	SENSOR_ID_CO2         SensorType = 7  // Carbon Dioxide
	SENSOR_ID_HCHO        SensorType = 8  // Formaldehyde
	SENSOR_ID_VOC         SensorType = 9  // Volatile Organic Compounds
	SENSOR_ID_NOX         SensorType = 10 // Nitrogen Oxides
	SENSOR_ID_PM005       SensorType = 11 // Particulate Matter 0.5
	SENSOR_ID_PM010       SensorType = 12 // Particulate Matter 1.0
	SENSOR_ID_PM025       SensorType = 13 // Particulate Matter 2.5
	SENSOR_ID_PM040       SensorType = 14 // Particulate Matter 4.0
	SENSOR_ID_PM100       SensorType = 15 // Particulate Matter 10.0
	SENSOR_ID_SOUND       SensorType = 16 // Noise/Sound Level
	SENSOR_ID_BINARY      SensorType = 17 // On/Off, Open/Closed, True/False, etc.
	SENSOR_ID_AMPLITUDE   SensorType = 18 // Like RSSI or other signal strength
	SENSOR_ID_DURATION    SensorType = 19 // Duration in milliseconds
	SENSOR_ID_FREQUENCY   SensorType = 20 // Frequency in Hertz
	SENSOR_ID_BATTERY     SensorType = 21 // Battery
	SENSOR_ID_VOLTAGE     SensorType = 22 // Voltage
	SENSOR_ID_CURRENT     SensorType = 23 // Current
	SENSOR_ID_POWER       SensorType = 24 // Power
	SENSOR_ID_ENERGY      SensorType = 25 // Energy
	SENSOR_ID_GAS_M3      SensorType = 26 // Gas Meter
	SENSOR_ID_WATER_M3    SensorType = 27 // Water Meter
	SENSOR_ID_kWAh        SensorType = 28 // Electric Meter (kilowatt-hours)
)

func ToSensorType(id uint16) SensorType {
	if id > uint16(SENSOR_ID_kWAh) {
		return SENSOR_ID_UNKNOWN
	}
	return SensorType(id)
}

type SensorTypeValueRange struct {
	Min int32 // int24
	Max int32 // int24
}

var SensorTypeValueRanges = map[SensorType]SensorTypeValueRange{
	SENSOR_ID_UNKNOWN:     {Min: -32768, Max: 32767},
	SENSOR_ID_TEMPERATURE: {Min: -400, Max: 1250},
	SENSOR_ID_HUMIDITY:    {Min: 0, Max: 100},
	SENSOR_ID_PRESSURE:    {Min: 30000, Max: 110000},
	SENSOR_ID_LIGHT:       {Min: 0, Max: 100000},
	SENSOR_ID_UV:          {Min: 0, Max: 15},
	SENSOR_ID_CO:          {Min: 0, Max: 10000},
	SENSOR_ID_CO2:         {Min: 400, Max: 5000},
	SENSOR_ID_HCHO:        {Min: 0, Max: 1000},
	SENSOR_ID_VOC:         {Min: 0, Max: 10000},
	SENSOR_ID_NOX:         {Min: 0, Max: 10000},
	SENSOR_ID_PM005:       {Min: 0, Max: 250},
	SENSOR_ID_PM010:       {Min: 0, Max: 500},
	SENSOR_ID_PM025:       {Min: 0, Max: 500},
	SENSOR_ID_PM040:       {Min: 0, Max: 500},
	SENSOR_ID_PM100:       {Min: 0, Max: 1000},
	SENSOR_ID_SOUND:       {Min: 0, Max: 130},     // Sound level in dB
	SENSOR_ID_BINARY:      {Min: 0, Max: 1},       // Binary sensor (0 or 1)
	SENSOR_ID_AMPLITUDE:   {Min: 0, Max: 100},     // Amplitude in percentage
	SENSOR_ID_DURATION:    {Min: 0, Max: 3600000}, // Duration in milliseconds (up to 1 hour)
	SENSOR_ID_FREQUENCY:   {Min: 0, Max: 1000000}, // Frequency in Hertz
	SENSOR_ID_BATTERY:     {Min: 0, Max: 100},     // Battery level in percentage
	SENSOR_ID_VOLTAGE:     {Min: 0, Max: 5000},
	SENSOR_ID_CURRENT:     {Min: -2000, Max: 2000},
	SENSOR_ID_POWER:       {Min: -100000, Max: 100000},
	SENSOR_ID_ENERGY:      {Min: 0, Max: 16000000},
	SENSOR_ID_GAS_M3:      {Min: 0, Max: 16000000},
	SENSOR_ID_WATER_M3:    {Min: 0, Max: 16000000},
	SENSOR_ID_kWAh:        {Min: 0, Max: 16000000},
}

var SensorTypeNames = map[SensorType]string{
	SENSOR_ID_UNKNOWN:     "Unknown",
	SENSOR_ID_TEMPERATURE: "Temperature",
	SENSOR_ID_HUMIDITY:    "Humidity",
	SENSOR_ID_PRESSURE:    "Pressure",
	SENSOR_ID_LIGHT:       "Light",
	SENSOR_ID_UV:          "UV",
	SENSOR_ID_CO:          "Carbon Monoxide",
	SENSOR_ID_CO2:         "Carbon Dioxide",
	SENSOR_ID_HCHO:        "Formaldehyde",
	SENSOR_ID_VOC:         "Volatile Organic Compounds",
	SENSOR_ID_NOX:         "Nitrogen Oxides",
	SENSOR_ID_PM005:       "Particulate Matter 0.5",
	SENSOR_ID_PM010:       "Particulate Matter 1.0",
	SENSOR_ID_PM025:       "Particulate Matter 2.5",
	SENSOR_ID_PM040:       "Particulate Matter 4.0",
	SENSOR_ID_PM100:       "Particulate Matter 10.0",
	SENSOR_ID_SOUND:       "Noise/Sound Level",
	SENSOR_ID_BINARY:      "On/Off",
	SENSOR_ID_AMPLITUDE:   "Amplitude",
	SENSOR_ID_DURATION:    "Duration",
	SENSOR_ID_FREQUENCY:   "Frequency",
	SENSOR_ID_BATTERY:     "Battery",
	SENSOR_ID_VOLTAGE:     "Voltage",
	SENSOR_ID_CURRENT:     "Current",
	SENSOR_ID_POWER:       "Power",
	SENSOR_ID_ENERGY:      "Energy",
	SENSOR_ID_GAS_M3:      "Gas Meter",
	SENSOR_ID_WATER_M3:    "Water Meter",
	SENSOR_ID_kWAh:        "Electric Meter",
}

func (sensorType SensorType) String() string {
	name, exists := SensorTypeNames[sensorType]
	if !exists {
		return fmt.Sprintf("Unknown(%d)", sensorType)
	}
	return name
}

var SensorNameToSensorType = map[string]SensorType{
	"Unknown":                    SENSOR_ID_UNKNOWN,
	"Temperature":                SENSOR_ID_TEMPERATURE,
	"Humidity":                   SENSOR_ID_HUMIDITY,
	"Pressure":                   SENSOR_ID_PRESSURE,
	"Light":                      SENSOR_ID_LIGHT,
	"UV":                         SENSOR_ID_UV,
	"Carbon Monoxide":            SENSOR_ID_CO,
	"Carbon Dioxide":             SENSOR_ID_CO2,
	"Formaldehyde":               SENSOR_ID_HCHO,
	"Volatile Organic Compounds": SENSOR_ID_VOC,
	"Nitrogen Oxides":            SENSOR_ID_NOX,
	"Particulate Matter 0.5":     SENSOR_ID_PM005,
	"Particulate Matter 1.0":     SENSOR_ID_PM010,
	"Particulate Matter 2.5":     SENSOR_ID_PM025,
	"Particulate Matter 4.0":     SENSOR_ID_PM040,
	"Particulate Matter 10.0":    SENSOR_ID_PM100,
	"Sound":                      SENSOR_ID_SOUND,
	"Binary":                     SENSOR_ID_BINARY,
	"Amplitude":                  SENSOR_ID_AMPLITUDE,
	"Duration":                   SENSOR_ID_DURATION,
	"Frequency":                  SENSOR_ID_FREQUENCY,
	"Battery":                    SENSOR_ID_BATTERY,
	"Voltage":                    SENSOR_ID_VOLTAGE,
	"Current":                    SENSOR_ID_CURRENT,
	"Power":                      SENSOR_ID_POWER,
	"Energy":                     SENSOR_ID_ENERGY,
	"Gas Meter":                  SENSOR_ID_GAS_M3,
	"Water Meter":                SENSOR_ID_WATER_M3,
	"Electric Meter":             SENSOR_ID_kWAh,
}
