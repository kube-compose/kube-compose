package fs

import (
	"os"
	"reflect"
	"testing"
)

func Test_VirtualFileSystem_Mkdir_Success(t *testing.T) {
	fs := NewVirtualFileSystem(map[string]VirtualFile{})
	expectedName := "asdf"
	err := fs.Mkdir(expectedName, os.ModePerm)
	if err != nil {
		t.Error(err)
	} else {
		dir := fs.root.extra.([]*node)
		expectedNode := newDirNode(
			nil,
			os.ModePerm,
			expectedName,
		)
		if !reflect.DeepEqual(dir, []*node{
			expectedNode,
		}) {
			t.Logf("actual  : %+v\n", dir[0])
			t.Logf("expected: %+v\n", expectedNode)
			t.Fail()
		}
	}
}