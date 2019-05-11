package k8s

import (
	"testing"
)

func TestEncodeName(t *testing.T) {
	r := EncodeName("\x00\x390a\x7B")
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
