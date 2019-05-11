package up

import (
	"testing"
)

func TestContainsTrue(t *testing.T) {
	f := contains([]string{""}, "")
	if !f {
		t.Fail()
	}
}

func TestContainsFalse(t *testing.T) {
	f := contains([]string{}, "")
	if f {
		t.Fail()
	}
}
