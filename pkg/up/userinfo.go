package up

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type userinfo struct {
	UID   *int64
	User  string
	Gid   *int64
	Group string
}

func tryParseUID(s string) *int64 {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		// If the string cannot be parsed into an int64 then interpret it as a user name or group name (e.g. to find their uid and gid we have
		// we can look in /etc/passwd and /etc/group). This is exactly the behavior of docker.
		return nil
	}
	pi := new(int64)
	*pi = i
	return pi
}

// parseUserinfo parses a string into a user and group. The string is interpreted exactly as the docker run command would interpret it.
func parseUserinfo(userinfoRaw string) (*userinfo, error) {
	if userinfoRaw == "" {
		r := &userinfo{}
		r.UID = new(int64)
		*r.UID = 0
		return r, nil
	}
	i := strings.IndexByte(userinfoRaw, ':')
	r := &userinfo{}
	if i < 0 {
		r.User = userinfoRaw
	} else {
		r.User = userinfoRaw[:i]
	}
	r.UID = tryParseUID(r.User)
	if r.UID != nil && (*r.UID > math.MaxInt32 || *r.UID < 0) {
		return nil, fmt.Errorf("linux spec user: uids and gids must be in range 0-2147483647")
	}
	if i >= 0 {
		r.Group = userinfoRaw[i+1:]
		r.Gid = tryParseUID(r.Group)
		if r.Gid != nil && (*r.Gid > math.MaxInt32 || *r.Gid < 0) {
			return nil, fmt.Errorf("linux spec user: uids and gids must be in range 0-2147483647")
		}
	}
	return r, nil
}
