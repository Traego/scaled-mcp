package aggregate

import (
	"context"
	"log/slog"
	"sync"
)

type ExampleAggregateStore struct {
	*BaseAggregateStore
	aggregates map[string]AggregateRoot
	mu         sync.RWMutex
}

func NewExampleAggregateStore(snapshotter Snapshotter, conf *SnapshotConfig) *ExampleAggregateStore {
	return &ExampleAggregateStore{
		BaseAggregateStore: NewBaseAggregateStore(snapshotter, conf),
		aggregates:         make(map[string]AggregateRoot),
	}
}

func (s *ExampleAggregateStore) buildKey(streamName, aggregateID string) string {
	return streamName + ":" + aggregateID
}

func (s *ExampleAggregateStore) Load(ctx context.Context, streamName, aggregateID string) (AggregateRoot, error) {
	s.mu.RLock()
	key := s.buildKey(streamName, aggregateID)
	aggregate, exists := s.aggregates[key]
	s.mu.RUnlock()

	if !exists {
		return nil, ErrStreamNotFound
	}

	_, err := s.LoadSnapshot(ctx, aggregate)
	if err != nil {
		slog.Error("Failed to load snapshot", "error", err)
	}

	return aggregate, nil
}

func (s *ExampleAggregateStore) Save(ctx context.Context, aggregate AggregateRoot) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	currentVersion := aggregate.GetVersion()
	newVersion := currentVersion + 1

	key := s.buildKey(aggregate.GetStreamName(), aggregate.GetID())
	s.aggregates[key] = aggregate

	err := s.SaveSnapshot(ctx, aggregate)
	if err != nil {
		slog.Error("Failed to create snapshot", "error", err)
	}

	return newVersion, nil
}
