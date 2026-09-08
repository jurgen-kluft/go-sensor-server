package sensorserver

// A light is a collection of bulbs, for example the 2 bulbs above
// the kitchen table is one light. Each light can be switched on
// or off, can have brightness, color temperature and HSV.

type LightFeatures uint8

const (
	LIGHT_FEATURE_COLOR      LightFeatures = 1 << iota // Light supports color control
	LIGHT_FEATURE_WHITE                                // Light supports white control
	LIGHT_FEATURE_KELVIN                               // Light supports color temperature control
	LIGHT_FEATURE_BRIGHTNESS                           // Light supports brightness control
)

type LightMode uint8

const (
	LIGHT_MODE_COLOR  LightMode = iota // Light is in color mode
	LIGHT_MODE_WHITE                   // Light is in white mode
	LIGHT_MODE_KELVIN                  // Light is in color temperature mode
)

type LightID uint8

type LightSettings struct {
	Brightness       uint8         `json:"brightness"`        // Brightness of the light (0-100)
	HSV              [3]uint8      `json:"hsv"`               // HSV values of the light (Hue, Saturation, Value)
	ColorTemperature uint16        `json:"color_temperature"` // Color temperature of the light (in Kelvin)
	LightFeatures    LightFeatures `json:"features"`          // Features of the light (Color, White, Kelvin)
	LightMode        LightMode     `json:"mode"`              // Mode of the light (Color, White, Kelvin)
}

type LightKey struct {
	Floor   FloorType `json:"floor"` // Floor number of the light
	Room    RoomType  `json:"room"`  // Room number of the light
	LightID LightID   `json:"id"`    // ID of the light
}

type Light struct {
	Name  string        `json:"name"`  // Name of the light
	Key   LightKey      `json:"key"`   // Key of the light (Floor, Room, LightID)
	Light LightSettings `json:"light"` // Settings of the light
}

type HouseLighting struct {
	Lights []Light             `json:"lights"` // Map of lights in the house
	Router map[LightKey]*Light // Map of light keys to their corresponding Light struct
}
