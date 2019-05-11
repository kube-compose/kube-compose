package docker

import (
	"bytes"
	"fmt"
	"testing"
)

func TestPullProgress(t *testing.T) {
	// If there is 1 layer that is only observed to be pulled then there should be 1 progress update of 100%.
	reader := bytes.NewReader([]byte(`{"id":"layer1","status":"Pull complete"}`))
	pull := NewPull(reader)
	var progress float64
	count := 0
	pull.Wait(func(_ *PullOrPush) {
		progress = pull.Progress()
		count++
	})
	if count != 1 || progress != 1.0 {
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
	digestExpected := "sha256:rgqatjyh3bx91qvkto2rytlrobzfprrbyv7h0fnm1soac55hqc6rpcev5qw9b9uj"
	reader := bytes.NewReader([]byte(fmt.Sprintf(`{"status":"%s"}`, digestExpected)))
	pull := NewPull(reader)
	digestActual, err := pull.Wait(func(_ *PullOrPush) {})
	if err != nil || digestActual != digestExpected {
		t.Fail()
	}
}

func TestPushProgress(t *testing.T) {
	reader := bytes.NewReader([]byte(`{"id":"layer1","status":"Pushed"}`))
	push := NewPush(reader)
	// If there is 1 layer that is only observed to be already pushed then there should be 1 progress update of 100%.
	var progress float64
	count := 0
	push.Wait(func(_ *PullOrPush) {
		progress = push.Progress()
		count++
	})
	if count != 1 || progress != 1.0 {
		t.Fail()
	}
}
