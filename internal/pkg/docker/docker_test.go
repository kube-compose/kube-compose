package docker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	dockerTypes "github.com/docker/docker/api/types"
)

// The encoded form of user:password to be used as docker registry authentication header value
// This is a test password, so ignore linting issue.
// nolint
const testToken = "eyJ1c2VybmFtZSI6InVzZXIiLCJwYXNzd29yZCI6InBhc3N3b3JkIn0="

func TestEncodeRegistryAuth(t *testing.T) {
	ret := EncodeRegistryAuth("user", "password")
	if ret != testToken {
		t.Fail()
	}
}

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

type testImagePusher struct {
	ctx         context.Context
	readCloser  io.ReadCloser
	err         error
	image       string
	pushOptions dockerTypes.ImagePushOptions
}

func (t *testImagePusher) ImagePush(ctx context.Context, image string, pushOptions dockerTypes.ImagePushOptions) (io.ReadCloser, error) {
	t.ctx = ctx
	t.image = image
	t.pushOptions = pushOptions
	return t.readCloser, t.err
}

func TestPushImage_Success(t *testing.T) {
	pusher := &testImagePusher{
		readCloser: newTestDigestStatusReadCloser(),
	}
	imageExpected := "myimagepush:latest"
	digest, err := PushImage(context.Background(), pusher, imageExpected, testToken, func(_ *PullOrPush) {})
	if imageExpected != pusher.image {
		t.Fail()
	}
	if testToken != pusher.pushOptions.RegistryAuth {
		t.Fail()
	}
	if digest != testDigest {
		t.Fail()
	}
	if err != nil {
		t.Error(err)
	}
}
func TestPushImage_Error(t *testing.T) {
	errExpected := errors.New("oopspush")
	pusher := &testImagePusher{
		err: errExpected,
	}
	_, err := PushImage(context.Background(), pusher, "myimagepusherror:latest", testToken, func(_ *PullOrPush) {})
	if err != errExpected {
		t.Error(err)
	}
}

type testImagePuller struct {
	ctx         context.Context
	readCloser  io.ReadCloser
	err         error
	image       string
	pullOptions dockerTypes.ImagePullOptions
}

func (t *testImagePuller) ImagePull(ctx context.Context, image string, pullOptions dockerTypes.ImagePullOptions) (io.ReadCloser, error) {
	t.ctx = ctx
	t.image = image
	t.pullOptions = pullOptions
	return t.readCloser, t.err
}

func TestPullImage_Success(t *testing.T) {
	puller := &testImagePuller{
		readCloser: newTestDigestStatusReadCloser(),
	}
	imageExpected := "myimagepull:latest"
	digest, err := PullImage(context.Background(), puller, imageExpected, testToken, func(_ *PullOrPush) {})
	if imageExpected != puller.image {
		t.Fail()
	}
	if testToken != puller.pullOptions.RegistryAuth {
		t.Fail()
	}
	if testDigest != digest {
		t.Fail()
	}
	if err != nil {
		t.Error(err)
	}
}

func TestPullImage_Error(t *testing.T) {
	errExpected := errors.New("oopspull")
	puller := &testImagePuller{
		err: errExpected,
	}
	_, err := PullImage(context.Background(), puller, "myimagepullerror:latest", testToken, func(_ *PullOrPush) {})
	if err != errExpected {
		t.Error(err)
	}
}
