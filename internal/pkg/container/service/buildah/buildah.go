package buildah

import (
	"fmt"
)

var errNotSupported = fmt.Errorf("not supported")

// IsErrNotSupported determines whether an error returned by New is caused by running on an unsupported platform.
func IsErrNotSupported(err error) bool {
	return err == errNotSupported
}
