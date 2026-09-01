package sensorserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	DefaultQueueCapacity         = 512
	DefaultMaximumTCPConnections = 100
	DefaultWriteBufferSize       = 64 * 1024
	DefaultRotationSize          = 64 * 1024 * 1024
)

var (
	ErrInvalidConfiguration = errors.New("invalid configuration")
	ErrRestartRequired      = errors.New("configuration change requires restart")
)

type Duration time.Duration

func (duration *Duration) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("duration must be a string: %w", err)
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("parse duration %q: %w", value, err)
	}
	*duration = Duration(parsed)
	return nil
}

func (duration Duration) Value() time.Duration {
	return time.Duration(duration)
}

type DeviceConfig struct {
	MAC  string `json:"mac"`
	Area string `json:"area"`
}

type SensorConfig struct {
	ID   uint16 `json:"id"`
	Type string `json:"type"`
	Unit string `json:"unit"`
}

type DataStreamConfig struct {
	QueueCapacity   int      `json:"queue_capacity"`
	EnqueueWait     Duration `json:"enqueue_wait"`
	RetryCount      int      `json:"retry_count"`
	InitialBackoff  Duration `json:"initial_backoff"`
	MaximumBackoff  Duration `json:"maximum_backoff"`
	WriteBufferSize int      `json:"write_buffer_size"`
	FlushInterval   Duration `json:"flush_interval"`
	RotationSize    int64    `json:"rotation_size"`
}

type NetworkConfig struct {
	MaximumTCPConnections int      `json:"maximum_tcp_connections"`
	PayloadDeadline       Duration `json:"payload_deadline"`
}

type LoggingConfig struct {
	Level  string `json:"level"`
	Output string `json:"output"`
}

type Config struct {
	TCPAddress       string           `json:"tcp_address"`
	UDPAddress       string           `json:"udp_address"`
	DataRoot         string           `json:"data_root"`
	QuarantineRoot   string           `json:"quarantine_root"`
	Devices          []DeviceConfig   `json:"devices"`
	Sensors          []SensorConfig   `json:"sensors"`
	DataStream       DataStreamConfig `json:"data_stream"`
	Network          NetworkConfig    `json:"network"`
	Logging          LoggingConfig    `json:"logging"`
	ShutdownDeadline Duration         `json:"shutdown_deadline"`
}

type Device struct {
	MAC  MACAddress
	Area string
}

type SensorDefinition struct {
	ID   uint16
	Type string
	Unit string
}

type ConfigSnapshot struct {
	config  Config
	devices map[MACAddress]Device
	sensors map[uint16]SensorDefinition
}

func LoadConfig(reader io.Reader) (*ConfigSnapshot, error) {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()

	config := Config{}
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("decode configuration: %w", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return nil, err
	}
	applyConfigDefaults(&config)
	return NewConfigSnapshot(config)
}

func NewConfigSnapshot(config Config) (*ConfigSnapshot, error) {
	applyConfigDefaults(&config)
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	snapshot := &ConfigSnapshot{
		config:  cloneConfig(config),
		devices: make(map[MACAddress]Device, len(config.Devices)),
		sensors: make(map[uint16]SensorDefinition, len(config.Sensors)),
	}
	for _, configured := range config.Devices {
		mac, _ := ParseMACAddress(configured.MAC)
		snapshot.devices[mac] = Device{MAC: mac, Area: configured.Area}
	}
	for _, configured := range config.Sensors {
		snapshot.sensors[configured.ID] = SensorDefinition(configured)
	}
	return snapshot, nil
}

func (snapshot *ConfigSnapshot) Config() Config {
	return cloneConfig(snapshot.config)
}

func (snapshot *ConfigSnapshot) Device(mac MACAddress) (Device, bool) {
	device, ok := snapshot.devices[mac]
	return device, ok
}

func (snapshot *ConfigSnapshot) Sensor(id uint16) (SensorDefinition, bool) {
	sensor, ok := snapshot.sensors[id]
	return sensor, ok
}

// ValidateReload permits only the device registry to change at runtime.
func (snapshot *ConfigSnapshot) ValidateReload(candidate *ConfigSnapshot) error {
	if candidate == nil {
		return fmt.Errorf("reload configuration: nil snapshot: %w", ErrInvalidConfiguration)
	}
	current := snapshot.config
	next := candidate.config
	current.Devices = nil
	next.Devices = nil
	if !configsEqual(current, next) {
		return ErrRestartRequired
	}
	return nil
}

func ParseMACAddress(value string) (MACAddress, error) {
	var mac MACAddress
	parts := strings.Split(value, ":")
	if len(parts) != len(mac) {
		return mac, fmt.Errorf("MAC address %q: expected six colon-separated bytes", value)
	}
	for index, part := range parts {
		if len(part) != 2 {
			return MACAddress{}, fmt.Errorf("MAC address %q: byte %d must have two hexadecimal digits", value, index)
		}
		var parsed uint8
		if _, err := fmt.Sscanf(part, "%02x", &parsed); err != nil {
			return MACAddress{}, fmt.Errorf("MAC address %q: byte %d: %w", value, index, err)
		}
		mac[index] = parsed
	}
	return mac, nil
}

func applyConfigDefaults(config *Config) {
	if config.DataStream.QueueCapacity == 0 {
		config.DataStream.QueueCapacity = DefaultQueueCapacity
	}
	if config.DataStream.EnqueueWait == 0 {
		config.DataStream.EnqueueWait = Duration(100 * time.Millisecond)
	}
	if config.DataStream.RetryCount == 0 {
		config.DataStream.RetryCount = 5
	}
	if config.DataStream.InitialBackoff == 0 {
		config.DataStream.InitialBackoff = Duration(50 * time.Millisecond)
	}
	if config.DataStream.MaximumBackoff == 0 {
		config.DataStream.MaximumBackoff = Duration(time.Second)
	}
	if config.DataStream.WriteBufferSize == 0 {
		config.DataStream.WriteBufferSize = DefaultWriteBufferSize
	}
	if config.DataStream.FlushInterval == 0 {
		config.DataStream.FlushInterval = Duration(time.Second)
	}
	if config.DataStream.RotationSize == 0 {
		config.DataStream.RotationSize = DefaultRotationSize
	}
	if config.Network.MaximumTCPConnections == 0 {
		config.Network.MaximumTCPConnections = DefaultMaximumTCPConnections
	}
	if config.Network.PayloadDeadline == 0 {
		config.Network.PayloadDeadline = Duration(5 * time.Second)
	}
	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}
	if config.Logging.Output == "" {
		config.Logging.Output = "stderr"
	}
	if config.ShutdownDeadline == 0 {
		config.ShutdownDeadline = Duration(30 * time.Second)
	}
}

func validateConfig(config Config) error {
	if err := validateListenAddress("tcp_address", config.TCPAddress); err != nil {
		return err
	}
	if err := validateListenAddress("udp_address", config.UDPAddress); err != nil {
		return err
	}
	if err := validateRoot("data_root", config.DataRoot); err != nil {
		return err
	}
	if err := validateRoot("quarantine_root", config.QuarantineRoot); err != nil {
		return err
	}

	devices := make(map[MACAddress]struct{}, len(config.Devices))
	for index, device := range config.Devices {
		mac, err := ParseMACAddress(device.MAC)
		if err != nil {
			return invalidConfig("devices[%d].mac: %v", index, err)
		}
		if _, exists := devices[mac]; exists {
			return invalidConfig("devices[%d].mac: duplicate MAC %q", index, device.MAC)
		}
		devices[mac] = struct{}{}
		if err := validateName(device.Area); err != nil {
			return invalidConfig("devices[%d].area: %v", index, err)
		}
	}

	sensors := make(map[uint16]struct{}, len(config.Sensors))
	for index, sensor := range config.Sensors {
		if sensor.ID == 0 {
			return invalidConfig("sensors[%d].id: zero is reserved", index)
		}
		if _, exists := sensors[sensor.ID]; exists {
			return invalidConfig("sensors[%d].id: duplicate ID %d", index, sensor.ID)
		}
		sensors[sensor.ID] = struct{}{}
		if err := validateName(sensor.Type); err != nil {
			return invalidConfig("sensors[%d].type: %v", index, err)
		}
		if strings.TrimSpace(sensor.Unit) == "" {
			return invalidConfig("sensors[%d].unit: empty", index)
		}
	}

	if config.DataStream.QueueCapacity < 1 {
		return invalidConfig("data_stream.queue_capacity must be positive")
	}
	if config.DataStream.EnqueueWait.Value() < 0 {
		return invalidConfig("data_stream.enqueue_wait must not be negative")
	}
	if config.DataStream.RetryCount < 0 {
		return invalidConfig("data_stream.retry_count must not be negative")
	}
	if config.DataStream.InitialBackoff.Value() <= 0 || config.DataStream.MaximumBackoff.Value() < config.DataStream.InitialBackoff.Value() {
		return invalidConfig("data_stream backoff range is invalid")
	}
	if config.DataStream.WriteBufferSize < 4*1024 || config.DataStream.WriteBufferSize > 1024*1024 {
		return invalidConfig("data_stream.write_buffer_size must be between 4 KiB and 1 MiB")
	}
	if config.DataStream.FlushInterval.Value() < 100*time.Millisecond || config.DataStream.FlushInterval.Value() > time.Minute {
		return invalidConfig("data_stream.flush_interval must be between 100ms and 1m")
	}
	if config.DataStream.RotationSize < SensorDataRecordSize {
		return invalidConfig("data_stream.rotation_size must fit at least one record")
	}
	if config.Network.MaximumTCPConnections < 1 {
		return invalidConfig("network.maximum_tcp_connections must be positive")
	}
	if config.Network.PayloadDeadline.Value() < time.Second || config.Network.PayloadDeadline.Value() > time.Minute {
		return invalidConfig("network.payload_deadline must be between 1s and 1m")
	}
	if config.ShutdownDeadline.Value() < time.Second || config.ShutdownDeadline.Value() > 5*time.Minute {
		return invalidConfig("shutdown_deadline must be between 1s and 5m")
	}
	if !oneOf(strings.ToLower(config.Logging.Level), "debug", "info", "warn", "error", "disabled") {
		return invalidConfig("logging.level %q is unsupported", config.Logging.Level)
	}
	if config.Logging.Output != "stdout" && config.Logging.Output != "stderr" && strings.TrimSpace(config.Logging.Output) == "" {
		return invalidConfig("logging.output is empty")
	}
	return nil
}

const SensorDataRecordSize int64 = 10

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return invalidConfig("multiple JSON values")
		}
		return fmt.Errorf("decode configuration trailer: %w", err)
	}
	return nil
}

func validateListenAddress(field, value string) error {
	value = strings.TrimSpace(value)
	_, portText, err := net.SplitHostPort(value)
	if err != nil {
		return invalidConfig("%s %q: %v", field, value, err)
	}
	port, err := strconv.ParseUint(portText, 10, 16)
	if err != nil || port == 0 {
		return invalidConfig("%s %q has an invalid port", field, value)
	}
	return nil
}

func validateRoot(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return invalidConfig("%s is empty", field)
	}
	if filepath.Clean(value) == "." {
		return invalidConfig("%s must identify a directory", field)
	}
	return nil
}

func validateName(value string) error {
	if value == "" || strings.TrimSpace(value) != value {
		return errors.New("name is empty or has surrounding whitespace")
	}
	if value == "." || value == ".." || strings.ContainsAny(value, `/\\`) {
		return errors.New("name contains a path component")
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return errors.New("name contains a control character")
		}
	}
	return nil
}

func invalidConfig(format string, values ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidConfiguration, fmt.Sprintf(format, values...))
}

func cloneConfig(config Config) Config {
	config.Devices = append([]DeviceConfig(nil), config.Devices...)
	config.Sensors = append([]SensorConfig(nil), config.Sensors...)
	return config
}

func configsEqual(left, right Config) bool {
	leftJSON, _ := json.Marshal(left)
	rightJSON, _ := json.Marshal(right)
	return string(leftJSON) == string(rightJSON)
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
