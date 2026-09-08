package sensorserver

// A house has floors, each floor has rooms, each room has windows, doors, sensors, lights and actuators.
// - floor; basement, first, second, third
// - room; living, dining, kitchen, bathroom, bedroom1, bedroom2, studyroom, master_bedroom, staircase, entrance
// - contacts; doors, windows, etc.
// - sensors; temperature, humidity, pressure, lux, motion, etc.
// - lights; ceiling lights, wall lights, etc.
// - actuator; switches, relays, etc.

type House struct {
	Global *HouseGlobal
	Rooms  []Room
}
