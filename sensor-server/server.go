package sensorserver

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	DefaultConfigPollInterval = 15 * time.Second
	DefaultUDPWorkerCount     = 4
	DefaultUDPQueueCapacity   = 1024
)

type ServerOptions struct {
	FileSystem           FileSystem
	Network              Network
	Clock                Clock
	ConfigPollInterval   time.Duration
	UDPWorkerCount       int
	UDPQueueCapacity     int
	OnWarning            func(error)
	OnReload             func(*ConfigSnapshot)
	OnSensorObservation  func(SensorObservation)
	OnDeviceConnected    func(connectionID uint64, mac MACAddress)
	OnDeviceDisconnected func(connectionID uint64, mac MACAddress)
}

type Server struct {
	config     Config
	registry   *ConfigRegistry
	engine     *DataEngine
	quarantine *QuarantineWriter
	router     *MessageRouter
	tcp        *TCPServer
	udp        *UDPServer
	poller     *ConfigPoller

	ctx       context.Context
	cancel    context.CancelFunc
	done      chan struct{}
	waitGroup sync.WaitGroup
	closeOnce sync.Once
	errMu     sync.Mutex
	runErr    error
	closeErr  error
}

type ServerCounters struct {
	TCP         TCPServerCounters
	UDP         UDPServerCounters
	Router      RouterCounters
	DataStreams []DataStreamSnapshot
}

func OpenServer(ctx context.Context, configPath string, options ServerOptions) (*Server, error) {
	if ctx == nil {
		return nil, errors.New("open server: nil context")
	}
	applyServerOptionDefaults(&options)
	if options.FileSystem == nil || options.Network == nil || options.Clock == nil {
		return nil, errors.New("open server: nil platform dependency")
	}

	snapshot, err := LoadConfigFile(options.FileSystem, configPath)
	if err != nil {
		return nil, err
	}
	config := snapshot.Config()
	registry, err := NewConfigRegistry(snapshot)
	if err != nil {
		return nil, err
	}
	engine, err := NewDataEngine(NewFileDataStreamFactory(options.FileSystem, options.Clock, config.DataRoot, config.DataStream, options.OnWarning))
	if err != nil {
		return nil, err
	}
	quarantineOptions := DefaultQuarantineOptions(config.QuarantineRoot)
	quarantineOptions.RotationSize = config.DataStream.RotationSize
	quarantine, err := OpenQuarantineWriter(options.FileSystem, quarantineOptions)
	if err != nil {
		_ = engine.Close()
		return nil, err
	}
	cleanupStorage := func() {
		_ = quarantine.Close()
		_ = engine.Close()
	}
	if _, err := ReplayQuarantine(ctx, quarantine, registry, engine); err != nil {
		cleanupStorage()
		return nil, fmt.Errorf("replay quarantine: %w", err)
	}
	router, err := NewMessageRouter(registry, engine, quarantine)
	if err != nil {
		cleanupStorage()
		return nil, err
	}
	router.onSensorObservation = options.OnSensorObservation
	tcpListener, err := options.Network.Listen("tcp", config.TCPAddress)
	if err != nil {
		cleanupStorage()
		return nil, fmt.Errorf("listen TCP on %q: %w", config.TCPAddress, err)
	}
	udpConnection, err := options.Network.ListenPacket("udp", config.UDPAddress)
	if err != nil {
		_ = tcpListener.Close()
		cleanupStorage()
		return nil, fmt.Errorf("listen UDP on %q: %w", config.UDPAddress, err)
	}
	tcp, err := NewTCPServer(tcpListener, router, options.Clock, TCPServerOptions{
		MaximumConnections:   config.Network.MaximumTCPConnections,
		PayloadDeadline:      config.Network.PayloadDeadline.Value(),
		OnWarning:            options.OnWarning,
		OnDeviceConnected:    options.OnDeviceConnected,
		OnDeviceDisconnected: options.OnDeviceDisconnected,
	})
	if err != nil {
		_ = udpConnection.Close()
		_ = tcpListener.Close()
		cleanupStorage()
		return nil, err
	}
	udp, err := NewUDPServer(udpConnection, router, options.Clock, UDPServerOptions{
		WorkerCount:   options.UDPWorkerCount,
		QueueCapacity: options.UDPQueueCapacity,
		OnWarning:     options.OnWarning,
	})
	if err != nil {
		_ = tcp.Close()
		_ = udpConnection.Close()
		cleanupStorage()
		return nil, err
	}
	poller, err := NewConfigPoller(options.FileSystem, options.Clock, registry, configPath, ConfigPollerOptions{
		Interval: options.ConfigPollInterval,
		OnError:  options.OnWarning,
		OnReload: options.OnReload,
	})
	if err != nil {
		_ = udp.Close()
		_ = tcp.Close()
		cleanupStorage()
		return nil, err
	}

	serverContext, cancel := context.WithCancel(context.Background())
	server := &Server{
		config: config, registry: registry, engine: engine, quarantine: quarantine,
		router: router, tcp: tcp, udp: udp, poller: poller, ctx: serverContext, cancel: cancel, done: make(chan struct{}),
	}
	server.start(options.OnWarning)
	go func() {
		select {
		case <-ctx.Done():
			_ = server.Close()
		case <-server.ctx.Done():
		}
	}()
	return server, nil
}

func (server *Server) Close() error {
	server.closeOnce.Do(func() {
		server.cancel()
		server.closeErr = errors.Join(server.udp.Close(), server.tcp.Close())
		server.waitGroup.Wait()
		server.closeErr = errors.Join(server.closeErr, server.quarantine.Close(), server.engine.Close())
		server.errMu.Lock()
		server.closeErr = errors.Join(server.runErr, server.closeErr)
		server.errMu.Unlock()
		close(server.done)
	})
	return server.closeErr
}

func (server *Server) Shutdown(ctx context.Context) error {
	if ctx == nil {
		return errors.New("shutdown server: nil context")
	}
	shutdownContext, cancel := context.WithTimeout(ctx, server.config.ShutdownDeadline.Value())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- server.Close() }()
	select {
	case err := <-done:
		return err
	case <-shutdownContext.Done():
		return fmt.Errorf("shutdown server: %w", shutdownContext.Err())
	}
}

func (server *Server) Registry() *ConfigRegistry { return server.registry }
func (server *Server) Router() *MessageRouter    { return server.router }
func (server *Server) TCPServer() *TCPServer     { return server.tcp }
func (server *Server) UDPServer() *UDPServer     { return server.udp }
func (server *Server) Done() <-chan struct{}     { return server.done }

func (server *Server) Counters() ServerCounters {
	return ServerCounters{
		TCP:         server.tcp.Counters(),
		UDP:         server.udp.Counters(),
		Router:      server.router.Counters(),
		DataStreams: server.engine.Counters(),
	}
}

func (server *Server) start(onWarning func(error)) {
	server.waitGroup.Add(3)
	go func() {
		defer server.componentDone("configuration poller", onWarning)
		server.poller.Run(server.ctx)
	}()
	go func() {
		defer server.componentDone("TCP server", onWarning)
		if err := server.tcp.Serve(); err != nil {
			server.fail(err, onWarning)
		}
	}()
	go func() {
		defer server.componentDone("UDP server", onWarning)
		if err := server.udp.Serve(); err != nil {
			server.fail(err, onWarning)
		}
	}()
}

func (server *Server) componentDone(name string, onWarning func(error)) {
	if recovered := recover(); recovered != nil {
		server.fail(fmt.Errorf("%s panic: %v", name, recovered), onWarning)
	}
	server.waitGroup.Done()
}

func (server *Server) fail(err error, onWarning func(error)) {
	server.errMu.Lock()
	if server.runErr == nil {
		server.runErr = err
	}
	server.errMu.Unlock()
	onWarning(err)
	go func() { _ = server.Close() }()
}

func LoadConfigFile(fileSystem FileSystem, path string) (*ConfigSnapshot, error) {
	if fileSystem == nil {
		return nil, errors.New("load configuration: nil filesystem")
	}
	if path == "" {
		return nil, errors.New("load configuration: empty path")
	}
	file, err := fileSystem.OpenFile(path, readOnlyFlags, 0)
	if err != nil {
		return nil, fmt.Errorf("open configuration %q: %w", path, err)
	}
	snapshot, loadErr := LoadConfig(file)
	closeErr := file.Close()
	if loadErr != nil || closeErr != nil {
		return nil, fmt.Errorf("load configuration %q: %w", path, errors.Join(loadErr, closeErr))
	}
	return snapshot, nil
}

func applyServerOptionDefaults(options *ServerOptions) {
	if options.ConfigPollInterval <= 0 {
		options.ConfigPollInterval = DefaultConfigPollInterval
	}
	if options.UDPWorkerCount <= 0 {
		options.UDPWorkerCount = DefaultUDPWorkerCount
	}
	if options.UDPQueueCapacity <= 0 {
		options.UDPQueueCapacity = DefaultUDPQueueCapacity
	}
	if options.OnWarning == nil {
		options.OnWarning = func(error) {}
	}
	if options.OnReload == nil {
		options.OnReload = func(*ConfigSnapshot) {}
	}
}
