package fs

import (
	"fmt"
	"os"
	"syscall"
	"testing"
)

func Test_VirtualFileSystem_Mkdir_Success(t *testing.T) {
	fs := NewVirtualFileSystem(map[string]VirtualFile{})
	name := "mkdirSuccess"
	err := fs.Mkdir(name, os.ModePerm)
	if err != nil {
		t.Error(err)
	} else {
		fileInfo, err := fs.Stat(name)
		if err != nil {
			t.Error(err)
		} else if !fileInfo.IsDir() || fileInfo.Name() != name {
			t.Fail()
		}
	}
}

func Test_VirtualFileSystem_Mkdir_ErrorBadMode(t *testing.T) {
	fs := NewVirtualFileSystem(map[string]VirtualFile{})
	err := fs.Mkdir("errbadmode", os.ModeSymlink)
	if err == nil {
		t.Fail()
	}
}
func Test_VirtualFileSystem_Mkdir_ErrorInjectedFault(t *testing.T) {
	fs := NewVirtualFileSystem(map[string]VirtualFile{})
	errExpected := fmt.Errorf("injectedFault")
	fs.root.err = errExpected
	errActual := fs.Mkdir("errinjectedfault", 0)
	if errActual != errExpected {
		t.Fail()
	}
}
func Test_VirtualFileSystem_Mkdir_ENOENT(t *testing.T) {
	fs := NewVirtualFileSystem(map[string]VirtualFile{})
	err := fs.Mkdir("asdf/asdf", 0)
	if !os.IsNotExist(err) {
		t.Fail()
	}
}

func Test_VirtualFileSystem_Mkdir_EEXIST(t *testing.T) {
	fs := NewVirtualFileSystem(map[string]VirtualFile{})
	err := fs.Mkdir("/", 0)
	if !os.IsExist(err) {
		t.Fail()
	}
}

func Test_VirtualFileSystem_MkdirAll_ENOTDIR(t *testing.T) {
	fs := NewVirtualFileSystem(map[string]VirtualFile{
		"file": {
			Content: []byte("filecontent"),
		},
	})
	err := fs.MkdirAll("file", 0)
	if err != syscall.ENOTDIR {
		t.Fail()
	}
}

func Test_VirtualFileSystem_MkdirAll_Success(t *testing.T) {
	fs := NewVirtualFileSystem(map[string]VirtualFile{})
	name := "asdf/asdf"
	err := fs.MkdirAll(name, os.ModePerm)
	if err != nil {
		t.Error(err)
	} else {
		fileInfo, err := fs.Stat(name)
		if err != nil {
			t.Error(err)
		} else if !fileInfo.IsDir() || fileInfo.Name() != "asdf" {
			t.Fail()
		}
	}
}
