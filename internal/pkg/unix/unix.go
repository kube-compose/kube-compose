package unix

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/kube-compose/kube-compose/internal/pkg/fs"
	"github.com/kube-compose/kube-compose/internal/pkg/util"
)

const EtcPasswd = "/etc/passwd"

// FindUIDByNameInPasswd finds the UID of a user by name in an /etc/passwd file. It can also find the GID of a group by name in an
// /etc/group file.
func FindUIDByNameInPasswd(file, user string) (*int64, error) {
	uid := new(int64)
	err := findCommon(file, findUIDByNameCallback(user, uid))
	if err != nil {
		return nil, err
	}
	return uid, nil
}

// FindUIDByNameInPasswdReader finds the UID of a user by name in a stream encoded like the contents of /etc/passwd. It can also find the
// GID of a group by name in a stream encoded like the contents of /etc/group.
func FindUIDByNameInPasswdReader(reader io.Reader, user string) (*int64, error) {
	uid := new(int64)
	err := findCommonReader(reader, findUIDByNameCallback(user, uid))
	if err != nil {
		return nil, err
	}
	return uid, nil
}

// FindHomeByUIDInPasswd finds the home directory of a user by the user's uid in an /etc/passwd file.
func FindHomeByUIDInPasswd(file string, uid int64) (string, error) {
	var home string
	err := findCommon(file, func(line string) error {
		parts := strings.SplitN(line, ":", 7)
		if len(parts) < 3 {
			return nil
		}
		uidString := parts[2]
		uidLocal := util.TryParseInt64(uidString)
		if uidLocal == nil {
			return errUnexpectedFileFormat
		}
		if *uidLocal == uid {
			if len(parts) < 6 {
				return errUnexpectedFileFormat
			}
			home = parts[5]
			return errFindCommonBreak
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return home, nil
}

// FindHomeByNameInPasswd finds the home directory of a user by the user's name in an /etc/passwd file.
func FindHomeByNameInPasswd(file, name string) (string, error) {
	var home string
	err := findCommon(file, func(line string) error {
		parts := strings.SplitN(line, ":", 7)
		if parts[0] == name {
			if len(parts) < 6 {
				return errUnexpectedFileFormat
			}
			home = parts[5]
			return errFindCommonBreak
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return home, nil
}

func findUIDByNameCallback(user string, uid *int64) findCommonCallback {
	return func(line string) error {
		parts := strings.SplitN(line, ":", 4)
		if parts[0] == user {
			if len(parts) < 3 {
				return errUnexpectedFileFormat
			}
			uidString := parts[2]
			uidLocal := util.TryParseInt64(uidString)
			if uidLocal == nil || *uidLocal < 0 {
				return errUnexpectedFileFormat
			}
			*uid = *uidLocal
			return errFindCommonBreak
		}
		return nil
	}
}

type findCommonCallback = func(line string) error

var errFindCommonBreak = fmt.Errorf("")
var errUnexpectedFileFormat = fmt.Errorf("unexpected file format")

func findCommon(file string, callback findCommonCallback) error {
	fd, err := fs.FS.Open(file)
	if err != nil {
		return err
	}
	defer util.CloseAndLogError(fd)
	return findCommonReader(fd, callback)
}

func findCommonReader(reader io.Reader, callback findCommonCallback) error {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		err := callback(line)
		if err == errFindCommonBreak {
			break
		} else if err != nil {
			return err
		}
	}
	return scanner.Err()
}
