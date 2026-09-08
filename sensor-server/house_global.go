package sensorserver

type HouseGlobal struct {
	Year                  uint16                 // Year (e.g., 2026)
	Month                 uint8                  // Month (1-12)
	Day                   uint8                  // Day of the month (1-31)
	DayOfWeek             uint8                  // Day of the week (0-6, where 0 = Sunday, 1 = Monday, ..., 6 = Saturday)
	Hour                  uint8                  // Hour of the day (0-23)
	Minute                uint8                  // Minute of the hour (0-59)
	Second                uint8                  // Second of the minute (0-59)
	OutsideHumidity       uint8                  // Percentage (0-100%)
	OutsideTemperature    int8                   // Celsius
	OutsidePressure       uint16                 // hPa
	OutsideLux            uint16                 // Lux (Illuminance)
	OutsideSunPercentage  uint8                  // 0-100%
	OutsideCloudCover     uint8                  // 0-100%
	OutsideWindSpeed      uint8                  // m/s
	OutsideWindDirection  uint8                  // 0-7 for N, NE, E, SE, S, SW, W, NW
	OutsideRainPercentage uint8                  // 0-100%
	OutsideSnowPercentage uint8                  // 0-100%
	ObjectsActive         uint32                 // a bit per object, 1 = active, 0 = inactive
	PresenceState         [FLOOR_COUNT]uint16    // m_presence_state[floor_id] |= (1 << room_id) if presence detected in that room
	ContactState          [CONTACT_COUNT]Contact // m_contact_state[CONTACT_COUNT]
}
