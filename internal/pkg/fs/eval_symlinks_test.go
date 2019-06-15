package fs

import (
	"fmt"
	"os"
	"syscall"
	"testing"
)

func Test_VirtualFileSystem_EvalSymlinks_AbsRootInjectedFault(t *testing.T) {
	fs := NewInMemoryFileSystem(map[string]InMemoryFile{})
	errExpected := fmt.Errorf("absRootInjectedFault")
	fs.root.err = errExpected
	_, errActual := fs.EvalSymlinks("/")
	if errActual != errExpected {
		t.Fail()
	}
}

func Test_VirtualFileSystem_EvalSymlinks_RelCwdInjectedFault(t *testing.T) {
	fs := NewInMemoryFileSystem(map[string]InMemoryFile{})
	errExpected := fmt.Errorf("relCwdInjectedFault")
	fs.root.err = errExpected
	_, errActual := fs.EvalSymlinks("")
	if errActual != errExpected {
		t.Fail()
	}
}

func Test_VirtualFileSystem_EvalSymlinks_ENOTDIR(t *testing.T) {
	fs := NewInMemoryFileSystem(map[string]InMemoryFile{
		"notadir": {
			Content: []byte("notadircontent"),
		},
	})
	_, err := fs.EvalSymlinks("notadir/huh")
	if err != syscall.ENOTDIR {
		t.Fail()
	}
}

func Test_VirtualFileSystem_EvalSymlinks_ENOENT(t *testing.T) {
	fs := NewInMemoryFileSystem(map[string]InMemoryFile{})
	_, err := fs.EvalSymlinks("doesnotexist")
	if !os.IsNotExist(err) {
		t.Fail()
	}
}
func Test_VirtualFileSystem_EvalSymlinks_NonRootInjectedFault(t *testing.T) {
	errExpected := fmt.Errorf("nonRootInjectedFault")
	fs := NewInMemoryFileSystem(map[string]InMemoryFile{
		"child": {
			Error: errExpected,
		},
	})
	_, errActual := fs.EvalSymlinks("child")
	if errActual != errExpected {
		t.Fail()
	}
}

func Test_VirtualFileSystem_EvalSymlinks_AbsTooManyLinks(t *testing.T) {
	fs := NewInMemoryFileSystem(map[string]InMemoryFile{
		"selflink": {
			Content: []byte("selflink"),
			Mode:    os.ModeSymlink,
		},
	})
	_, err := fs.EvalSymlinks("/selflink")
	if err != errTooManyLinks {
		t.Fail()
	}
}

func Test_VirtualFileSystem_EvalSymlinks_Success(t *testing.T) {
	fs := NewInMemoryFileSystem(map[string]InMemoryFile{
		"/dir1/link1": {
			Content: []byte("/link2"),
			Mode:    os.ModeSymlink,
		},
		"/link2": {
			Content: []byte("dir2"),
			Mode:    os.ModeSymlink,
		},
		"/dir2/file": {},
	})
	resolved, err := fs.EvalSymlinks("/dir1/link1/file")
	if err != nil {
		t.Error(err)
	} else if resolved != "/dir2/file" {
		t.Logf("%s\n", resolved)
		t.Fail()
	}
}
