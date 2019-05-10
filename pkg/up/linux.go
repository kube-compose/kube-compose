package up

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func findUserInPasswd(file, user string) (*int64, error) {
	fd, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = fd.Close()
		if err != nil {
			fmt.Println(err)
		}
	}()
	scanner := bufio.NewScanner(fd)
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
