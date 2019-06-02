package fs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// FileDescriptor is an abstraction of os.File to improve testability of code.
type FileDescriptor interface {
	io.ReadCloser
	Readdir(n int) ([]os.FileInfo, error)
}

// FileSystem is an abstraction of the file system to improve testability of code.
type FileSystem interface {
	EvalSymlinks(path string) (string, error)
	Lstat(name string) (os.FileInfo, error)
	Open(name string) (FileDescriptor, error)
	Stat(name string) (os.FileInfo, error)
}

type osFileSystem struct {
}

func (fs *osFileSystem) EvalSymlinks(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}

func (fs *osFileSystem) Lstat(name string) (os.FileInfo, error) {
	return os.Lstat(name)
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
	root *mockINode
}

type mockINode struct {
	mode os.FileMode
	// if err != nil then err is returned when this node is accessed.
	err error
	// Either mockINodeExtraDir or mockINodeExtraBytes, depending on the type of this node.
	extra interface{}
}

type mockINodeExtraDir = map[string]*mockINode
type mockINodeExtraBytes = []byte

var (
	errMockBadMode                = fmt.Errorf("mock file has a bad mode")
	errMockDirectoryInconsistency = fmt.Errorf("data contains a name X that is not a directory, but another name Y indicates " +
		"that X must be a directory")
	errMockReadNonRegularFile = fmt.Errorf("cannot read non regular file")
)

// returns -1 if an item along the path is not a directory
// panics if the supplied name is not normalized (we do not want to complicate things by requiring normalization)
// otherwise, returns the index of the deepest path component and the parent inode of that component
func (fs *mockFileSystem) find(name string) (node *mockINode, start int) {
	node = fs.root
	// All relative files are relative to the root.
	if name == "" {
		return
	}
	if name == "/" {
		start = 1
		return
	}
	if name[0] == '/' {
		start = 1
	}
	for {
		nameComp := name[start:]
		end := strings.IndexByte(nameComp, '/')
		if end >= 0 {
			end += start
			nameComp = name[start:end]
		}
		validateNameComp(nameComp)
		if end < 0 {
			return node, start
		}
		if (node.mode & os.ModeDir) == 0 {
			return node, -1
		}
		dir := node.extra.(mockINodeExtraDir)
		childNode := dir[nameComp]
		if childNode == nil {
			return node, start
		}
		node = childNode
		start = end + 1
	}
}

func (fs *mockFileSystem) findOrError(name string) (*mockINode, error) {
	node := fs.root
	// All relative files are relative to the root.
	if name == "" {
		return node, node.err
	}
	start := 0
	if name[0] == '/' {
		start = 1
	}
	for start < len(name) {
		if node.err != nil {
			return node, node.err
		}
		nameComp := name[start:]
		end := strings.IndexByte(nameComp, '/')
		if end >= 0 {
			end += start
			nameComp = name[start:end]
		}
		validateNameComp(nameComp)
		if (node.mode & os.ModeDir) == 0 {
			return node, syscall.ENOTDIR
		}
		dir := node.extra.(mockINodeExtraDir)
		childNode := dir[nameComp]
		if childNode == nil {
			return node, os.ErrNotExist
		}
		node = childNode
		start = end + 1
	}
	return node, node.err
}

func validateNameComp(nameComp string) {
	if nameComp == "" || nameComp == "." || nameComp == ".." {
		panic(fmt.Errorf("name must not contain '//' and must not have a path component that is one of  '..' and '.'"))
	}
}

func (fs *mockFileSystem) createChildren(node *mockINode, name string, mockFile MockFile) {
	start := 0
	for {
		nameComp := name[start:]
		end := strings.IndexByte(nameComp, '/')
		if end >= 0 {
			end += start
			nameComp = name[start:end]
		}
		validateNameComp(nameComp)
		var childNode *mockINode
		if end >= 0 {
			// initialize directory with defaults
			childNode = &mockINode{
				mode:  os.ModeDir,
				extra: mockINodeExtraDir{},
			}
			node.extra.(mockINodeExtraDir)[nameComp] = childNode
			node = childNode
		} else {
			// initialize file or directory as per MockFile
			childNode = &mockINode{
				err:  mockFile.Error,
				mode: mockFile.Mode,
			}
			if (mockFile.Mode & os.ModeDir) != 0 {
				childNode.extra = mockINodeExtraDir{}
			} else {
				childNode.extra = mockFile.Content
			}
			node.extra.(mockINodeExtraDir)[nameComp] = childNode
			return
		}
		start = end + 1
	}
}

// MockFile is a helper struct used to initialize a file or directory in a mock file system. If Error is set then all file system
// operations will produce an error when this file is accessed. If Mode is a regular file then Content is the content of that file.
// If Mode is not a regular file or directory then an error is produced in MockFileSystem.
type MockFile struct {
	Content []byte
	Mode    os.FileMode
	Error   error
}

// MockFileSystem creates a mock file system based on the provided data.
func MockFileSystem(data map[string]MockFile) FileSystem {
	fs := &mockFileSystem{
		root: &mockINode{
			mode:  os.ModeDir,
			extra: mockINodeExtraDir{},
		},
	}
	for name, mockFile := range data {
		var flag os.FileMode
		switch {
		case mockFile.Mode.IsDir():
			flag = os.ModeDir
		case (mockFile.Mode & os.ModeSymlink) != 0:
			flag = os.ModeSymlink
		case mockFile.Mode.IsRegular():
			flag = 0
		}
		if (mockFile.Mode & (os.ModeType &^ flag)) != 0 {
			panic(errMockBadMode)
		}
		node, start := fs.find(name)
		if start == -1 {
			panic(errMockDirectoryInconsistency)
		}
		if start < len(name) {
			fs.createChildren(node, name[start:], mockFile)
		} else {
			nodeIsDir := (node.mode & os.ModeDir) != 0
			mockFileIsDir := (mockFile.Mode & os.ModeDir) != 0
			if nodeIsDir != mockFileIsDir {
				panic(errMockDirectoryInconsistency)
			}
			node.mode = mockFile.Mode
			node.err = mockFile.Error
			if !mockFileIsDir {
				node.extra = mockFile.Content
			}
		}
	}
	return fs
}

type mockFileDescriptor struct {
	node    *mockINode
	readPos int
}

func (r *mockFileDescriptor) Close() error {
	return nil
}

func (r *mockFileDescriptor) Read(p []byte) (n int, err error) {
	if !r.node.mode.IsRegular() {
		err = errMockReadNonRegularFile
		return
	}
	if len(p) == 0 {
		return
	}
	fileContents := r.node.extra.(mockINodeExtraBytes)
	n = copy(p, fileContents[r.readPos:])
	r.readPos += n
	if n == 0 {
		err = io.EOF
	}
	return
}

func (r *mockFileDescriptor) Readdir(n int) ([]os.FileInfo, error) {
	if !r.node.mode.IsDir() {
		return nil, syscall.ENOTDIR
	}
	if n > 0 {
		panic(fmt.Errorf("not supported"))
	}
	var ret []os.FileInfo
	for nameComp, childNode := range r.node.extra.(mockINodeExtraDir) {
		ret = append(ret, newMockFileInfo(childNode, nameComp))
	}
	return ret, nil
}

func (fs *mockFileSystem) EvalSymlinks(path string) (string, error) {
	_, err := fs.findOrError(path)
	if err != nil {
		return "", err
	}
	// Symbolic links are not supported in the mock file system.
	return path, nil
}

func (fs *mockFileSystem) Lstat(name string) (os.FileInfo, error) {
	// Symbolic links are not supported in the mock file system.
	return fs.Stat(name)
}

func (fs *mockFileSystem) Open(name string) (FileDescriptor, error) {
	node, err := fs.findOrError(name)
	if err != nil {
		return nil, err
	}
	return &mockFileDescriptor{
		node: node,
	}, nil
}

type mockFileInfo struct {
	mode os.FileMode
	name string
	size int64
}

func (fileInfo *mockFileInfo) IsDir() bool {
	return fileInfo.mode.IsDir()
}

func (fileInfo *mockFileInfo) Mode() os.FileMode {
	return fileInfo.mode
}

func (fileInfo *mockFileInfo) ModTime() time.Time {
	return time.Time{}
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

func newMockFileInfo(node *mockINode, nameComp string) *mockFileInfo {
	fileInfo := &mockFileInfo{
		mode: node.mode,
		name: nameComp,
	}
	if node.mode.IsRegular() {
		fileInfo.size = int64(len(node.extra.(mockINodeExtraBytes)))
	}
	return fileInfo
}

func (fs *mockFileSystem) Stat(name string) (os.FileInfo, error) {
	node, err := fs.findOrError(name)
	if err != nil {
		return nil, err
	}
	i := strings.LastIndexByte(name, '/')
	return newMockFileInfo(node, name[i+1:]), nil
}
