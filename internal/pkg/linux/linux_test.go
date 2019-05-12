package linux

import (
	"strings"
	"testing"
)

func TestFindUserInPasswdReaderSuccess(t *testing.T) {
	reader := strings.NewReader("root:x:0:\nbin:x:1:")
	_, err := FindUserInPasswdReader(reader, "bin")
	if err != nil {
		t.Fail()
	}
}
func TestFindUserInPasswdReaderNotFound(t *testing.T) {
	reader := strings.NewReader("root:x:0:\nbin:x:1:")
	_, err := FindUserInPasswdReader(reader, "henk")
	if err != nil {
		t.Fail()
	}
}

func TestFindUserInPasswdReaderInvalidUID(t *testing.T) {
	reader := strings.NewReader("root:x:0:\nbin:x:-1:")
	_, err := FindUserInPasswdReader(reader, "bin")
	if err == nil {
		t.Fail()
	}
}

func TestFindUserInPasswdReaderInvalidFormat(t *testing.T) {
	reader := strings.NewReader("root")
	_, err := FindUserInPasswdReader(reader, "root")
	if err == nil {
		t.Fail()
	}
}
