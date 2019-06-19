package fs

import (
	"os"
	"strings"
	"syscall"
)

func (fs *InMemoryFileSystem) lstatNode(name string) (*node, error) {
	if name == "" {
		name = fs.cwd
	}
	name = trimTrailingSlashes(name)
	var n *node
	if name != "" {
		i := strings.LastIndexByte(name, '/')
		var err error
		n, _, err = fs.find(name[:i+1], false, true)
		if err != nil {
			return nil, err
		}
		if (n.mode & os.ModeDir) == 0 {
			return nil, syscall.ENOTDIR
		}
		nameComp := name
		if i >= 0 {
			nameComp = name[i+1:]
		}
		validateNameComp(nameComp)
		n = n.dirLookup(nameComp)
		if n == nil {
			return nil, os.ErrNotExist
		}
	} else {
		n = fs.root
	}
	if n.err != nil {
		return nil, n.err
	}
	return n, nil
}

// Lstat should behave the same as os.Lstat but operates on the virtual file system.
func (fs *InMemoryFileSystem) Lstat(name string) (os.FileInfo, error) {
	n, err := fs.lstatNode(name)
	if err != nil {
		return nil, err
	}
	return n, nil
}
