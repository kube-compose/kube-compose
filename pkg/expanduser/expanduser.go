package expanduser

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	fsPackage "github.com/kube-compose/kube-compose/internal/pkg/fs"
	"github.com/kube-compose/kube-compose/internal/pkg/unix"
)

var isNT = runtime.GOOS == "windows"
var getUID = os.Getuid

// LookupEnvFunc is a function used to get environment variables. It is useful for unit testing.
var LookupEnvFunc = os.LookupEnv

// ExpandUser expands a tidle prefix of the specified path by calling either
// Posix or NT.
func ExpandUser(path string) string {
	if isNT {
		return NT(path)
	}
	return Posix(path)
}

// Home returns the home directory by calling either HomeNT or HomePosix.
func Home() (string, error) {
	if isNT {
		return HomeNT()
	}
	return HomePosix()
}

// HomePosix returns the home directory of the current user using non-Windows semantics. If the HOME environment variable is present it
// returns its value. Otherwise, it looks for an entry in the /etc/passwd file with at least 6 fields and a uid equal to os.Getuid(). If
// such an entry is found then returns the 6th field of the entry. If the before mentioned return conditions cannot be met an error is
// returned.
func HomePosix() (string, error) {
	home, ok := LookupEnvFunc("HOME")
	if !ok {
		uid := getUID()
		if uid < 0 {
			return "", fmt.Errorf("not available on Windows")
		}
		var err error
		home, err = unix.FindHomeByUIDInPasswd(unix.EtcPasswd, int64(uid))
		if err != nil {
			return "", err
		}
	}
	return home, nil
}

// Posix expands a tidle prefix of the specified path using posix semantics.
// Ported from expanduser https://github.com/python/cpython/blob/master/Lib/posixpath.py.
func Posix(path string) string {
	if path == "" || path[0] != '~' {
		return path
	}
	i := 1
	for i < len(path) && path[i] != '/' {
		i++
	}
	var err error
	var userhome string
	if i == 1 {
		userhome, err = HomePosix()
		if err != nil {
			// Ignore error here
			return path
		}
	} else {
		name := path[1:i]
		userhome, err = unix.FindHomeByNameInPasswd(unix.EtcPasswd, name)
		if err != nil {
			// Ignore error here
			return path
		}
	}
	path = strings.TrimSuffix(userhome, "/") + path[i:]
	if path == "" {
		path = "/"
	}
	return path
}

// HomeNT returns the home directory of the current user using Windows semantics.
func HomeNT() (string, error) {
	userhome, ok := LookupEnvFunc("USERPROFILE")
	if !ok {
		homepath, ok := LookupEnvFunc("HOMEPATH")
		if !ok {
			return "", fmt.Errorf("the environment variables USERPROFILE and HOMEPATH are both unset")
		}
		homedrive, _ := LookupEnvFunc("HOMEDRIVE")
		// This behaves slightly differently than the Python implementation because
		// Go's Join will also clean the path, whereas Python's join does not.
		// This is ok.
		userhome = filepath.Join(homedrive, homepath)
	}
	return userhome, nil
}

// NT expands a tidle prefix of the specified path using Windows semantics.
// Ported from expanduser in https://github.com/python/cpython/blob/master/Lib/ntpath.py.
func NT(path string) string {
	if path == "" || path[0] != '~' {
		return path
	}
	i := 1
	for i < len(path) && !fsPackage.NTIsPathSeparator(path[i]) {
		i++
	}
	userhome, err := HomeNT()
	if err != nil {
		return path
	}
	if i > 1 {
		j := dirname(userhome)
		userhome = userhome[:j] + path[1:i]
	}
	return userhome + path[i:]
}

func dirname(name string) int {
	volLen := fsPackage.NTVolumeNameLength(name)
	j := len(name)
	for j > volLen && fsPackage.NTIsPathSeparator(name[j-1]) {
		j--
	}
	for j > volLen && !fsPackage.NTIsPathSeparator(name[j-1]) {
		j--
	}
	return j
}
