package fs

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"time"
)

// FileDescriptor is an abstraction of os.File to improve testability of code.
type FileDescriptor interface {
	io.ReadCloser
}

// FileSystem is an abstraction of the file system to improve testability of code.
type FileSystem interface {
	EvalSymlinks(path string) (string, error)
	Open(name string) (FileDescriptor, error)
	Stat(name string) (os.FileInfo, error)
}

type osFileSystem struct {
}

func (fs *osFileSystem) EvalSymlinks(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}

func (fs *osFileSystem) Open(name string) (FileDescriptor, error) {
	return os.Open(name)
}

func (fs *osFileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

var osfs FileSystem = &osFileSystem{}

// OSFileSystem returns a FileSystem instance that is backed by the os.
func OSFileSystem() FileSystem {
	return osfs
}

type mockFileSystem struct {
	data map[string]MockFile
}

// MockFile represents a file in a mock file system. If Error is set then all file system operations will produce an error when this file
// is accessed, otherwise Content is the content of the file.
type MockFile struct {
	Content []byte
	Error   error
}

// MockFileSystem creates a mock file system based on the provided data.
func MockFileSystem(data map[string]MockFile) FileSystem {
	return &mockFileSystem{
		data: data,
	}
}

type mockFileDescriptor struct {
	reader *bytes.Reader
}

func (r *mockFileDescriptor) Read(b []byte) (n int, err error) {
	return r.reader.Read(b)
}

func (r *mockFileDescriptor) Close() error {
	return nil
}

func (fs *mockFileSystem) EvalSymlinks(path string) (string, error) {
	mockFile, ok := fs.data[path]
	if !ok {
		return "", os.ErrNotExist
	}
	if mockFile.Error != nil {
		return "", mockFile.Error
	}
	// Symbolic links are not supported in the mock file system.
	return path, nil
}

func (fs *mockFileSystem) Open(name string) (FileDescriptor, error) {
	mockFile, ok := fs.data[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	if mockFile.Error != nil {
		return nil, mockFile.Error
	}
	return &mockFileDescriptor{
		reader: bytes.NewReader(mockFile.Content),
	}, nil
}

type mockFileInfo struct {
	name string
	size int64
}

func (fileInfo *mockFileInfo) IsDir() bool {
	return false
}

func (fileInfo *mockFileInfo) Mode() os.FileMode {
	return os.ModePerm
}

func (fileInfo *mockFileInfo) ModTime() time.Time {
	return time.Now()
}

func (fileInfo *mockFileInfo) Name() string {
	return fileInfo.name
}

func (fileInfo *mockFileInfo) Size() int64 {
	return fileInfo.size
}

func (fileInfo *mockFileInfo) Sys() interface{} {
	return nil
}

func (fs *mockFileSystem) Stat(name string) (os.FileInfo, error) {
	mockFile, ok := fs.data[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	if mockFile.Error != nil {
		return nil, mockFile.Error
	}
	return &mockFileInfo{
		name: filepath.Base(name),
		size: int64(len(mockFile.Content)),
	}, nil
}
