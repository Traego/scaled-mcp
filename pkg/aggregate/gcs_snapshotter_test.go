package aggregate

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGCSSnapshotter_BuildObjectPath(t *testing.T) {
	snapshotter := &GCSSnapshotter{
		bucketName: "test-bucket",
		basePrefix: "snapshots",
	}
	
	path := snapshotter.buildObjectPath("test-stream", "test-1", 42)
	expected := "snapshots/test-stream/test-1/snapshots/00000000000000000042"
	assert.Equal(t, expected, path)
}

func TestGCSSnapshotter_Integration(t *testing.T) {
	t.Skip("Skipping GCS integration test - requires GCS credentials")
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	snapshotter, err := NewGCSSnapshotter(ctx, "test-bucket", "test-snapshots")
	require.NoError(t, err)
	defer snapshotter.Close()
	
	err = snapshotter.SaveSnapshot(ctx, "test-stream", "test-1", 10, []byte("test-data"))
	require.NoError(t, err)
	
	data, version, err := snapshotter.GetLatestSnapshot(ctx, "test-stream", "test-1", 20)
	require.NoError(t, err)
	assert.Equal(t, int64(10), version)
	assert.Equal(t, []byte("test-data"), data)
}
