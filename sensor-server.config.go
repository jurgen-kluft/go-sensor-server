package sensor_server

import (
	"fmt"
	"os"

	corepkg "github.com/jurgen-kluft/go-core"
)

type SensorServerConfig struct {
	StoragePath string
	TcpPort     int
	Devices     []*SensorGroupConfig
	DevicesMap  map[string]int // map from Mac address to store Index
}

func newSensorServerConfig() *SensorServerConfig {
	return &SensorServerConfig{
		StoragePath: "",
		TcpPort:     0,
		Devices:     nil,
		DevicesMap:  make(map[string]int),
	}
}

func LoadSensorServerConfig(filePath string) (*SensorServerConfig, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read all file content
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}

	fileSize := fileInfo.Size()
	if fileSize <= 0 || fileSize > 10*1024*1024 {
		return nil, fmt.Errorf("the size of configuration file is invalid: %v", fileSize)
	}

	buffer := make([]byte, fileSize)
	if _, err := file.Read(buffer); err != nil {
		return nil, err
	}

	decoder := corepkg.NewJsonDecoder()
	if decoder.Begin(string(buffer)) {
		return decodeSensorServerConfig(decoder), nil
	}

	return nil, fmt.Errorf("invalid configuration file format")
}

func decodeSensorServerConfig(decoder *corepkg.JsonDecoder) *SensorServerConfig {
	object := newSensorServerConfig()
	fields := map[string]corepkg.JsonDecode{
		"storage":  func(decoder *corepkg.JsonDecoder) { object.StoragePath = decoder.DecodeString() },
		"tcp_port": func(decoder *corepkg.JsonDecoder) { object.TcpPort = int(decoder.DecodeInt32()) },
		"devices": func(decoder *corepkg.JsonDecoder) {
			object.Devices = make([]*SensorGroupConfig, 0, 4)
			for !decoder.ReadUntilArrayEnd() {
				object.Devices = append(object.Devices, newSensorStoreConfig())
				decodeSensorStoreConfig(decoder, object.Devices[len(object.Devices)-1])
			}
		},
	}

	decoder.Decode(fields)

	// Create the map from Mac address to store Index
	object.DevicesMap = make(map[string]int)
	for i, store := range object.Devices {
		object.DevicesMap[store.Mac] = i
	}

	return object
}

type SensorGroupConfig struct {
	Mac  string
	Name string
}

func newSensorStoreConfig() *SensorGroupConfig {
	return &SensorGroupConfig{
		Mac:  "",
		Name: "",
	}
}

func decodeSensorStoreConfig(decoder *corepkg.JsonDecoder, object *SensorGroupConfig) {
	fields := map[string]corepkg.JsonDecode{
		"mac":  func(decoder *corepkg.JsonDecoder) { object.Mac = decoder.DecodeString() },
		"name": func(decoder *corepkg.JsonDecoder) { object.Name = decoder.DecodeString() },
	}
	decoder.Decode(fields)
}
