package docker

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

const testDigest = "sha256:f0b6db8bb4b757d0c3c9e120f4ac091286be5815ad576fbd48d8b953e8d2b06d"

func TestPullProgress_Done(t *testing.T) {
	// If there is 1 layer that is only observed to be pulled then there should be 1 progress update of 100%.
	reader := bytes.NewReader([]byte(`{"id":"layer1","status":"Pull complete"}`))
	pull := NewPull(reader)
	var progress float64
	count := 0
	_, _ = pull.Wait(func(_ *PullOrPush) {
		progress = pull.Progress()
		count++
	})
	if count != 1 || progress != 1.0 {
		t.Fail()
	}
}
func TestPullProgress_Empty(t *testing.T) {
	// If there is 1 layer that is only observed to be pulled then there should be 1 progress update of 100%.
	reader := bytes.NewReader([]byte(`{"id":"layer1","status":"Pull complete"}`))
	pull := NewPull(reader)
	progress := pull.Progress()
	if progress != 0.0 {
		t.Fail()
	}
}

func TestPullWait_KnownError(t *testing.T) {
	// If the server returns an error then it should be forwarded by Wait (pull).
	reader := bytes.NewReader([]byte(`{"errorDetail":{"message":"asdf"}}`))
	pull := NewPull(reader)
	_, err := pull.Wait(func(_ *PullOrPush) {})
	if err == nil {
		t.Fail()
	} else if !strings.Contains(err.Error(), "asdf") {
		t.Error(err)
	}
}

type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

func TestPushWait_ReaderError(t *testing.T) {
	errExpected := errors.New("readerroroops")
	push := NewPush(&errorReader{
		err: errExpected,
	})
	_, err := push.Wait(func(_ *PullOrPush) {})
	if err != errExpected {
		t.Error(err)
	}
}

func TestPushWait_KnownError(t *testing.T) {
	// If the server returns an error then it should be forwarded by Wait (push).
	reader := bytes.NewReader([]byte(`{"errorDetail":{"message":"asdf"}}`))
	push := NewPush(reader)
	_, err := push.Wait(func(_ *PullOrPush) {})
	if err == nil {
		t.Fail()
	} else if !strings.Contains(err.Error(), "asdf") {
		t.Error(err)
	}
}
func TestPullWait_UnknownError(t *testing.T) {
	// If there is no digest then we expect an error.
	reader := bytes.NewReader([]byte(`{"id":"layer1","status":"Pull complete"}`))
	pull := NewPull(reader)
	_, err := pull.Wait(func(_ *PullOrPush) {})
	if err == nil {
		t.Fail()
	}
}

func TestPullWait_Digest(t *testing.T) {
	// Wait should return the image digest.
	reader := bytes.NewReader([]byte(fmt.Sprintf(`{"status":"%s "}`, testDigest)))
	pull := NewPull(reader)
	digest, err := pull.Wait(func(_ *PullOrPush) {})
	if err != nil {
		t.Error(err)
	}
	if digest != testDigest {
		t.Fail()
	}
}

func TestPushProgress_Done(t *testing.T) {
	reader := bytes.NewReader([]byte(`{"id":"layer1","status":"Pushed"}`))
	push := NewPush(reader)
	// If there is 1 layer that is only observed to be already pushed then there should be 1 progress update of 100%.
	var progress float64
	count := 0
	_, _ = push.Wait(func(_ *PullOrPush) {
		progress = push.Progress()
		count++
	})
	if count != 1 || progress != 1.0 {
		t.Fail()
	}
}

func TestPushProgress_Partial(t *testing.T) {
	reader := bytes.NewReader([]byte(`{"id":"layer1","status":"Pushing","progressDetail":{"current":1,"total":2}}`))
	push := NewPush(reader)
	// If there is 1 layer that is only observed to be already pushed then there should be 1 progress update of 100%.
	var progress float64
	count := 0
	_, err := push.Wait(func(_ *PullOrPush) {
		progress = push.Progress()
		count++
	})
	if err != nil {
		// Don't fail the test here because we expect an error due to no digest being reported from the server.
		t.Log(err)
	}
	if count != 1 || progress >= 1 || progress <= 0 {
		t.Fail()
	}
}

func TestNewDigestRegexp_Success(t *testing.T) {
	r := NewDigestRegexp()
	if r == nil {
		t.Fail()
	}
}
