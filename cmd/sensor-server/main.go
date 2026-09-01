package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jurgen-kluft/go-sensor-server/logging"
	sensorserver "github.com/jurgen-kluft/go-sensor-server/sensor-server"
)

func main() {
	configPath := flag.String("config", "sensor-server.json", "configuration file path")
	flag.Parse()
	if err := run(*configPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(configPath string) error {
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
	server, err := sensorserver.OpenServer(signalContext, configPath, sensorserver.ServerOptions{
		FileSystem: fileSystem,
		Network:    sensorserver.StandardNetwork{},
		Clock:      sensorserver.SystemClock{},
		OnWarning:  limiter.WarningCallback(logger),
		OnReload: func(snapshot *sensorserver.ConfigSnapshot) {
			logger.LogInfof("configuration reloaded: devices=%d", len(snapshot.Config().Devices))
		},
	})
	if err != nil {
		return err
	}
	logger.LogInfof("server ready: tcp=%s udp=%s", config.TCPAddress, config.UDPAddress)
	select {
	case <-signalContext.Done():
	case <-server.Done():
	}
	logger.LogInfo("graceful shutdown started")
	if err := server.Shutdown(context.Background()); err != nil {
		return err
	}
	logger.LogInfof("shutdown counters: %+v", server.Counters())
	return nil
}
