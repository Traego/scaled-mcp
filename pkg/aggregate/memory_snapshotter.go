package aggregate

import (
	"context"
	"fmt"
	"sync"
)

type MemorySnapshotter struct {
	snapshots map[string]map[int64][]byte
	mu        sync.RWMutex
}

func NewMemorySnapshotter() *MemorySnapshotter {
	return &MemorySnapshotter{
		snapshots: make(map[string]map[int64][]byte),
	}
}

func (s *MemorySnapshotter) buildKey(streamName string, aggregateID string) string {
	return fmt.Sprintf("%s:%s", streamName, aggregateID)
}

func (s *MemorySnapshotter) SaveSnapshot(ctx context.Context, streamName string, aggregateID string, version int64, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.buildKey(streamName, aggregateID)

	if _, ok := s.snapshots[key]; !ok {
		s.snapshots[key] = make(map[int64][]byte)
	}

	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	s.snapshots[key][version] = dataCopy
	return nil
}

func (s *MemorySnapshotter) GetLatestSnapshot(ctx context.Context, streamName string, aggregateID string, version int64) ([]byte, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := s.buildKey(streamName, aggregateID)

	versionMap, ok := s.snapshots[key]
	if !ok {
		return nil, 0, nil
	}

	var latestVersion int64 = -1
	var latestData []byte

	for v, data := range versionMap {
		if v <= version && v > latestVersion {
			latestVersion = v
			latestData = data
		}
	}

	if latestVersion == -1 {
		return nil, 0, nil
	}

	dataCopy := make([]byte, len(latestData))
	copy(dataCopy, latestData)

	return dataCopy, latestVersion, nil
}
