package fs

import (
	"os"
	"syscall"
	"testing"
)

func Test_OSFileSystem_Chdir(t *testing.T) {
	_ = OS.Chdir("/")
}

func Test_VirtualFileSystem_Chdir_Success(t *testing.T) {
	name := "/chdirsuccess"
	vfs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{
		name: {
			Mode: os.ModeDir,
		},
	})
	err := vfs.Chdir(name)
	if err != nil {
		t.Error(err)
	} else if vfs.cwd != name {
		t.Fail()
	}
}

func Test_VirtualFileSystem_Chdir_ENOTDIR(t *testing.T) {
	name := "/chdirenotdir"
	vfs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{
		name: {},
	})
	err := vfs.Chdir(name)
	if err != syscall.ENOTDIR {
		t.Fail()
	}
}

func Test_VirtualFileSystem_Chdir_ENOENT(t *testing.T) {
	name := "/chdirenoent"
	vfs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{})
	err := vfs.Chdir(name)
	if !os.IsNotExist(err) {
		t.Fail()
	}
}
