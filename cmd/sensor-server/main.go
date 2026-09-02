package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jurgen-kluft/go-sensor-server/logging"
	httpplugin "github.com/jurgen-kluft/go-sensor-server/plugins/http"
	sensorserver "github.com/jurgen-kluft/go-sensor-server/sensor-server"
)

func main() {
	configPath := flag.String("config", "sensor-server.json", "configuration file path")
	httpConfigPath := flag.String("http-config", "", "optional HTTP plugin configuration file path")
	flag.Parse()
	if err := run(*configPath, *httpConfigPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(configPath, httpConfigPath string) error {
	fileSystem := sensorserver.OSFileSystem{}
	snapshot, err := sensorserver.LoadConfigFile(fileSystem, configPath)
	if err != nil {
		return err
	}
	config := snapshot.Config()
	logger, err := logging.New(config.Logging.Level, config.Logging.Output)
	if err != nil {
		return err
	}
	defer logger.Close()
	limiter, err := logging.NewRateLimiter(time.Second, time.Now)
	if err != nil {
		return err
	}

	signalContext, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	var monitoring *httpplugin.MonitoringState
	var httpConfig httpplugin.Config
	if httpConfigPath != "" {
		configFile, openErr := os.Open(httpConfigPath)
		if openErr != nil {
			return fmt.Errorf("open HTTP plugin configuration %q: %w", httpConfigPath, openErr)
		}
		httpConfig, err = httpplugin.LoadConfig(configFile)
		closeErr := configFile.Close()
		if err != nil || closeErr != nil {
			return errors.Join(err, closeErr)
		}
		if httpConfig.Enabled {
			monitoring, err = httpplugin.NewMonitoringState(httpConfig)
			if err != nil {
				return err
			}
		}
	}

	serverOptions := sensorserver.ServerOptions{
		FileSystem: fileSystem,
		Network:    sensorserver.StandardNetwork{},
		Clock:      sensorserver.SystemClock{},
		OnWarning:  limiter.WarningCallback(logger),
		OnReload: func(snapshot *sensorserver.ConfigSnapshot) {
			logger.LogInfof("configuration reloaded: devices=%d", len(snapshot.Config().Devices))
		},
	}
	if monitoring != nil {
		serverOptions.OnSensorObservation = monitoring.OnSensorObservation
		serverOptions.OnDeviceConnected = monitoring.OnDeviceConnected
		serverOptions.OnDeviceDisconnected = monitoring.OnDeviceDisconnected
	}
	server, err := sensorserver.OpenServer(context.Background(), configPath, serverOptions)
	if err != nil {
		return err
	}

	var httpServer *httpplugin.Server
	var httpDone <-chan struct{}
	if monitoring != nil {
		httpServer, err = httpplugin.NewServer(httpConfig, server, monitoring)
		if err != nil {
			return errors.Join(err, server.Close())
		}
		httpServer.Serve()
		httpDone = httpServer.Done()
		logger.LogInfof("HTTP plugin ready: http=%s", httpServer.Address())
	}
	logger.LogInfof("server ready: tcp=%s udp=%s", config.TCPAddress, config.UDPAddress)
	select {
	case <-signalContext.Done():
	case <-server.Done():
	case <-httpDone:
	}
	logger.LogInfo("graceful shutdown started")
	if httpServer != nil {
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err = httpServer.Shutdown(shutdownContext)
		cancel()
		if err != nil {
			_ = server.Close()
			return err
		}
	}
	if err := server.Shutdown(context.Background()); err != nil {
		return err
	}
	logger.LogInfof("shutdown counters: %+v", server.Counters())
	return nil
}
