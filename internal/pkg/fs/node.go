package fs

import (
	"os"
	"time"
)

type node struct {
	name string
	mode os.FileMode
	// if err != nil then err is returned when this node is accessed.
	err error
	// Either []byte or []*node, depending on the type of this node.
	extra interface{}
}

func newDirNode(mode os.FileMode, name string) *node {
	return &node{
		extra: []*node{},
		mode:  mode | os.ModeDir,
		name:  name,
	}
}

func (n *node) dirAppend(childN *node) {
	dir := n.extra.([]*node)
	dir = append(dir, childN)
	n.extra = dir
}

func (n *node) dirLookup(nameComp string) *node {
	dir := n.extra.([]*node)
	for _, childN := range dir {
		if childN.name == nameComp {
			return childN
		}
	}
	return nil
}

func (n *node) IsDir() bool {
	return n.mode.IsDir()
}

func (n *node) Mode() os.FileMode {
	return n.mode
}

func (n *node) ModTime() time.Time {
	return time.Time{}
}

func (n *node) Name() string {
	return n.name
}

func (n *node) Size() int64 {
	if n.mode.IsRegular() {
		return int64(len(n.extra.([]byte)))
	}
	return 0
}

func (n *node) Sys() interface{} {
	return nil
}
