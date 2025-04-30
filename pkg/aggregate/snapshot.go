package aggregate

import (
	"context"
)

type Snapshotter interface {
	SaveSnapshot(ctx context.Context, streamName string, aggregateID string, version int64, data []byte) error

	GetLatestSnapshot(ctx context.Context, streamName string, aggregateID string, version int64) (data []byte, snapshotVersion int64, err error)
}

type SnapshotConfig struct {
	SnapshotInterval int64

	Enabled bool
}

func DefaultSnapshotConfig() *SnapshotConfig {
	return &SnapshotConfig{
		SnapshotInterval: 10, // Create snapshots every 10 versions
		Enabled:          true,
	}
}
