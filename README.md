# Sensor Server

A tcp server for receiving sensor data in real-time and writing them to disk in an custom structured format.

Some notes:

- Every sensor is a data stream, and falls under a sensor group which is a device.
- Data is very minimal and it should be quite easy to load/use it in a visualization tool
- Snappy is used upon writing, and reading data to/from disk

## Status

- Implemention is quite done, there is a simple tcp-server that writes the sensor packets to disk.
  Status: Beta version, needs testing.

## Data Format

Example for a Temperature sensor:

Storing the temperature every minute for a year with 1 signed byte would require:
```
    1 byte * 1 (time per minute) * 60 (minutes per hour) * 24 (hours per day) * 365 (days per year) = 525,600 bytes = ~513 KB.
    Note: Applying (snappy) compression would reduce this to about ~100 KB.
```
Note: That is ~100KB for one year, for one sensor.

Example for a Motion sensor:

Storing the motion state twice a second for a year with 1 bit would require:
```
    1 bit * 2 (times per second) * 60 (seconds per minute) * 60 (minutes per hour) * 24 (hours per day) * 365 (days per year) = 63,072,000 bits = ~7.5 MB.
    Note: Since there will be mostly large runs of 0 bits, applying (snappy) compression would reduce this to about ~300 KB.
```

Note: That is ~300KB for one year, for one motion sensor
