package bfsos

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bsm/bfs"
)

// bucket emulates bfs.Bucket behaviour for local file system.
type bucket struct {
	root string
}

// New initiates an bfs.Bucket backed by local file system.
func New(root string) (bfs.Bucket, error) {
	if root == "" {
		root = "."
	}

	return &bucket{
		root: filepath.Clean(root) + string(filepath.Separator), // root should always have trailing slash to trim file names properly
	}, nil
}

// Glob lists the files mathing a glob pattern.
func (b *bucket) Glob(_ context.Context, pattern string) (bfs.Iterator, error) {
	if pattern == "" { // would return just current dir
		return newIterator(nil), nil
	}

	matches := make([]string, 0, 10)

	// filepath.Glob is not suitable, returns dirs as well:
	err := filepath.Walk(b.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}

		name := strings.TrimPrefix(path, b.root)
		if matched, _ := filepath.Match(pattern, name); matched {
			matches = append(matches, name)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return newIterator(matches), nil
}

// Head returns an object's meta Info.
func (b *bucket) Head(ctx context.Context, name string) (*bfs.MetaInfo, error) {
	fi, err := os.Stat(b.resolve(name))
	if err != nil {
		return nil, normError(err)
	}
	return &bfs.MetaInfo{
		Name:    name,
		Size:    fi.Size(),
		ModTime: fi.ModTime(),
	}, nil
}

// Open opens an object for reading.
func (b *bucket) Open(ctx context.Context, name string) (io.ReadCloser, error) {
	f, err := os.Open(b.resolve(name))
	if err != nil {
		return nil, normError(err)
	}
	return f, nil
}

// Create creates/opens a object for writing.
func (b *bucket) Create(ctx context.Context, name string) (io.WriteCloser, error) {
	path := b.resolve(name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, normError(err)
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return nil, normError(err)
	}
	return f, nil
}

// Remove removes a object.
func (b *bucket) Remove(ctx context.Context, name string) error {
	err := os.Remove(b.resolve(name))
	if err != nil && os.IsExist(err) {
		return err
	}
	return nil
}

// Close closes the bucket.
func (b *bucket) Close() error {
	return nil // noop
}

// Copy supports copying of objects within the bucket.
func (b *bucket) Copy(ctx context.Context, srcName, dstName string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	src, err := b.Open(ctx, srcName)
	if err != nil {
		cancel()
		return err
	}
	defer src.Close()

	dst, err := b.Create(ctx, dstName)
	if err != nil {
		cancel()
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		cancel()
		return normError(err)
	}
	return normError(dst.Close())
}

// resolve returns full safely rooted path.
func (b *bucket) resolve(name string) string {
	return filepath.Join(b.root, filepath.Join("/", name))
}
