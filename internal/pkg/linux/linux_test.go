package linux

import (
	"strings"
	"testing"

	fsPackage "github.com/kube-compose/kube-compose/internal/pkg/fs"
)

var mockFileSystem fsPackage.FileSystem = fsPackage.NewMockFileSystem(map[string]fsPackage.MockFile{
	"/passwd": {
		Content: []byte("root:x:0:"),
	},
})

func withMockFS(cb func()) {
	fsOld := fs
	defer func() {
		fs = fsOld
	}()
	fs = mockFileSystem
	cb()
}

func TestFindUIDByNameInPasswd_Success(t *testing.T) {
	withMockFS(func() {
		_, _ = FindUIDByNameInPasswd("/passwd", "")
	})
}
func TestFindUIDByNameInPasswd_ENOENT(t *testing.T) {
	withMockFS(func() {
		_, err := FindUIDByNameInPasswd("/asdf", "")
		if err == nil {
			t.Fail()
		}
	})
}

func TestFindUIDByNameInPasswdReader_Success(t *testing.T) {
	reader := strings.NewReader("root:x:0:\nbin:x:1:")
	_, err := FindUIDByNameInPasswdReader(reader, "bin")
	if err != nil {
		t.Fail()
	}
}
func TestFindUIDByNameInPasswdReader_NotFound(t *testing.T) {
	reader := strings.NewReader("root:x:0:\nbin:x:1:")
	_, err := FindUIDByNameInPasswdReader(reader, "henk")
	if err != nil {
		t.Fail()
	}
}

func TestFindUIDByNameInPasswdReader_InvalidUID(t *testing.T) {
	reader := strings.NewReader("root:x:0:\nbin:x:-1:")
	_, err := FindUIDByNameInPasswdReader(reader, "bin")
	if err == nil {
		t.Fail()
	}
}

func TestFindUIDByNameInPasswdReader_InvalidFormat(t *testing.T) {
	reader := strings.NewReader("root")
	_, err := FindUIDByNameInPasswdReader(reader, "root")
	if err == nil {
		t.Fail()
	}
}
