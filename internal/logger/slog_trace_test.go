package logger

import (
	"bytes"
	"log/slog"
	"testing"

	"disorder.dev/shandler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlogTrace(t *testing.T) {
	t.Run("With Trace log level", func(t *testing.T) {
		buffer := new(bytes.Buffer)
		handler := slog.NewJSONHandler(buffer, &slog.HandlerOptions{Level: shandler.LevelTrace})
		logger := NewSlog(handler)
		
		logger.Trace("test trace")
		
		actual, err := extractMessage(buffer.Bytes())
		require.NoError(t, err)
		assert.Equal(t, "test trace", actual)
		
		lvl, err := extractLevel(buffer.Bytes())
		require.NoError(t, err)
		assert.Equal(t, "DEBUG-2", lvl)
	})
	
	t.Run("With Debug log level", func(t *testing.T) {
		buffer := new(bytes.Buffer)
		logger := NewSlog(slog.NewJSONHandler(buffer, &slog.HandlerOptions{Level: slog.LevelDebug}))
		
		logger.Trace("test trace")
		
		assert.Empty(t, buffer.String())
	})
}

func TestSlogLogOutput(t *testing.T) {
	buffer := new(bytes.Buffer)
	logger := NewSlog(slog.NewJSONHandler(buffer, &slog.HandlerOptions{Level: slog.LevelDebug}))
	
	writers := logger.LogOutput()
	
	assert.Nil(t, writers)
}

func TestSlogStdLogger(t *testing.T) {
	buffer := new(bytes.Buffer)
	logger := NewSlog(slog.NewJSONHandler(buffer, &slog.HandlerOptions{Level: slog.LevelDebug}))
	
	stdLogger := logger.StdLogger()
	
	assert.NotNil(t, stdLogger)
}

func TestSlogFatal(t *testing.T) {
	t.Skip("Skipping test that calls os.Exit")
	
}

func TestSlogFatalMessage(t *testing.T) {
	t.Skip("Skipping test for Fatal method because it calls os.Exit which would terminate the test process")
	
}
