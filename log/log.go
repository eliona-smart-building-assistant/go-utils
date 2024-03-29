//  This file is part of the eliona project.
//  Copyright © 2022 LEICOM iTEC AG. All Rights Reserved.
//  ______ _ _
// |  ____| (_)
// | |__  | |_  ___  _ __   __ _
// |  __| | | |/ _ \| '_ \ / _` |
// | |____| | | (_) | | | | (_| |
// |______|_|_|\___/|_| |_|\__,_|
//
//  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING
//  BUT NOT LIMITED  TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
//  NON INFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM,
//  DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
//  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package log

import (
	"bytes"
	"fmt"
	"github.com/eliona-smart-building-assistant/go-utils/common"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Level type
type Level uint32

// These are the different logging levels. You can set the logging level to log
// on your instance of logger.
const (
	// FatalLevel level. Fatal errors which make it necessary to abort.
	FatalLevel Level = iota
	// ErrorLevel level. Used for errors that should definitely be noted.
	ErrorLevel
	// WarnLevel level. Non-critical entries that deserve eyes.
	WarnLevel
	// InfoLevel level. General operational entries about what's going on inside the application.
	InfoLevel
	// DebugLevel level. Output information necessary for debugging. Verbose logging.
	DebugLevel
	// TraceLevel level. Enabled when everything should be outputted. Maximum detailed logging.
	TraceLevel
)

// Convert the Level to a string. E.g. ErrorLevel becomes "ERROR".
func (level Level) String() string {
	if b, err := level.MarshalText(); err == nil {
		return string(b)
	} else {
		return "null"
	}
}

func (level Level) MarshalText() ([]byte, error) {
	switch level {
	case TraceLevel:
		return []byte("TRACE"), nil
	case DebugLevel:
		return []byte("DEBUG"), nil
	case InfoLevel:
		return []byte("INFO"), nil
	case WarnLevel:
		return []byte("WARNING"), nil
	case ErrorLevel:
		return []byte("ERROR"), nil
	case FatalLevel:
		return []byte("FATAL"), nil
	}

	return nil, fmt.Errorf("not a valid logit level %d", level)
}

// parseLevel takes a string level and returns log level constant. If unable to parse the string, debug level is returned
func parseLevel(lvl string) Level {
	switch strings.ToLower(lvl) {
	case "fatal":
		return FatalLevel
	case "error":
		return ErrorLevel
	case "warn", "warning":
		return WarnLevel
	case "info":
		return InfoLevel
	case "trace":
		return TraceLevel
	default:
		return DebugLevel
	}
}

// A Logger represents an active logging object that generates lines of
// output to an io.Writer. Each logging operation makes a single call to
// the Writer's Write method. A Logger can be used simultaneously from
// multiple goroutines; it guarantees to serialize access to the Writer.
type Logger struct {
	mu       sync.Mutex // ensures atomic writes; protects the following fields
	out      io.Writer  // destination for output
	buf      []byte     // for accumulating text to write
	bufLimit int64      // buflimit 0 means no limit
	lev      Level      // level for logging
}

// A logWriter implements the io.Writer interface and logs all data send to the writer line by line.
// The logWriter contains a Logger inside to log the data. Each logger needs a level and a tag used for logging.
type logWriter struct {
	log *Logger
	lev Level
	buf bytes.Buffer
	tag string
}

func (lw *logWriter) Write(p []byte) (n int, err error) {
	var count int
	lines := bytes.Split(p, []byte("\n"))
	for idx, line := range lines {
		lw.buf.Write(line)
		if lw.buf.Len() > 0 && idx < (len(lines)-1) {
			err := lw.log.Output(lw.lev, lw.tag, lw.buf.String())
			if err != nil {
				return 0, err
			}
			count += lw.buf.Len()
			lw.buf.Reset()
		}
	}
	return count, nil
}

// New creates a new Logger. The out variable sets the
// destination to which log data will be written. The log level is taken
// from LOG_LEVEL environment variable.
func New(out io.Writer) *Logger {
	var level = parseLevel(common.Getenv("LOG_LEVEL", "info"))
	return &Logger{out: out, lev: level}
}

// GetWriter returns an io.Writer implementation which log all data to the standard logger with log level and tag
func (l *Logger) GetWriter(level Level, tag string) io.Writer {
	return &logWriter{
		log: l,
		tag: tag,
		lev: level,
	}
}

func (l *Logger) level() Level {
	return Level(atomic.LoadUint32((*uint32)(&l.lev)))
}

// IsLevelEnabled checks if the log level of the logger is greater than the level param
func (l *Logger) IsLevelEnabled(level Level) bool {
	return l.level() >= level
}

// SetOutput sets the output destination for the logger.
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.out = w
}

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
//   * level
//   * date and time,
//   * tag.
// The values are separated with tab
func (l *Logger) formatHeader(buf *[]byte, t time.Time, level Level, tag string) {

	*buf = append(*buf, level.String()...)
	*buf = append(*buf, '\t')

	// "2006-01-02 15:04:05.000000"
	year, month, day := t.Date()
	itoa(buf, year, 4)
	*buf = append(*buf, '-')
	itoa(buf, int(month), 2)
	*buf = append(*buf, '-')
	itoa(buf, day, 2)
	*buf = append(*buf, ' ')

	hour, min, sec := t.Clock()
	itoa(buf, hour, 2)
	*buf = append(*buf, ':')
	itoa(buf, min, 2)
	*buf = append(*buf, ':')
	itoa(buf, sec, 2)
	*buf = append(*buf, '.')
	itoa(buf, t.Nanosecond()/1e3, 6)
	*buf = append(*buf, '\t')

	*buf = append(*buf, strings.ToUpper(tag)...)
	*buf = append(*buf, '\t')
}

// Output writes the output for a logging event. The string s contains
// the text to print after the tag specified by the flags of the
// Logger. A newline is appended if the last character of s is not
// already a newline. Output won't write more bytes than bufLimit (if it's set)
func (l *Logger) Output(level Level, tag, s string) error {
	return l.Timestamp(level, time.Now(), tag, s)
}

// Timestamp writes the output with an own timestamp for a logging event. The string s contains
// the text to print after the tag specified by the flags of the
// Logger. A newline is appended if the last character of s is not
// already a newline. Output won't write more bytes than bufLimit (if it's set)
func (l *Logger) Timestamp(level Level, timestamp time.Time, tag, s string) error {
	if !l.IsLevelEnabled(level) {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.buf = l.buf[:0]
	l.formatHeader(&l.buf, timestamp, level, tag)
	l.buf = append(l.buf, s...)
	// if limit is set truncate buf
	if l.bufLimit > 0 && int64(len(l.buf)) >= l.bufLimit {
		// bufLimit-1 so we will set to \n as last char
		l.buf = l.buf[:l.bufLimit-1]
	}
	if len(l.buf) == 0 || l.buf[len(l.buf)-1] != '\n' {
		l.buf = append(l.buf, '\n')
	}
	_, err := l.out.Write(l.buf)
	return err
}

// Printf calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Printf(level Level, tag, format string, v ...interface{}) {
	if l.IsLevelEnabled(level) {
		_ = l.Output(level, tag, fmt.Sprintf(format, v...))
	}
}

// TimestampPrintf calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) TimestampPrintf(level Level, timestamp time.Time, tag, format string, v ...interface{}) {
	if l.IsLevelEnabled(level) {
		_ = l.Timestamp(level, timestamp, tag, fmt.Sprintf(format, v...))
	}
}

// Print calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Print.
func (l *Logger) Print(level Level, tag string, v ...interface{}) {
	if l.IsLevelEnabled(level) {
		_ = l.Output(level, tag, fmt.Sprint(v...))
	}
}

// Println calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Println.
func (l *Logger) Println(level Level, tag string, v ...interface{}) {
	if l.IsLevelEnabled(level) {
		_ = l.Output(level, tag, fmt.Sprintln(v...))
	}
}

// Level returns logging level for Logger.
func (l *Logger) Level() Level {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.level()
}

// SetLevel sets log level.
func (l *Logger) SetLevel(level Level) {
	atomic.StoreUint32((*uint32)(&l.lev), uint32(level))
}

// BufLimit returns current bufLimit value.
func (l *Logger) BufLimit() int64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.bufLimit
}

// SetBufLimit sets bufLimit.
func (l *Logger) SetBufLimit(bufLimit int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.bufLimit = bufLimit
}

// Writer returns the output destination for the logger.
func (l *Logger) Writer() io.Writer {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.out
}

// Error calls Printf to print to the standard logger with the error level. As tag
// the app name is taken. Other arguments are handled in the manner of fmt.Printf.
func (l *Logger) Error(tag, format string, v ...interface{}) {
	l.Printf(ErrorLevel, tag, format, v...)
}

// Warn calls Printf to print to the standard logger with the warning level. As tag
// the app name is taken. Other arguments are handled in the manner of fmt.Printf.
func (l *Logger) Warn(tag, format string, v ...interface{}) {
	l.Printf(WarnLevel, tag, format, v...)
}

// Info calls Printf to print to the standard logger with the info level. As tag
// the app name is taken. Other arguments are handled in the manner of fmt.Printf.
func (l *Logger) Info(tag, format string, v ...interface{}) {
	l.Printf(InfoLevel, tag, format, v...)
}

// Debug calls Printf to print to the standard logger with the debug level. As tag
// the app name is taken. Other arguments are handled in the manner of fmt.Printf.
func (l *Logger) Debug(tag, format string, v ...interface{}) {
	l.Printf(DebugLevel, tag, format, v...)
}

// Trace calls Printf to print to the standard logger with the trace level. As tag
// the app name is taken. Other arguments are handled in the manner of fmt.Printf.
func (l *Logger) Trace(tag, format string, v ...interface{}) {
	l.Printf(TraceLevel, tag, format, v...)
}

// Fatal calls Printf to print to the standard logger with the debug level. As tag
// the app name is taken. Other arguments are handled in the manner of fmt.Printf.
// After logging aborting with exit status 1
func (l *Logger) Fatal(tag, format string, v ...interface{}) {
	l.Printf(FatalLevel, tag, format, v...)
	os.Exit(1)
}

var std = New(os.Stderr)

// SetOutput sets the output destination for the standard logger.
func SetOutput(w io.Writer) {
	std.SetOutput(w)
}

// Lev returns logging level for standard Logger.
func Lev() Level {
	return std.Level()
}

// SetLevel sets logging level for standard Logger.
func SetLevel(level Level) {
	std.SetLevel(level)
}

// BufLimit returns bufLimit for standard Logger.
func BufLimit() int64 {
	return std.BufLimit()
}

// SetBufLimit sets bufLimit for standard Logger.
func SetBufLimit(bufLimit int64) {
	std.SetBufLimit(bufLimit)
}

// Writer returns the output destination for the standard logger.
func Writer() io.Writer {
	return std.Writer()
}

// Print calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Print.
func Print(level Level, tag string, v ...interface{}) {
	std.Print(level, tag, v...)
}

// Printf calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf.
func Printf(level Level, tag, format string, v ...interface{}) {
	std.Printf(level, tag, format, v...)
}

// Println calls Output to print to the standard logger.
// Arguments are handled in the manner of fmt.Println.
func Println(level Level, tag string, v ...interface{}) {
	std.Println(level, tag, v...)
}

// Output writes the output for a logging event. The string s contains
// the text to print after the tag specified by the flags of the
// Logger. A newline is appended if the last character of s is not
// already a newline.
func Output(level Level, tag string, s string) error {
	return std.Output(level, tag, s)
}

// Error calls Printf to print to the standard logger with the error level. As tag
// the app name is taken. Other arguments are handled in the manner of fmt.Printf.
func Error(tag, format string, v ...interface{}) {
	std.Error(tag, format, v...)
}

// GetWriter returns an io.Writer implementation which log all data to the standard logger with log level and tag
func GetWriter(level Level, tag string) io.Writer {
	return std.GetWriter(level, tag)
}

// Warn calls Printf to print to the standard logger with the warning level. As tag
// the app name is taken. Other arguments are handled in the manner of fmt.Printf.
func Warn(tag, format string, v ...interface{}) {
	std.Warn(tag, format, v...)
}

// Info calls Printf to print to the standard logger with the info level. As tag
// the app name is taken. Other arguments are handled in the manner of fmt.Printf.
func Info(tag, format string, v ...interface{}) {
	std.Info(tag, format, v...)
}

// Debug calls Printf to print to the standard logger with the debug level. As tag
// the app name is taken. Other arguments are handled in the manner of fmt.Printf.
func Debug(tag, format string, v ...interface{}) {
	std.Debug(tag, format, v...)
}

// Trace calls Printf to print to the standard logger with the trace level. As tag
// the app name is taken. Other arguments are handled in the manner of fmt.Printf.
func Trace(tag, format string, v ...interface{}) {
	std.Trace(tag, format, v...)
}

// Timestamp calls Printf to print to the standard logger with the trace level and own timestamp. As tag
// the app name is taken. Other arguments are handled in the manner of fmt.Printf.
func Timestamp(level Level, timestamp time.Time, tag, format string, v ...interface{}) {
	std.TimestampPrintf(level, timestamp, tag, format, v...)
}

// Fatal calls Printf to print to the standard logger with the debug level. As tag
// the app name is taken. Other arguments are handled in the manner of fmt.Printf.
// After logging aborting with exit status 1
func Fatal(tag, format string, v ...interface{}) {
	std.Fatal(tag, format, v...)
}
