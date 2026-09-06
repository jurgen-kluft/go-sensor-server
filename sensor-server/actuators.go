package sensorserver

import "fmt"

type ActuatorType uint8

const (
	ACTUATOR_ID_UNKNOWN    ActuatorType = 0 // Unknown
	ACTUATOR_ID_SWITCH     ActuatorType = 1 // Switch
	ACTUATOR_ID_COLOR      ActuatorType = 1 // Color
	ACTUATOR_ID_BRIGHTNESS ActuatorType = 1 // Brightness
	A
	ACTUATOR_ID_MAX ActuatorType = 2 // Max
)

func ToActuatorType(id uint16) ActuatorType {
	if id >= uint16(ACTUATOR_ID_MAX) {
		return ACTUATOR_ID_UNKNOWN
	}
	return ActuatorType(id)
}

type ActuatorTypeValueRange struct {
	Min int32 // int24
	Max int32 // int24
}

var ActuatorTypeValueRanges = map[ActuatorType]ActuatorTypeValueRange{
	ACTUATOR_ID_UNKNOWN: {Min: -32768, Max: 32767},
}

var ActuatorTypeNames = map[ActuatorType]string{
	ACTUATOR_ID_UNKNOWN: "Unknown",
}

func (actuatorType ActuatorType) String() string {
	name, exists := ActuatorTypeNames[actuatorType]
	if !exists {
		return fmt.Sprintf("Unknown(%d)", actuatorType)
	}
	return name
}

var ActuatorNameToActuatorType = map[string]ActuatorType{
	"Unknown": ACTUATOR_ID_UNKNOWN,
}
