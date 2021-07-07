package docker

import (
	"bytes"
	"fmt"
	"io"
)

// The encoded form of user:password to be used as docker registry authentication header value
// This is a test password, so ignore linting issue.
// nolint
const (
	testToken   = "eyJ1c2VybmFtZSI6InVzZXIiLCJwYXNzd29yZCI6InBhc3N3b3JkIn0="
	testDigest  = "sha256:" + testImageID
	testImageID = "18cd47570b4fd77db3c0d9be067f82b1aeb2299ff6df7e765c9d087b7549d24d"
)

type testReadCloser struct {
	reader io.Reader
}

func (t *testReadCloser) Read(p []byte) (n int, err error) {
	return t.reader.Read(p)
}

func (t *testReadCloser) Close() error {
	return nil
}

func newTestDigestStatusReadCloser() *testReadCloser {
	reader := bytes.NewReader([]byte(fmt.Sprintf(`{"status":"%s "}`, testDigest)))
	return &testReadCloser{
		reader: reader,
	}
}
