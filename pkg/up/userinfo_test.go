package up

import "testing"

func TestTryParseUIDError(t *testing.T) {
	uid := tryParseUID("asdf")
	if uid != nil {
		t.Fail()
	}
}
func TestTryParseUIDSuccess(t *testing.T) {
	uid := tryParseUID("234")
	if uid == nil || *uid != 234 {
		t.Fail()
	}
}

func TestParseUserinfoEmptyUserGroup(t *testing.T) {
	_, err := parseUserinfo(":")
	if err != nil {
		t.Fail()
	}
}

func TestParseUserinfoUID(t *testing.T) {
	user, err := parseUserinfo("0")
	if err != nil {
		t.Fail()
	}
	if user.UID == nil || *user.UID != 0 {
		t.Fail()
	}
}
func TestParseUserinfoGID(t *testing.T) {
	user, err := parseUserinfo(":1234")
	if err != nil {
		t.Fail()
	}
	if user.GID == nil || *user.GID != 1234 {
		t.Fail()
	}
}

func TestParseUserinfoUIDOutOfRange(t *testing.T) {
	_, err := parseUserinfo("-1")
	if err == nil {
		t.Fail()
	}
}

func TestParseUserinfoGIDOutOfRange(t *testing.T) {
	_, err := parseUserinfo(":-1")
	if err == nil {
		t.Fail()
	}
}
