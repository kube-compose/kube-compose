package fs

import (
	"fmt"
	"os"
	"syscall"
	"testing"
)

func Test_Lstat_RootSuccess(t *testing.T) {
	fs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{})
	fileInfo, err := fs.Lstat("")
	if err != nil {
		t.Error(err)
	} else if fileInfo.Name() != "/" {
		t.Fail()
	}
}

func Test_Lstat_InjectedFault1(t *testing.T) {
	fs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{})
	errExpected := fmt.Errorf("injectedFault1")
	fs.root.err = errExpected
	_, errActual := fs.Lstat("")
	if errActual != errExpected {
		t.Fail()
	}
}

func Test_Lstat_InjectedFault2(t *testing.T) {
	errExpected := fmt.Errorf("injectedFault2")
	fs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{
		"/dir": {
			Error: errExpected,
			Mode:  os.ModeDir,
		},
	})
	fs.Set("/dir/file", &InMemoryFile{})
	_, errActual := fs.Lstat("/dir/file")
	if errActual != errExpected {
		t.Fail()
	}
}

func Test_Lstat_ENOTDIR(t *testing.T) {
	fs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{
		"/notadir": {},
	})
	_, err := fs.Lstat("/notadir/file")
	if err != syscall.ENOTDIR {
		t.Fail()
	}
}

func Test_Lstat_ENOENT(t *testing.T) {
	fs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{})
	_, err := fs.Lstat("/doesnotexist")
	if !os.IsNotExist(err) {
		t.Fail()
	}
}
