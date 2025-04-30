package aggregate

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloud.google.com/go/storage"
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

func TestGCSSnapshotter_SaveSnapshot(t *testing.T) {
	mockClient := NewMockGCSClient()
	
	snapshotter := NewGCSSnapshotterWithClient(mockClient, "test-bucket", "snapshots")
	
	ctx := context.Background()
	err := snapshotter.SaveSnapshot(ctx, "test-stream", "test-1", 10, []byte("test-data"))
	require.NoError(t, err)
	
	bucket := mockClient.Bucket("test-bucket").(*MockBucket)
	objectPath := "snapshots/test-stream/test-1/snapshots/00000000000000000010"
	
	bucket.mu.RLock()
	data, exists := bucket.objects[objectPath]
	bucket.mu.RUnlock()
	
	assert.True(t, exists, "Snapshot object should exist")
	assert.Equal(t, []byte("test-data"), data, "Snapshot data should match")
}

func TestGCSSnapshotter_GetLatestSnapshot(t *testing.T) {
	mockClient := NewMockGCSClient()
	bucket := mockClient.Bucket("test-bucket").(*MockBucket)
	
	bucket.mu.Lock()
	bucket.objects["snapshots/test-stream/test-1/snapshots/00000000000000000010"] = []byte("snapshot-v10")
	bucket.objects["snapshots/test-stream/test-1/snapshots/00000000000000000020"] = []byte("snapshot-v20")
	bucket.objects["snapshots/test-stream/test-1/snapshots/00000000000000000030"] = []byte("snapshot-v30")
	bucket.mu.Unlock()
	
	snapshotter := NewGCSSnapshotterWithClient(mockClient, "test-bucket", "snapshots")
	
	ctx := context.Background()
	
	t.Run("get snapshot at version 15", func(t *testing.T) {
		data, version, err := snapshotter.GetLatestSnapshot(ctx, "test-stream", "test-1", 15)
		require.NoError(t, err)
		assert.Equal(t, int64(10), version)
		assert.Equal(t, []byte("snapshot-v10"), data)
	})
	
	t.Run("get snapshot at version 25", func(t *testing.T) {
		data, version, err := snapshotter.GetLatestSnapshot(ctx, "test-stream", "test-1", 25)
		require.NoError(t, err)
		assert.Equal(t, int64(20), version)
		assert.Equal(t, []byte("snapshot-v20"), data)
	})
	
	t.Run("get snapshot at version 40", func(t *testing.T) {
		data, version, err := snapshotter.GetLatestSnapshot(ctx, "test-stream", "test-1", 40)
		require.NoError(t, err)
		assert.Equal(t, int64(30), version)
		assert.Equal(t, []byte("snapshot-v30"), data)
	})
	
	t.Run("no snapshots available", func(t *testing.T) {
		data, version, err := snapshotter.GetLatestSnapshot(ctx, "nonexistent", "nonexistent", 10)
		require.NoError(t, err)
		assert.Nil(t, data)
		assert.Equal(t, int64(0), version)
	})
	
	t.Run("version too low", func(t *testing.T) {
		data, version, err := snapshotter.GetLatestSnapshot(ctx, "test-stream", "test-1", 5)
		require.NoError(t, err)
		assert.Nil(t, data)
		assert.Equal(t, int64(0), version)
	})
}

func TestGCSSnapshotter_GetLatestSnapshot_IteratorError(t *testing.T) {
	mockClient := NewMockGCSClient()
	
	snapshotter := NewGCSSnapshotterWithClient(mockClient, "test-bucket", "snapshots")
	
	bucket := mockClient.Bucket("test-bucket").(*MockBucket)
	bucket.objectsFunc = func(ctx context.Context, q *storage.Query) ObjectIterator {
		return &MockObjectIteratorWithError{
			err: errors.New("iterator error"),
		}
	}
	
	ctx := context.Background()
	data, version, err := snapshotter.GetLatestSnapshot(ctx, "test-stream", "test-1", 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error iterating snapshots")
	assert.Nil(t, data)
	assert.Equal(t, int64(0), version)
}

func TestGCSSnapshotter_GetLatestSnapshot_ReaderError(t *testing.T) {
	mockClient := NewMockGCSClient()
	bucket := mockClient.Bucket("test-bucket").(*MockBucket)
	
	bucket.mu.Lock()
	bucket.objects["snapshots/test-stream/test-1/snapshots/00000000000000000010"] = []byte("snapshot-v10")
	bucket.mu.Unlock()
	
	bucket.objectReaderError = errors.New("reader error")
	
	snapshotter := NewGCSSnapshotterWithClient(mockClient, "test-bucket", "snapshots")
	
	ctx := context.Background()
	data, version, err := snapshotter.GetLatestSnapshot(ctx, "test-stream", "test-1", 20)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error reading snapshot")
	assert.Nil(t, data)
	assert.Equal(t, int64(0), version)
}

func TestGCSSnapshotter_SaveSnapshot_WriteError(t *testing.T) {
	mockClient := NewMockGCSClient()
	bucket := mockClient.Bucket("test-bucket").(*MockBucket)
	
	bucket.objectWriterError = errors.New("write error")
	
	snapshotter := NewGCSSnapshotterWithClient(mockClient, "test-bucket", "snapshots")
	
	ctx := context.Background()
	err := snapshotter.SaveSnapshot(ctx, "test-stream", "test-1", 10, []byte("test-data"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write snapshot data")
}

func TestGCSSnapshotter_SaveSnapshot_CloseError(t *testing.T) {
	mockClient := NewMockGCSClient()
	bucket := mockClient.Bucket("test-bucket").(*MockBucket)
	
	bucket.objectWriterCloseError = errors.New("close error")
	
	snapshotter := NewGCSSnapshotterWithClient(mockClient, "test-bucket", "snapshots")
	
	ctx := context.Background()
	err := snapshotter.SaveSnapshot(ctx, "test-stream", "test-1", 10, []byte("test-data"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to close writer")
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
