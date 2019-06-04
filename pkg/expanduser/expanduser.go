package expanduser

import (
	"fmt"
	"path/filepath"
	"os"
	"runtime"
	"strings"

	fsPackage "github.com/kube-compose/kube-compose/internal/pkg/fs"
	"github.com/kube-compose/kube-compose/internal/pkg/linux"
)

var goos = runtime.GOOS

// ExpandUser expands a tidle prefix of the specified path by calling either
// Posix or NT.
func ExpandUser(path string) string {
	if goos == "windows" {
		return NT(path)
	}
	return Posix(path)
}

// Home returns the home directory by calling either HomeNT or HomePosix.
func Home() (string, error) {
	if goos == "windows" {
		return HomeNT()
	}
	return HomePosix()
}

// HomePosix returns the home directory of the current user using non-Windows semantics. If the HOME environment variable is present it
// returns its value. Otherwise, it looks for an entry with at least 6 fields and a uid equal to os.Getuid() in the /etc/passwd file. If
// such an entry is found then returns the 6th field of the entry. An error is returned if and only if before mentioned return conditions
// were not met.
func HomePosix() (string, error) {
	home, ok := os.LookupEnv("HOME")
	if !ok {
		uid := os.Getuid()
		if uid < 0 {
			return "", fmt.Errorf("not available on Windows")
		}
		var err error
		home, err = linux.FindHomeByUIDInPasswd(linux.EtcPasswd, int64(uid))
		if err != nil {
			return "", err
		}
	}
	return home, nil
}

// Posix expands a tidle prefix of the specified path using posix semantics.
// Ported from expanduser https://github.com/python/cpython/blob/master/Lib/posixpath.py.
func Posix(path string) string {
	if path == "" || path[0] == '~' {
		return path
	}
	i := 1
	for i < len(path) && path[i] != os.PathSeparator {
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
		userhome, err = linux.FindHomeByNameInPasswd(linux.EtcPasswd, name)
		if err != nil {
			// Ignore error here
			return path
		}
	}
	path = strings.TrimSuffix(userhome, string(os.PathSeparator)) + path[i:]
	if path == "" {
		path = string(os.PathSeparator)
	}
	return path
}

// HomeNT returns the home directory of the current user using Windows semantics.
func HomeNT() (string, error) {
	userhome, ok := os.LookupEnv("USERPROFILE")
	if !ok {
		homepath, ok := os.LookupEnv("HOMEPATH")
		if !ok {
			return "", fmt.Errorf("the environment variables USERPROFILE and HOMEPATH are both unset")
		}
		homedrive := os.Getenv("HOMEDRIVE")
		// This behaves sligthly differently than the Python implementation because
		// Go's Join will also clean the path, whereas Python's join does not.
		// This is ok.
		userhome = filepath.Join(homedrive, homepath)
	}
	return userhome, nil
}

// NT expands a tidle prefix of the specified path using Windows semantics.
// Ported from expanduser in https://github.com/python/cpython/blob/master/Lib/ntpath.py.
func NT(path string) string {
	if path == "" || path[0] == '~' {
		return path
	}
	i := 1
	for i < len(path) && !fsPackage.IsPathSeparatorWindows(path[i]) {
		i++
	}
	userhome, err := HomeNT()
	if err != nil {
		return path
	}
	if i > 1 {
		// Go's Dir might not behave exactly the same as Python's dirname, but that's ok.
		userhome = filepath.Join(filepath.Dir(userhome), path[1:i])
	}
	return userhome + path[i:]
}
