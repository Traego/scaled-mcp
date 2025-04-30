package aggregate

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

type GCSSnapshotter struct {
	client      GCSClientInterface
	bucketName  string
	basePrefix  string
}

func NewGCSSnapshotter(ctx context.Context, bucketName string, basePrefix string) (*GCSSnapshotter, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}
	
	adapter := NewGCSClientAdapter(client)
	return NewGCSSnapshotterWithClient(adapter, bucketName, basePrefix), nil
}

func NewGCSSnapshotterWithClient(client GCSClientInterface, bucketName string, basePrefix string) *GCSSnapshotter {
	return &GCSSnapshotter{
		client:     client,
		bucketName: bucketName,
		basePrefix: basePrefix,
	}
}

func (s *GCSSnapshotter) Close() error {
	return s.client.Close()
}

func (s *GCSSnapshotter) buildObjectPath(streamName string, aggregateID string, version int64) string {
	versionStr := fmt.Sprintf("%020d", version) // Pad version for correct sorting
	return path.Join(s.basePrefix, streamName, aggregateID, "snapshots", versionStr)
}

func (s *GCSSnapshotter) SaveSnapshot(ctx context.Context, streamName string, aggregateID string, version int64, data []byte) error {
	bucket := s.client.Bucket(s.bucketName)
	objectPath := s.buildObjectPath(streamName, aggregateID, version)
	obj := bucket.Object(objectPath)

	w := obj.NewWriter(ctx)
	w.SetContentType("application/octet-stream")
	w.SetMetadata(map[string]string{
		"streamName":  streamName,
		"aggregateID": aggregateID,
		"version":     strconv.FormatInt(version, 10),
		"created":     time.Now().UTC().Format(time.RFC3339),
	})

	if _, err := w.Write(data); err != nil {
		w.Close()
		return fmt.Errorf("failed to write snapshot data: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	slog.Debug("Saved snapshot to GCS",
		"bucket", s.bucketName,
		"object", objectPath,
		"stream", streamName,
		"aggregateID", aggregateID,
		"version", version)

	return nil
}

func (s *GCSSnapshotter) GetLatestSnapshot(ctx context.Context, streamName string, aggregateID string, version int64) ([]byte, int64, error) {
	bucket := s.client.Bucket(s.bucketName)
	prefix := path.Join(s.basePrefix, streamName, aggregateID, "snapshots")

	query := &storage.Query{
		Prefix: prefix,
	}

	it := bucket.Objects(ctx, query)

	var latestObj *storage.ObjectAttrs
	var latestVersion int64 = -1

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, 0, fmt.Errorf("error iterating snapshots: %w", err)
		}

		parts := strings.Split(attrs.Name, "/")
		versionStr := parts[len(parts)-1]
		objVersion, err := strconv.ParseInt(versionStr, 10, 64)
		if err != nil {
			slog.Warn("Ignoring invalid snapshot version format", "name", attrs.Name)
			continue
		}

		if objVersion <= version && objVersion > latestVersion {
			latestObj = attrs
			latestVersion = objVersion
		}
	}

	if latestObj == nil {
		return nil, 0, nil // No snapshot found
	}

	obj := bucket.Object(latestObj.Name)
	r, err := obj.NewReader(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("error reading snapshot: %w", err)
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, 0, fmt.Errorf("error reading snapshot data: %w", err)
	}

	slog.Debug("Retrieved snapshot from GCS",
		"bucket", s.bucketName,
		"object", latestObj.Name,
		"stream", streamName,
		"aggregateID", aggregateID,
		"version", latestVersion)

	return data, latestVersion, nil
}
