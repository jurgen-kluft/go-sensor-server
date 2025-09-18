# Sensor Server

This is the README for Sensor Server, a tcp server for processing sensor data in real-time and writing them to disk in an structured format.

Some notes:

- Data is organized per day, per sensor and per sensor group.
- Data is very minimal and it should be quite easy to load/use it in a visualization tool
- Snappy is used upon writing, and reading, data to/from disk.
