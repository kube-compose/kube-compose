package service

import (
	"testing"
)

func Test_EncodeRegistryAuth(t *testing.T) {
	ret := EncodeRegistryAuth("user", "password")
	if ret != testToken {
		t.Fail()
	}
}
