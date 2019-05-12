package docker

import "testing"

func TestParseUserinfoEmptyUserGroup(t *testing.T) {
	_, err := ParseUserinfo(":")
	if err != nil {
		t.Fail()
	}
}

func TestParseUserinfoUID(t *testing.T) {
	user, err := ParseUserinfo("0")
	if err != nil {
		t.Fail()
	}
	if user.UID == nil || *user.UID != 0 {
		t.Fail()
	}
}
func TestParseUserinfoGID(t *testing.T) {
	user, err := ParseUserinfo(":1234")
	if err != nil {
		t.Fail()
	}
	if user.GID == nil || *user.GID != 1234 {
		t.Fail()
	}
}

func TestParseUserinfoUIDOutOfRange(t *testing.T) {
	_, err := ParseUserinfo("-1")
	if err == nil {
		t.Fail()
	}
}

func TestParseUserinfoGIDOutOfRange(t *testing.T) {
	_, err := ParseUserinfo(":-1")
	if err == nil {
		t.Fail()
	}
}
