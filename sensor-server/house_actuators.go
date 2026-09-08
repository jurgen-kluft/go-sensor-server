package sensorserver

import "fmt"

// An Actuator is a switch, for example a wall panel can have 1, 2 or 3 actuators.
// Each actuator can be switched on or off, this should not be confused with lights.
// Lights have their own way of being controlled (see lights.go).

// An actuator is addressable as device (floor -> room -> panel) = actuators
// we configure it in JSON as follows:
// {
//     "02:3a:4b:5c:6d:7e": {
//         "floor": 1,
//         "room": 2,
//         "panel": 1,
//         "actuators": [
//             0,
//             1,
//             2
//         ]
//     }
// }

type ActuatorType uint8

const (
	ACTUATOR_ID_UNKNOWN ActuatorType = 0  // Unknown
	ACTUATOR_ID_1       ActuatorType = 1  // Actuator 1
	ACTUATOR_ID_2       ActuatorType = 2  // Actuator 2
	ACTUATOR_ID_3       ActuatorType = 3  // Actuator 3
	ACTUATOR_ID_4       ActuatorType = 4  // Actuator 4
	ACTUATOR_ID_5       ActuatorType = 5  // Actuator 5
	ACTUATOR_ID_6       ActuatorType = 6  // Actuator 6
	ACTUATOR_ID_7       ActuatorType = 7  // Actuator 7
	ACTUATOR_ID_8       ActuatorType = 8  // Actuator 8
	ACTUATOR_ID_9       ActuatorType = 9  // Actuator 9
	ACTUATOR_ID_10      ActuatorType = 10 // Actuator 10
	ACTUATOR_ID_MAX     ActuatorType = 11 // Max
)

func ToActuatorType(id uint8) ActuatorType {
	if id >= uint8(ACTUATOR_ID_MAX) {
		return ACTUATOR_ID_UNKNOWN
	}
	return ActuatorType(id)
}

var ActuatorTypeNames = map[ActuatorType]string{
	ACTUATOR_ID_UNKNOWN: "Unknown",
	ACTUATOR_ID_1:       "Actuator1",
	ACTUATOR_ID_2:       "Actuator2",
	ACTUATOR_ID_3:       "Actuator3",
	ACTUATOR_ID_4:       "Actuator4",
	ACTUATOR_ID_5:       "Actuator5",
	ACTUATOR_ID_6:       "Actuator6",
	ACTUATOR_ID_7:       "Actuator7",
	ACTUATOR_ID_8:       "Actuator8",
	ACTUATOR_ID_9:       "Actuator9",
	ACTUATOR_ID_10:      "Actuator10",
}

func (actuatorType ActuatorType) String() string {
	name, exists := ActuatorTypeNames[actuatorType]
	if !exists {
		return fmt.Sprintf("Unknown(%d)", actuatorType)
	}
	return name
}

var ActuatorNameToActuatorType = map[string]ActuatorType{
	"Unknown":    ACTUATOR_ID_UNKNOWN,
	"Actuator1":  ACTUATOR_ID_1,
	"Actuator2":  ACTUATOR_ID_2,
	"Actuator3":  ACTUATOR_ID_3,
	"Actuator4":  ACTUATOR_ID_4,
	"Actuator5":  ACTUATOR_ID_5,
	"Actuator6":  ACTUATOR_ID_6,
	"Actuator7":  ACTUATOR_ID_7,
	"Actuator8":  ACTUATOR_ID_8,
	"Actuator9":  ACTUATOR_ID_9,
	"Actuator10": ACTUATOR_ID_10,
}
