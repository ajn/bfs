// Package bfs outlines an abstraction for bucket-based fyle systems with
// mock-implmentations.
package bfs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"sync"
	"time"
)

// ErrNotFound must be returned by all implementations
// when a requested object cannot be found.
var ErrNotFound = errors.New("bfs: object not found")

// Bucket is an abstract storage bucket.
type Bucket interface {
	// Glob lists the files matching a glob pattern. It supports
	// `*`, `**`, `?` wildcards, character classes and alternative sequences.
	// Please see https://github.com/bmatcuk/doublestar#patterns for more details.
	Glob(ctx context.Context, pattern string) (Iterator, error)

	// Head returns an object's meta info.
	Head(ctx context.Context, name string) (*MetaInfo, error)

	// Open opens an object for reading.
	Open(ctx context.Context, name string) (io.ReadCloser, error)

	// Create creates/opens a object for writing.
	Create(ctx context.Context, name string, opts *WriteOptions) (io.WriteCloser, error)

	// Remove removes a object.
	Remove(ctx context.Context, name string) error

	// Close closes the bucket.
	Close() error
}

// WriteOptions provide optional configuration when creating/writing objects.
type WriteOptions struct {
	ContentType string
	Metadata    map[string]string
}

// GetContentType returns a content type.
func (o *WriteOptions) GetContentType() string {
	if o != nil {
		return o.ContentType
	}
	return ""
}

// GetMetadata returns a content type.
func (o *WriteOptions) GetMetadata() map[string]string {
	if o != nil {
		return o.Metadata
	}
	return nil
}

// MetaInfo contains meta information about an object.
type MetaInfo struct {
	Name        string            // base name of the object
	Size        int64             // length of the content in bytes
	ModTime     time.Time         // modification time
	ContentType string            // content type
	Metadata    map[string]string // metadata
}

// Iterator iterates over objects
type Iterator interface {
	// Next advances the cursor to the next position.
	Next() bool
	// Name returns the name at the current cursor position.
	Name() string
	// Size returns the length of the content in bytes for the current object.
	Size() int64
	// ModTime returns the modification time for the current object.
	ModTime() time.Time
	// Error returns the last iterator error, if any.
	Error() error
	// Close closes the iterator, should always be deferred.
	Close() error
}

type supportsCopying interface {
	Copy(context.Context, string, string) error
}

// --------------------------------------------------------------------

var (
	registry     = map[string]Resolver{}
	registryLock sync.Mutex
)

// Resolver constructs a bucket from a URL.
type Resolver func(context.Context, *url.URL) (Bucket, error)

// Resolve opens a bucket from a URL. Example (from bfs/bfsfs):
//
//   bfs.Register("file", func(_ context.Context, u *url.URL) (bfs.Bucket, error) {
//     return bfsfs.New(u.Path, "")
//   })
//
//   u, err := url.Parse("file:///home/user/Documents")
//   ...
//   bucket, err := bfs.Resolve(context.TODO(), u)
//   ...
func Resolve(ctx context.Context, u *url.URL) (Bucket, error) {
	registryLock.Lock()
	resv, ok := registry[u.Scheme]
	registryLock.Unlock()
	if !ok {
		return nil, fmt.Errorf("bfs: unknown URL scheme %q", u.Scheme)
	}

	return resv(ctx, u)
}

// Connect connects to a bucket via URL. Example (from bfs/bfsfs):
//
//   bfs.Register("file", func(_ context.Context, u *url.URL) (bfs.Bucket, error) {
//     return bfsfs.New(u.Path, "")
//   })
//
//   bucket, err := bfs.Connect(context.TODO(), "file:///home/user/Documents")
func Connect(ctx context.Context, urlStr string) (Bucket, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	return Resolve(ctx, u)
}

// Register registers a new protocol with a scheme and a corresponding resolver.
// Example (from bfs/bfsfs):
//
//   bfs.Register("file", func(_ context.Context, u *url.URL) (bfs.Bucket, error) {
//     return bfsfs.New(u.Path, "")
//   })
//
//   bucket, err := bfs.Connect(context.TODO(), "file:///home/user/Documents")
//   ...
func Register(scheme string, resv Resolver) {
	registryLock.Lock()
	defer registryLock.Unlock()

	if _, exists := registry[scheme]; exists {
		panic("protocol " + scheme + " already registered")
	}
	registry[scheme] = resv
}
