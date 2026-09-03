package sensorserver

type AreaType uint8

const (
	AreaUnknown  AreaType = iota
	AreaBasement          = 0x01
	Area1stFloor          = 0x10
	Area2ndFloor          = 0x20
	Area3rdFloor          = 0x30

	AreaFrontGarden = Area1stFloor + 0x01
	AreaBackGarden  = Area1stFloor + 0x02
	AreaCarPark     = Area1stFloor + 0x03

	AreaBasementFrontRoom = AreaBasement + 0x01
	AreaBasementBackRoom  = AreaBasement + 0x02
	AreaBasementStairs    = AreaBasement + 0x03

	AreaLivingRoom = Area1stFloor + 0x04
	AreaKitchen    = Area1stFloor + 0x05
	AreaBedroom    = Area1stFloor + 0x06
	AreaBathroom   = Area1stFloor + 0x07
	AreaOffice     = Area1stFloor + 0x08

	Area2ndLivingRoom = Area2ndFloor + 0x02
	Area2ndBedroom    = Area2ndFloor + 0x04
	Area2ndBathroom   = Area2ndFloor + 0x05
	Area2ndStudy      = Area2ndFloor + 0x06
	Area2ndWashRoom   = Area2ndFloor + 0x07
	Area2ndStairs     = Area2ndFloor + 0x08

	Area3rdLivingRoom = Area3rdFloor + 0x02
	Area3rdBedroom    = Area3rdFloor + 0x04
	Area3rdBathroom   = Area3rdFloor + 0x05
	Area3rdStairs     = Area3rdFloor + 0x08

	AreaAttic = Area3rdFloor + 0x01
)

var AreaNames = map[AreaType]string{
	AreaUnknown:           "Unknown",
	AreaBasement:          "Basement",
	Area1stFloor:          "",
	Area2ndFloor:          "2nd Floor",
	Area3rdFloor:          "3rd Floor",
	AreaFrontGarden:       "Front Garden",
	AreaBackGarden:        "Back Garden",
	AreaCarPark:           "Car Park",
	AreaBasementFrontRoom: "Front Room",
	AreaBasementBackRoom:  "Back Room",
	AreaBasementStairs:    "Stairs",
	AreaLivingRoom:        "Living Room",
	AreaKitchen:           "Kitchen",
	AreaBedroom:           "Bedroom",
	AreaBathroom:          "Bathroom",
	AreaOffice:            "Office",
	Area2ndLivingRoom:     "Living Room",
	Area2ndBedroom:        "Bedroom",
	Area2ndBathroom:       "Bathroom",
	Area2ndStudy:          "Study",
	Area2ndWashRoom:       "Wash Room",
	Area2ndStairs:         "Stairs",
	Area3rdLivingRoom:     "Living Room",
	Area3rdBedroom:        "Bedroom",
	Area3rdBathroom:       "Bathroom",
	Area3rdStairs:         "Stairs",
	AreaAttic:             "Attic",
}

var FloorAreaNames = []string{
	"Basement",
	"",
	"2nd Floor",
	"3rd Floor",
}

func (a AreaType) String() string {
	if name, ok := AreaNames[a]; ok {
		floor := (int(a) >> 4) & 0x0f
		if floor < len(FloorAreaNames) {
			if len(FloorAreaNames[floor]) == 0 {
				return name
			}
			return name + " (" + FloorAreaNames[floor] + ")"
		}
		return name
	}
	return "Unknown"
}

var AreaNamesToType = map[string]AreaType{
	"Unknown":         AreaUnknown,
	"Basement":        AreaBasement,
	"1st Floor":       Area1stFloor,
	"2nd Floor":       Area2ndFloor,
	"3rd Floor":       Area3rdFloor,
	"Front Garden":    AreaFrontGarden,
	"Back Garden":     AreaBackGarden,
	"Car Park":        AreaCarPark,
	"LivingRoom":      AreaLivingRoom,
	"1st Living Room": AreaLivingRoom,
	"1st Kitchen":     AreaKitchen,
	"1st Bedroom":     AreaBedroom,
	"1st Bathroom":    AreaBathroom,
	"1st Office":      AreaOffice,
	"2nd Living Room": Area2ndLivingRoom,
	"2nd Bedroom":     Area2ndBedroom,
	"2nd Bathroom":    Area2ndBathroom,
	"2nd Study":       Area2ndStudy,
	"2nd Wash Room":   Area2ndWashRoom,
	"2nd Stairs":      Area2ndStairs,
	"3rd Living Room": Area3rdLivingRoom,
	"3rd Bedroom":     Area3rdBedroom,
	"3rd Bathroom":    Area3rdBathroom,
	"3rd Stairs":      Area3rdStairs,
	"Attic":           AreaAttic,
}

func AreaTypeFromString(name string) AreaType {
	if areaType, ok := AreaNamesToType[name]; ok {
		return areaType
	}
	return AreaUnknown
}
