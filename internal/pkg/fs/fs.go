package fs

import (
	"bytes"
	"io"
	"os"
)

// File is an abstraction of os.File to improve testability of code.
type File interface {
	io.ReadCloser
}

// FileSystem is an abstraction of the file system to improve testability of code.
type FileSystem interface {
	Open(name string) (File, error)
}

type osFileSystem struct {
}

func (fs *osFileSystem) Open(name string) (File, error) {
	return os.Open(name)
}

var osfs FileSystem = &osFileSystem{}

// OSFileSystem returns a FileSystem instance that is backed by the os.
func OSFileSystem() FileSystem {
	return osfs
}

type mockFileSystem struct {
	data map[string][]byte
}

func MockFileSystem(data map[string][]byte) FileSystem {
	return &mockFileSystem{
		data: data,
	}
}

type mockFile struct {
	reader *bytes.Reader
}

func (r *mockFile) Read(b []byte) (n int, err error) {
	return r.reader.Read(b)
}

func (r *mockFile) Close() error {
	return nil
}

func (fs *mockFileSystem) Open(name string) (File, error) {
	data, ok := fs.data[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	return &mockFile{
		reader: bytes.NewReader(data),
	}, nil
}
