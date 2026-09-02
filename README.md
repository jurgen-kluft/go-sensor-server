# Sensor Server

The sensor server receives binary sensor messages over TCP and UDP, routes
known devices into segmented data files, and durably quarantines unknown-device
messages for replay on the next startup.

## Run

Go 1.23 or newer is required.

```sh
go run ./cmd/sensor-server -config sensor-server.json
```

To also run the optional monitoring API:

```sh
go run ./cmd/sensor-server \
	-config sensor-server.json \
	-http-config cmd/sensor-server/http-config.json
```

The HTTP plugin is not started unless `-http-config` is provided and its
configuration has `"enabled": true`.

The process handles `SIGINT` and `SIGTERM`, stops network ingress, drains
accepted records, synchronizes storage, and logs a final counter snapshot.

## Configuration

```json
{
	"tcp_address": ":9000",
	"udp_address": ":9001",
	"data_root": "./data",
	"quarantine_root": "./quarantine",
	"devices": [
		{"mac": "02:00:00:ab:cd:ef", "area": "1st Living Room"}
	],
	"data_stream": {},
	"network": {},
	"logging": {"level": "info", "output": "stderr"},
	"shutdown_deadline": "30s"
}
```

The configuration file is polled every 15 seconds. Device assignments can be
reloaded without restarting; changes to network, storage, sensors, or runtime
policy are rejected until restart. Logging output accepts `stdout`, `stderr`,
or a file path. See [docs/design.md](docs/design.md) for the wire and storage
formats and all validated limits.

## HTTP monitoring plugin

The optional plugin lives in `plugins/http` and depends on the core
`sensor-server` package. The core package does not depend on HTTP. The command
under `cmd/sensor-server` is responsible for composing and shutting down both
components.

Example HTTP configuration:

```json
{
	"enabled": true,
	"address": ":8080",
	"history_capacity": 1000
}
```

Sensor inactivity does not create a warning.

The plugin exposes these read-only endpoints:

- `GET /health`
- `GET /api/v1/status`
- `GET /api/v1/events`
- `GET /api/v1/rooms`
- `GET /api/v1/rooms/{room}/devices`
- `GET /api/v1/devices/{mac}`
- `GET /api/v1/devices/{mac}/sensors/{sensorID}/readings`

Per-device sensor histories are fixed-size in-memory buffers. They begin empty
on startup and are not restored from the room-level data files. TCP devices
report their current MAC-bound connection state. UDP devices are connectionless
and report only their last observed transport and timestamp.

### Live updates

`GET /api/v1/events` is a Server-Sent Events stream for live dashboard
updates. It emits `sensor_observation`, `device_connected`, and
`device_disconnected` events. Each event contains one JSON `data` value.

```js
const events = new EventSource("http://mac-hostname.local:8080/api/v1/events");

events.addEventListener("sensor_observation", event => {
	const observation = JSON.parse(event.data);
	console.log(observation.mac, observation.sensor_type, observation.sample);
});

events.addEventListener("device_connected", event => {
	console.log("connected", JSON.parse(event.data).mac);
});

events.addEventListener("device_disconnected", event => {
	console.log("disconnected", JSON.parse(event.data).mac);
});
```

The stream is live-only and sends heartbeat comments every 15 seconds. It has
bounded per-client buffering and may drop updates for a client that cannot keep
up rather than delaying sensor ingestion. A web application should load an
initial REST snapshot and reload that snapshot whenever `EventSource`
reconnects; event replay and `Last-Event-ID` are not supported.

The API allows cross-origin GET requests so a separately hosted web application
can call it. It has no authentication or TLS and is intended only for a trusted
local network. Do not expose its listening port directly to the public internet.

## Verification

```sh
go test -race ./...
go test ./sensor-server -run '^$' -fuzz '^FuzzDecodeDatagram$'
go test ./sensor-server -run '^$' -fuzz '^FuzzLoadConfig$'
```

