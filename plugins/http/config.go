package httpplugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"

	sensorserver "github.com/jurgen-kluft/go-sensor-server/sensor-server"
)

const DefaultHistoryCapacity = 1000

type Threshold struct {
	SensorType sensorserver.SensorType `json:"sensor_type"`
	Minimum    *int16                  `json:"minimum,omitempty"`
	Maximum    *int16                  `json:"maximum,omitempty"`
}

type Config struct {
	Enabled         bool        `json:"enabled"`
	Address         string      `json:"address"`
	HistoryCapacity int         `json:"history_capacity"`
	Thresholds      []Threshold `json:"thresholds"`
}

func LoadConfig(reader io.Reader) (Config, error) {
	if reader == nil {
		return Config{}, errors.New("load HTTP plugin config: nil reader")
	}
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	config := Config{}
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("decode HTTP plugin config: %w", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return Config{}, err
	}
	if config.Address == "" {
		config.Address = ":8080"
	}
	if config.HistoryCapacity == 0 {
		config.HistoryCapacity = DefaultHistoryCapacity
	}
	if err := config.Validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

func (config Config) Validate() error {
	if _, _, err := net.SplitHostPort(config.Address); err != nil {
		return fmt.Errorf("HTTP plugin address %q: %w", config.Address, err)
	}
	if config.HistoryCapacity < 1 || config.HistoryCapacity > 100000 {
		return errors.New("HTTP plugin history_capacity must be between 1 and 100000")
	}
	return nil
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("decode HTTP plugin config: multiple JSON values")
		}
		return fmt.Errorf("decode HTTP plugin config: %w", err)
	}
	return nil
}
