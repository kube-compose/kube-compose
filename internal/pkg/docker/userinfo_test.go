package docker

import "testing"

func TestParseUserinfo_EmptyUserGroup(t *testing.T) {
	_, err := ParseUserinfo(":")
	if err != nil {
		t.Fail()
	}
}

func TestParseUserinfo_UID(t *testing.T) {
	user, err := ParseUserinfo("0")
	if err != nil {
		t.Fail()
	}
	if user.UID == nil || *user.UID != 0 {
		t.Fail()
	}
}
func TestParseUserinfo_GID(t *testing.T) {
	user, err := ParseUserinfo(":1234")
	if err != nil {
		t.Fail()
	}
	if user.GID == nil || *user.GID != 1234 {
		t.Fail()
	}
}

func TestParseUserinfo_UIDOutOfRange(t *testing.T) {
	_, err := ParseUserinfo("-1")
	if err == nil {
		t.Fail()
	}
}

func TestParseUserinfo_GIDOutOfRange(t *testing.T) {
	_, err := ParseUserinfo(":-1")
	if err == nil {
		t.Fail()
	}
}
