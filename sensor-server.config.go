package sensor_server

import (
	corepkg "github.com/jurgen-kluft/go-core"
)

type SensorServerConfig struct {
	StoragePath    string
	TcpPort        int
	MaximumStores  int
	MaximumStreams int
	Groups         []*SensorGroupConfig
	StoreMap       map[string]int // map from Mac address to store Index
}

func newSensorServerConfig() *SensorServerConfig {
	return &SensorServerConfig{
		StoragePath:    "",
		TcpPort:        0,
		MaximumStores:  0,
		MaximumStreams: 0,
		Groups:         nil,
	}
}

func DecodeSensorServerConfig(decoder *corepkg.JsonDecoder) *SensorServerConfig {
	object := newSensorServerConfig()
	fields := map[string]corepkg.JsonDecode{
		"storage":         func(decoder *corepkg.JsonDecoder) { object.StoragePath = decoder.DecodeString() },
		"tcp_port":        func(decoder *corepkg.JsonDecoder) { object.TcpPort = int(decoder.DecodeInt32()) },
		"maximum_stores":  func(decoder *corepkg.JsonDecoder) { object.MaximumStores = int(decoder.DecodeInt32()) },
		"maximum_Streams": func(decoder *corepkg.JsonDecoder) { object.MaximumStreams = int(decoder.DecodeInt32()) },
		"groups": func(decoder *corepkg.JsonDecoder) {
			object.Groups = make([]*SensorGroupConfig, 0, 4)
			for !decoder.ReadUntilArrayEnd() {
				object.Groups = append(object.Groups, newSensorStoreConfig())
				decodeSensorStoreConfig(decoder, object.Groups[len(object.Groups)-1])
			}
		},
	}

	decoder.Decode(fields)

	// Create the map from Mac address to store Index
	object.StoreMap = make(map[string]int)
	for _, store := range object.Groups {
		object.StoreMap[store.Mac] = store.Index
	}

	return object
}

type SensorGroupConfig struct {
	Index       int
	Mac         string
	Name        string
	Sensors     []*SensorConfig
	NameToIndex map[string]int // map from sensor Name to ID
}

func newSensorStoreConfig() *SensorGroupConfig {
	return &SensorGroupConfig{
		Index:   0,
		Mac:     "",
		Name:    "",
		Sensors: make([]*SensorConfig, 0, 4),
	}
}

func decodeSensorStoreConfig(decoder *corepkg.JsonDecoder, object *SensorGroupConfig) {
	fields := map[string]corepkg.JsonDecode{
		"Index": func(decoder *corepkg.JsonDecoder) { object.Index = int(decoder.DecodeInt32()) },
		"Mac":   func(decoder *corepkg.JsonDecoder) { object.Mac = decoder.DecodeString() },
		"Name":  func(decoder *corepkg.JsonDecoder) { object.Name = decoder.DecodeString() },
		"Sensors": func(decoder *corepkg.JsonDecoder) {
			object.Sensors = make([]*SensorConfig, 0, 4)
			for !decoder.ReadUntilArrayEnd() {
				object.Sensors = append(object.Sensors, newSensorStreamConfig())
				decodeSensorStreamConfig(decoder, object.Sensors[len(object.Sensors)-1])
			}
		},
	}
	decoder.Decode(fields)

	// Create the map from sensor Name to ID
	object.NameToIndex = make(map[string]int)
	for _, stream := range object.Sensors {
		object.NameToIndex[stream.Type] = stream.Index
	}
}

type SensorConfig struct {
	Index int    //
	Type  string //
	Freq  int    // samples per hour
}

func newSensorStreamConfig() *SensorConfig {
	return &SensorConfig{
		Index: 0,
		Type:  "",
		Freq:  0,
	}
}

func decodeSensorStreamConfig(decoder *corepkg.JsonDecoder, object *SensorConfig) {
	fields := map[string]corepkg.JsonDecode{
		"Index": func(decoder *corepkg.JsonDecoder) { object.Index = int(decoder.DecodeInt32()) },
		"type":  func(decoder *corepkg.JsonDecoder) { object.Type = decoder.DecodeString() },
		"freq":  func(decoder *corepkg.JsonDecoder) { object.Freq = int(decoder.DecodeInt32()) },
	}
	decoder.Decode(fields)
}
