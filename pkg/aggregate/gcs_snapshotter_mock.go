package aggregate

import (
	"context"
	"io"
	"strings"
	"sync"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

type GCSClientInterface interface {
	Bucket(name string) BucketHandle
	Close() error
}

type BucketHandle interface {
	Object(name string) ObjectHandle
	Objects(ctx context.Context, q *storage.Query) ObjectIterator
}

type GCSObjectWriter interface {
	io.WriteCloser
	SetContentType(contentType string)
	SetMetadata(metadata map[string]string)
}

type ObjectHandle interface {
	NewWriter(ctx context.Context) GCSObjectWriter
	NewReader(ctx context.Context) (io.ReadCloser, error)
}

type ObjectIterator interface {
	Next() (*storage.ObjectAttrs, error)
}

type MockGCSClient struct {
	buckets map[string]*MockBucket
	mu      sync.RWMutex
}

func NewMockGCSClient() *MockGCSClient {
	return &MockGCSClient{
		buckets: make(map[string]*MockBucket),
	}
}

func (c *MockGCSClient) Bucket(name string) BucketHandle {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if bucket, ok := c.buckets[name]; ok {
		return bucket
	}
	
	bucket := NewMockBucket()
	c.buckets[name] = bucket
	return bucket
}

func (c *MockGCSClient) Close() error {
	return nil
}

type MockBucket struct {
	objects               map[string][]byte
	mu                    sync.RWMutex
	objectsFunc           func(ctx context.Context, q *storage.Query) ObjectIterator
	objectReaderError     error
	objectWriterError     error
	objectWriterCloseError error
}

func NewMockBucket() *MockBucket {
	return &MockBucket{
		objects: make(map[string][]byte),
	}
}

func (b *MockBucket) Object(name string) ObjectHandle {
	return &MockObject{
		bucket: b,
		name:   name,
	}
}

func (b *MockBucket) Objects(ctx context.Context, q *storage.Query) ObjectIterator {
	if b.objectsFunc != nil {
		return b.objectsFunc(ctx, q)
	}
	
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	var objects []*storage.ObjectAttrs
	prefix := ""
	if q != nil {
		prefix = q.Prefix
	}
	
	for name := range b.objects {
		if prefix == "" || strings.HasPrefix(name, prefix) {
			objects = append(objects, &storage.ObjectAttrs{
				Name: name,
			})
		}
	}
	
	return &MockObjectIterator{
		objects: objects,
		index:   0,
	}
}

type MockObject struct {
	bucket *MockBucket
	name   string
}

func (o *MockObject) NewWriter(ctx context.Context) GCSObjectWriter {
	return &MockObjectWriter{
		object:     o,
		buffer:     &strings.Builder{},
		writeError: o.bucket.objectWriterError,
		closeError: o.bucket.objectWriterCloseError,
		metadata:   make(map[string]string),
	}
}

func (o *MockObject) NewReader(ctx context.Context) (io.ReadCloser, error) {
	if o.bucket.objectReaderError != nil {
		return nil, o.bucket.objectReaderError
	}
	
	o.bucket.mu.RLock()
	defer o.bucket.mu.RUnlock()
	
	data, ok := o.bucket.objects[o.name]
	if !ok {
		return nil, storage.ErrObjectNotExist
	}
	
	return &MockObjectReader{
		reader: strings.NewReader(string(data)),
	}, nil
}

type MockObjectWriter struct {
	object      *MockObject
	buffer      *strings.Builder
	writeError  error
	closeError  error
	contentType string
	metadata    map[string]string
}

func (w *MockObjectWriter) Write(p []byte) (n int, err error) {
	if w.writeError != nil {
		return 0, w.writeError
	}
	return w.buffer.Write(p)
}

func (w *MockObjectWriter) Close() error {
	if w.closeError != nil {
		return w.closeError
	}
	
	w.object.bucket.mu.Lock()
	defer w.object.bucket.mu.Unlock()
	
	w.object.bucket.objects[w.object.name] = []byte(w.buffer.String())
	return nil
}

func (w *MockObjectWriter) SetContentType(contentType string) {
	w.contentType = contentType
}

func (w *MockObjectWriter) SetMetadata(metadata map[string]string) {
	w.metadata = metadata
}

type MockObjectReader struct {
	reader *strings.Reader
}

func (r *MockObjectReader) Read(p []byte) (n int, err error) {
	return r.reader.Read(p)
}

func (r *MockObjectReader) Close() error {
	return nil
}

type MockObjectIterator struct {
	objects []*storage.ObjectAttrs
	index   int
}

func (it *MockObjectIterator) Next() (*storage.ObjectAttrs, error) {
	if it.index >= len(it.objects) {
		return nil, iterator.Done
	}
	
	obj := it.objects[it.index]
	it.index++
	return obj, nil
}

type MockObjectIteratorWithError struct {
	err error
}

func (it *MockObjectIteratorWithError) Next() (*storage.ObjectAttrs, error) {
	return nil, it.err
}
