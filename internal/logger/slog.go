package logger

import (
	"context"
	"fmt"
	"io"
	golog "log"
	"log/slog"
	"os"
	"strings"

	"disorder.dev/shandler"
	"github.com/tochemey/goakt/v3/log"
)

// DefaultSlogLogger represents the default Log to use
// This Log wraps slog under the hood
var DefaultSlogLogger = NewSlog(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
var DiscardSlogLogger = NewSlog(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))

// Slog represents a logger that wraps the slog logger
// It implements the Logger interface
type Slog struct {
	logger *slog.Logger
	level  slog.Level
}

// enforce compilation and linter error
var _ log.Logger = &Slog{}

// New creates an instance of Slog
func NewSlog(handler slog.Handler) *Slog {
	levelPanic := shandler.LevelFatal + 2
	levels := []slog.Level{levelPanic, shandler.LevelFatal, slog.LevelError, slog.LevelWarn, slog.LevelInfo, slog.LevelDebug, shandler.LevelTrace}
	l := levelPanic

	for i, level := range levels {
		if !handler.Enabled(context.TODO(), level) {
			l = levels[i-1]
			break
		}
	}

	logger := slog.New(handler)
	return &Slog{
		logger: logger,
		level:  l,
	}
}

// Debug starts a message with debug level
func (l *Slog) Debug(v ...any) {
	s := strings.Builder{}
	for i, a := range v {
		if i != len(v)-1 {
			_, _ = s.WriteString(fmt.Sprint(a) + " ")
		} else {
			_, _ = s.WriteString(fmt.Sprint(a))
		}
	}
	l.logger.Debug(s.String())
}

// Debugf starts a message with debug level
func (l *Slog) Debugf(format string, v ...any) {
	l.logger.Debug(fmt.Sprintf(format, v...))
}

// Panic starts a new message with panic level. The panic() function
// is called which stops the ordinary flow of a goroutine.
func (l *Slog) Panic(v ...any) {
	s := strings.Builder{}
	for i, a := range v {
		if i != len(v)-1 {
			_, _ = s.WriteString(fmt.Sprint(a) + " ")
		} else {
			_, _ = s.WriteString(fmt.Sprint(a))
		}
	}
	l.logger.Log(context.TODO(), shandler.LevelFatal+2, s.String())
	panic(v)
}

// Panicf starts a new message with panic level. The panic() function
// is called which stops the ordinary flow of a goroutine.
func (l *Slog) Panicf(format string, v ...any) {
	l.logger.Log(context.TODO(), shandler.LevelFatal+2, fmt.Sprintf(format, v...))
	panic(v)
}

// Fatal starts a new message with fatal level. The os.Exit(1) function
// is called which terminates the program immediately.
func (l *Slog) Fatal(v ...any) {
	s := strings.Builder{}
	for i, a := range v {
		if i != len(v)-1 {
			_, _ = s.WriteString(fmt.Sprint(a) + " ")
		} else {
			_, _ = s.WriteString(fmt.Sprint(a))
		}
	}
	l.logger.Log(context.TODO(), shandler.LevelFatal, s.String())
	os.Exit(1)
}

// Fatalf starts a new message with fatal level. The os.Exit(1) function
// is called which terminates the program immediately.
func (l *Slog) Fatalf(format string, v ...any) {
	l.logger.Log(context.TODO(), shandler.LevelFatal, fmt.Sprintf(format, v...))
	os.Exit(1)
}

// Error starts a new message with error level.
func (l *Slog) Error(v ...any) {
	s := strings.Builder{}
	for i, a := range v {
		if i != len(v)-1 {
			_, _ = s.WriteString(fmt.Sprint(a) + " ")
		} else {
			_, _ = s.WriteString(fmt.Sprint(a))
		}
	}
	l.logger.Error(s.String())
}

// Errorf starts a new message with error level.
func (l *Slog) Errorf(format string, v ...any) {
	l.logger.Error(fmt.Sprintf(format, v...))
}

// Warn starts a new message with warn level
func (l *Slog) Warn(v ...any) {
	s := strings.Builder{}
	for i, a := range v {
		if i != len(v)-1 {
			_, _ = s.WriteString(fmt.Sprint(a) + " ")
		} else {
			_, _ = s.WriteString(fmt.Sprint(a))
		}
	}
	l.logger.Warn(s.String())
}

// Warnf starts a new message with warn level
func (l *Slog) Warnf(format string, v ...any) {
	l.logger.Warn(fmt.Sprintf(format, v...))
}

// Info starts a message with info level
func (l *Slog) Info(v ...any) {
	s := strings.Builder{}
	for i, a := range v {
		if i != len(v)-1 {
			_, _ = s.WriteString(fmt.Sprint(a) + " ")
		} else {
			_, _ = s.WriteString(fmt.Sprint(a))
		}
	}
	l.logger.Info(s.String())
}

// Infof starts a message with info level
func (l *Slog) Infof(format string, v ...any) {
	l.logger.Info(fmt.Sprintf(format, v...))
}

// Info starts a message with info level
func (l *Slog) Trace(v ...any) {
	s := strings.Builder{}
	for i, a := range v {
		if i != len(v)-1 {
			_, _ = s.WriteString(fmt.Sprint(a) + " ")
		} else {
			_, _ = s.WriteString(fmt.Sprint(a))
		}
	}
	l.logger.Log(context.TODO(), shandler.LevelTrace, s.String())
}

// LogLevel returns the log level that is used
func (l *Slog) LogLevel() log.Level {
	var traceLevel log.Level = log.DebugLevel + 2
	switch l.level {
	case shandler.LevelFatal:
		return log.FatalLevel
	case slog.LevelError:
		return log.ErrorLevel
	case slog.LevelInfo:
		return log.InfoLevel
	case slog.LevelDebug:
		return log.DebugLevel
	case slog.LevelWarn:
		return log.WarningLevel
	case shandler.LevelTrace:
		return traceLevel
	default:
		return log.InvalidLevel
	}
}

// LogOutput returns the log output that is set
func (l *Slog) LogOutput() []io.Writer {
	return nil
}

// StdLogger returns the standard logger associated to the logger
func (l *Slog) StdLogger() *golog.Logger {
	return slog.NewLogLogger(l.logger.Handler(), l.level)
}
