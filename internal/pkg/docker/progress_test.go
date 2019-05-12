package docker

import (
	"bytes"
	"fmt"
	"testing"
)

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

func TestPullWaitKnownError(t *testing.T) {
	// If the server returns an error then it should be forwarded by Wait (pull).
	reader := bytes.NewReader([]byte(`{"error":{"message":"asdf"}}`))
	pull := NewPull(reader)
	_, err := pull.Wait(func(_ *PullOrPush) {})
	if err == nil {
		t.Fail()
	}
}

func TestPushWaitKnownError(t *testing.T) {
	// If the server returns an error then it should be forwarded by Wait (push).
	reader := bytes.NewReader([]byte(`{"error":{"message":"asdf"}}`))
	push := NewPush(reader)
	_, err := push.Wait(func(_ *PullOrPush) {})
	if err == nil {
		t.Fail()
	}
}
func TestPullWaitUnknownError(t *testing.T) {
	// If there is no digest then we expect an error.
	reader := bytes.NewReader([]byte(`{"id":"layer1","status":"Pull complete"}`))
	pull := NewPull(reader)
	_, err := pull.Wait(func(_ *PullOrPush) {})
	if err == nil {
		t.Fail()
	}
}

func TestPullWaitDigest(t *testing.T) {
	// Wait should return the image digest.
	digestExpected := "sha256:f0b6db8bb4b757d0c3c9e120f4ac091286be5815ad576fbd48d8b953e8d2b06d"
	reader := bytes.NewReader([]byte(fmt.Sprintf(`{"status":"%s "}`, digestExpected)))
	pull := NewPull(reader)
	digestActual, err := pull.Wait(func(_ *PullOrPush) {})
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if digestActual != digestExpected {
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
	reader := bytes.NewReader([]byte(`{"id":"layer1","status":"Pushing","progress":{"current":1,"total":2}}`))
	push := NewPush(reader)
	// If there is 1 layer that is only observed to be already pushed then there should be 1 progress update of 100%.
	var progress float64
	count := 0
	_, _ = push.Wait(func(_ *PullOrPush) {
		progress = push.Progress()
		count++
	})
	if count != 1 || progress >= 1 || progress <= 0 {
		t.Fail()
	}
}
