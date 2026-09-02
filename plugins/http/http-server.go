package httpplugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	sensorserver "github.com/jurgen-kluft/go-sensor-server/sensor-server"
)

type CoreServer interface {
	Counters() sensorserver.ServerCounters
	Registry() *sensorserver.ConfigRegistry
	Done() <-chan struct{}
}

type Server struct {
	listener net.Listener
	server   *http.Server
	cancel   context.CancelFunc
	done     chan struct{}
	errMu    sync.Mutex
	err      error
}

type StatusResponse struct {
	Status   string                      `json:"status"`
	Issues   []string                    `json:"issues"`
	Counters sensorserver.ServerCounters `json:"counters"`
}

func NewServer(config Config, core CoreServer, state *MonitoringState) (*Server, error) {
	if core == nil || state == nil {
		return nil, errors.New("new HTTP plugin server: nil dependency")
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp", config.Address)
	if err != nil {
		return nil, fmt.Errorf("listen HTTP on %q: %w", config.Address, err)
	}
	serverContext, cancel := context.WithCancel(context.Background())
	server := &Server{listener: listener, cancel: cancel, done: make(chan struct{})}
	server.server = &http.Server{
		Handler:           corsHandler(newHandler(serverContext, core, state)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return server, nil
}

func (server *Server) Address() string       { return server.listener.Addr().String() }
func (server *Server) Done() <-chan struct{} { return server.done }

func (server *Server) Serve() {
	go func() {
		defer close(server.done)
		if err := server.server.Serve(server.listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			server.errMu.Lock()
			server.err = err
			server.errMu.Unlock()
		}
	}()
}

func (server *Server) Shutdown(ctx context.Context) error {
	if ctx == nil {
		return errors.New("shutdown HTTP plugin server: nil context")
	}
	server.cancel()
	err := server.server.Shutdown(ctx)
	<-server.done
	server.errMu.Lock()
	defer server.errMu.Unlock()
	return errors.Join(err, server.err)
}

func newHandler(serverContext context.Context, core CoreServer, state *MonitoringState) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(writer http.ResponseWriter, request *http.Request) {
		status := buildStatus(core)
		code := http.StatusOK
		if status.Status != "healthy" {
			code = http.StatusServiceUnavailable
		}
		writeJSON(writer, code, struct {
			Status string   `json:"status"`
			Issues []string `json:"issues"`
		}{Status: status.Status, Issues: status.Issues})
	})
	mux.HandleFunc("GET /api/v1/status", func(writer http.ResponseWriter, request *http.Request) {
		writeJSON(writer, http.StatusOK, buildStatus(core))
	})
	mux.HandleFunc("GET /api/v1/events", func(writer http.ResponseWriter, request *http.Request) {
		handleEvents(serverContext, core, state, writer, request)
	})
	mux.HandleFunc("GET /api/v1/rooms", func(writer http.ResponseWriter, request *http.Request) {
		writeJSON(writer, http.StatusOK, state.Rooms(core.Registry().Snapshot()))
	})
	mux.HandleFunc("GET /api/v1/rooms/{room}/devices", func(writer http.ResponseWriter, request *http.Request) {
		room := request.PathValue("room")
		devices := state.Devices(core.Registry().Snapshot())
		filtered := make([]DeviceSnapshot, 0)
		for _, device := range devices {
			if device.Area == room {
				filtered = append(filtered, device)
			}
		}
		if len(filtered) == 0 {
			writeError(writer, http.StatusNotFound, "room not found")
			return
		}
		writeJSON(writer, http.StatusOK, filtered)
	})
	mux.HandleFunc("GET /api/v1/devices/{mac}", func(writer http.ResponseWriter, request *http.Request) {
		mac, err := sensorserver.ParseMACAddress(request.PathValue("mac"))
		if err != nil {
			writeError(writer, http.StatusBadRequest, "invalid MAC address")
			return
		}
		device, exists := state.Device(core.Registry().Snapshot(), mac)
		if !exists {
			writeError(writer, http.StatusNotFound, "device not found")
			return
		}
		writeJSON(writer, http.StatusOK, device)
	})
	mux.HandleFunc("GET /api/v1/devices/{mac}/sensors/{sensorID}/readings", func(writer http.ResponseWriter, request *http.Request) {
		mac, err := sensorserver.ParseMACAddress(request.PathValue("mac"))
		if err != nil {
			writeError(writer, http.StatusBadRequest, "invalid MAC address")
			return
		}
		sensorID, err := strconv.ParseUint(request.PathValue("sensorID"), 10, 16)
		if err != nil || sensorID == 0 {
			writeError(writer, http.StatusBadRequest, "invalid sensor ID")
			return
		}
		snapshot := core.Registry().Snapshot()
		if _, exists := snapshot.Device(mac); !exists {
			writeError(writer, http.StatusNotFound, "device not found")
			return
		}
		sensorType := sensorserver.ToSensorType(uint16(sensorID))
		if _, exists := snapshot.Sensor(sensorType); !exists {
			writeError(writer, http.StatusNotFound, "sensor not found")
			return
		}
		readings, exists := state.Readings(mac, sensorType)
		if !exists {
			readings = []Sample{}
		}
		writeJSON(writer, http.StatusOK, readings)
	})
	return mux
}

func handleEvents(serverContext context.Context, core CoreServer, state *MonitoringState, writer http.ResponseWriter, request *http.Request) {
	flusher, ok := writer.(http.Flusher)
	if !ok {
		writeError(writer, http.StatusInternalServerError, "streaming is unsupported")
		return
	}
	events, unsubscribe := state.Subscribe()
	defer unsubscribe()

	writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-cache, no-transform")
	writer.Header().Set("Connection", "keep-alive")
	writer.Header().Set("X-Accel-Buffering", "no")
	_, _ = writer.Write([]byte(": connected\nretry: 5000\n\n"))
	flusher.Flush()

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case event := <-events:
			if _, err := fmt.Fprintf(writer, "event: %s\ndata: %s\n\n", event.Name, event.Data); err != nil {
				return
			}
			flusher.Flush()
		case <-heartbeat.C:
			if _, err := writer.Write([]byte(": heartbeat\n\n")); err != nil {
				return
			}
			flusher.Flush()
		case <-request.Context().Done():
			return
		case <-serverContext.Done():
			return
		case <-core.Done():
			return
		}
	}
}

func buildStatus(core CoreServer) StatusResponse {
	counters := core.Counters()
	issues := make([]string, 0)
	select {
	case <-core.Done():
		issues = append(issues, "sensor server is stopped")
	default:
	}
	if counters.Router.DataEngineFailures > 0 {
		issues = append(issues, "sensor data writes have failed")
	}
	if counters.Router.QuarantineFailures > 0 {
		issues = append(issues, "unknown message quarantine writes have failed")
	}
	for _, stream := range counters.DataStreams {
		if stream.Counters.Errors > 0 || stream.Counters.Dropped > 0 {
			issues = append(issues, fmt.Sprintf("data stream %s/%s has write errors or dropped samples", stream.Area, stream.SensorType))
		}
	}
	sort.Strings(issues)
	status := "healthy"
	if len(issues) > 0 {
		status = "unhealthy"
	}
	return StatusResponse{Status: status, Issues: issues, Counters: counters}
}

func corsHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Access-Control-Allow-Origin", "*")
		writer.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if request.Method == http.MethodOptions {
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func writeError(writer http.ResponseWriter, status int, message string) {
	writeJSON(writer, status, struct {
		Error string `json:"error"`
	}{Error: strings.TrimSpace(message)})
}
