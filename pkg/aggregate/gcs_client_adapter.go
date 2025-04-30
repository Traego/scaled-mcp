package aggregate

import (
	"context"
	"io"

	"cloud.google.com/go/storage"
)

type GCSClientAdapter struct {
	client *storage.Client
}

func NewGCSClientAdapter(client *storage.Client) *GCSClientAdapter {
	return &GCSClientAdapter{
		client: client,
	}
}

func (a *GCSClientAdapter) Bucket(name string) BucketHandle {
	return &GCSBucketAdapter{
		bucket: a.client.Bucket(name),
	}
}

func (a *GCSClientAdapter) Close() error {
	return a.client.Close()
}

type GCSBucketAdapter struct {
	bucket *storage.BucketHandle
}

func (a *GCSBucketAdapter) Object(name string) ObjectHandle {
	return &GCSObjectAdapter{
		object: a.bucket.Object(name),
	}
}

func (a *GCSBucketAdapter) Objects(ctx context.Context, q *storage.Query) ObjectIterator {
	return a.bucket.Objects(ctx, q)
}

type GCSObjectAdapter struct {
	object *storage.ObjectHandle
}

func (a *GCSObjectAdapter) NewWriter(ctx context.Context) GCSObjectWriter {
	return &GCSWriterAdapter{
		writer: a.object.NewWriter(ctx),
	}
}

func (a *GCSObjectAdapter) NewReader(ctx context.Context) (io.ReadCloser, error) {
	return a.object.NewReader(ctx)
}

type GCSWriterAdapter struct {
	writer *storage.Writer
}

func (a *GCSWriterAdapter) Write(p []byte) (n int, err error) {
	return a.writer.Write(p)
}

func (a *GCSWriterAdapter) Close() error {
	return a.writer.Close()
}

func (a *GCSWriterAdapter) SetContentType(contentType string) {
	a.writer.ContentType = contentType
}

func (a *GCSWriterAdapter) SetMetadata(metadata map[string]string) {
	a.writer.Metadata = metadata
}
