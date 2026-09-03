package sensorserver

import "fmt"

/*
   typedef u8 sensor_type_t;

   enum sensor_type_e
   {
       SENSOR_ID_UNKNOWN       = 0,       // Unknown
       SENSOR_ID_TEMPERATURE   = 1,       // Temperature
       SENSOR_ID_HUMIDITY      = 2,       // Humidity
       SENSOR_ID_PRESSURE      = 3,       // Pressure
       SENSOR_ID_LIGHT         = 4,       // Light
       SENSOR_ID_UV            = 5,       // UV
       SENSOR_ID_CO            = 6,       // Carbon Monoxide
       SENSOR_ID_CO2           = 7,       // Carbon Dioxide
       SENSOR_ID_HCHO          = 8,       // Formaldehyde
       SENSOR_ID_VOC           = 9,       // Volatile Organic Compounds
       SENSOR_ID_NOX           = 10,      // Nitrogen Oxides
       SENSOR_ID_PM005         = 11,      // Particulate Matter 0.5
       SENSOR_ID_PM010         = 12,      // Particulate Matter 1.0
       SENSOR_ID_PM025         = 13,      // Particulate Matter 2.5
       SENSOR_ID_PM040         = 14,      // Particulate Matter 4.0
       SENSOR_ID_PM100         = 15,      // Particulate Matter 10.0
       SENSOR_ID_NOISE         = 16,      // Noise
       SENSOR_ID_VIBRATION     = 17,      // Vibration
       SENSOR_ID_STATE         = 18,      // State
       SENSOR_ID_BATTERY       = 19,      // Battery
       SENSOR_ID_SWITCH1       = 21,      // On/Off, Open/Close (same as ID_SWITCH)
       SENSOR_ID_SWITCH2       = 22,      // On/Off, Open/Close
       SENSOR_ID_SWITCH3       = 23,      // On/Off, Open/Close
       SENSOR_ID_SWITCH4       = 24,      // On/Off, Open/Close
       SENSOR_ID_SWITCH5       = 25,      // On/Off, Open/Close
       SENSOR_ID_SWITCH6       = 26,      // On/Off, Open/Close
       SENSOR_ID_SWITCH7       = 27,      // On/Off, Open/Close
       SENSOR_ID_SWITCH8       = 28,      // On/Off, Open/Close
       SENSOR_ID_SWITCH9       = 29,      // On/Off, Open/Close
       SENSOR_ID_PRESENCE1     = 51,      // Presence1
       SENSOR_ID_PRESENCE2     = 52,      // Presence2
       SENSOR_ID_PRESENCE3     = 53,      // Presence3
       SENSOR_ID_DISTANCE1     = 54,      // Distance1
       SENSOR_ID_DISTANCE2     = 55,      // Distance2
       SENSOR_ID_DISTANCE3     = 56,      // Distance3
       SENSOR_ID_POS1_X        = 57,      // X
       SENSOR_ID_POS1_Y        = 58,      // Y
       SENSOR_ID_POS1_Z        = 59,      // Z
       SENSOR_ID_POS2_X        = 60,      // X
       SENSOR_ID_POS2_Y        = 61,      // Y
       SENSOR_ID_POS2_Z        = 62,      // Z
       SENSOR_ID_POS3_X        = 63,      // X
       SENSOR_ID_POS3_Y        = 64,      // Y
       SENSOR_ID_POS3_Z        = 65,      // Z
       SENSOR_ID_RSSI          = 66,      // RSSI
       SENSOR_ID_PERF1         = 67,      // Performance Metric 1
       SENSOR_ID_PERF2         = 68,      // Performance Metric 2
       SENSOR_ID_PERF3         = 69,      // Performance Metric 3
       SENSOR_ID_VOLTAGE       = 70,      // Voltage
       SENSOR_ID_CURRENT       = 71,      // Current
       SENSOR_ID_POWER         = 72,      // Power
       SENSOR_ID_ENERGY        = 73,      // Energy
       SENSOR_ID_JSON          = 75,      // JSON Data
       SENSOR_ID_IMAGE         = 76,      // Image Data
       SENSOR_ID_GAS_M3        = 77,      // Gas Meter
       SENSOR_ID_WATER_M3      = 78,      // Water Meter
       SENSOR_ID_kWAh          = 79,      // Electric Meter (kilowatt-hours)
       SENSOR_ID_COUNT,                   // The maximum number of ID (highest index + 1)
   };

*/

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
	SENSOR_ID_SWITCH1     SensorType = 21 // On/Off, Open/Close (same as ID_SWITCH)
	SENSOR_ID_SWITCH2     SensorType = 22 // On/Off, Open/Close
	SENSOR_ID_SWITCH3     SensorType = 23 // On/Off, Open/Close
	SENSOR_ID_SWITCH4     SensorType = 24 // On/Off, Open/Close
	SENSOR_ID_SWITCH5     SensorType = 25 // On/Off, Open/Close
	SENSOR_ID_SWITCH6     SensorType = 26 // On/Off, Open/Close
	SENSOR_ID_SWITCH7     SensorType = 27 // On/Off, Open/Close
	SENSOR_ID_SWITCH8     SensorType = 28 // On/Off, Open/Close
	SENSOR_ID_SWITCH9     SensorType = 29 // On/Off, Open/Close
	SENSOR_ID_PRESENCE1   SensorType = 51 // Presence1
	SENSOR_ID_PRESENCE2   SensorType = 52 // Presence2
	SENSOR_ID_PRESENCE3   SensorType = 53 // Presence3
	SENSOR_ID_DISTANCE1   SensorType = 54 // Distance1
	SENSOR_ID_DISTANCE2   SensorType = 55 // Distance2
	SENSOR_ID_DISTANCE3   SensorType = 56 // Distance3
	SENSOR_ID_POS1_X      SensorType = 57 // X
	SENSOR_ID_POS1_Y      SensorType = 58 // Y
	SENSOR_ID_POS1_Z      SensorType = 59 // Z
	SENSOR_ID_POS2_X      SensorType = 60 // X
	SENSOR_ID_POS2_Y      SensorType = 61 // Y
	SENSOR_ID_POS2_Z      SensorType = 62 // Z
	SENSOR_ID_POS3_X      SensorType = 63 // X
	SENSOR_ID_POS3_Y      SensorType = 64 // Y
	SENSOR_ID_POS3_Z      SensorType = 65 // Z
	SENSOR_ID_RSSI        SensorType = 66 // RSSI
	SENSOR_ID_PERF1       SensorType = 67 // Performance Metric 1
	SENSOR_ID_PERF2       SensorType = 68 // Performance Metric 2
	SENSOR_ID_PERF3       SensorType = 69 // Performance Metric 3
	SENSOR_ID_VOLTAGE     SensorType = 70 // Voltage
	SENSOR_ID_CURRENT     SensorType = 71 // Current
	SENSOR_ID_POWER       SensorType = 72 // Power
	SENSOR_ID_ENERGY      SensorType = 73 // Energy
	SENSOR_ID_JSON        SensorType = 75 // JSON Data
	SENSOR_ID_IMAGE       SensorType = 76 // Image Data
	SENSOR_ID_GAS_M3      SensorType = 77 // Gas Meter
	SENSOR_ID_WATER_M3    SensorType = 78 // Water Meter
	SENSOR_ID_kWAh        SensorType = 79 // Electric Meter (kilowatt-hours)
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
	SENSOR_ID_SWITCH1:     {Min: 0, Max: 1},
	SENSOR_ID_SWITCH2:     {Min: 0, Max: 1},
	SENSOR_ID_SWITCH3:     {Min: 0, Max: 1},
	SENSOR_ID_SWITCH4:     {Min: 0, Max: 1},
	SENSOR_ID_SWITCH5:     {Min: 0, Max: 1},
	SENSOR_ID_SWITCH6:     {Min: 0, Max: 1},
	SENSOR_ID_SWITCH7:     {Min: 0, Max: 1},
	SENSOR_ID_SWITCH8:     {Min: 0, Max: 1},
	SENSOR_ID_SWITCH9:     {Min: 0, Max: 1},
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
	SENSOR_ID_SWITCH1:     "Switch 1",
	SENSOR_ID_SWITCH2:     "Switch 2",
	SENSOR_ID_SWITCH3:     "Switch 3",
	SENSOR_ID_SWITCH4:     "Switch 4",
	SENSOR_ID_SWITCH5:     "Switch 5",
	SENSOR_ID_SWITCH6:     "Switch 6",
	SENSOR_ID_SWITCH7:     "Switch 7",
	SENSOR_ID_SWITCH8:     "Switch 8",
	SENSOR_ID_SWITCH9:     "Switch 9",
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
	"Switch 1":                   SENSOR_ID_SWITCH1,
	"Switch 2":                   SENSOR_ID_SWITCH2,
	"Switch 3":                   SENSOR_ID_SWITCH3,
	"Switch 4":                   SENSOR_ID_SWITCH4,
	"Switch 5":                   SENSOR_ID_SWITCH5,
	"Switch 6":                   SENSOR_ID_SWITCH6,
	"Switch 7":                   SENSOR_ID_SWITCH7,
	"Switch 8":                   SENSOR_ID_SWITCH8,
	"Switch 9":                   SENSOR_ID_SWITCH9,
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
