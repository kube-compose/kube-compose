package config

import (
	"path/filepath"

	"github.com/kube-compose/kube-compose/pkg/expanduser"
)

func expandPath(workingDirChild, path string) string {
	path = expanduser.ExpandUser(path)
	if !filepath.IsAbs(path) {
		path = filepath.Join(filepath.Dir(workingDirChild), path)
	}
	return path
}
