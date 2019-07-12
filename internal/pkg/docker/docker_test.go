package docker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	dockerRef "github.com/docker/distribution/reference"
	dockerTypes "github.com/docker/docker/api/types"
)

// The encoded form of user:password to be used as docker registry authentication header value
// This is a test password, so ignore linting issue.
// nolint
const (
	testToken   = "eyJ1c2VybmFtZSI6InVzZXIiLCJwYXNzd29yZCI6InBhc3N3b3JkIn0="
	testDigest  = "sha256:" + testImageID
	testImageID = "18cd47570b4fd77db3c0d9be067f82b1aeb2299ff6df7e765c9d087b7549d24d"
)

func Test_EncodeRegistryAuth(t *testing.T) {
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

func Test_PushImage_Success(t *testing.T) {
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
func Test_PushImage_Error(t *testing.T) {
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

func Test_PullImage_Success(t *testing.T) {
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

func Test_PullImage_Error(t *testing.T) {
	errExpected := errors.New("oopspull")
	puller := &testImagePuller{
		err: errExpected,
	}
	_, err := PullImage(context.Background(), puller, "myimagepullerror:latest", testToken, func(_ *PullOrPush) {})
	if err != errExpected {
		t.Error(err)
	}
}

type testImageLister struct {
	err            error
	imageSummaries []dockerTypes.ImageSummary
}

func (t *testImageLister) ImageList(ctx context.Context, listOptions dockerTypes.ImageListOptions) (
	[]dockerTypes.ImageSummary, error) {
	if t.err != nil {
		return nil, t.err
	}
	return t.imageSummaries, nil
}

func Test_ResolveLocalImageAfterPull_SuccessNotFound(t *testing.T) {
	lister := &testImageLister{}
	named, _ := dockerRef.ParseNormalizedNamed("myimage")
	imageID, repoDigest, err := ResolveLocalImageAfterPull(context.Background(), lister, named, testDigest)
	if err != nil {
		t.Error(err)
	} else if imageID != "" || repoDigest != "" {
		t.Fail()
	}
}

func Test_ResolveLocalImageAfterPull_Error(t *testing.T) {
	errExpected := fmt.Errorf("resolveLocalImageAfterPullError")
	lister := &testImageLister{
		err: errExpected,
	}
	named, _ := dockerRef.ParseNormalizedNamed("myimage")
	_, _, errActual := ResolveLocalImageAfterPull(context.Background(), lister, named, testDigest)
	if errActual != errExpected {
		t.Error(errActual)
	}
}

func Test_ResolveLocalImageAfterPull_SuccessFound(t *testing.T) {
	imageIDExpected := testImageID
	imageName := "myimage"
	repoDigestExpected := imageName + "@" + testDigest
	lister := &testImageLister{
		imageSummaries: []dockerTypes.ImageSummary{
			{
				ID: imageIDExpected,
				RepoDigests: []string{
					repoDigestExpected,
				},
			},
		},
	}
	named, _ := dockerRef.ParseNormalizedNamed(imageName)
	imageIDActual, repoDigestActual, err := ResolveLocalImageAfterPull(context.Background(), lister, named, testDigest)
	switch {
	case err != nil:
		t.Error(err)
	case imageIDActual != imageIDExpected:
		t.Fail()
	case repoDigestActual != repoDigestExpected:
		t.Fail()
	}
}

func Test_DefaultDomain(t *testing.T) {
	// In case our logic is broken, or the default domain changes: fail CI.
	s := DefaultDomain()
	if s != "docker.io" {
		t.Fail()
	}
}

func Test_OfficialRepoName(t *testing.T) {
	// In case our logic is broken, or the default domain changes: fail CI.
	s := OfficialRepoName()
	if s != "library" {
		t.Fail()
	}
}
