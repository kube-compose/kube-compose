package fs

import (
	"os"
	"strings"
	"syscall"
)

type evalSymlinksHelper struct {
	fs       *InMemoryFileSystem
	n        *node
	links    int
	resolved string
	nameRem  string
}

func (h *evalSymlinksHelper) getNameComp(slashPos int) string {
	if slashPos < 0 {
		return h.nameRem
	}
	return h.nameRem[:slashPos]
}

func (h *evalSymlinksHelper) run() error {
	for h.nameRem != "" {
		slashPos := strings.IndexByte(h.nameRem, '/')
		nameComp := h.getNameComp(slashPos)
		var childN *node
		if nameComp != "" {
			validateNameComp(nameComp)
			if (h.n.mode & os.ModeDir) == 0 {
				return syscall.ENOTDIR
			}
			childN = h.n.dirLookup(nameComp)
			if childN == nil {
				return os.ErrNotExist
			}
			if childN.err != nil {
				return childN.err
			}
		}
		h.updateNameRemFromSlashPos(slashPos)
		if nameComp != "" {
			err := h.updateFromChildN(childN, nameComp)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *evalSymlinksHelper) updateFromChildN(childN *node, nameComp string) error {
	if (childN.mode & os.ModeSymlink) != 0 {
		h.links++
		if h.links > 255 {
			return errTooManyLinks
		}
		target := childN.extra.([]byte)
		j := 0
		if len(target) > 0 && target[0] == '/' {
			h.resolved = "/"
			h.n = h.fs.root
			j = 1
		}
		if h.nameRem != "" {
			h.nameRem = string(target)[j:] + "/" + h.nameRem
		} else {
			h.nameRem = string(target)[j:]
		}
	} else {
		if h.resolved == "/" {
			h.resolved += nameComp
		} else {
			h.resolved += "/" + nameComp
		}
		h.n = childN
	}
	return nil
}

func (h *evalSymlinksHelper) updateNameRemFromSlashPos(slashPos int) {
	if slashPos < 0 {
		h.nameRem = ""
	} else {
		h.nameRem = h.nameRem[slashPos+1:]
	}
}

// EvalSymlinks should behave the same as "path/filepath".EvalSymlinks, but on the virtual file system.
func (fs *InMemoryFileSystem) EvalSymlinks(path string) (string, error) {
	h := &evalSymlinksHelper{
		fs: fs,
	}
	var err error
	if path != "" && path[0] == '/' {
		h.resolved = "/"
		h.nameRem = path[1:]
		h.n = fs.root
		if h.n.err != nil {
			return "", h.n.err
		}
	} else {
		h.n, _, err = h.fs.find(h.fs.cwd, false, true)
		if err != nil {
			return "", err
		}
		h.nameRem = path
	}
	err = h.run()
	if err != nil {
		return "", err
	}
	return h.resolved, nil
}
