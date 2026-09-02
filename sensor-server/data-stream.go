package sensorserver

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrDataStreamClosed = errors.New("data stream is closed")
	ErrDataStreamFull   = errors.New("data stream queue is full")
)

type SensorData struct {
	Timestamp int64
	Value     int32
}

type RecordWriter interface {
	WriteRecord(timestamp int64, value int32) error
	Flush() error
	Close() error
}

type DataStreamOptions struct {
	QueueCapacity  int
	EnqueueWait    time.Duration
	FlushInterval  time.Duration
	RetryCount     int
	InitialBackoff time.Duration
	MaximumBackoff time.Duration
	Retryable      func(error) bool
	OnError        func(error)
}

type DataStreamCounters struct {
	Accepted     uint64
	Written      uint64
	Dropped      uint64
	Retries      uint64
	Errors       uint64
	QueueDepth   int
	BytesWritten uint64
	Rotations    uint64
}

type DataStream struct {
	writer  RecordWriter
	clock   Clock
	options DataStreamOptions
	queue   chan SensorData
	done    chan struct{}

	closeOnce sync.Once
	submitMu  sync.RWMutex
	closed    atomic.Bool
	errMu     sync.Mutex
	closeErr  error

	accepted atomic.Uint64
	written  atomic.Uint64
	dropped  atomic.Uint64
	retries  atomic.Uint64
	errors   atomic.Uint64
}

func NewDataStream(writer RecordWriter, clock Clock, options DataStreamOptions) (*DataStream, error) {
	if writer == nil {
		return nil, errors.New("new data stream: nil writer")
	}
	if clock == nil {
		return nil, errors.New("new data stream: nil clock")
	}
	if options.QueueCapacity < 1 || options.EnqueueWait < 0 || options.FlushInterval <= 0 {
		return nil, errors.New("new data stream: invalid queue or flush settings")
	}
	if options.RetryCount < 0 || options.InitialBackoff <= 0 || options.MaximumBackoff < options.InitialBackoff {
		return nil, errors.New("new data stream: invalid retry settings")
	}
	if options.Retryable == nil {
		options.Retryable = func(error) bool { return false }
	}
	if options.OnError == nil {
		options.OnError = func(error) {}
	}

	stream := &DataStream{
		writer:  writer,
		clock:   clock,
		options: options,
		queue:   make(chan SensorData, options.QueueCapacity),
		done:    make(chan struct{}),
	}
	go stream.run()
	return stream, nil
}

func (stream *DataStream) Write(ctx context.Context, data SensorData) error {
	stream.submitMu.RLock()
	defer stream.submitMu.RUnlock()
	if stream.closed.Load() {
		stream.dropped.Add(1)
		return ErrDataStreamClosed
	}

	timer := stream.clock.NewTimer(stream.options.EnqueueWait)
	defer timer.Stop()
	select {
	case stream.queue <- data:
		stream.accepted.Add(1)
		return nil
	case <-ctx.Done():
		stream.dropped.Add(1)
		return ctx.Err()
	case <-timer.C():
		stream.dropped.Add(1)
		return ErrDataStreamFull
	}
}

func (stream *DataStream) Close() error {
	stream.closeOnce.Do(func() {
		stream.submitMu.Lock()
		stream.closed.Store(true)
		close(stream.queue)
		stream.submitMu.Unlock()
	})
	<-stream.done
	stream.errMu.Lock()
	defer stream.errMu.Unlock()
	return stream.closeErr
}

func (stream *DataStream) Counters() DataStreamCounters {
	counters := DataStreamCounters{
		Accepted:   stream.accepted.Load(),
		Written:    stream.written.Load(),
		Dropped:    stream.dropped.Load(),
		Retries:    stream.retries.Load(),
		Errors:     stream.errors.Load(),
		QueueDepth: len(stream.queue),
	}
	if source, ok := stream.writer.(interface{ Counters() DataFileCounters }); ok {
		fileCounters := source.Counters()
		counters.BytesWritten = fileCounters.BytesWritten
		counters.Rotations = fileCounters.Rotations
	}
	return counters
}

func (stream *DataStream) run() {
	ticker := stream.clock.NewTicker(stream.options.FlushInterval)
	defer ticker.Stop()
	defer close(stream.done)

	dirty := false
	for {
		select {
		case data, ok := <-stream.queue:
			if !ok {
				stream.setCloseError(errors.Join(stream.flushIfDirty(dirty), stream.writer.Close()))
				return
			}
			if err := stream.writeWithRetry(data); err != nil {
				stream.errors.Add(1)
				stream.dropped.Add(1)
				stream.options.OnError(err)
				continue
			}
			stream.written.Add(1)
			dirty = true
		case <-ticker.C():
			if !dirty {
				continue
			}
			if err := stream.writer.Flush(); err != nil {
				stream.errors.Add(1)
				stream.options.OnError(fmt.Errorf("periodic flush: %w", err))
				continue
			}
			dirty = false
		}
	}
}

func (stream *DataStream) writeWithRetry(data SensorData) error {
	backoff := stream.options.InitialBackoff
	for attempt := 0; ; attempt++ {
		err := stream.writer.WriteRecord(data.Timestamp, data.Value)
		if err == nil {
			return nil
		}
		if attempt >= stream.options.RetryCount || !stream.options.Retryable(err) {
			return fmt.Errorf("write sensor record after %d retries: %w", attempt, err)
		}
		stream.retries.Add(1)
		timer := stream.clock.NewTimer(backoff)
		<-timer.C()
		timer.Stop()
		backoff *= 2
		if backoff > stream.options.MaximumBackoff {
			backoff = stream.options.MaximumBackoff
		}
	}
}

func (stream *DataStream) flushIfDirty(dirty bool) error {
	if !dirty {
		return nil
	}
	if err := stream.writer.Flush(); err != nil {
		stream.errors.Add(1)
		return fmt.Errorf("final flush: %w", err)
	}
	return nil
}

func (stream *DataStream) setCloseError(err error) {
	stream.errMu.Lock()
	stream.closeErr = err
	stream.errMu.Unlock()
}
