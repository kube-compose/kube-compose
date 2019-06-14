package unix

import (
	"strings"
	"testing"

	"github.com/kube-compose/kube-compose/internal/pkg/fs"
)

var vfs fs.FileSystem = fs.NewVirtualFileSystem(map[string]fs.VirtualFile{
	EtcPasswd: {
		Content: []byte("root:x:0:\ndaemon:x:1:1:daemon:/daemonhomelol\nasdf\nuiderr:x::"),
	},
})

func TestFindUIDByNameInPasswd_Success(t *testing.T) {
	fs.ScopedFS(vfs, func() {
		_, _ = FindUIDByNameInPasswd(EtcPasswd, "")
	})
}
func TestFindUIDByNameInPasswd_ENOENT(t *testing.T) {
	fs.ScopedFS(vfs, func() {
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

func TestFindHomeByUIDInPasswd_Success(t *testing.T) {
	fs.ScopedFS(vfs, func() {
		home, err := FindHomeByUIDInPasswd(EtcPasswd, 1)
		if err != nil {
			t.Error(err)
		} else if home != "/daemonhomelol" {
			t.Fail()
		}
	})
}

func TestFindHomeByUIDInPasswd_ErrorUIDInvalidFormat(t *testing.T) {
	fs.ScopedFS(vfs, func() {
		_, err := FindHomeByUIDInPasswd(EtcPasswd, 5)
		if err == nil {
			t.Fail()
		}
	})
}

func TestFindHomeByNameInPasswd_Success(t *testing.T) {
	fs.ScopedFS(vfs, func() {
		_, err := FindHomeByNameInPasswd(EtcPasswd, "notfoundtest")
		if err != nil {
			t.Error(err)
		}
	})
}
