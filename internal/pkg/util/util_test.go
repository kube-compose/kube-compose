package util

import (
	"testing"
)

func TestEscapeName(t *testing.T) {
	r := EscapeName("\x00\x390a\x7B")
	// Each character that is not [a-z0-8] is replaced by a three-letter sequence 9[a-z0-9]{2}, i.e.:
	// "\x00" => "9aa"
	// "\x39" => "9bv"
	// "0" 	  => "0"
	// "a"    => "a"
	// "\x7B" => "9dp"
	if r != "9aa9bv0a9dp" {
		t.Fail()
	}
}

func TestUnescapeNameSuccess(t *testing.T) {
	r, err := UnescapeName("9aa9bv0a9dp")
	if r != "\x00\x390a\x7B" || err != nil {
		t.Fail()
	}
}

func TestUnescapeNameError(t *testing.T) {
	_, err := UnescapeName("9zz")
	if err == nil {
		t.Fail()
	}
}

func TestUnescapeByteError1(t *testing.T) {
	_, err := unescapeByte("9\x00a", 0)
	if err == nil {
		t.Fail()
	}
}

func TestUnescapeByteError2(t *testing.T) {
	_, err := unescapeByte("9a\x00", 0)
	if err == nil {
		t.Fail()
	}
}

func TestUnescapeByteError3(t *testing.T) {
	_, err := unescapeByte("", 0)
	if err == nil {
		t.Fail()
	}
}
