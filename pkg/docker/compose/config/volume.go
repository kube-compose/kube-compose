package config

import (
	"strings"

	fsPackage "github.com/kube-compose/kube-compose/internal/pkg/fs"
	"github.com/kube-compose/kube-compose/pkg/expanduser"
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
	// Since we know that s does not start with '/' or '\\', the function NTVolumeNameLength is overkill.
	// But we leave it here to maintain a similar structure to docker-compose.
	return fsPackage.NTVolumeNameLength(s)
}

// Copy of the resolve_volume_path function:
// https://github.com/docker/compose/blob/99e67d0c061fa3d9b9793391f3b7c8bdf8e841fc/compose/config/config.py#L1354
func resolveBindMountVolumeHostPath(resolvedFile string, sv *ServiceVolume) {
	if sv.Short != nil && sv.Short.HasHostPath && sv.Short.HostPath != "" {
		// The intent of the following if is to resolve relative file paths, but not all relative file paths start with a full stop. We
		// still perform the check as follows, because docker compose also allows specifying named volumes.
		if sv.Short.HostPath[0] == '.' {
			sv.Short.HostPath = expandPath(resolvedFile, sv.Short.HostPath)
		} else {
			sv.Short.HostPath = expanduser.ExpandUser(sv.Short.HostPath)
		}
	}
	// TODO https://github.com/kube-compose/kube-compose/issues/161 expanding source of long volume syntax
}
