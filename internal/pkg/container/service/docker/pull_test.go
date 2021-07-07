//nolint
// ignoring dupl of image_push_test.go warning
package docker

import (
	"context"
	"io"
	"testing"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/pkg/errors"
)

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

func Test_ImagePull_Success(t *testing.T) {
	puller := &testImagePuller{
		readCloser: newTestDigestStatusReadCloser(),
	}
	imageExpected := "myimagepull:latest"
	digest, err := imagePull(context.Background(), puller, imageExpected, testToken, func(_ *pullOrPush) {})
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

func Test_ImagePull_Error(t *testing.T) {
	errExpected := errors.New("oopspull")
	puller := &testImagePuller{
		err: errExpected,
	}
	_, err := imagePull(context.Background(), puller, "myimagepullerror:latest", testToken, func(_ *pullOrPush) {})
	if err != errExpected {
		t.Error(err)
	}
}
