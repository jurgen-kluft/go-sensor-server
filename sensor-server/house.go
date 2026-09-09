package sensorserver

// A house has floors, each floor has rooms, each room has windows, doors, sensors, lights and actuators.
// - floors; basement, first, second, third
// - rooms; living, dining, kitchen, bathroom, bedroom1, bedroom2, studyroom, master_bedroom, staircase, entrance
// - contacts; doors, windows, etc. (mailbox?)
// - sensors; temperature, humidity, pressure, lux, motion, etc (see house_sensors.go)
// - actuators; switches, relays, etc.
// - displays; wall panel with display, environmental sensor unit with display
// - lights; ceiling lights, wall lights, etc.

type House struct {
	Global *HouseGlobal
	Rooms  []Room
}
