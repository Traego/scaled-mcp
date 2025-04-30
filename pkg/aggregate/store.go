package aggregate

import (
	"context"
	"log/slog"
)

type BaseAggregateStore struct {
	snapshotter  Snapshotter
	snapshotConf *SnapshotConfig
}

func NewBaseAggregateStore(snapshotter Snapshotter, conf *SnapshotConfig) *BaseAggregateStore {
	if conf == nil {
		conf = DefaultSnapshotConfig()
	}

	return &BaseAggregateStore{
		snapshotter:  snapshotter,
		snapshotConf: conf,
	}
}

func (s *BaseAggregateStore) LoadSnapshot(ctx context.Context, aggregate AggregateRoot) (bool, error) {
	if !s.snapshotConf.Enabled || s.snapshotter == nil {
		return false, nil
	}

	streamName := aggregate.GetStreamName()
	aggregateID := aggregate.GetID()

	snapshotData, version, err := s.snapshotter.GetLatestSnapshot(ctx, streamName, aggregateID, aggregate.GetVersion())
	if err != nil {
		slog.Debug("No snapshot found or error occurred", "streamName", streamName, "aggregateID", aggregateID, "error", err)
		return false, nil
	}

	if snapshotData == nil {
		return false, nil
	}

	err = aggregate.ApplySnapshot(ctx, snapshotData)
	if err != nil {
		return false, err
	}

	slog.Debug("Snapshot loaded", "streamName", streamName, "aggregateID", aggregateID, "version", version)
	return true, nil
}

func (s *BaseAggregateStore) SaveSnapshot(ctx context.Context, aggregate AggregateRoot) error {
	if !s.snapshotConf.Enabled || s.snapshotter == nil {
		return nil
	}

	version := aggregate.GetVersion()

	if version%s.snapshotConf.SnapshotInterval != 0 {
		return nil
	}

	snapshotData, err := aggregate.CreateSnapshot()
	if err != nil {
		return err
	}

	streamName := aggregate.GetStreamName()
	aggregateID := aggregate.GetID()

	err = s.snapshotter.SaveSnapshot(ctx, streamName, aggregateID, version, snapshotData)
	if err != nil {
		return err
	}

	slog.Debug("Snapshot created", "streamName", streamName, "aggregateID", aggregateID, "version", version)
	return nil
}
