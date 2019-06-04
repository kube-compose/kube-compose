package linux

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	fsPackage "github.com/kube-compose/kube-compose/internal/pkg/fs"
	"github.com/kube-compose/kube-compose/internal/pkg/util"
)

var fs = fsPackage.OSFileSystem()

// FindUserInPasswd finds the UID of a user by name in an /etc/passwd file. It can also find the GID of a group by name in an /etc/group
// file.
func FindUserInPasswd(file, user string) (*int64, error) {
	fd, err := fs.Open(file)
	if err != nil {
		return nil, err
	}
	defer util.CloseAndLogError(fd)
	return FindUserInPasswdReader(fd, user)
}

// FindUserInPasswdReader finds the UID of a user by name in a stream encoded like the contents of /etc/passwd. It can also find the GID of
// a group by name in a stream encoded like the contents of /etc/group.
func FindUserInPasswdReader(reader io.Reader, user string) (*int64, error) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 4)
		if parts[0] == user {
			if len(parts) < 3 {
				return nil, fmt.Errorf("unexpected file format")
			}
			user := parts[2]
			uid := util.TryParseInt64(user)
			if uid == nil || *uid < 0 {
				return nil, fmt.Errorf("unexpected file format")
			}
			return uid, nil
		}
	}
	return nil, scanner.Err()
}
