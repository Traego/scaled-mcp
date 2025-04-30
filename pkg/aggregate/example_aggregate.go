package aggregate

import (
	"context"
	"encoding/json"
	"fmt"
)

type ExampleAggregate struct {
	ID         string
	Version    int64
	StreamName string
	Data       map[string]interface{}
}

func NewExampleAggregate(id, streamName string) *ExampleAggregate {
	return &ExampleAggregate{
		ID:         id,
		Version:    0,
		StreamName: streamName,
		Data:       make(map[string]interface{}),
	}
}

func (a *ExampleAggregate) GetID() string {
	return a.ID
}

func (a *ExampleAggregate) GetVersion() int64 {
	return a.Version
}

func (a *ExampleAggregate) GetStreamName() string {
	return a.StreamName
}

func (a *ExampleAggregate) SetData(key string, value interface{}) {
	a.Data[key] = value
}

func (a *ExampleAggregate) GetData(key string) interface{} {
	return a.Data[key]
}

func (a *ExampleAggregate) CreateSnapshot() ([]byte, error) {
	snapshot := map[string]interface{}{
		"id":          a.ID,
		"version":     a.Version,
		"stream_name": a.StreamName,
		"data":        a.Data,
	}

	data, err := json.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize snapshot: %w", err)
	}

	return data, nil
}

func (a *ExampleAggregate) ApplySnapshot(ctx context.Context, data []byte) error {
	var snapshot map[string]interface{}
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return fmt.Errorf("failed to deserialize snapshot: %w", err)
	}

	if id, ok := snapshot["id"].(string); ok {
		a.ID = id
	}

	if version, ok := snapshot["version"].(float64); ok {
		a.Version = int64(version)
	}

	if streamName, ok := snapshot["stream_name"].(string); ok {
		a.StreamName = streamName
	}

	if data, ok := snapshot["data"].(map[string]interface{}); ok {
		a.Data = data
	}

	return nil
}
