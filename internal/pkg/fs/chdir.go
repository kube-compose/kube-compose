package fs

import (
	"os"
	"syscall"
)

func (fs *osFileSystem) Chdir(dir string) error {
	return os.Chdir(dir)
}

func (fs *InMemoryFileSystem) Chdir(dir string) error {
	h := evalSymlinksHelper{
		fs:      fs,
		nameRem: dir,
	}
	err := h.run()
	if err != nil {
		return err
	}
	if !h.n.Mode().IsDir() {
		return syscall.ENOTDIR
	}
	fs.cwd = h.resolved
	return nil
}
