package linux

import (
	"strings"
	"testing"

	fsPackage "github.com/jbrekelmans/kube-compose/internal/pkg/fs"
)

var mockFileSystem = fsPackage.MockFileSystem(map[string][]byte{
	"/passwd": []byte(""),
})

func TestFindUserInPasswd(t *testing.T) {
	fsOld := fs
	defer func() {
		fs = fsOld
	}()
	fs = mockFileSystem
	_, _ = FindUserInPasswd("/passwd", "")
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
