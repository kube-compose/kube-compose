package fs

import (
	"os"
	"strings"
	"syscall"
)

func (fs *VirtualFileSystem) mkdirCommon(name string, perm os.FileMode, all bool) error {
	if (perm & os.ModeType) != 0 {
		return errBadMode
	}
	n, nameRem, err := fs.find(name, false, true)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if !all {
		slashPos := strings.IndexByte(nameRem, '/')
		if slashPos >= 0 {
			return os.ErrNotExist
		}
		if nameRem == "" {
			return os.ErrExist
		}
	}
	if !n.mode.IsDir() {
		return syscall.ENOTDIR
	}
	if all {
		fs.mkdirCommonAll(n, nameRem, perm)
		return nil
	}
	n.dirAppend(newDirNode(
		nil,
		perm,
		nameRem,
	))
	return nil
}

func (fs *VirtualFileSystem) mkdirCommonAll(n *node, nameRem string, perm os.FileMode) {
	for nameRem != "" {
		slashPos := strings.IndexByte(nameRem, '/')
		nameComp := nameRem
		if slashPos >= 0 {
			nameComp = nameComp[:slashPos]
		}
		if nameComp != "" {
			validateNameComp(nameComp)
			childN := newDirNode(
				nil,
				perm,
				nameComp,
			)
			n.dirAppend(childN)
			n = childN
		}
		if slashPos < 0 {
			nameRem = ""
		} else {
			nameRem = nameRem[slashPos+1:]
		}
	}
}

func (fs *VirtualFileSystem) Mkdir(name string, perm os.FileMode) error {
	return fs.mkdirCommon(name, perm, false)
}

func (fs *VirtualFileSystem) MkdirAll(name string, perm os.FileMode) error {
	return fs.mkdirCommon(name, perm, true)
}
