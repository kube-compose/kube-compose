package fs

import (
	"os"
	"testing"
)

func Test_Node_Size_NotRegularFile(t *testing.T) {
	n := &node{
		mode: os.ModeDir,
	}
	if n.Size() != 0 {
		t.Fail()
	}
}