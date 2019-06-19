package fs

import (
	"fmt"
	"os"
	"testing"
)

func Test_Readlink_Success(t *testing.T) {
	targetExpected := "successtarget"
	fs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{
		"/success": {
			Mode:    os.ModeSymlink,
			Content: []byte(targetExpected),
		},
	})
	targetActual, err := fs.Readlink("/success")
	if err != nil {
		t.Error(err)
	} else if targetActual != targetExpected {
		t.Fail()
	}
}
func Test_Readlink_ErrorNotSymlink(t *testing.T) {
	fs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{
		"/errornotsymlink": {},
	})
	_, err := fs.Readlink("/errornotsymlink")
	if err != errBadMode {
		t.Fail()
	}
}
func Test_Readlink_ErrorInjectedFault(t *testing.T) {
	errExpected := fmt.Errorf("errorInjectedFault")
	fs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{})
	fs.root.err = errExpected
	_, errActual := fs.Readlink("")
	if errActual != errExpected {
		t.Fail()
	}
}

func Test_Readlink_ErrorRead(t *testing.T) {
	errExpected := fmt.Errorf("readlinkError")
	fs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{
		"readlinkerror": {
			Content:   []byte("readlinkerror"),
			Mode:      os.ModeSymlink,
			ReadError: errExpected,
		},
	})
	_, errActual := fs.Readlink("readlinkerror")
	if errActual != errExpected {
		t.Fail()
	}
}
