package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

// PathMapping is a representation of a short docker-compose volume.
// Instead of a string pointer we use a pair of boolean and string for host path and mode. This is because
// merging (and detection of duplicates) is to be implemented with struct equality. Defining the struct this way would align
// the merging behavior with docker compose.
type PathMapping struct {
	HasHostPath   bool   // false if and only if the docker engine should create a volume.
	HasMode       bool   // true if and only if the path mapping explicitly set the mode.
	HostPath      string // If this starts with a . or ~ then those should be expanded as appropriate.
	Mode          string
	ContainerPath string
}

// parsePathMapping has the same logic as split_path_mapping:
// https://github.com/docker/compose/blob/99e67d0c061fa3d9b9793391f3b7c8bdf8e841fc/compose/config/config.py#L1440
func parsePathMapping(shortSyntax string) PathMapping {
	r := PathMapping{}
	i := volumeNameLength(shortSyntax)
	hostDrive := shortSyntax[:i]
	remaining := shortSyntax[i:]

	i = strings.IndexByte(remaining, ':')
	if i < 0 {
		r.ContainerPath = shortSyntax
	} else {
		r.HostPath = hostDrive + remaining[:i]
		r.HasHostPath = true
		remaining = remaining[i+1:]

		i = volumeNameLength(remaining)
		containerDrive := remaining[:i]
		remaining = remaining[i:]

		i = strings.IndexByte(remaining, ':')
		if i < 0 {
			r.ContainerPath = containerDrive + remaining
		} else {
			r.ContainerPath = containerDrive + remaining[:i]
			remaining = remaining[i+1:]

			r.Mode = remaining
			r.HasMode = true
		}
	}
	return r
}

// volumeNameLength is used to correctly interpret volumes of a docker compose service.
// This function has the same logic as splitdrive:
// https://github.com/docker/compose/blob/d563a6640539ad6395d69561c719d886c0d1861c/compose/utils.py#L133
func volumeNameLength(s string) int {
	if s == "" {
		return 0
	}
	switch s[0] {
	case '.':
		return 0
	case '\\':
		return 0
	case '/':
		return 0
	case '~':
		return 0
	}
	// Since we know that s does not start with '/' or '\\', the function ntpathVolumeNameLength is overkill.
	// But we leave it here to maintain a similar structure to docker-compose.
	return ntpathVolumeNameLength(s)
}

// ntpathVolumeNameLength is similar to Go's "file/filepath".VolumeName, but is used to interpret the volume of a docker-compose service
// exactly like docker compose and interprets UNC paths and drive letters on non-Windows platforms.
// This function has the same logic as ntpath.splitdrive:
// https://github.com/python/cpython/blob/74510e2a57f6d4b51ac1ab4f778cd7a4c54b541e/Lib/ntpath.py#L116.
// Even on Windows we cannot use "file/filepath".VolumeName because it differs from Python's ntpath.splitdrive:
// 1. Go requires ASCII letter to precede colon for drive letters, but Python does not.
// 2. Go never considers paths that have a . after the third slash a UNC path, but Python does.
func ntpathVolumeNameLength(s string) int {
	n := len(s)
	if n >= 2 {
		if isSlash(s[0]) && isSlash(s[1]) && (n < 3 || !isSlash(s[2])) {
			return ntpathVolumeNameLengthCore(s)
		}
		if s[1] == ':' {
			return 2
		}
	}
	return 0
}

func ntpathVolumeNameLengthCore(s string) int {
	n := len(s)
	index := 3
	for {
		if index >= n {
			return 0
		}
		if isSlash(s[index]) {
			break
		}
		index++
	}
	if index+1 < n && isSlash(s[index+1]) {
		return 0
	}
	index2 := index + 2
	for {
		if index2 >= n {
			return n
		}
		if isSlash(s[index2]) {
			return index2
		}
		index2++
	}
}

// isSlash returns true if and only if b is the ASCII code of a forward or backward slash.
func isSlash(b byte) bool {
	return b == '/' || b == '\\'
}

// Copy of the resolve_volume_path function:
// https://github.com/docker/compose/blob/99e67d0c061fa3d9b9793391f3b7c8bdf8e841fc/compose/config/config.py#L1354
func resolveHostPath(resolvedFile string, sv *ServiceVolume) error {
	if sv.Short != nil && sv.Short.HasHostPath && len(sv.Short.HostPath) > 0 {
		if sv.Short.HostPath[0] == '.' {
			sv.Short.HostPath = filepath.Join(filepath.Dir(resolvedFile), sv.Short.HostPath)
		}
		if sv.Short.HostPath[0] == '~' {
			// TODO https://github.com/kube-compose/kube-compose/issues/162 support expanding tilde
			return fmt.Errorf("a docker compose service has a volume that includes a ~, but expanding users is not supported")
		}
	}
	// TODO https://github.com/kube-compose/kube-compose/issues/161 expanding source of long volume syntax
	return nil
}
