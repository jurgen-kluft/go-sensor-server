# Sensor Server

A udp/tcp server for receiving sensor data in real-time and flushing them to disk in a custom structured format.

Some notes:

- Every sensor is a data stream
- Data is very minimal and it should be quite easy to load/use it in a visualization tool
- TODO: Snappy is used upon writing, and reading data to/from disk
- One or more local clients can connect through unix domain sockets to receive real-time data. 
  The data is a simple array of the following format:
  - `{uint16 Sensor Index, uint16 Sensor Value}`

## Status

- Implemention is quite done
  - ipc, udp and tcp server 
  - updating sensor streams with incoming sensor data
  - flushing sensor streams to disk
  Status: Beta version, needs testing.

## Sensor Stream Format

- 64 bytes header
  	- uint16, Year
    - uint8, Month
    - uint8, Day
    - uint64, Full Sensor Identifier
    - uint64, Flags (compressed or not)
    - uint32, Data Length (length of data in bytes)
- Data (variable length, depends on SampleType and SampleFreq)

## Data Size

Example for a Temperature sensor:

Storing the temperature every minute for a year with 1 signed byte would require:
```
    1 byte * 1 (time per minute) * 60 (minutes per hour) * 24 (hours per day) * 365 (days per year) = 525,600 bytes = ~513 KB.
    Note: Applying (snappy) compression would reduce this to about ~100 KB.
```
Note: That is ~100KB for one year, for one sensor.

Example for a Motion sensor:

Storing the motion state twice a second for a year with 2 bits would require:
```
    2 bits * 2 (times per second) * 60 (seconds per minute) * 60 (minutes per hour) * 24 (hours per day) * 365 (days per year) = 63,072,000 bits = ~7.5 MB.
    Note: Since there will be mostly large runs of 0 (or 1) bits, applying (snappy) compression would reduce this to about ~300 KB.
```

Note: That is ~300KB for one year, for one motion sensor
