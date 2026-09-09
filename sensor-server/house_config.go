package sensorserver

// House Configuration is JSON format and can be easily loaded.
// After loading, the following things have to be done:
//
// Presence Mapping:
// floor:room => index
//
// Contact Mapping:
// floor:room:contact => index
//
// Actuator Mapping:
// floor:room:actuator => index
//
// Room Sensors:
// floor:room:sensor => offset (based on value type that is associated with the sensor type)
//
// Collecting all:
// - Sprite Pack JSON files
// - Font Pack JSON files
// - Script COVA files
//
//
