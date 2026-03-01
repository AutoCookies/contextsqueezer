package runtime

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

type Logger struct {
	mu     sync.Mutex
	out    io.Writer
	level  LogLevel
	prefix string
}

var defaultLogger = &Logger{
	out:    os.Stderr,
	level:  LevelInfo,
	prefix: "[CSQ]",
}

func init() {
	if os.Getenv("CSQ_DEBUG") == "1" {
		defaultLogger.level = LevelDebug
	}
}

func SetOutput(w io.Writer) {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()
	defaultLogger.out = w
}

func SetLevel(l LogLevel) {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()
	defaultLogger.level = l
}

func log(l LogLevel, msg string) {
	if l < defaultLogger.level {
		return
	}
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()
	fmt.Fprintf(defaultLogger.out, "%s %s %s\n",
		time.Now().Format("2006-01-02 15:04:05.000"),
		defaultLogger.prefix, msg)
}

func Debugf(format string, args ...interface{}) { log(LevelDebug, fmt.Sprintf(format, args...)) }
func Infof(format string, args ...interface{})  { log(LevelInfo, fmt.Sprintf(format, args...)) }
func Warnf(format string, args ...interface{})  { log(LevelWarn, fmt.Sprintf(format, args...)) }
func Errorf(format string, args ...interface{}) { log(LevelError, fmt.Sprintf(format, args...)) }
