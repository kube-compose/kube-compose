//nolint
// ignoring dupl of image_pull_test.go warning
package docker

import (
	"context"
	"io"
	"testing"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/pkg/errors"
)

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

func Test_ImagePush_Success(t *testing.T) {
	pusher := &testImagePusher{
		readCloser: newTestDigestStatusReadCloser(),
	}
	imageExpected := "myimagepush:latest"
	digest, err := imagePush(context.Background(), pusher, imageExpected, testToken, func(_ *pullOrPush) {})
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
func Test_ImagePush_Error(t *testing.T) {
	errExpected := errors.New("oopspush")
	pusher := &testImagePusher{
		err: errExpected,
	}
	_, err := imagePush(context.Background(), pusher, "myimagepusherror:latest", testToken, func(_ *pullOrPush) {})
	if err != errExpected {
		t.Error(err)
	}
}
