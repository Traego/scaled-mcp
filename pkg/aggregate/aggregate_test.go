package aggregate

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExampleAggregate(t *testing.T) {
	t.Run("create and manipulate aggregate", func(t *testing.T) {
		agg := NewExampleAggregate("test-1", "test-stream")

		assert.Equal(t, "test-1", agg.GetID())
		assert.Equal(t, "test-stream", agg.GetStreamName())
		assert.Equal(t, int64(0), agg.GetVersion())

		agg.SetData("key1", "value1")
		agg.SetData("key2", 42)

		assert.Equal(t, "value1", agg.GetData("key1"))
		assert.Equal(t, 42, agg.GetData("key2"))
	})

	t.Run("create and apply snapshot", func(t *testing.T) {
		agg := NewExampleAggregate("test-1", "test-stream")
		agg.Version = 5 // Set version

		agg.SetData("key1", "value1")
		agg.SetData("key2", 42)

		snapshot, err := agg.CreateSnapshot()
		require.NoError(t, err)
		require.NotEmpty(t, snapshot)

		newAgg := NewExampleAggregate("", "")
		err = newAgg.ApplySnapshot(context.Background(), snapshot)
		require.NoError(t, err)

		assert.Equal(t, "test-1", newAgg.GetID())
		assert.Equal(t, "test-stream", newAgg.GetStreamName())
		assert.Equal(t, int64(5), newAgg.GetVersion())
		assert.Equal(t, "value1", newAgg.GetData("key1"))
		assert.Equal(t, float64(42), newAgg.GetData("key2")) // JSON unmarshals numbers as float64
	})
}

func TestMemorySnapshotter(t *testing.T) {
	ctx := context.Background()
	snapshotter := NewMemorySnapshotter()

	streamName := "test-stream"
	aggregateID := "test-1"

	t.Run("save and retrieve snapshots", func(t *testing.T) {
		err := snapshotter.SaveSnapshot(ctx, streamName, aggregateID, 10, []byte("snapshot-v10"))
		require.NoError(t, err)

		err = snapshotter.SaveSnapshot(ctx, streamName, aggregateID, 20, []byte("snapshot-v20"))
		require.NoError(t, err)

		err = snapshotter.SaveSnapshot(ctx, streamName, aggregateID, 30, []byte("snapshot-v30"))
		require.NoError(t, err)

		data, ver, err := snapshotter.GetLatestSnapshot(ctx, streamName, aggregateID, 15)
		require.NoError(t, err)
		assert.Equal(t, int64(10), ver)
		assert.Equal(t, []byte("snapshot-v10"), data)

		data, ver, err = snapshotter.GetLatestSnapshot(ctx, streamName, aggregateID, 25)
		require.NoError(t, err)
		assert.Equal(t, int64(20), ver)
		assert.Equal(t, []byte("snapshot-v20"), data)

		data, ver, err = snapshotter.GetLatestSnapshot(ctx, streamName, aggregateID, 40)
		require.NoError(t, err)
		assert.Equal(t, int64(30), ver)
		assert.Equal(t, []byte("snapshot-v30"), data)
	})

	t.Run("no snapshots available", func(t *testing.T) {
		data, ver, err := snapshotter.GetLatestSnapshot(ctx, "nonexistent", "nonexistent", 10)
		require.NoError(t, err)
		assert.Nil(t, data)
		assert.Equal(t, int64(0), ver)
	})

	t.Run("version too low", func(t *testing.T) {
		data, ver, err := snapshotter.GetLatestSnapshot(ctx, streamName, aggregateID, 5)
		require.NoError(t, err)
		assert.Nil(t, data)
		assert.Equal(t, int64(0), ver)
	})
}

func TestExampleAggregateStore(t *testing.T) {
	ctx := context.Background()
	snapshotter := NewMemorySnapshotter()

	t.Run("snapshots every 10 versions", func(t *testing.T) {
		config := &SnapshotConfig{
			SnapshotInterval: 10,
			Enabled:          true,
		}
		store := NewExampleAggregateStore(snapshotter, config)

		agg := NewExampleAggregate("test-1", "test-stream")
		agg.SetData("key1", "value1")

		for i := 0; i < 25; i++ {
			agg.Version = int64(i) // Manually set version for testing
			_, err := store.Save(ctx, agg)
			require.NoError(t, err)
		}

		_, v0, err := snapshotter.GetLatestSnapshot(ctx, "test-stream", "test-1", 9)
		require.NoError(t, err)
		assert.Equal(t, int64(0), v0)

		_, v10, err := snapshotter.GetLatestSnapshot(ctx, "test-stream", "test-1", 15)
		require.NoError(t, err)
		assert.Equal(t, int64(10), v10)

		_, v20, err := snapshotter.GetLatestSnapshot(ctx, "test-stream", "test-1", 25)
		require.NoError(t, err)
		assert.Equal(t, int64(20), v20)
	})

	t.Run("disabled snapshots", func(t *testing.T) {
		config := &SnapshotConfig{
			SnapshotInterval: 10,
			Enabled:          false,
		}
		store := NewExampleAggregateStore(snapshotter, config)

		agg := NewExampleAggregate("test-disabled", "test-stream")
		agg.Version = 10 // Version that would normally trigger a snapshot

		_, err := store.Save(ctx, agg)
		require.NoError(t, err)

		data, _, err := snapshotter.GetLatestSnapshot(ctx, "test-stream", "test-disabled", 10)
		require.NoError(t, err)
		assert.Nil(t, data)
	})

	t.Run("load with snapshot", func(t *testing.T) {
		snapshotter := NewMemorySnapshotter()
		store := NewExampleAggregateStore(snapshotter, DefaultSnapshotConfig())

		agg := NewExampleAggregate("test-load", "test-stream")
		agg.SetData("key1", "original-value")
		agg.Version = 10 // Version that will trigger a snapshot

		_, err := store.Save(ctx, agg)
		require.NoError(t, err)

		agg.SetData("key1", "modified-value")

		loadedAgg, err := store.Load(ctx, "test-stream", "test-load")
		require.NoError(t, err)

		assert.Equal(t, "original-value", loadedAgg.(*ExampleAggregate).GetData("key1"))
	})
}
