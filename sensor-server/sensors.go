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
	SENSOR_ID_NOISE       SensorType = 16 // Noise
	SENSOR_ID_VIBRATION   SensorType = 17 // Vibration
	SENSOR_ID_STATE       SensorType = 18 // State
	SENSOR_ID_BATTERY     SensorType = 19 // Battery
	SENSOR_ID_PRESENCE1   SensorType = 20 // Presence1
	SENSOR_ID_PRESENCE2   SensorType = 21 // Presence2
	SENSOR_ID_PRESENCE3   SensorType = 22 // Presence3
	SENSOR_ID_DISTANCE1   SensorType = 23 // Distance1
	SENSOR_ID_DISTANCE2   SensorType = 24 // Distance2
	SENSOR_ID_DISTANCE3   SensorType = 25 // Distance3
	SENSOR_ID_POS1_X      SensorType = 26 // X
	SENSOR_ID_POS1_Y      SensorType = 27 // Y
	SENSOR_ID_POS1_Z      SensorType = 28 // Z
	SENSOR_ID_POS2_X      SensorType = 29 // X
	SENSOR_ID_POS2_Y      SensorType = 30 // Y
	SENSOR_ID_POS2_Z      SensorType = 31 // Z
	SENSOR_ID_POS3_X      SensorType = 32 // X
	SENSOR_ID_POS3_Y      SensorType = 33 // Y
	SENSOR_ID_POS3_Z      SensorType = 34 // Z
	SENSOR_ID_RSSI        SensorType = 35 // RSSI
	SENSOR_ID_PERF1       SensorType = 36 // Performance Metric 1
	SENSOR_ID_PERF2       SensorType = 37 // Performance Metric 2
	SENSOR_ID_PERF3       SensorType = 38 // Performance Metric 3
	SENSOR_ID_VOLTAGE     SensorType = 39 // Voltage
	SENSOR_ID_CURRENT     SensorType = 40 // Current
	SENSOR_ID_POWER       SensorType = 41 // Power
	SENSOR_ID_ENERGY      SensorType = 42 // Energy
	SENSOR_ID_JSON        SensorType = 43 // JSON Data
	SENSOR_ID_IMAGE       SensorType = 44 // Image Data
	SENSOR_ID_GAS_M3      SensorType = 45 // Gas Meter
	SENSOR_ID_WATER_M3    SensorType = 46 // Water Meter
	SENSOR_ID_kWAh        SensorType = 47 // Electric Meter (kilowatt-hours)
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
	SENSOR_ID_NOISE:       {Min: 0, Max: 130},
	SENSOR_ID_VIBRATION:   {Min: 0, Max: 10000},
	SENSOR_ID_STATE:       {Min: 0, Max: 1},
	SENSOR_ID_BATTERY:     {Min: 0, Max: 100},
	SENSOR_ID_PRESENCE1:   {Min: 0, Max: 1},
	SENSOR_ID_PRESENCE2:   {Min: 0, Max: 1},
	SENSOR_ID_PRESENCE3:   {Min: 0, Max: 1},
	SENSOR_ID_DISTANCE1:   {Min: 0, Max: 10000},
	SENSOR_ID_DISTANCE2:   {Min: 0, Max: 10000},
	SENSOR_ID_DISTANCE3:   {Min: 0, Max: 10000},
	SENSOR_ID_POS1_X:      {Min: -10000, Max: 10000},
	SENSOR_ID_POS1_Y:      {Min: -10000, Max: 10000},
	SENSOR_ID_POS1_Z:      {Min: -10000, Max: 10000},
	SENSOR_ID_POS2_X:      {Min: -10000, Max: 10000},
	SENSOR_ID_POS2_Y:      {Min: -10000, Max: 10000},
	SENSOR_ID_POS2_Z:      {Min: -10000, Max: 10000},
	SENSOR_ID_POS3_X:      {Min: -10000, Max: 10000},
	SENSOR_ID_POS3_Y:      {Min: -10000, Max: 10000},
	SENSOR_ID_POS3_Z:      {Min: -10000, Max: 10000},
	SENSOR_ID_RSSI:        {Min: -128, Max: 127},
	SENSOR_ID_PERF1:       {Min: 0, Max: 1000000},
	SENSOR_ID_PERF2:       {Min: 0, Max: 1000000},
	SENSOR_ID_PERF3:       {Min: 0, Max: 1000000},
	SENSOR_ID_VOLTAGE:     {Min: 0, Max: 5000},
	SENSOR_ID_CURRENT:     {Min: -2000, Max: 2000},
	SENSOR_ID_POWER:       {Min: -100000, Max: 100000},
	SENSOR_ID_ENERGY:      {Min: 0, Max: 16000000},
	SENSOR_ID_JSON:        {Min: 0, Max: 0},
	SENSOR_ID_IMAGE:       {Min: 0, Max: 0},
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
	SENSOR_ID_NOISE:       "Noise",
	SENSOR_ID_VIBRATION:   "Vibration",
	SENSOR_ID_STATE:       "State",
	SENSOR_ID_BATTERY:     "Battery",
	SENSOR_ID_PRESENCE1:   "Presence 1",
	SENSOR_ID_PRESENCE2:   "Presence 2",
	SENSOR_ID_PRESENCE3:   "Presence 3",
	SENSOR_ID_DISTANCE1:   "Distance 1",
	SENSOR_ID_DISTANCE2:   "Distance 2",
	SENSOR_ID_DISTANCE3:   "Distance 3",
	SENSOR_ID_POS1_X:      "Position 1 X",
	SENSOR_ID_POS1_Y:      "Position 1 Y",
	SENSOR_ID_POS1_Z:      "Position 1 Z",
	SENSOR_ID_POS2_X:      "Position 2 X",
	SENSOR_ID_POS2_Y:      "Position 2 Y",
	SENSOR_ID_POS2_Z:      "Position 2 Z",
	SENSOR_ID_POS3_X:      "Position 3 X",
	SENSOR_ID_POS3_Y:      "Position 3 Y",
	SENSOR_ID_POS3_Z:      "Position 3 Z",
	SENSOR_ID_RSSI:        "RSSI",
	SENSOR_ID_PERF1:       "Performance Metric 1",
	SENSOR_ID_PERF2:       "Performance Metric 2",
	SENSOR_ID_PERF3:       "Performance Metric 3",
	SENSOR_ID_VOLTAGE:     "Voltage",
	SENSOR_ID_CURRENT:     "Current",
	SENSOR_ID_POWER:       "Power",
	SENSOR_ID_ENERGY:      "Energy",
	SENSOR_ID_JSON:        "JSON Data",
	SENSOR_ID_IMAGE:       "Image Data",
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
	"Noise":                      SENSOR_ID_NOISE,
	"Vibration":                  SENSOR_ID_VIBRATION,
	"State":                      SENSOR_ID_STATE,
	"Battery":                    SENSOR_ID_BATTERY,
	"Presence 1":                 SENSOR_ID_PRESENCE1,
	"Presence 2":                 SENSOR_ID_PRESENCE2,
	"Presence 3":                 SENSOR_ID_PRESENCE3,
	"Distance 1":                 SENSOR_ID_DISTANCE1,
	"Distance 2":                 SENSOR_ID_DISTANCE2,
	"Distance 3":                 SENSOR_ID_DISTANCE3,
	"Position 1 X":               SENSOR_ID_POS1_X,
	"Position 1 Y":               SENSOR_ID_POS1_Y,
	"Position 1 Z":               SENSOR_ID_POS1_Z,
	"Position 2 X":               SENSOR_ID_POS2_X,
	"Position 2 Y":               SENSOR_ID_POS2_Y,
	"Position 2 Z":               SENSOR_ID_POS2_Z,
	"Position 3 X":               SENSOR_ID_POS3_X,
	"Position 3 Y":               SENSOR_ID_POS3_Y,
	"Position 3 Z":               SENSOR_ID_POS3_Z,
	"RSSI":                       SENSOR_ID_RSSI,
	"Performance Metric 1":       SENSOR_ID_PERF1,
	"Performance Metric 2":       SENSOR_ID_PERF2,
	"Performance Metric 3":       SENSOR_ID_PERF3,
	"Voltage":                    SENSOR_ID_VOLTAGE,
	"Current":                    SENSOR_ID_CURRENT,
	"Power":                      SENSOR_ID_POWER,
	"Energy":                     SENSOR_ID_ENERGY,
	"JSON Data":                  SENSOR_ID_JSON,
	"Image Data":                 SENSOR_ID_IMAGE,
	"Gas Meter":                  SENSOR_ID_GAS_M3,
	"Water Meter":                SENSOR_ID_WATER_M3,
	"Electric Meter":             SENSOR_ID_kWAh,
}
