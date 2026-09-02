package sensorserver

type AreaType uint16

const (
	AreaUnknown  AreaType = iota
	AreaBasement          = 0x0001
	Area1stFloor          = 0x0100
	Area2ndFloor          = 0x0200
	Area3rdFloor          = 0x0300

	AreaFrontGarden = Area1stFloor + 0x0001
	AreaBackGarden  = Area1stFloor + 0x0002
	AreaCarPark     = Area1stFloor + 0x0003

	Area1stLivingRoom = Area1stFloor + 0x0004
	Area1stKitchen    = Area1stFloor + 0x0005
	Area1stBedroom    = Area1stFloor + 0x0006
	Area1stBathroom   = Area1stFloor + 0x0007
	Area1stOffice     = Area1stFloor + 0x0008

	Area2ndLivingRoom = Area2ndFloor + 0x0002
	Area2ndBedroom    = Area2ndFloor + 0x0004
	Area2ndBathroom   = Area2ndFloor + 0x0005
	Area2ndStudy      = Area2ndFloor + 0x0006
	Area2ndWashRoom   = Area2ndFloor + 0x0007
	Area2ndStairs     = Area2ndFloor + 0x0008

	Area3rdLivingRoom = Area3rdFloor + 0x0002
	Area3rdBedroom    = Area3rdFloor + 0x0004
	Area3rdBathroom   = Area3rdFloor + 0x0005
	Area3rdStairs     = Area3rdFloor + 0x0008

	AreaAttic = Area3rdFloor + 0x0001
)

var AreaNames = map[AreaType]string{
	AreaUnknown:       "Unknown",
	AreaBasement:      "Basement",
	Area1stFloor:      "1st Floor",
	Area2ndFloor:      "2nd Floor",
	Area3rdFloor:      "3rd Floor",
	AreaFrontGarden:   "Front Garden",
	AreaBackGarden:    "Back Garden",
	AreaCarPark:       "Car Park",
	Area1stLivingRoom: "1st Living Room",
	Area1stKitchen:    "1st Kitchen",
	Area1stBedroom:    "1st Bedroom",
	Area1stBathroom:   "1st Bathroom",
	Area1stOffice:     "1st Office",
	Area2ndLivingRoom: "2nd Living Room",
	Area2ndBedroom:    "2nd Bedroom",
	Area2ndBathroom:   "2nd Bathroom",
	Area2ndStudy:      "2nd Study",
	Area2ndWashRoom:   "2nd Wash Room",
	Area2ndStairs:     "2nd Stairs",
	Area3rdLivingRoom: "3rd Living Room",
	Area3rdBedroom:    "3rd Bedroom",
	Area3rdBathroom:   "3rd Bathroom",
	Area3rdStairs:     "3rd Stairs",
	AreaAttic:         "Attic",
}

func (a AreaType) String() string {
	if name, ok := AreaNames[a]; ok {
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
	"1st Living Room": Area1stLivingRoom,
	"1st Kitchen":     Area1stKitchen,
	"1st Bedroom":     Area1stBedroom,
	"1st Bathroom":    Area1stBathroom,
	"1st Office":      Area1stOffice,
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
