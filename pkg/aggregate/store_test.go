package aggregate

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockSnapshotter struct {
	mock.Mock
}

func (m *mockSnapshotter) SaveSnapshot(ctx context.Context, streamName string, aggregateID string, version int64, data []byte) error {
	args := m.Called(ctx, streamName, aggregateID, version, data)
	return args.Error(0)
}

func (m *mockSnapshotter) GetLatestSnapshot(ctx context.Context, streamName string, aggregateID string, version int64) ([]byte, int64, error) {
	args := m.Called(ctx, streamName, aggregateID, version)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]byte), args.Get(1).(int64), args.Error(2)
}

type mockAggregateRoot struct {
	mock.Mock
}

func (m *mockAggregateRoot) GetID() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockAggregateRoot) GetVersion() int64 {
	args := m.Called()
	return args.Get(0).(int64)
}

func (m *mockAggregateRoot) GetStreamName() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockAggregateRoot) ApplySnapshot(ctx context.Context, snapshot []byte) error {
	args := m.Called(ctx, snapshot)
	return args.Error(0)
}

func (m *mockAggregateRoot) CreateSnapshot() ([]byte, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func TestNewBaseAggregateStore(t *testing.T) {
	t.Run("with config", func(t *testing.T) {
		snapshotter := new(mockSnapshotter)
		config := &SnapshotConfig{
			SnapshotInterval: 5,
			Enabled:          true,
		}
		
		store := NewBaseAggregateStore(snapshotter, config)
		
		assert.Equal(t, snapshotter, store.snapshotter)
		assert.Equal(t, config, store.snapshotConf)
	})
	
	t.Run("with nil config", func(t *testing.T) {
		snapshotter := new(mockSnapshotter)
		
		store := NewBaseAggregateStore(snapshotter, nil)
		
		assert.Equal(t, snapshotter, store.snapshotter)
		assert.NotNil(t, store.snapshotConf)
		assert.Equal(t, int64(10), store.snapshotConf.SnapshotInterval)
		assert.True(t, store.snapshotConf.Enabled)
	})
}

func TestBaseAggregateStore_LoadSnapshot(t *testing.T) {
	ctx := context.Background()
	
	t.Run("disabled snapshots", func(t *testing.T) {
		snapshotter := new(mockSnapshotter)
		config := &SnapshotConfig{
			SnapshotInterval: 10,
			Enabled:          false,
		}
		store := NewBaseAggregateStore(snapshotter, config)
		
		aggregate := new(mockAggregateRoot)
		
		loaded, err := store.LoadSnapshot(ctx, aggregate)
		require.NoError(t, err)
		assert.False(t, loaded)
		
		snapshotter.AssertNotCalled(t, "GetLatestSnapshot")
	})
	
	t.Run("nil snapshotter", func(t *testing.T) {
		config := &SnapshotConfig{
			SnapshotInterval: 10,
			Enabled:          true,
		}
		store := NewBaseAggregateStore(nil, config)
		
		aggregate := new(mockAggregateRoot)
		
		loaded, err := store.LoadSnapshot(ctx, aggregate)
		require.NoError(t, err)
		assert.False(t, loaded)
	})
	
	t.Run("no snapshot found", func(t *testing.T) {
		snapshotter := new(mockSnapshotter)
		config := &SnapshotConfig{
			SnapshotInterval: 10,
			Enabled:          true,
		}
		store := NewBaseAggregateStore(snapshotter, config)
		
		aggregate := new(mockAggregateRoot)
		aggregate.On("GetStreamName").Return("test-stream")
		aggregate.On("GetID").Return("test-1")
		aggregate.On("GetVersion").Return(int64(10))
		
		snapshotter.On("GetLatestSnapshot", ctx, "test-stream", "test-1", int64(10)).Return(nil, int64(0), nil)
		
		loaded, err := store.LoadSnapshot(ctx, aggregate)
		require.NoError(t, err)
		assert.False(t, loaded)
		
		aggregate.AssertExpectations(t)
		snapshotter.AssertExpectations(t)
	})
	
	t.Run("snapshot error", func(t *testing.T) {
		snapshotter := new(mockSnapshotter)
		config := &SnapshotConfig{
			SnapshotInterval: 10,
			Enabled:          true,
		}
		store := NewBaseAggregateStore(snapshotter, config)
		
		aggregate := new(mockAggregateRoot)
		aggregate.On("GetStreamName").Return("test-stream")
		aggregate.On("GetID").Return("test-1")
		aggregate.On("GetVersion").Return(int64(10))
		
		snapshotter.On("GetLatestSnapshot", ctx, "test-stream", "test-1", int64(10)).Return(nil, int64(0), errors.New("snapshot error"))
		
		loaded, err := store.LoadSnapshot(ctx, aggregate)
		require.NoError(t, err) // Error is logged but not returned
		assert.False(t, loaded)
		
		aggregate.AssertExpectations(t)
		snapshotter.AssertExpectations(t)
	})
	
	t.Run("apply snapshot error", func(t *testing.T) {
		snapshotter := new(mockSnapshotter)
		config := &SnapshotConfig{
			SnapshotInterval: 10,
			Enabled:          true,
		}
		store := NewBaseAggregateStore(snapshotter, config)
		
		aggregate := new(mockAggregateRoot)
		aggregate.On("GetStreamName").Return("test-stream")
		aggregate.On("GetID").Return("test-1")
		aggregate.On("GetVersion").Return(int64(10))
		
		snapshotData := []byte("snapshot-data")
		snapshotter.On("GetLatestSnapshot", ctx, "test-stream", "test-1", int64(10)).Return(snapshotData, int64(5), nil)
		
		aggregate.On("ApplySnapshot", ctx, snapshotData).Return(errors.New("apply error"))
		
		loaded, err := store.LoadSnapshot(ctx, aggregate)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "apply error")
		assert.False(t, loaded)
		
		aggregate.AssertExpectations(t)
		snapshotter.AssertExpectations(t)
	})
	
	t.Run("successful load", func(t *testing.T) {
		snapshotter := new(mockSnapshotter)
		config := &SnapshotConfig{
			SnapshotInterval: 10,
			Enabled:          true,
		}
		store := NewBaseAggregateStore(snapshotter, config)
		
		aggregate := new(mockAggregateRoot)
		aggregate.On("GetStreamName").Return("test-stream")
		aggregate.On("GetID").Return("test-1")
		aggregate.On("GetVersion").Return(int64(10))
		
		snapshotData := []byte("snapshot-data")
		snapshotter.On("GetLatestSnapshot", ctx, "test-stream", "test-1", int64(10)).Return(snapshotData, int64(5), nil)
		
		aggregate.On("ApplySnapshot", ctx, snapshotData).Return(nil)
		
		loaded, err := store.LoadSnapshot(ctx, aggregate)
		require.NoError(t, err)
		assert.True(t, loaded)
		
		aggregate.AssertExpectations(t)
		snapshotter.AssertExpectations(t)
	})
}

func TestBaseAggregateStore_SaveSnapshot(t *testing.T) {
	ctx := context.Background()
	
	t.Run("disabled snapshots", func(t *testing.T) {
		snapshotter := new(mockSnapshotter)
		config := &SnapshotConfig{
			SnapshotInterval: 10,
			Enabled:          false,
		}
		store := NewBaseAggregateStore(snapshotter, config)
		
		aggregate := new(mockAggregateRoot)
		
		err := store.SaveSnapshot(ctx, aggregate)
		require.NoError(t, err)
		
		snapshotter.AssertNotCalled(t, "SaveSnapshot")
	})
	
	t.Run("nil snapshotter", func(t *testing.T) {
		config := &SnapshotConfig{
			SnapshotInterval: 10,
			Enabled:          true,
		}
		store := NewBaseAggregateStore(nil, config)
		
		aggregate := new(mockAggregateRoot)
		
		err := store.SaveSnapshot(ctx, aggregate)
		require.NoError(t, err)
	})
	
	t.Run("version not at interval", func(t *testing.T) {
		snapshotter := new(mockSnapshotter)
		config := &SnapshotConfig{
			SnapshotInterval: 10,
			Enabled:          true,
		}
		store := NewBaseAggregateStore(snapshotter, config)
		
		aggregate := new(mockAggregateRoot)
		aggregate.On("GetVersion").Return(int64(5)) // Not divisible by 10
		
		err := store.SaveSnapshot(ctx, aggregate)
		require.NoError(t, err)
		
		aggregate.AssertExpectations(t)
		snapshotter.AssertNotCalled(t, "SaveSnapshot")
	})
	
	t.Run("create snapshot error", func(t *testing.T) {
		snapshotter := new(mockSnapshotter)
		config := &SnapshotConfig{
			SnapshotInterval: 10,
			Enabled:          true,
		}
		store := NewBaseAggregateStore(snapshotter, config)
		
		aggregate := new(mockAggregateRoot)
		aggregate.On("GetVersion").Return(int64(10)) // Divisible by 10
		aggregate.On("CreateSnapshot").Return(nil, errors.New("create error"))
		
		err := store.SaveSnapshot(ctx, aggregate)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create error")
		
		aggregate.AssertExpectations(t)
		snapshotter.AssertNotCalled(t, "SaveSnapshot")
	})
	
	t.Run("save snapshot error", func(t *testing.T) {
		snapshotter := new(mockSnapshotter)
		config := &SnapshotConfig{
			SnapshotInterval: 10,
			Enabled:          true,
		}
		store := NewBaseAggregateStore(snapshotter, config)
		
		aggregate := new(mockAggregateRoot)
		aggregate.On("GetVersion").Return(int64(10)) // Divisible by 10
		
		snapshotData := []byte("snapshot-data")
		aggregate.On("CreateSnapshot").Return(snapshotData, nil)
		aggregate.On("GetStreamName").Return("test-stream")
		aggregate.On("GetID").Return("test-1")
		
		snapshotter.On("SaveSnapshot", ctx, "test-stream", "test-1", int64(10), snapshotData).Return(errors.New("save error"))
		
		err := store.SaveSnapshot(ctx, aggregate)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "save error")
		
		aggregate.AssertExpectations(t)
		snapshotter.AssertExpectations(t)
	})
	
	t.Run("successful save", func(t *testing.T) {
		snapshotter := new(mockSnapshotter)
		config := &SnapshotConfig{
			SnapshotInterval: 10,
			Enabled:          true,
		}
		store := NewBaseAggregateStore(snapshotter, config)
		
		aggregate := new(mockAggregateRoot)
		aggregate.On("GetVersion").Return(int64(10)) // Divisible by 10
		
		snapshotData := []byte("snapshot-data")
		aggregate.On("CreateSnapshot").Return(snapshotData, nil)
		aggregate.On("GetStreamName").Return("test-stream")
		aggregate.On("GetID").Return("test-1")
		
		snapshotter.On("SaveSnapshot", ctx, "test-stream", "test-1", int64(10), snapshotData).Return(nil)
		
		err := store.SaveSnapshot(ctx, aggregate)
		require.NoError(t, err)
		
		aggregate.AssertExpectations(t)
		snapshotter.AssertExpectations(t)
	})
}
