package sensorserver

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type ConfigPollerOptions struct {
	Interval time.Duration
	OnError  func(error)
	OnReload func(*ConfigSnapshot)
}

type ConfigPoller struct {
	fileSystem FileSystem
	clock      Clock
	registry   *ConfigRegistry
	path       string
	options    ConfigPollerOptions
	observed   configFileFingerprint
}

type configFileFingerprint struct {
	modified time.Time
	size     int64
}

func NewConfigPoller(fileSystem FileSystem, clock Clock, registry *ConfigRegistry, path string, options ConfigPollerOptions) (*ConfigPoller, error) {
	if fileSystem == nil || clock == nil || registry == nil || registry.Snapshot() == nil {
		return nil, errors.New("new config poller: nil dependency")
	}
	if path == "" {
		return nil, errors.New("new config poller: empty path")
	}
	if options.Interval <= 0 {
		return nil, errors.New("new config poller: interval must be positive")
	}
	if options.OnError == nil {
		options.OnError = func(error) {}
	}
	if options.OnReload == nil {
		options.OnReload = func(*ConfigSnapshot) {}
	}
	info, err := fileSystem.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat configuration %q: %w", path, err)
	}
	return &ConfigPoller{
		fileSystem: fileSystem,
		clock:      clock,
		registry:   registry,
		path:       path,
		options:    options,
		observed:   fingerprint(info.ModTime(), info.Size()),
	}, nil
}

func (poller *ConfigPoller) Run(ctx context.Context) {
	ticker := poller.clock.NewTicker(poller.options.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C():
			if _, err := poller.Poll(); err != nil {
				poller.options.OnError(err)
			}
		}
	}
}

// Poll checks once for a changed file and publishes a valid device-only reload.
func (poller *ConfigPoller) Poll() (bool, error) {
	info, err := poller.fileSystem.Stat(poller.path)
	if err != nil {
		return false, fmt.Errorf("stat configuration %q: %w", poller.path, err)
	}
	current := fingerprint(info.ModTime(), info.Size())
	if current == poller.observed {
		return false, nil
	}
	poller.observed = current

	file, err := poller.fileSystem.OpenFile(poller.path, readOnlyFlags, 0)
	if err != nil {
		return false, fmt.Errorf("open configuration %q: %w", poller.path, err)
	}
	candidate, loadErr := LoadConfig(file)
	closeErr := file.Close()
	if loadErr != nil || closeErr != nil {
		return false, fmt.Errorf("load configuration %q: %w", poller.path, errors.Join(loadErr, closeErr))
	}
	if err := poller.registry.Reload(candidate); err != nil {
		return false, fmt.Errorf("reload configuration %q: %w", poller.path, err)
	}
	poller.options.OnReload(candidate)
	return true, nil
}

func fingerprint(modified time.Time, size int64) configFileFingerprint {
	return configFileFingerprint{modified: modified, size: size}
}
