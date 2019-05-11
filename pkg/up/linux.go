package up

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jbrekelmans/kube-compose/internal/pkg/util"
)

func findUserInPasswd(file, user string) (*int64, error) {
	fd, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer util.CloseAndLogError(fd)
	return findUserInPasswdReader(fd, user)
}

func findUserInPasswdReader(reader io.Reader, user string) (*int64, error) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 4)
		if parts[0] == user {
			if len(parts) < 3 {
				return nil, fmt.Errorf("unexpected file format")
			}
			user := parts[2]
			uid := tryParseUID(user)
			if uid == nil || *uid < 0 {
				return nil, fmt.Errorf("unexpected file format")
			}
			return uid, nil
		}
	}
	return nil, scanner.Err()
}
