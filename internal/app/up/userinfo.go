package up

import (
	"fmt"
	"math"
	"strings"

	"github.com/kube-compose/kube-compose/internal/pkg/util"
)

type Userinfo struct {
	UID   *int64
	User  string
	GID   *int64
	Group string
}

// ParseUserinfo parses a string into a user and group. The string is interpreted exactly as the docker run command would interpret it.
func ParseUserinfo(userinfoRaw string) (*Userinfo, error) {
	i := strings.IndexByte(userinfoRaw, ':')
	r := &Userinfo{}
	if i < 0 {
		r.User = userinfoRaw
	} else {
		r.User = userinfoRaw[:i]
	}
	if r.User == "" {
		r.UID = new(int64)
		*r.UID = 0
	} else {
		r.UID = util.TryParseInt64(r.User)
		if r.UID != nil && (*r.UID > math.MaxInt32 || *r.UID < 0) {
			return nil, fmt.Errorf("linux spec user: uids and gids must be in range 0-2147483647")
		}
		// If the string cannot be parsed into an int64 then interpret it as a user name (r.UID == nil).
		// To find the uid we look in /etc/passwd. This is exactly the behavior of docker.
	}
	err := r.parseUserinfoGroup(userinfoRaw, i)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Userinfo) parseUserinfoGroup(userinfoRaw string, i int) error {
	if i >= 0 {
		r.Group = userinfoRaw[i+1:]
		if r.Group != "" {
			r.GID = util.TryParseInt64(r.Group)
			if r.GID != nil && (*r.GID > math.MaxInt32 || *r.GID < 0) {
				return fmt.Errorf("linux spec user: uids and gids must be in range 0-2147483647")
			}
			// If the string cannot be parsed into an int64 then interpret it as a group name (r.GID == nil but r.Group != "").
			// To find the gid we look in /etc/group. This is exactly the behavior of docker.
		}
	}
	return nil
}
