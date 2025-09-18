# Sensor Server

This is the README for Sensor Server, a tcp server for processing sensor data in real-time and writing them to disk in an structured format.

Some notes:

- Every sensor is a data stream that is 'per day', and falls under a sensor group which is a device.
- Data is very minimal and it should be quite easy to load/use it in a visualization tool
- Snappy is used upon writing, and reading, data to/from disk.

## Data Format

Example for a Tempature sensor:

Storing the temperature every minute for a year with 1 signed byte would require:
```
    1 byte * 1 (times per minute) * 60 (minutes per hour) * 24 (hours per day) * 365 (days per year) = 525,600 bytes = ~513 KB.
    Note: Applying (snappy) compression would reduce this to about ~100 KB.
```
That is for ONE YEAR, for ONE SENSOR, so as you can see this is quite efficient.


Example for a Motion detection:

Storing the motion state twice a second for a year with 1 bit would require:
```
    1 bit * 2 (times per second) * 60 (seconds per minute) * 60 (minutes per hour) * 24 (hours per day) * 365 (days per year) = 63,072,000 bits = ~7.5 MB.
    Note: Since there will be mostly large runs of 0 bits, applying (snappy) compression would reduce this to about ~300 KB.
```

