package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Logger is a component that underneath has N number of log.Logger instances.
// Here we have different log levels where each level has its own log.Logger instance.

type LogLevel int

const (
	LevelNone LogLevel = iota
	LevelFatal
	LevelError
	LevelWarning
	LevelInfo
)

type Logger interface {
	SetLevel(level LogLevel)

	LogFatal(err error, v ...any)
	LogError(err error, v ...any)
	LogWarning(v ...any)
	LogInfo(v ...any)

	LogFatalf(err error, format string, v ...any)
	LogErrorf(err error, format string, v ...any)
	LogWarningf(format string, v ...any)
	LogInfof(format string, v ...any)
}

type DefaultLogger struct {
	ogLogLevel    LogLevel
	ogFatalLogger *StdLogger
	ogErrLogger   *StdLogger
	ogWarnLogger  *StdLogger
	ogInfoLogger  *StdLogger
}

func NewDefault() Logger {
	return &DefaultLogger{
		ogLogLevel:    LevelInfo,
		ogFatalLogger: NewStdLogger(os.Stderr, "FATAL: ", log.Ldate|log.Ltime|log.Lshortfile),
		ogErrLogger:   NewStdLogger(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile),
		ogWarnLogger:  NewStdLogger(os.Stdout, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile),
		ogInfoLogger:  NewStdLogger(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

func (l *DefaultLogger) SetLevel(level LogLevel) {
	l.ogLogLevel = level
}

func (l *DefaultLogger) LogFatal(err error, v ...any) {
	if LevelFatal <= l.ogLogLevel {
		l.ogFatalLogger.Print(err)
		l.ogFatalLogger.Println(v...)
	}
}

func (l *DefaultLogger) LogError(err error, v ...any) {
	if LevelError <= l.ogLogLevel {
		l.ogErrLogger.Print(err)
		l.ogErrLogger.Println(v...)
	}
}

func (l *DefaultLogger) LogWarning(v ...any) {
	if LevelWarning <= l.ogLogLevel {
		l.ogWarnLogger.Println(v...)
	}
}

func (l *DefaultLogger) LogInfo(v ...any) {
	if LevelInfo <= l.ogLogLevel {
		l.ogInfoLogger.Print(v...)
	}
}

func (l *DefaultLogger) LogFatalf(err error, format string, v ...any) {
	if LevelFatal <= l.ogLogLevel {
		l.ogFatalLogger.Print(err)
		l.ogFatalLogger.Printf(format, v...)
	}
}

func (l *DefaultLogger) LogErrorf(err error, format string, v ...any) {
	if LevelError <= l.ogLogLevel {
		l.ogErrLogger.Print(err)
		l.ogErrLogger.Printf(format, v...)
	}
}

func (l *DefaultLogger) LogWarningf(format string, v ...any) {
	if LevelWarning <= l.ogLogLevel {
		l.ogWarnLogger.Printf(format, v...)
	}
}

func (l *DefaultLogger) LogInfof(format string, v ...any) {
	if LevelInfo <= l.ogLogLevel {
		l.ogInfoLogger.Printf(format, v...)
		//l.ogInfoLogger.Println()
	}
}

// ---------------------------------------------------------------------------------
// standard logger has a hardcoded stack depth of 2, we need to change this

// A Logger represents an active logging object that generates lines of
// output to an [io.Writer]. Each logging operation makes a single call to
// the Writer's Write method. A Logger can be used simultaneously from
// multiple goroutines; it guarantees to serialize access to the Writer.
type StdLogger struct {
	outMu sync.Mutex
	out   io.Writer // destination for output

	depth     int                    // stack depth to recover file/line info
	prefix    atomic.Pointer[string] // prefix on each line to identify the logger (but see Lmsgprefix)
	flag      atomic.Int32           // properties
	isDiscard atomic.Bool
}

// New creates a new [Logger]. The out variable sets the
// destination to which log data will be written.
// The prefix appears at the beginning of each generated log line, or
// after the log header if the [Lmsgprefix] flag is provided.
// The flag argument defines the logging properties.
func NewStdLogger(out io.Writer, prefix string, flag int) *StdLogger {
	l := new(StdLogger)
	l.depth = 3 // stack depth to recover file/line info
	l.SetOutput(out)
	l.SetPrefix(prefix)
	l.SetFlags(flag)
	return l
}

// SetOutput sets the output destination for the logger.
func (l *StdLogger) SetOutput(w io.Writer) {
	l.outMu.Lock()
	defer l.outMu.Unlock()
	l.out = w
	l.isDiscard.Store(w == io.Discard)
}

var std = NewStdLogger(os.Stderr, "", log.LstdFlags)

// Default returns the standard logger used by the package-level output functions.
func Default() *StdLogger { return std }

// Cheap integer to fixed-width decimal ASCII. Give a negative width to avoid zero-padding.
func itoa(buf *[]byte, i int, wid int) {
	// Assemble decimal in reverse order.
	var b [20]byte
	bp := len(b) - 1
	for i >= 10 || wid > 1 {
		wid--
		q := i / 10
		b[bp] = byte('0' + i - q*10)
		bp--
		i = q
	}
	// i < 10
	b[bp] = byte('0' + i)
	*buf = append(*buf, b[bp:]...)
}

// formatHeader writes log header to buf in following order:
//   - l.prefix (if it's not blank and Lmsgprefix is unset),
//   - date and/or time (if corresponding flags are provided),
//   - file and line number (if corresponding flags are provided),
//   - l.prefix (if it's not blank and Lmsgprefix is set).
func formatHeader(buf *[]byte, t time.Time, prefix string, flag int, file string, line int) {
	if flag&log.Lmsgprefix == 0 {
		*buf = append(*buf, prefix...)
	}
	if flag&(log.Ldate|log.Ltime|log.Lmicroseconds) != 0 {
		if flag&log.LUTC != 0 {
			t = t.UTC()
		}
		if flag&log.Ldate != 0 {
			year, month, day := t.Date()
			itoa(buf, year, 4)
			*buf = append(*buf, '/')
			itoa(buf, int(month), 2)
			*buf = append(*buf, '/')
			itoa(buf, day, 2)
			*buf = append(*buf, ' ')
		}
		if flag&(log.Ltime|log.Lmicroseconds) != 0 {
			hour, min, sec := t.Clock()
			itoa(buf, hour, 2)
			*buf = append(*buf, ':')
			itoa(buf, min, 2)
			*buf = append(*buf, ':')
			itoa(buf, sec, 2)
			if flag&log.Lmicroseconds != 0 {
				*buf = append(*buf, '.')
				itoa(buf, t.Nanosecond()/1e3, 6)
			}
			*buf = append(*buf, ' ')
		}
	}
	if flag&(log.Lshortfile|log.Llongfile) != 0 {
		if flag&log.Lshortfile != 0 {
			short := file
			for i := len(file) - 1; i > 0; i-- {
				if file[i] == '/' {
					short = file[i+1:]
					break
				}
			}
			file = short
		}
		*buf = append(*buf, file...)
		*buf = append(*buf, ':')
		itoa(buf, line, -1)
		*buf = append(*buf, ": "...)
	}
	if flag&log.Lmsgprefix != 0 {
		*buf = append(*buf, prefix...)
	}
}

var bufferPool = sync.Pool{New: func() any { return new([]byte) }}

func getBuffer() *[]byte {
	p := bufferPool.Get().(*[]byte)
	*p = (*p)[:0]
	return p
}

func putBuffer(p *[]byte) {
	// Proper usage of a sync.Pool requires each entry to have approximately
	// the same memory cost. To obtain this property when the stored type
	// contains a variably-sized buffer, we add a hard limit on the maximum buffer
	// to place back in the pool.
	//
	// See https://go.dev/issue/23199
	if cap(*p) > 64<<10 {
		*p = nil
	}
	bufferPool.Put(p)
}

// Output writes the output for a logging event. The string s contains
// the text to print after the prefix specified by the flags of the
// Logger. A newline is appended if the last character of s is not
// already a newline. Calldepth is used to recover the PC and is
// provided for generality, although at the moment on all pre-defined
// paths it will be 2.
func (l *StdLogger) Output(calldepth int, s string) error {
	calldepth++ // +1 for this frame.
	return l.output(0, calldepth, func(b []byte) []byte {
		return append(b, s...)
	})
}

// output can take either a calldepth or a pc to get source line information.
// It uses the pc if it is non-zero.
func (l *StdLogger) output(pc uintptr, calldepth int, appendOutput func([]byte) []byte) error {
	if l.isDiscard.Load() {
		return nil
	}

	now := time.Now() // get this early.

	// Load prefix and flag once so that their value is consistent within
	// this call regardless of any concurrent changes to their value.
	prefix := l.Prefix()
	flag := l.Flags()

	var file string
	var line int
	if flag&(log.Lshortfile|log.Llongfile) != 0 {
		if pc == 0 {
			var ok bool
			_, file, line, ok = runtime.Caller(calldepth)
			if !ok {
				file = "???"
				line = 0
			}
		} else {
			fs := runtime.CallersFrames([]uintptr{pc})
			f, _ := fs.Next()
			file = f.File
			if file == "" {
				file = "???"
			}
			line = f.Line
		}
	}

	buf := getBuffer()
	defer putBuffer(buf)
	formatHeader(buf, now, prefix, flag, file, line)
	*buf = appendOutput(*buf)
	if len(*buf) == 0 || (*buf)[len(*buf)-1] != '\n' {
		*buf = append(*buf, '\n')
	}

	l.outMu.Lock()
	defer l.outMu.Unlock()
	_, err := l.out.Write(*buf)
	return err
}

// Print calls l.Output to print to the logger.
// Arguments are handled in the manner of [fmt.Print].
func (l *StdLogger) Print(v ...any) {
	l.output(0, l.depth, func(b []byte) []byte {
		return fmt.Append(b, v...)
	})
}

// Printf calls l.Output to print to the logger.
// Arguments are handled in the manner of [fmt.Printf].
func (l *StdLogger) Printf(format string, v ...any) {
	l.output(0, l.depth, func(b []byte) []byte {
		return fmt.Appendf(b, format, v...)
	})
}

// Println calls l.Output to print to the logger.
// Arguments are handled in the manner of [fmt.Println].
func (l *StdLogger) Println(v ...any) {
	l.output(0, l.depth, func(b []byte) []byte {
		return fmt.Appendln(b, v...)
	})
}

// Fatal is equivalent to l.Print() followed by a call to [os.Exit](1).
func (l *StdLogger) Fatal(v ...any) {
	l.Output(l.depth, fmt.Sprint(v...))
	os.Exit(1)
}

// Fatalf is equivalent to l.Printf() followed by a call to [os.Exit](1).
func (l *StdLogger) Fatalf(format string, v ...any) {
	l.Output(l.depth, fmt.Sprintf(format, v...))
	os.Exit(1)
}

// Fatalln is equivalent to l.Println() followed by a call to [os.Exit](1).
func (l *StdLogger) Fatalln(v ...any) {
	l.Output(l.depth, fmt.Sprintln(v...))
	os.Exit(1)
}

// Panic is equivalent to l.Print() followed by a call to panic().
func (l *StdLogger) Panic(v ...any) {
	s := fmt.Sprint(v...)
	l.Output(l.depth, s)
	panic(s)
}

// Panicf is equivalent to l.Printf() followed by a call to panic().
func (l *StdLogger) Panicf(format string, v ...any) {
	s := fmt.Sprintf(format, v...)
	l.Output(l.depth, s)
	panic(s)
}

// Panicln is equivalent to l.Println() followed by a call to panic().
func (l *StdLogger) Panicln(v ...any) {
	s := fmt.Sprintln(v...)
	l.Output(l.depth, s)
	panic(s)
}

// Flags returns the output flags for the logger.
// The flag bits are [Ldate], [Ltime], and so on.
func (l *StdLogger) Flags() int {
	return int(l.flag.Load())
}

// SetFlags sets the output flags for the logger.
// The flag bits are [Ldate], [Ltime], and so on.
func (l *StdLogger) SetFlags(flag int) {
	l.flag.Store(int32(flag))
}

// Prefix returns the output prefix for the logger.
func (l *StdLogger) Prefix() string {
	if p := l.prefix.Load(); p != nil {
		return *p
	}
	return ""
}

// SetPrefix sets the output prefix for the logger.
func (l *StdLogger) SetPrefix(prefix string) {
	l.prefix.Store(&prefix)
}

// Writer returns the output destination for the logger.
func (l *StdLogger) Writer() io.Writer {
	l.outMu.Lock()
	defer l.outMu.Unlock()
	return l.out
}

// SetOutput sets the output destination for the standard logger.
func SetOutput(w io.Writer) {
	std.SetOutput(w)
}

// Flags returns the output flags for the standard logger.
// The flag bits are [Ldate], [Ltime], and so on.
func Flags() int {
	return std.Flags()
}

// SetFlags sets the output flags for the standard logger.
// The flag bits are [Ldate], [Ltime], and so on.
func SetFlags(flag int) {
	std.SetFlags(flag)
}

// Prefix returns the output prefix for the standard logger.
func Prefix() string {
	return std.Prefix()
}

// SetPrefix sets the output prefix for the standard logger.
func SetPrefix(prefix string) {
	std.SetPrefix(prefix)
}

// Writer returns the output destination for the standard logger.
func Writer() io.Writer {
	return std.Writer()
}

// These functions write to the standard logger.

// Print calls Output to print to the standard logger.
// Arguments are handled in the manner of [fmt.Print].
func Print(v ...any) {
	std.output(0, 3, func(b []byte) []byte {
		return fmt.Append(b, v...)
	})
}

// Printf calls Output to print to the standard logger.
// Arguments are handled in the manner of [fmt.Printf].
func Printf(format string, v ...any) {
	std.output(0, 3, func(b []byte) []byte {
		return fmt.Appendf(b, format, v...)
	})
}

// Println calls Output to print to the standard logger.
// Arguments are handled in the manner of [fmt.Println].
func Println(v ...any) {
	std.output(0, 3, func(b []byte) []byte {
		return fmt.Appendln(b, v...)
	})
}

// Fatal is equivalent to [Print] followed by a call to [os.Exit](1).
func Fatal(v ...any) {
	std.Output(3, fmt.Sprint(v...))
	os.Exit(1)
}

// Fatalf is equivalent to [Printf] followed by a call to [os.Exit](1).
func Fatalf(format string, v ...any) {
	std.Output(3, fmt.Sprintf(format, v...))
	os.Exit(1)
}

// Fatalln is equivalent to [Println] followed by a call to [os.Exit](1).
func Fatalln(v ...any) {
	std.Output(3, fmt.Sprintln(v...))
	os.Exit(1)
}

// Panic is equivalent to [Print] followed by a call to panic().
func Panic(v ...any) {
	s := fmt.Sprint(v...)
	std.Output(3, s)
	panic(s)
}

// Panicf is equivalent to [Printf] followed by a call to panic().
func Panicf(format string, v ...any) {
	s := fmt.Sprintf(format, v...)
	std.Output(3, s)
	panic(s)
}

// Panicln is equivalent to [Println] followed by a call to panic().
func Panicln(v ...any) {
	s := fmt.Sprintln(v...)
	std.Output(3, s)
	panic(s)
}

// Output writes the output for a logging event. The string s contains
// the text to print after the prefix specified by the flags of the
// Logger. A newline is appended if the last character of s is not
// already a newline. Calldepth is the count of the number of
// frames to skip when computing the file name and line number
// if [Llongfile] or [Lshortfile] is set; a value of 1 will print the details
// for the caller of Output.
func Output(calldepth int, s string) error {
	return std.Output(calldepth+1, s) // +1 for this frame.
}
