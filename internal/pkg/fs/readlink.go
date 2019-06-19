package fs

import (
	"os"
)

// Readlink should behave the same as os.Readlink but operates on the virtual file system.
func (fs *InMemoryFileSystem) Readlink(name string) (string, error) {
	n, err := fs.lstatNode(name)
	if err != nil {
		return "", err
	}
	if (n.mode & os.ModeSymlink) == 0 {
		return "", errBadMode
	}
	if n.errRead != nil {
		return "", n.errRead
	}
	return string(n.extra.([]byte)), nil
}
