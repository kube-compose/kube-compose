package docker

import (
	"testing"
)

func TestEncodeRegistryAuth_Success(t *testing.T) {
	_, err := EncodeRegistryAuth("user", "password")
	if err != nil {
		t.Fail()
	}
}
