package util

import (
	"fmt"
	"io"
)

// CloseAndLogError closes the closer and logs any error it returns.
func CloseAndLogError(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		fmt.Println(err)
	}
}
