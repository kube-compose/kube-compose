package fs

// NTVolumeNameLength is similar to Go's "file/filepath".VolumeName, but is used to interpret the volume of a docker-compose service
// exactly like docker compose and interprets UNC paths and drive letters on non-Windows platforms.
// This function has the same logic as ntpath.splitdrive:
// https://github.com/python/cpython/blob/74510e2a57f6d4b51ac1ab4f778cd7a4c54b541e/Lib/ntpath.py#L116.
// Even on Windows we cannot use "file/filepath".VolumeName because it differs from Python's ntpath.splitdrive:
// 1. Go requires ASCII letter to precede colon for drive letters, but Python does not.
// 2. Go never considers paths that have a . after the third slash a UNC path, but Python does.
func NTVolumeNameLength(s string) int {
	n := len(s)
	if n >= 2 {
		if IsPathSeparatorWindows(s[0]) && IsPathSeparatorWindows(s[1]) && (n < 3 || !IsPathSeparatorWindows(s[2])) {
			return NTVolumeNameLengthCore(s)
		}
		if s[1] == ':' {
			return 2
		}
	}
	return 0
}

func NTVolumeNameLengthCore(s string) int {
	n := len(s)
	index := 3
	for {
		if index >= n {
			return 0
		}
		if IsPathSeparatorWindows(s[index]) {
			break
		}
		index++
	}
	if index+1 < n && IsPathSeparatorWindows(s[index+1]) {
		return 0
	}
	index2 := index + 2
	for {
		if index2 >= n {
			return n
		}
		if IsPathSeparatorWindows(s[index2]) {
			return index2
		}
		index2++
	}
}
