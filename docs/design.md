# Sensor Server Technical Design

## 1. Purpose

The Sensor Server receives sensor readings from ESP32 devices over TCP and UDP
and appends those readings to binary files on disk.

The target is a production-ready home automation service. It must support 50 to
100 concurrent TCP connections, concurrent UDP traffic, configuration reloads, 
graceful shutdown, and recovery from malformed network input without crashing.

Sensor value scaling and interpretation are outside the scope of this design.
The server stores each sensor value as the signed 16-bit value received on the
wire.

## 2. Terminology

* **Device**: An ESP32 identified by its six-byte MAC address.
* **Area**: A room or other location to which one or more devices belong.
* **Sensor definition**: A configured sensor type, and unit.
* **Data stream**: The writer for one `(area, sensor type)` pair.
* **Known device**: A MAC address present in the current configuration.
* **Unknown device**: A MAC address absent from the current configuration.
* **Quarantine**: Durable storage for valid messages from unknown devices.

Multiple devices in one area may report the same sensor type. Their readings
are intentionally consolidated into the same data stream and file sequence.

An optional monitoring plugin exists under `plugins/http` . Dependency flows
from that plugin to the core server package. The executable is the composition
root; the core server neither imports nor manages HTTP.

### 2.1 Core observation boundary

The core server can publish generic observations to callbacks supplied at
startup:

* A validated known-device sensor observation, including MAC, area, transport, 
  sensor definition, server timestamp, and raw value.
* Successful binding of a TCP connection ID to a device MAC.
* Disconnection of a bound TCP connection ID and MAC.

These observations contain no HTTP-specific types. Consumers must keep callback
work bounded so they do not delay ingestion. Connection IDs allow a consumer to
ignore the delayed disconnection of an older connection after the same MAC has
already bound a replacement connection.

### 2.2 HTTP live events

The optional HTTP plugin exposes `/api/v1/events` using Server-Sent Events.
Monitoring callbacks update in-memory state under a lock, release that lock, 
and then publish a copied event payload. Each subscriber has a bounded queue; 
publishing never waits for a slow client. The stream sends sensor observations
and effective TCP connection transitions, plus periodic heartbeat comments.

Events are not retained or replayed. Clients obtain authoritative state from
the REST endpoints before opening the stream and after reconnecting. Plugin
shutdown explicitly cancels active streams before waiting for HTTP shutdown; 
the core sensor server remains independent of this lifecycle.

## 3. Configuration

The server loads a JSON configuration file containing:

* TCP listen address and port.
* UDP listen address and port.
* Sensor data root directory.
* Quarantine directory.
* Device definitions containing a unique MAC address and area name.
* Sensor definitions containing a unique sensor ID, type, and unit.
* Data stream queue, retry, flush, and rotation settings.
* Network connection limits and timeout settings.
* Logging level and output settings.

Configuration loading is transactional:

1. Read and decode the complete candidate file.
2. Apply defaults.
3. Validate the complete candidate configuration.
4. Publish it only if every validation succeeds.

An invalid initial configuration prevents startup. An invalid reload is logged
and leaves the previous configuration active.

Validation must reject:

* Duplicate MAC addresses.
* Duplicate sensor IDs.
* Invalid MAC address representations.
* Ports outside the valid range.
* Empty or unsafe area and sensor type names.
* Paths that escape the configured storage roots after cleaning.
* Queue, timeout, retry, or rotation values outside supported ranges.

Area and sensor type names are logical identifiers, not arbitrary paths. The
configuration package must normalize or reject path separators, `.` and `..` , 
control characters, and platform-specific unsafe names before they reach the
storage package.

### 3.1 Live Reload

A live reload may atomically replace only the MAC-to-area device registry.
Existing requests use either the old or new immutable registry snapshot; they
must never observe a partially updated registry.

Changes to listener addresses, ports, storage paths, sensor definitions, wire
settings, queue settings, or rotation settings require a process restart. A
reload containing such changes is rejected and logged.

Removed or changed MAC mappings affect future messages only. Existing records
are not moved automatically. Quarantined records are replayed only during
startup, after the initial configuration has been loaded.

The server polls the configuration file modification time every 15 seconds and
reloads it when the file has changed.

## 4. Wire Protocol

All multi-byte integer fields use little-endian byte order.

Each TCP message or UDP datagram contains one 16-byte header followed by the
payload declared by that header.

### 4.1 Header

|  Offset | Size | Field | Encoding |
|---:|---:|---|---|
| 0  |  2 | Magic | Unsigned 16-bit integer, fixed value `0xF00D` |
| 2  |  2 | Message type | Unsigned 16-bit integer |
| 4  |  2 | Payload length | Unsigned 16-bit byte count |
| 6  |  6 | MAC address | Raw device MAC bytes |
| 12 |  4 | Checksum | IEEE CRC-32 of the payload |

The two magic bytes on the wire are therefore `0D F0` .

The checksum uses the standard IEEE CRC-32 algorithm supported by Go as
`crc32.ChecksumIEEE` . Its polynomial is `0xEDB88320` in reflected form. The
checksum covers the payload only and is encoded little-endian in the header.
The standard check value for the ASCII bytes `123456789` is `0xCBF43926` .
Firmware and server tests must share golden message vectors.

Message types are represented by an enum. Value `0` is invalid, value `1` is
sensor data, and all other values are reserved for future use. Messages with an
unknown or reserved type are dropped and warning-logged.

The maximum accepted payload is 8 KiB for both TCP and UDP. This is an
application protocol limit, not a network MTU. An 8 KiB UDP datagram exceeds a
typical Ethernet or Wi-Fi MTU and may be fragmented by IP, so UDP senders should
use smaller datagrams that fit their network path MTU whenever possible.

### 4.2 Sensor Payload

A sensor-data payload contains zero or more four-byte records:

| Offset | Size | Field | Encoding |
|---:|---:|---|---|
| 0  |  1 | Sensor Type | Unsigned 8-bit integer |
| 1  |  3 | Sensor Value | Signed 24-bit integer |

The payload length of a sensor-data message must be divisible by four. Each
record in one message receives the same server timestamp.

A zero-length payload is valid for a sensor-data message and is treated as a
keepalive. It produces no sensor records and is warning-logged. A zero-length
payload for any other message type is malformed unless that message type
explicitly permits it.

### 4.3 Timestamp

The server assigns a signed 64-bit Unix timestamp in microseconds to each valid
message. This timestamp is wall-clock time and does not include Go's monotonic
clock component.

The timestamp is captured immediately after the complete message has been read
and before checksum validation and configuration lookup.

### 4.4 Malformed Messages

A message is malformed if any applicable validation fails, including:

* Invalid magic.
* Unsupported message type.
* Payload length over the accepted limit.
* Invalid payload length for its message type.
* Truncated header or payload.
* Extra bytes in a UDP datagram.
* Checksum mismatch.
* Unknown sensor Type.

Malformed messages are not sent to the Data Engine. The server drops them and
emits a rate-limited warning containing the transport, remote address, reason, 
and MAC address when it can be decoded safely.

For UDP, only the current datagram is dropped. For TCP, a bad checksum or
semantically invalid but correctly framed payload can be dropped while keeping
the connection open. An invalid header or unusable payload length closes the
connection because the next message boundary cannot be recovered reliably.

## 5. TCP Server

TCP is a byte stream and does not preserve message boundaries. For each message, 
the server must:

1. Read exactly 16 bytes using full-read semantics.
2. Decode and validate the header fields required for safe framing.
3. Read exactly `Payload Length` bytes using full-read semantics.
4. Validate the checksum and payload.
5. Route the valid message.

Short reads are normal and must not be treated as complete headers or payloads.
Multiple messages received in one socket read must be processed separately.

Each TCP connection belongs to one device. The first valid message binds the
connection to its header MAC address. Every later message on that connection
must contain the same MAC address.

Connections are tracked by a server-generated connection ID and include the
bound MAC, remote address, connection time, last activity time, cancellation
function, and underlying network connection. The server must not use a fixed
array as its primary connection registry. It may enforce a configured maximum
with a semaphore or equivalent admission mechanism.

A correctly framed message whose MAC differs from the MAC bound to the
connection is warning-logged and dropped. The connection remains open.

At most one TCP connection may be active for a MAC. When a new connection is
bound to a MAC that already has an active connection, the server closes and
replaces the older connection.

TCP connections are long-lived and have no idle timeout. A connection may
remain quiet indefinitely until the peer closes it or the server shuts down.
Once a valid header declares a non-empty payload, the complete payload must
arrive within five seconds so a partial message cannot hold connection resources
indefinitely. The payload deadline is cleared after the payload has been read
and is configurable from 1 to 60 seconds.

## 6. UDP Server

One UDP datagram contains exactly one complete message. Its byte length must be
exactly `16 + Payload Length` ; trailing or missing bytes make it malformed.

UDP has no connection lifecycle, delivery guarantee, ordering guarantee, or
backpressure. The server processes each valid datagram independently and must
remain responsive when individual data stream queues are unavailable.

Truncated datagrams and datagrams larger than the receive buffer are dropped and
warning-logged subject to rate limiting.

## 7. Routing Known and Unknown Devices

After wire validation, the server looks up the message MAC in one immutable
configuration snapshot.

For a known MAC, each sensor record is resolved through the sensor registry and
pushed to the Data Engine using:

```go
WriteSensorData(area, sensorType string, timestamp int64, value int32) error
```

The final API may use typed identifiers, but it must carry the same information
and report whether the record was accepted.

For an unknown MAC, the complete valid message and its server timestamp are
written to quarantine. Unknown-device messages must not be written to normal
area streams until their MAC has a configured area.

### 7.1 Quarantine Format and Replay

Quarantine is an append-only, rotated binary log. Each entry must preserve
enough information to reproduce normal routing:

* Format version.
* Complete entry length.
* Server timestamp.
* Source transport.
* Message type.
* MAC address.
* Payload length and payload.
* Entry integrity checksum.

Length and integrity fields allow startup to detect and truncate an incomplete
trailing entry. Quarantine segments are immutable after rotation.

Replay reads quarantined messages, uses the current device registry, and routes
messages whose MAC is now known. Messages that remain unknown stay available
for a future replay. A replay operation must not race with mutation of the same
source segment.

Replay runs only during startup, after configuration and quarantine recovery
and before the network listeners start. Configuration reload does not trigger
replay. Newly recognized MAC addresses are replayed the next time the server
starts.

Replay provides at-least-once delivery using immutable source segments and
atomic per-segment completion markers. Because replay runs only during startup, 
it cannot race with live network ingestion. A crash between writing normal
output and recording replay progress may cause records to be written again on
the next startup.

Processed quarantine segments are retained indefinitely in place.

## 8. Data Engine

The Data Engine owns all Data Streams. It maintains a synchronized map keyed by
normalized `(area type, sensor type)` rather than preallocating inactive streams for
every possible combination.

On the first write to a key, the Data Engine atomically creates one Data Stream.
Concurrent first writes for the same key must resolve to the same stream. Each
active Data Stream owns:

* One bounded channel of records.
* One writer goroutine.
* The active file and buffered writer.
* Rotation and flush state.
* Its terminal error and lifecycle state.

Only that stream's writer goroutine writes its files. This removes the need for
locking around individual file writes and preserves queue arrival order.

Arrival order is not necessarily timestamp order when TCP and UDP messages or
multiple devices feed the same stream.

Data Streams preserve queue arrival order without reordering by timestamp.

### 8.1 Queue and Write Failure Policy

Queues are bounded so unavailable storage cannot consume unbounded memory.
Writes use bounded retries with backoff. If the queue cannot accept a record or
a write still fails after all retries, that record is dropped, an error is
rate-limited and logged with stream context, and a loss counter is incremented.
Failure of one stream must not stop unrelated streams.

TCP acceptance does not imply that a record is durable. UDP cannot provide
end-to-end backpressure. Protocol acknowledgements and durable acceptance are
outside the current protocol.

The queue capacity is 512 records and the enqueue wait duration is 100
milliseconds. Transient file errors are retried up to five times with
exponential backoff starting at 50 milliseconds and capped at one second.
Permanent file errors are not retried.

## 9. Sensor Data Storage

Data is stored beneath the configured root using normalized area and sensor type
names. The logical layout is:

```text
<data-root>/<area>/<sensor-type>/<segment-name>.dat
```

Each file is append-only and contains fixed-size 12-byte records:

| Offset | Size | Field | Encoding |
|---:|---:|---|---|
| 0 | 8 | Timestamp | Signed Unix microseconds, little-endian |
| 8 | 4 | Sensor value | Signed integer, little-endian |

No Go struct may be serialized directly because compiler padding would change
the format. Fields must be encoded explicitly.

On startup, a data file whose size is not divisible by 12 has an incomplete
trailing record. The server warning-logs the condition and truncates the file to
the last complete record before appending.

### 9.1 Rotation

The active file rotates before appending a record that would make it exceed 64
MiB. Rotation never splits a record. Completed segments are immutable and are
retained indefinitely by the server; retention is an external responsibility.

Segment discovery and creation must prevent overwriting an existing segment, 
including after an unclean restart.

Segments use a fixed-width monotonically increasing sequence beginning with
`00000001.dat` . The highest existing valid sequence is discovered at startup.

Each Data Stream uses a 64 KiB write buffer and flushes it every second while
records are pending. Rotation and graceful shutdown flush the buffer and call
`fsync` before closing the file. Individual records do not trigger `fsync` .

The buffer size is configurable from 4 KiB to 1 MiB. The periodic flush interval
is configurable from 100 milliseconds to one minute.

Directories are created with mode `0755` and files with mode `0644` , subject to
the process umask, so other processes can read stored data.

## 10. Lifecycle

### 10.1 Startup

Startup proceeds in this order:

1. Load and validate configuration.
2. Validate or create storage and quarantine directories.
3. Discover existing data and quarantine segments and repair incomplete tails.
4. Create the Data Engine and quarantine writer.
5. Start UDP and TCP listeners.
6. Report readiness.

Failure before readiness stops startup and closes all resources already opened.

### 10.2 Graceful Shutdown

Shutdown proceeds in this order:

1. Mark the service as stopping.
2. Stop accepting TCP connections and UDP datagrams.
3. Cancel and close active TCP connections.
4. Reject new Data Engine and quarantine writes.
5. Close stream queues and drain accepted records.
6. Flush, synchronize as configured, and close all files.
7. Wait for all owned goroutines to exit.

Every long-running component accepts a `context.Context` or exposes an explicit
`Close` / `Shutdown` operation. Shutdown operations must be idempotent.

The default global shutdown deadline is 30 seconds and is configurable from one
second to five minutes. On expiry, the server logs the number of records not
drained, closes remaining resources, and returns an error.

## 11. Error Handling

Expected network, parsing, configuration, and file errors are returned or
logged; they must not panic the process. Panics at goroutine boundaries should
be recovered only to log context and initiate controlled service shutdown, not
to silently restart a corrupted component.

Errors should include operation and identity context while preserving the
underlying error for `errors.Is` and `errors.As` .

## 12. Logging and Metrics

Logging supports DEBUG, INFO, WARN, ERROR, and disabled levels. Output may be
stdout/stderr or an owned file. Library code returns fatal errors to the process
orchestrator instead of calling `os.Exit` .

Logs should include structured or consistently formatted context such as:

* Transport and remote address.
* Connection ID and MAC address.
* Area and sensor type.
* Message type and rejection reason.
* File path and storage operation.

Repeated malformed input, unknown MAC, queue-full, and storage errors must be
rate-limited so a device cannot flood logs. Log level changes must be safe while
other goroutines are logging.

The server maintains concurrency-safe internal counters for:

* Active and total TCP connections.
* UDP datagrams and TCP messages received.
* Sensor records accepted and written.
* Rejections by reason, including checksum failures and unknown sensors.
* Unknown-device messages quarantined.
* Queue depth and dropped records by stream.
* File retries and terminal write failures.
* Bytes written, rotations, and replay progress.

Counters have no external HTTP or Prometheus endpoint. Components expose
in-process snapshot methods for diagnostics and tests. Counter snapshots are
logged at INFO level during graceful shutdown.

## 13. Code Organization

The initial package organization is:

* `config.go`: configuration schema, validation, immutable snapshots, reload.
* `protocol.go`: header and payload decoding, validation, CRC-32.
* `tcp-server.go`: listener and TCP connection lifecycle.
* `udp-server.go`: UDP receive loop and datagram processing.
* `data-engine.go`: stream registry and routing API.
* `data-stream.go`: queue, writer, binary records, flush, and rotation.
* `quarantine.go`: unknown-device log and replay.
* `server.go`: process lifecycle and component orchestration.

Constructors accept option structures when a component has meaningful optional
behavior. Required dependencies are explicit constructor parameters. A
constructor validates its inputs and returns an error. The design does not
require both `NewXOptions` and `NewDefaultXOptions` for every type when a useful
zero value or simpler constructor is clearer.

Dependencies such as logging, clocks, and file operations should be injectable
at the narrow boundary where tests need deterministic behavior. Avoid mutable
package globals.

## 14. Verification Strategy

Implementation is complete only with tests covering:

* Golden protocol vectors shared with ESP32 firmware.
* TCP headers and payloads split across reads and multiple messages coalesced in
  one read.
* Exact UDP datagram boundaries.
* Bad magic, lengths, checksums, message types, sensor IDs, and MAC changes.
* Configuration validation and concurrent atomic reload.
* Concurrent creation of one stream for the same area and sensor type.
* Consolidation from multiple devices into one stream.
* Queue saturation, bounded retries, and isolated stream failure.
* File rotation and incomplete-tail recovery.
* Unknown-MAC quarantine and interrupted replay.
* Graceful shutdown with accepted records draining.
* At least 100 mixed TCP/UDP senders under the Go race detector.
* Fuzz testing of protocol and configuration decoders.

## 15. Resolved Decision Checklist

* [X] Configuration reload trigger.
* [X] Message type values and unknown-type behavior.
* [X] Maximum accepted payload size.
* [X] Empty sensor payload behavior.
* [X] Timestamp assignment point.
* [X] TCP MAC mismatch behavior.
* [X] Duplicate TCP connection behavior.
* [X] TCP connection and payload deadline policy.
* [X] Quarantine replay trigger.
* [X] Replay delivery guarantee.
* [X] Processed quarantine retention.
* [X] Cross-source ordering requirement.
* [X] Queue and retry values.
* [X] Segment naming.
* [X] Buffering and durability policy.
* [X] File permissions.
* [X] Shutdown deadline.
* [X] Internal counters with no external metrics endpoint.
