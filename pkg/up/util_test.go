package up

import (
	"testing"
)

func TestNewFalsePointer(t *testing.T) {
	f := newFalsePointer()
	if f == nil || *f {
		t.Fail()
	}
}

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
