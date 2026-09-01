# Sensor Server

The sensor server receives binary sensor messages over TCP and UDP, routes
known devices into segmented data files, and durably quarantines unknown-device
messages for replay on the next startup.

## Run

Go 1.23 or newer is required.

```sh
go run ./cmd/sensor-server -config sensor-server.json
```

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
		{"mac": "02:00:00:ab:cd:ef", "area": "LivingRoom"}
	],
	"sensors": [
		{"id": 1, "type": "Temperature", "unit": "Celsius"}
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

## Verification

```sh
go test -race ./...
go test ./sensor-server -run '^$' -fuzz '^FuzzDecodeDatagram$'
go test ./sensor-server -run '^$' -fuzz '^FuzzLoadConfig$'
```

