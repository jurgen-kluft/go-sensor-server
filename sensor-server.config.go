package sensor_server

import (
	"fmt"
	"os"

	corepkg "github.com/jurgen-kluft/go-core"
)

type SensorServerConfig struct {
	StoragePath          string                //
	TcpPort              int                   //
	UdpPort              int                   //
	FlushPeriodInSeconds int                   // how many seconds before we flush all sensor data again
	Sensors              []*SensorConfig       //
	SensorMap            map[int]*SensorConfig //
}

func newSensorServerConfig() *SensorServerConfig {
	return &SensorServerConfig{
		StoragePath:          "",
		TcpPort:              0,
		UdpPort:              0,
		FlushPeriodInSeconds: 15 * 60, // Every 15 minutes, flush to disk
		Sensors:              nil,
		SensorMap:            make(map[int]*SensorConfig),
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
		"udp_port": func(decoder *corepkg.JsonDecoder) { object.UdpPort = int(decoder.DecodeInt32()) },
		"flush":    func(decoder *corepkg.JsonDecoder) { object.FlushPeriodInSeconds = int(decoder.DecodeInt32()) },
		"sensors": func(decoder *corepkg.JsonDecoder) {
			object.Sensors = make([]*SensorConfig, 0, 4)
			for !decoder.ReadUntilArrayEnd() {
				sensor := NewSensorConfig(0, "", Unknown, TypeNone, 0)
				object.Sensors = append(object.Sensors, sensor)
				decodeSensorConfig(decoder, sensor)
			}
		},
	}
	decoder.Decode(fields)

	for _, sensor := range object.Sensors {
		if sensor.Index() >= 0 {
			object.SensorMap[sensor.Index()] = sensor
		}
	}

	return object
}

type SensorConfig struct {
	mIndex     int
	mName      string
	mType      SensorType
	mFieldType SensorFieldType
	mFrequency int32
}

func NewSensorConfig(index int, name string, sensorType SensorType, fieldType SensorFieldType, frequency int32) *SensorConfig {
	return &SensorConfig{
		mIndex:     index,
		mName:      name,
		mType:      sensorType,
		mFieldType: fieldType,
		mFrequency: frequency,
	}
}

func decodeSensorConfig(decoder *corepkg.JsonDecoder, object *SensorConfig) {
	fields := map[string]corepkg.JsonDecode{
		"index": func(decoder *corepkg.JsonDecoder) { object.mIndex = int(decoder.DecodeInt32()) },
		"name":  func(decoder *corepkg.JsonDecoder) { object.mName = decoder.DecodeString() },
		"type": func(decoder *corepkg.JsonDecoder) {
			sensorTypeName := decoder.DecodeString()
			if st, ok := SensorNameToSensorTypeMap[sensorTypeName]; ok {
				object.mType = st
			} else {
				object.mType = Unknown
			}
		},
		"field_type": func(decoder *corepkg.JsonDecoder) {
			fieldTypeName := decoder.DecodeString()
			if ft, ok := SensorFieldNameToSensorFieldTypeMap[fieldTypeName]; ok {
				object.mFieldType = ft
			} else {
				object.mFieldType = TypeNone
			}
		},
		"frequency": func(decoder *corepkg.JsonDecoder) { object.mFrequency = decoder.DecodeInt32() },
	}
	decoder.Decode(fields)
}

func (t SensorConfig) Name() string {
	return t.mName
}

func (t SensorConfig) FullIdentifier() uint64 {
	return (uint64(t.mFieldType)<<8)&0xFF | uint64(t.mType)&0xFF | ((uint64(t.mFrequency) << 32) & 0xFFFFFFFF00000000)
}

func (t SensorConfig) SizeInBits() int {
	return int(t.mFieldType) & 0x7F
}

func (t SensorConfig) Index() int {
	return int(t.mType)
}

func (t SensorConfig) IsValid() bool {
	return t.mType != Unknown
}

func (t SensorConfig) IsBattery() bool {
	return t.mType == Battery
}

func (t SensorConfig) String() string {
	return t.mName
}

func (t SensorConfig) SampleFrequency() int32 {
	return t.mFrequency
}

func (t SensorConfig) SamplePeriodInMs() int32 {
	return 60 * 60 * 1000 / t.mFrequency
}

func (t SensorConfig) Type() SensorType {
	return t.mType
}

func (t SensorConfig) FieldType() SensorFieldType {
	return t.mFieldType
}
