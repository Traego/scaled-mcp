package utils

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlogWrapper(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer

	// Create a new logger with the buffer as output
	logger := NewLogger(DebugLevel, &buf)
	require.NotNil(t, logger)

	// Test that the logger implements the Logger interface
	var _ Logger = logger

	t.Run("LogLevels", func(t *testing.T) {
		// Clear the buffer
		buf.Reset()

		// Test debug level
		logger.Debug("debug message")
		assert.Contains(t, buf.String(), "DEBUG")
		assert.Contains(t, buf.String(), "debug message")
		buf.Reset()

		logger.Debugf("debug %s", "formatted")
		assert.Contains(t, buf.String(), "DEBUG")
		assert.Contains(t, buf.String(), "debug formatted")
		buf.Reset()

		// Test info level
		logger.Info("info message")
		assert.Contains(t, buf.String(), "INFO")
		assert.Contains(t, buf.String(), "info message")
		buf.Reset()

		logger.Infof("info %s", "formatted")
		assert.Contains(t, buf.String(), "INFO")
		assert.Contains(t, buf.String(), "info formatted")
		buf.Reset()

		// Test warn level
		logger.Warn("warn message")
		assert.Contains(t, buf.String(), "WARN")
		assert.Contains(t, buf.String(), "warn message")
		buf.Reset()

		logger.Warnf("warn %s", "formatted")
		assert.Contains(t, buf.String(), "WARN")
		assert.Contains(t, buf.String(), "warn formatted")
		buf.Reset()

		// Test error level
		logger.Error("error message")
		assert.Contains(t, buf.String(), "ERROR")
		assert.Contains(t, buf.String(), "error message")
		buf.Reset()

		logger.Errorf("error %s", "formatted")
		assert.Contains(t, buf.String(), "ERROR")
		assert.Contains(t, buf.String(), "error formatted")
		buf.Reset()
	})

	t.Run("LogLevel", func(t *testing.T) {
		assert.Equal(t, DebugLevel, logger.LogLevel())

		// Create a new logger with info level
		infoLogger := NewLogger(InfoLevel, &buf)
		assert.Equal(t, InfoLevel, infoLogger.LogLevel())
	})

	t.Run("LogOutput", func(t *testing.T) {
		outputs := logger.LogOutput()
		require.Len(t, outputs, 1)
		assert.Equal(t, &buf, outputs[0])
	})

	t.Run("StdLogger", func(t *testing.T) {
		stdLogger := logger.StdLogger()
		require.NotNil(t, stdLogger)
	})

	t.Run("WithField", func(t *testing.T) {
		buf.Reset()
		fieldLogger := logger.WithField("key", "value")
		fieldLogger.Info("with field")

		output := buf.String()
		assert.Contains(t, output, "key")
		assert.Contains(t, output, "value")
		assert.Contains(t, output, "with field")
	})

	t.Run("WithFields", func(t *testing.T) {
		buf.Reset()
		fieldsLogger := logger.WithFields(map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		})
		fieldsLogger.Info("with fields")

		output := buf.String()
		assert.Contains(t, output, "key1")
		assert.Contains(t, output, "value1")
		assert.Contains(t, output, "key2")
		assert.Contains(t, output, "value2")
		assert.Contains(t, output, "with fields")
	})

	t.Run("DefaultLogger", func(t *testing.T) {
		// Just test that the default logger functions don't panic
		// We can't easily capture their output since they write to os.Stderr
		assert.NotPanics(t, func() {
			Info("info")
			Infof("info %s", "formatted")
			Warn("warn")
			Warnf("warn %s", "formatted")
			Error("error")
			Errorf("error %s", "formatted")
			Debug("debug")
			Debugf("debug %s", "formatted")
		})
	})

	t.Run("PanicFunctions", func(t *testing.T) {
		// Test that Panic and Panicf actually panic
		assert.Panics(t, func() {
			logger.Panic("panic message")
		})

		assert.Panics(t, func() {
			logger.Panicf("panic %s", "formatted")
		})
	})

	// We can't easily test Fatal functions since they call os.Exit
}

func TestSetLogLevel(t *testing.T) {
	// Save the original default logger
	originalLogger := defaultLogger

	// Restore the default logger after the test
	defer func() {
		defaultLogger = originalLogger
	}()

	// Set the log level to debug
	SetLogLevel(DebugLevel)
	assert.Equal(t, DebugLevel, defaultLogger.LogLevel())

	// Set the log level to error
	SetLogLevel(ErrorLevel)
	assert.Equal(t, ErrorLevel, defaultLogger.LogLevel())
}

func TestGetLogger(t *testing.T) {
	logger := GetLogger()
	assert.Equal(t, defaultLogger, logger)
}
