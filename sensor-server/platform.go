package sensorserver

import (
	"io"
	"io/fs"
	"net"
	"os"
	"time"
)

const (
	readOnlyFlags                = os.O_RDONLY
	readWriteFlags               = os.O_RDWR
	createAppendReadWriteFlags   = os.O_CREATE | os.O_APPEND | os.O_RDWR
	createTruncateReadWriteFlags = os.O_CREATE | os.O_TRUNC | os.O_RDWR
)

// File is the storage handle used by data streams and quarantine logs.
type File interface {
	io.Reader
	io.Writer
	io.Closer
	Stat() (fs.FileInfo, error)
	Sync() error
	Truncate(size int64) error
}

// FileSystem contains the filesystem operations required by storage components.
type FileSystem interface {
	OpenFile(name string, flag int, perm fs.FileMode) (File, error)
	MkdirAll(path string, perm fs.FileMode) error
	ReadDir(name string) ([]fs.DirEntry, error)
	Stat(name string) (fs.FileInfo, error)
	Rename(oldPath, newPath string) error
	Remove(name string) error
}

// OSFileSystem delegates filesystem operations to the os package.
type OSFileSystem struct{}

func (OSFileSystem) OpenFile(name string, flag int, perm fs.FileMode) (File, error) {
	return os.OpenFile(name, flag, perm)
}

func (OSFileSystem) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (OSFileSystem) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(name)
}

func (OSFileSystem) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

func (OSFileSystem) Rename(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

func (OSFileSystem) Remove(name string) error {
	return os.Remove(name)
}

// Network creates standard library connection interfaces. Tests can inject
// net.Pipe connections, fake packet connections, or loopback listeners.
type Network interface {
	Listen(network, address string) (net.Listener, error)
	ListenPacket(network, address string) (net.PacketConn, error)
}

// StandardNetwork delegates listener creation to the net package.
type StandardNetwork struct{}

func (StandardNetwork) Listen(network, address string) (net.Listener, error) {
	return net.Listen(network, address)
}

func (StandardNetwork) ListenPacket(network, address string) (net.PacketConn, error) {
	return net.ListenPacket(network, address)
}

// Timer is the subset of time.Timer needed by retry and deadline handling.
type Timer interface {
	C() <-chan time.Time
	Stop() bool
	Reset(duration time.Duration) bool
}

// Ticker is the subset of time.Ticker needed by periodic work.
type Ticker interface {
	C() <-chan time.Time
	Stop()
}

// Clock allows timestamps and scheduled work to be deterministic in tests.
type Clock interface {
	Now() time.Time
	NewTimer(duration time.Duration) Timer
	NewTicker(duration time.Duration) Ticker
}

// SystemClock delegates time operations to the time package.
type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now()
}

func (SystemClock) NewTimer(duration time.Duration) Timer {
	return systemTimer{timer: time.NewTimer(duration)}
}

func (SystemClock) NewTicker(duration time.Duration) Ticker {
	return systemTicker{ticker: time.NewTicker(duration)}
}

type systemTimer struct {
	timer *time.Timer
}

func (timer systemTimer) C() <-chan time.Time {
	return timer.timer.C
}

func (timer systemTimer) Stop() bool {
	return timer.timer.Stop()
}

func (timer systemTimer) Reset(duration time.Duration) bool {
	return timer.timer.Reset(duration)
}

type systemTicker struct {
	ticker *time.Ticker
}

func (ticker systemTicker) C() <-chan time.Time {
	return ticker.ticker.C
}

func (ticker systemTicker) Stop() {
	ticker.ticker.Stop()
}

var (
	_ FileSystem = OSFileSystem{}
	_ Network    = StandardNetwork{}
	_ Clock      = SystemClock{}
)
