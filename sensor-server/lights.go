package sensorserver

type LightFeatures uint8

const (
	LIGHT_FEATURE_COLOR      LightFeatures = 1 << iota // Light supports color control
	LIGHT_FEATURE_WHITE                                // Light supports white control
	LIGHT_FEATURE_KELVIN                               // Light supports color temperature control
	LIGHT_FEATURE_BRIGHTNESS                           // Light supports brightness control
)

// A light is a collection of bulbs, for example the 2 bulbs above
// the kitchen table is one light. Each light can be switched on
// or off, can have brightness, color temperature and HSV.

type Light struct {
	Brightness       uint8         `json:"brightness"`        // Brightness of the light (0-100)
	ColorTemperature uint16        `json:"color_temperature"` // Color temperature of the light (in Kelvin)
	HSV              [3]uint8      `json:"hsv"`               // HSV values of the light (Hue, Saturation, Value)
	LightFeatures    LightFeatures `json:"light_features"`    // Features of the light (Color, White, Kelvin)
	LightMode        uint8         `json:"light_mode"`        // Mode of the light (Color, White, Kelvin)
}

type HouseLighting struct {
	Lights []Light `json:"lights"` // Map of lights in the house
}
