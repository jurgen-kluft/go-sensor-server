package httpplugin

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sensorserver "github.com/jurgen-kluft/go-sensor-server/sensor-server"
)

type testCoreServer struct {
	registry *sensorserver.ConfigRegistry
	counters sensorserver.ServerCounters
	done     chan struct{}
}

func (server *testCoreServer) Counters() sensorserver.ServerCounters  { return server.counters }
func (server *testCoreServer) Registry() *sensorserver.ConfigRegistry { return server.registry }
func (server *testCoreServer) Done() <-chan struct{}                  { return server.done }

func TestHandlerReportsHealthAndMonitoringData(t *testing.T) {
	core, state := newHandlerTestDependencies(t)
	mac := sensorserver.MACAddress{2, 0, 0, 0xab, 0xcd, 0xef}
	state.OnSensorObservation(sensorserver.SensorObservation{
		MAC: mac, Area: sensorserver.Area1stLivingRoom, Transport: sensorserver.TransportUDP,
		SensorType: sensorserver.SENSOR_ID_BATTERY, UnitType: sensorserver.UCelcius, Timestamp: 1234, Value: 21,
	})
	handler := corsHandler(newHandler(context.Background(), core, state))

	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/health", nil))
	if health.Code != http.StatusOK || health.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("health status = %d, CORS = %q", health.Code, health.Header().Get("Access-Control-Allow-Origin"))
	}

	detail := httptest.NewRecorder()
	handler.ServeHTTP(detail, httptest.NewRequest(http.MethodGet, "/api/v1/devices/02:00:00:ab:cd:ef", nil))
	if detail.Code != http.StatusOK {
		t.Fatalf("device status = %d, body = %s", detail.Code, detail.Body.String())
	}
	var device DeviceSnapshot
	if err := json.NewDecoder(detail.Body).Decode(&device); err != nil {
		t.Fatalf("decode device: %v", err)
	}
	if device.ObservedSensors != 1 || len(device.Sensors) != 1 || device.Sensors[0].Current.Value != 21 {
		t.Fatalf("device = %+v", device)
	}
}

func TestHandlerHealthFailsForPipelineErrors(t *testing.T) {
	core, state := newHandlerTestDependencies(t)
	core.counters.Router.DataEngineFailures = 1
	recorder := httptest.NewRecorder()
	newHandler(context.Background(), core, state).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}

func TestHandlerReturnsEmptyHistoryForUnseenConfiguredSensor(t *testing.T) {
	core, state := newHandlerTestDependencies(t)
	recorder := httptest.NewRecorder()
	newHandler(context.Background(), core, state).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/devices/02:00:00:ab:cd:ef/sensors/1/readings", nil))
	if recorder.Code != http.StatusOK || recorder.Body.String() != "[]\n" {
		t.Fatalf("status = %d, body = %q", recorder.Code, recorder.Body.String())
	}
}

func TestSSEStreamReceivesSensorObservation(t *testing.T) {
	core, state := newHandlerTestDependencies(t)
	server := httptest.NewServer(corsHandler(newHandler(context.Background(), core, state)))
	defer server.Close()

	requestContext, cancel := context.WithCancel(context.Background())
	defer cancel()
	request, err := http.NewRequestWithContext(requestContext, http.MethodGet, server.URL+"/api/v1/events", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext() error = %v", err)
	}
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.HasPrefix(response.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("status = %d, content type = %q", response.StatusCode, response.Header.Get("Content-Type"))
	}

	state.OnSensorObservation(sensorserver.SensorObservation{
		MAC: sensorserver.MACAddress{2, 0, 0, 0xab, 0xcd, 0xef}, Area: sensorserver.Area1stLivingRoom,
		Transport: sensorserver.TransportUDP, SensorType: sensorserver.SENSOR_ID_TEMPERATURE,
		UnitType: sensorserver.UCelcius, Timestamp: 1234, Value: 21,
	})

	lines := make(chan string, 8)
	scanErrors := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(response.Body)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		scanErrors <- scanner.Err()
	}()
	deadline := time.After(2 * time.Second)
	foundName := false
	foundData := false
	for !foundName || !foundData {
		select {
		case line := <-lines:
			foundName = foundName || line == "event: sensor_observation"
			foundData = foundData || strings.Contains(line, `"value":21`)
		case err := <-scanErrors:
			t.Fatalf("SSE stream ended early: %v", err)
		case <-deadline:
			t.Fatalf("SSE event incomplete: name=%v data=%v", foundName, foundData)
		}
	}
}

func TestServerShutdownClosesActiveSSEStream(t *testing.T) {
	core, state := newHandlerTestDependencies(t)
	server, err := NewServer(Config{Address: "127.0.0.1:0", HistoryCapacity: 10}, core, state)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	server.Serve()
	response, err := http.Get("http://" + server.Address() + "/api/v1/events")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer response.Body.Close()

	shutdownDone := make(chan error, 1)
	go func() {
		shutdownContext, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		shutdownDone <- server.Shutdown(shutdownContext)
	}()
	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown() blocked on active SSE stream")
	}
}

func newHandlerTestDependencies(t *testing.T) (*testCoreServer, *MonitoringState) {
	t.Helper()
	registry, err := sensorserver.NewConfigRegistry(testSnapshot(t))
	if err != nil {
		t.Fatalf("NewConfigRegistry() error = %v", err)
	}
	state, err := NewMonitoringState(Config{Address: ":8080", HistoryCapacity: 10})
	if err != nil {
		t.Fatalf("NewMonitoringState() error = %v", err)
	}
	return &testCoreServer{registry: registry, done: make(chan struct{})}, state
}
