// Package utils provides utility functions and wrappers for the scaled-mcp library.
package utils

import (
	"context"
	"fmt"
	"io"
	golog "log"
	"log/slog"
	"os"
)

// Level represents the severity level of a log message.
type Level int

const (
	// DebugLevel logs are typically voluminous, and are usually disabled in production.
	DebugLevel Level = iota
	// InfoLevel is the default logging priority.
	InfoLevel
	// WarnLevel logs are more important than Info, but don't need individual human review.
	WarnLevel
	// ErrorLevel logs are high-priority. If an application is running smoothly,
	// it shouldn't generate any error-level logs.
	ErrorLevel
	// FatalLevel logs are particularly important errors. In development the logger will panic after writing the message.
	FatalLevel
	// PanicLevel logs a message, then panics.
	PanicLevel
)

// SlogWrapper is a wrapper around slog that implements the Logger interface.
type SlogWrapper struct {
	logger    *slog.Logger
	level     Level
	outputs   []io.Writer
	stdLogger *golog.Logger
}

// NewLogger creates a new SlogWrapper with the specified level and output.
// If output is nil, os.Stderr is used.
func NewLogger(level Level, output io.Writer) *SlogWrapper {
	if output == nil {
		output = os.Stderr
	}

	// Convert our level to slog level
	var slogLevel slog.Level
	switch level {
	case DebugLevel:
		slogLevel = slog.LevelDebug
	case InfoLevel:
		slogLevel = slog.LevelInfo
	case WarnLevel:
		slogLevel = slog.LevelWarn
	case ErrorLevel, FatalLevel, PanicLevel:
		slogLevel = slog.LevelError
	}

	// Create handler with appropriate level
	handler := slog.NewTextHandler(output, &slog.HandlerOptions{
		Level: slogLevel,
	})

	logger := slog.New(handler)

	// Create standard logger that writes to the same output
	stdLogger := golog.New(output, "", golog.LstdFlags)

	return &SlogWrapper{
		logger:    logger,
		level:     level,
		outputs:   []io.Writer{output},
		stdLogger: stdLogger,
	}
}

// Info logs a message at InfoLevel.
func (l *SlogWrapper) Info(args ...any) {
	l.logger.Info(fmt.Sprint(args...))
}

// Infof logs a formatted message at InfoLevel.
func (l *SlogWrapper) Infof(format string, args ...any) {
	l.logger.Info(fmt.Sprintf(format, args...))
}

// Warn logs a message at WarnLevel.
func (l *SlogWrapper) Warn(args ...any) {
	l.logger.Warn(fmt.Sprint(args...))
}

// Warnf logs a formatted message at WarnLevel.
func (l *SlogWrapper) Warnf(format string, args ...any) {
	l.logger.Warn(fmt.Sprintf(format, args...))
}

// Error logs a message at ErrorLevel.
func (l *SlogWrapper) Error(args ...any) {
	l.logger.Error(fmt.Sprint(args...))
}

// Errorf logs a formatted message at ErrorLevel.
func (l *SlogWrapper) Errorf(format string, args ...any) {
	l.logger.Error(fmt.Sprintf(format, args...))
}

// Fatal logs a message at FatalLevel and then calls os.Exit(1).
func (l *SlogWrapper) Fatal(args ...any) {
	l.logger.Error(fmt.Sprint(args...), slog.String("level", "FATAL"))
	os.Exit(1)
}

// Fatalf logs a formatted message at FatalLevel and then calls os.Exit(1).
func (l *SlogWrapper) Fatalf(format string, args ...any) {
	l.logger.Error(fmt.Sprintf(format, args...), slog.String("level", "FATAL"))
	os.Exit(1)
}

// Panic logs a message at PanicLevel and then panics.
func (l *SlogWrapper) Panic(args ...any) {
	msg := fmt.Sprint(args...)
	l.logger.Error(msg, slog.String("level", "PANIC"))
	panic(msg)
}

// Panicf logs a formatted message at PanicLevel and then panics.
func (l *SlogWrapper) Panicf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	l.logger.Error(msg, slog.String("level", "PANIC"))
	panic(msg)
}

// Debug logs a message at DebugLevel.
func (l *SlogWrapper) Debug(args ...any) {
	l.logger.Debug(fmt.Sprint(args...))
}

// Debugf logs a formatted message at DebugLevel.
func (l *SlogWrapper) Debugf(format string, args ...any) {
	l.logger.Debug(fmt.Sprintf(format, args...))
}

// LogLevel returns the current log level.
func (l *SlogWrapper) LogLevel() Level {
	return l.level
}

// LogOutput returns the current log outputs.
func (l *SlogWrapper) LogOutput() []io.Writer {
	return l.outputs
}

// StdLogger returns the standard logger.
func (l *SlogWrapper) StdLogger() *golog.Logger {
	return l.stdLogger
}

// WithContext returns a new logger with the provided context.
func (l *SlogWrapper) WithContext(ctx context.Context) *SlogWrapper {
	// slog doesn't have a direct WithContext method
	// We can store the context in a new logger instance
	return &SlogWrapper{
		logger:    l.logger,
		level:     l.level,
		outputs:   l.outputs,
		stdLogger: l.stdLogger,
	}
}

// WithField returns a new logger with the provided field.
func (l *SlogWrapper) WithField(key string, value interface{}) *SlogWrapper {
	return &SlogWrapper{
		logger:    l.logger.With(key, value),
		level:     l.level,
		outputs:   l.outputs,
		stdLogger: l.stdLogger,
	}
}

// WithFields returns a new logger with the provided fields.
func (l *SlogWrapper) WithFields(fields map[string]interface{}) *SlogWrapper {
	attrs := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		attrs = append(attrs, k, v)
	}

	return &SlogWrapper{
		logger:    l.logger.With(attrs...),
		level:     l.level,
		outputs:   l.outputs,
		stdLogger: l.stdLogger,
	}
}

// Logger is the interface that wraps the basic logging methods.
type Logger interface {
	// Info starts a new message with info level.
	Info(...any)
	// Infof starts a new message with info level.
	Infof(string, ...any)
	// Warn starts a new message with warn level.
	Warn(...any)
	// Warnf starts a new message with warn level.
	Warnf(string, ...any)
	// Error starts a new message with error level.
	Error(...any)
	// Errorf starts a new message with error level.
	Errorf(string, ...any)
	// Fatal starts a new message with fatal level. The os.Exit(1) function
	// is called which terminates the program immediately.
	Fatal(...any)
	// Fatalf starts a new message with fatal level. The os.Exit(1) function
	// is called which terminates the program immediately.
	Fatalf(string, ...any)
	// Panic starts a new message with panic level. The panic() function
	// is called which stops the ordinary flow of a goroutine.
	Panic(...any)
	// Panicf starts a new message with panic level. The panic() function
	// is called which stops the ordinary flow of a goroutine.
	Panicf(string, ...any)
	// Debug starts a new message with debug level.
	Debug(...any)
	// Debugf starts a new message with debug level.
	Debugf(string, ...any)
	// LogLevel returns the log level being used
	LogLevel() Level
	// LogOutput returns the log output that is set
	LogOutput() []io.Writer
	// StdLogger returns the standard logger associated to the logger
	StdLogger() *golog.Logger
}

// Ensure SlogWrapper implements the Logger interface
var _ Logger = (*SlogWrapper)(nil)

// Default logger instance
var defaultLogger = NewLogger(InfoLevel, os.Stderr)

// Default logger functions for package-level logging
func Info(args ...any)                  { defaultLogger.Info(args...) }
func Infof(format string, args ...any)  { defaultLogger.Infof(format, args...) }
func Warn(args ...any)                  { defaultLogger.Warn(args...) }
func Warnf(format string, args ...any)  { defaultLogger.Warnf(format, args...) }
func Error(args ...any)                 { defaultLogger.Error(args...) }
func Errorf(format string, args ...any) { defaultLogger.Errorf(format, args...) }
func Fatal(args ...any)                 { defaultLogger.Fatal(args...) }
func Fatalf(format string, args ...any) { defaultLogger.Fatalf(format, args...) }
func Panic(args ...any)                 { defaultLogger.Panic(args...) }
func Panicf(format string, args ...any) { defaultLogger.Panicf(format, args...) }
func Debug(args ...any)                 { defaultLogger.Debug(args...) }
func Debugf(format string, args ...any) { defaultLogger.Debugf(format, args...) }

// SetLogLevel sets the log level for the default logger
func SetLogLevel(level Level) {
	defaultLogger = NewLogger(level, defaultLogger.outputs[0])
}

// GetLogger returns the default logger
func GetLogger() Logger {
	return defaultLogger
}
