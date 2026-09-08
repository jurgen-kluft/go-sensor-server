package sensorserver

type UnitType uint8

const (
	// Temperature (0–2)
	UTemperature UnitType = 0
	UCelcius     UnitType = 1
	UFahrenheit  UnitType = 2
	UKelvin      UnitType = 3

	// Pressure (6–9)
	UPressure   UnitType = 6
	UBar        UnitType = 7
	UPascal     UnitType = 8
	UAtmosphere UnitType = 9

	// Humidity (13)
	UHumidity UnitType = 13

	// Speed & Acceleration (17–21)
	UVelocity          UnitType = 17
	UKilometersPerHour UnitType = 18
	UMilesPerHour      UnitType = 19
	UAcceleration      UnitType = 20
	UGForce            UnitType = 21

	// Concentration (25–26)
	UPpm     UnitType = 25
	UPpb     UnitType = 26
	UPercent UnitType = 27

	// Mass (30–36)
	UKilograms  UnitType = 30
	UGrams      UnitType = 31
	UMilligrams UnitType = 32
	UMicrograms UnitType = 33
	UTons       UnitType = 34
	UPounds     UnitType = 35
	UOunces     UnitType = 36

	// Length & Distance (40–46)
	UKilometers  UnitType = 40
	UMeters      UnitType = 41
	UCentimeters UnitType = 42
	UMillimeters UnitType = 43
	UMiles       UnitType = 44
	UFeet        UnitType = 45
	UInches      UnitType = 46

	// Time (50–55)
	UHours        UnitType = 50
	UMinutes      UnitType = 51
	USeconds      UnitType = 52
	UMilliseconds UnitType = 53
	UMicroseconds UnitType = 54
	UNanoseconds  UnitType = 55

	// Volume (59–62)
	ULiters      UnitType = 59
	UMilliliters UnitType = 60
	UCubicMeters UnitType = 61
	UGallons     UnitType = 62

	// Energy & Power (66–71)
	UJoules       UnitType = 66
	UKiloJoules   UnitType = 67
	UWatts        UnitType = 68
	UKiloWatts    UnitType = 69
	UMegaWatts    UnitType = 70
	UKiloWattHour UnitType = 71

	// Data Size (75–79)
	UBytes     UnitType = 75
	UKiloBytes UnitType = 76
	UMegaBytes UnitType = 77
	UGigaBytes UnitType = 78
	UTeraBytes UnitType = 79

	// Electrical (83–89)
	UVolt        UnitType = 83
	UMilliVolt   UnitType = 84
	UAmpere      UnitType = 85
	UMilliAmpere UnitType = 86
	UOhm         UnitType = 87
	UFarad       UnitType = 88
	UHenry       UnitType = 89
	UdBm         UnitType = 90 // Decibel-milliwatts (signal strength)

	// Frequency (93–96)
	UHertz     UnitType = 93
	UKiloHertz UnitType = 94
	UMegaHertz UnitType = 95
	UGigaHertz UnitType = 96

	// Angle (100–101)
	UDegrees UnitType = 100
	URadians UnitType = 101

	// Logical / State Units (105–115)
	UOnOff               UnitType = 105
	UOpenClose           UnitType = 106
	UTrueFalse           UnitType = 107
	UActiveInactive      UnitType = 108
	UEnabledDisabled     UnitType = 109
	UStartStop           UnitType = 110
	UAlarmNormal         UnitType = 111
	UFaultNormal         UnitType = 112
	UPresentAbsent       UnitType = 113
	UDetectedNotDetected UnitType = 114

	// Sound
	UDecibels UnitType = 116

	// Light
	ULux     UnitType = 120
	UUvIndex UnitType = 121

	// mg/m3, ug/m3
	Uugm3 UnitType = 125
	Umgm3 UnitType = 126

	UUnknown UnitType = 255
)

var SensorTypeUnits = map[SensorType]UnitType{
	SENSOR_ID_UNKNOWN:     UUnknown,
	SENSOR_ID_TEMPERATURE: UTemperature,
	SENSOR_ID_HUMIDITY:    UHumidity,
	SENSOR_ID_PRESSURE:    UPressure,
	SENSOR_ID_LIGHT:       ULux,
	SENSOR_ID_UV:          UUvIndex,
	SENSOR_ID_CO:          UPpm,
	SENSOR_ID_CO2:         UPpm,
	SENSOR_ID_HCHO:        UPpm,
	SENSOR_ID_VOC:         UPpm,
	SENSOR_ID_NOX:         UPpm,
	SENSOR_ID_PM005:       Uugm3,
	SENSOR_ID_PM010:       Uugm3,
	SENSOR_ID_PM025:       Uugm3,
	SENSOR_ID_PM040:       Uugm3,
	SENSOR_ID_PM100:       Uugm3,
	SENSOR_ID_SOUND:       UDecibels,
	SENSOR_ID_BINARY:      UOnOff,
	SENSOR_ID_AMPLITUDE:   UPercent,
	SENSOR_ID_DURATION:    UMilliseconds,
	SENSOR_ID_FREQUENCY:   UHertz,
	SENSOR_ID_BATTERY:     UPercent,
	SENSOR_ID_VOLTAGE:     UVolt,
	SENSOR_ID_CURRENT:     UAmpere,
	SENSOR_ID_POWER:       UKiloWatts,
	SENSOR_ID_ENERGY:      UKiloWattHour,
	SENSOR_ID_GAS_M3:      UCubicMeters,
	SENSOR_ID_WATER_M3:    UCubicMeters,
	SENSOR_ID_kWAh:        UKiloWattHour,
}

func UnitForSensorType(sensorType SensorType) UnitType {
	if unit, exists := SensorTypeUnits[sensorType]; exists {
		return unit
	}
	return UUnknown
}

func (u UnitType) String() string {
	if name, exists := UnitTypeToString[u]; exists {
		return name
	}
	return "Unknown"
}

func (u UnitType) Symbol() string {
	if symbol, exists := UnitTypeToSymbolString[u]; exists {
		return symbol
	}
	return "?"
}

var UnitTypeToSymbolString = map[UnitType]string{
	// Temperature (0–2)
	UTemperature: "°C",
	UCelcius:     "°C",
	UFahrenheit:  "°F",
	UKelvin:      "K",

	// Pressure (6–9)
	UPressure:   "Pa",
	UBar:        "bar",
	UPascal:     "Pa",
	UAtmosphere: "atm",

	// Humidity (13)
	UHumidity: "%",

	// Speed & Acceleration (17–21)
	UVelocity:          "m/s",
	UKilometersPerHour: "km/h",
	UMilesPerHour:      "mph",
	UAcceleration:      "m/s²",
	UGForce:            "g",

	// Concentration (25–26)
	UPpm:     "ppm",
	UPpb:     "ppb",
	UPercent: "%",

	// Mass (30–36)
	UKilograms:  "kg",
	UGrams:      "g",
	UMilligrams: "mg",
	UMicrograms: "µg",
	UTons:       "t",
	UPounds:     "lb",
	UOunces:     "oz",

	// Length & Distance (40–46)
	UKilometers:  "km",
	UMeters:      "m",
	UCentimeters: "cm",
	UMillimeters: "mm",
	UMiles:       "mi",
	UFeet:        "ft",
	UInches:      "in",

	// Time (50–55)
	UHours:        "h",
	UMinutes:      "min",
	USeconds:      "s",
	UMilliseconds: "ms",
	UMicroseconds: "µs",
	UNanoseconds:  "ns",

	// Volume (59–62)
	ULiters:      "L",
	UMilliliters: "mL",
	UCubicMeters: "m³",
	UGallons:     "gal",

	// Energy & Power (66–71)
	UJoules:       "J",
	UKiloJoules:   "kJ",
	UWatts:        "W",
	UKiloWatts:    "kW",
	UMegaWatts:    "MW",
	UKiloWattHour: "kWh",

	// Data Size (75–79)
	UBytes:     "B",
	UKiloBytes: "kB",
	UMegaBytes: "MB",
	UGigaBytes: "GB",
	UTeraBytes: "TB",

	// Electrical (83–89)
	UVolt:        "V",
	UMilliVolt:   "mV",
	UAmpere:      "A",
	UMilliAmpere: "mA",
	UOhm:         "Ω",
	UFarad:       "F",
	UHenry:       "H",
	UdBm:         "dBm",

	// Frequency (93–96)
	UHertz:     "Hz",
	UKiloHertz: "kHz",
	UMegaHertz: "MHz",
	UGigaHertz: "GHz",

	// Angle (100–101)
	UDegrees: "°",
	URadians: "rad",

	// Logical / State Units (105–115)
	UOnOff:               "On/Off",
	UOpenClose:           "Open/Close",
	UTrueFalse:           "True/False",
	UActiveInactive:      "Active/Inactive",
	UEnabledDisabled:     "Enabled/Disabled",
	UStartStop:           "Start/Stop",
	UAlarmNormal:         "Alarm/Normal",
	UFaultNormal:         "Fault/Normal",
	UPresentAbsent:       "Present/Absent",
	UDetectedNotDetected: "Detected/Not Detected",

	// Sound
	UDecibels: "dB",

	// Light
	ULux:     "lx",
	UUvIndex: "UV Index",

	// mg/m3, ug/m3
	Uugm3: "µg/m³",
	Umgm3: "mg/m³",

	UUnknown: "?",
}

var UnitTypeToString = map[UnitType]string{
	UTemperature:         "Temperature",
	UCelcius:             "Celsius",
	UFahrenheit:          "Fahrenheit",
	UKelvin:              "Kelvin",
	UPressure:            "Pressure",
	UBar:                 "Bar",
	UPascal:              "Pascal",
	UAtmosphere:          "Atmosphere",
	UHumidity:            "Humidity",
	UVelocity:            "Velocity",
	UKilometersPerHour:   "Kilometers per Hour",
	UMilesPerHour:        "Miles per Hour",
	UAcceleration:        "Acceleration",
	UGForce:              "G-Force",
	UPpm:                 "Parts per Million (PPM)",
	UPpb:                 "Parts per Billion (PPB)",
	UPercent:             "Percent (%)",
	UKilograms:           "Kilograms (kg)",
	UGrams:               "Grams (g)",
	UMilligrams:          "Milligrams (mg)",
	UMicrograms:          "Micrograms (µg)",
	UTons:                "Tons (t)",
	UPounds:              "Pounds (lb)",
	UOunces:              "Ounces (oz)",
	UKilometers:          "Kilometers (km)",
	UMeters:              "Meters (m)",
	UCentimeters:         "Centimeters (cm)",
	UMillimeters:         "Millimeters (mm)",
	UMiles:               "Miles (mi)",
	UFeet:                "Feet (ft)",
	UInches:              "Inches (in)",
	UHours:               "Hours (h)",
	UMinutes:             "Minutes (min)",
	USeconds:             "Seconds (s)",
	UMilliseconds:        "Milliseconds (ms)",
	UMicroseconds:        "Microseconds (µs)",
	UNanoseconds:         "Nanoseconds (ns)",
	ULiters:              "Liters (L)",
	UMilliliters:         "Milliliters (mL)",
	UCubicMeters:         "Cubic Meters (m³)",
	UGallons:             "Gallons (gal)",
	UJoules:              "Joules (J)",
	UKiloJoules:          "Kilojoules (kJ)",
	UWatts:               "Watts (W)",
	UKiloWatts:           "Kilowatts (kW)",
	UMegaWatts:           "Megawatts (MW)",
	UKiloWattHour:        "Kilowatt-hour (kWh)",
	UBytes:               "Bytes (B)",
	UKiloBytes:           "Kilobytes (KB)",
	UMegaBytes:           "Megabytes (MB)",
	UGigaBytes:           "Gigabytes (GB)",
	UTeraBytes:           "Terabytes (TB)",
	UVolt:                "Volts (V)",
	UMilliVolt:           "Millivolts (mV)",
	UAmpere:              "Amperes (A)",
	UMilliAmpere:         "Milliamperes (mA)",
	UOhm:                 "Ohms (Ω)",
	UFarad:               "Farads (F)",
	UHenry:               "Henrys (H)",
	UdBm:                 "Decibel-milliwatts (dBm)",
	UHertz:               "Hertz (Hz)",
	UKiloHertz:           "Kilohertz (kHz)",
	UMegaHertz:           "Megahertz (MHz)",
	UGigaHertz:           "Gigahertz (GHz)",
	UDegrees:             "Degrees (°)",
	URadians:             "Radians (rad)",
	UOnOff:               "On/Off",
	UOpenClose:           "Open/Close",
	UTrueFalse:           "True/False",
	UActiveInactive:      "Active/Inactive",
	UEnabledDisabled:     "Enabled/Disabled",
	UStartStop:           "Start/Stop",
	UAlarmNormal:         "Alarm/Normal",
	UFaultNormal:         "Fault/Normal",
	UPresentAbsent:       "Present/Absent",
	UDetectedNotDetected: "Detected/Not Detected",
	UDecibels:            "Decibels (dB)",
	ULux:                 "Lux (lx)",
	UUvIndex:             "UV Index",
	Uugm3:                "Micrograms per Cubic Meter (µg/m³)",
	Umgm3:                "Milligrams per Cubic Meter (mg/m³)",
}
