package aggregate

import (
	"context"
	"errors"
)

var (
	ErrStreamNotFound  = errors.New("stream not found")
	ErrVersionConflict = errors.New("version conflict")
	ErrSnapshotFailed  = errors.New("snapshot creation failed")
)

type AggregateRoot interface {
	GetID() string

	GetVersion() int64

	GetStreamName() string

	ApplySnapshot(ctx context.Context, snapshot []byte) error

	CreateSnapshot() ([]byte, error)
}

type AggregateStore interface {
	Load(ctx context.Context, streamName, aggregateID string) (AggregateRoot, error)

	Save(ctx context.Context, aggregate AggregateRoot) (int64, error)
}
