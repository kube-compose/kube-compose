package linux

import (
	"strings"
	"testing"

	fsPackage "github.com/jbrekelmans/kube-compose/internal/pkg/fs"
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

func TestFindUserInPasswd_Success(t *testing.T) {
	withMockFS(func() {
		_, _ = FindUserInPasswd("/passwd", "")
	})
}
func TestFindUserInPasswd_ENOENT(t *testing.T) {
	withMockFS(func() {
		_, err := FindUserInPasswd("", "")
		if err == nil {
			t.Fail()
		}
	})
}

func TestFindUserInPasswdReader_Success(t *testing.T) {
	reader := strings.NewReader("root:x:0:\nbin:x:1:")
	_, err := FindUserInPasswdReader(reader, "bin")
	if err != nil {
		t.Fail()
	}
}
func TestFindUserInPasswdReader_NotFound(t *testing.T) {
	reader := strings.NewReader("root:x:0:\nbin:x:1:")
	_, err := FindUserInPasswdReader(reader, "henk")
	if err != nil {
		t.Fail()
	}
}

func TestFindUserInPasswdReader_InvalidUID(t *testing.T) {
	reader := strings.NewReader("root:x:0:\nbin:x:-1:")
	_, err := FindUserInPasswdReader(reader, "bin")
	if err == nil {
		t.Fail()
	}
}

func TestFindUserInPasswdReader_InvalidFormat(t *testing.T) {
	reader := strings.NewReader("root")
	_, err := FindUserInPasswdReader(reader, "root")
	if err == nil {
		t.Fail()
	}
}
