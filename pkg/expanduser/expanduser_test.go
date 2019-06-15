package expanduser

import (
	"testing"

	"github.com/kube-compose/kube-compose/internal/pkg/fs"
	"github.com/kube-compose/kube-compose/internal/pkg/unix"
)

func TestNT_Noop(t *testing.T) {
	if NT("") != "" {
		t.Fail()
	}
}

func withMockEnv(env map[string]string, cb func()) {
	orig := LookupEnvFunc
	defer func() {
		LookupEnvFunc = orig
	}()
	LookupEnvFunc = func(name string) (string, bool) {
		value, ok := env[name]
		return value, ok
	}
	cb()
}

func TestNT_ExplicitUser(t *testing.T) {
	withMockEnv(map[string]string{
		"USERPROFILE": "\\Test\\Users\\Henk\\",
	}, func() {
		actual := NT("~Klaas\\.bash_profile")
		if actual != "\\Test\\Users\\Klaas\\.bash_profile" {
			t.Log(actual)
			t.Fail()
		}
	})
}
func TestNT_Error(t *testing.T) {
	withMockEnv(map[string]string{}, func() {
		if NT("~") != "~" {
			t.Fail()
		}
	})
}
func TestHomeNT_Error(t *testing.T) {
	withMockEnv(map[string]string{}, func() {
		_, err := HomeNT()
		if err == nil {
			t.Fail()
		}
	})
}
func TestHomeNT_SuccessAlternative(t *testing.T) {
	withMockEnv(map[string]string{
		"HOMEPATH":  "\\Users\\Henk",
		"HOMEDRIVE": "C:",
	}, func() {
		expected := "C:/\\Users\\Henk"
		actual, err := HomeNT()
		if err != nil {
			t.Error(err)
		} else if actual != expected {
			t.Log(actual)
			t.Log(expected)
			t.Fail()
		}
	})
}

func withMockEtcPasswd(s string, cb func()) {
	original := fs.OS
	defer func() {
		fs.OS = original
	}()
	fs.OS = fs.NewInMemoryFileSystem(map[string]fs.InMemoryFile{
		unix.EtcPasswd: {
			Content: []byte(s),
		},
	})
	cb()
}

func TestPosix_Error(t *testing.T) {
	withMockUID(-1, func() {
		withMockEnv(map[string]string{}, func() {
			withMockEtcPasswd("henk:x:1:1:henkie penkie:/home/henkiehome", func() {
				actual := Posix("~/")
				if actual != "~/" {
					t.Fail()
				}
			})
		})
	})
}

func TestPosix_NeverEmpty(t *testing.T) {
	withMockEnv(map[string]string{
		"HOME": "",
	}, func() {
		actual := Posix("~")
		if actual != "/" {
			t.Fail()
		}
	})
}

func TestPosix_ExplicitUserCase1(t *testing.T) {
	withMockEtcPasswd("henk:x:1:1:henkie penkie:/home/henkiehome", func() {
		actual := Posix("~henk")
		if actual != "/home/henkiehome" {
			t.Fail()
		}
	})
}

func TestPosix_ExplicitUserCase2(t *testing.T) {
	withMockEtcPasswd("henk:x:1", func() {
		expected := "~henk"
		actual := Posix(expected)
		if actual != expected {
			t.Fail()
		}
	})
}

func TestPosix_ExplicitUserCase3(t *testing.T) {
	withMockEtcPasswd("root:x:0:0::/", func() {
		expected := "/henk"
		actual := Posix("~root/henk")
		if actual != expected {
			t.Fail()
		}
	})
}

func withMockUID(uid int, cb func()) {
	orig := getUID
	defer func() {
		getUID = orig
	}()
	getUID = func() int {
		return uid
	}
	cb()
}

func TestHomePosix_ErrorWindows(t *testing.T) {
	withMockEnv(map[string]string{}, func() {
		withMockUID(-1, func() {
			_, err := HomePosix()
			if err == nil {
				t.Fail()
			}
		})
	})
}
func TestHomePosix_ErrorInvalidEtcPasswd(t *testing.T) {
	withMockEnv(map[string]string{}, func() {
		withMockUID(0, func() {
			withMockEtcPasswd("root:x:0", func() {
				_, err := HomePosix()
				if err == nil {
					t.Fail()
				}
			})
		})
	})
}

func withMockNT(nt bool, cb func()) {
	orig := isNT
	defer func() {
		isNT = orig
	}()
	isNT = nt
	cb()
}

func TestHome_PosixSuccess(t *testing.T) {
	withMockNT(false, func() {
		expected := "/testhome/posixsuccess"
		withMockEnv(map[string]string{
			"HOME": expected,
		}, func() {
			actual, err := Home()
			if err != nil {
				t.Error(err)
			} else if actual != expected {
				t.Fail()
			}
		})
	})
}
func TestHome_WindowsSuccess(t *testing.T) {
	withMockNT(true, func() {
		expected := "C:\\Users\\windowssuccess"
		withMockEnv(map[string]string{
			"USERPROFILE": expected,
		}, func() {
			actual, err := Home()
			if err != nil {
				t.Error(err)
			} else if actual != expected {
				t.Fail()
			}
		})
	})
}

func TestExpandUser_NTSuccess(t *testing.T) {
	withMockNT(true, func() {
		withMockEnv(map[string]string{
			"USERPROFILE": "\\home\\user1",
		}, func() {
			actual := ExpandUser("~user2\\.bash_profile")
			if actual != "\\home\\user2\\.bash_profile" {
				t.Log(actual)
				t.Fail()
			}
		})
	})
}
