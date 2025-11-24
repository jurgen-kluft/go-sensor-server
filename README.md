# Sensor Server

A udp/tcp server for receiving sensor data in real-time and flushing them to disk in a custom structured format.
We also have a UDP discovery server for devices to find the server on the local network.

Some notes:

- Data is very minimal and it should be quite easy to load/use it in a visualization tool or otherwise
  preprocess the data to convert it to a more convenient format.
- One or more (local) clients can connect through unix domain sockets to receive real-time home state. 
  Home state is a simple array of the following format:
  - `{byte[6] Mac, byte[6] TimeStamp, uint16 Sensor Index, uint16 Sensor Value}`

## Status

- Implemention is WIP
  - ipc, udp and tcp server 
  - updating sensor append-only log with incoming sensor data
  - update home state with incoming sensor data
  - flushing home state to disk every N minutes
  Status: Beta version, needs testing.

## Sensor Log Format

- 16 bytes header
- Mac Address (6 bytes) + Timestamp (4 bytes) + Sensor Index (2 bytes) + Sensor Value (2 bytes) + Padding (2 bytes)
- Occasional TimePoint: FF FF FF FF FF FF (6 bytes) + Timestamp (8 bytes) + Padding (2 bytes)

