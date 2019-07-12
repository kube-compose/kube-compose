package docker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/docker/distribution/digestset"
	dockerRef "github.com/docker/distribution/reference"
	dockerTypes "github.com/docker/docker/api/types"
	digestPackage "github.com/opencontainers/go-digest"
)

// The encoded form of user:password to be used as docker registry authentication header value
// This is a test password, so ignore linting issue.
// nolint
const testToken = "eyJ1c2VybmFtZSI6InVzZXIiLCJwYXNzd29yZCI6InBhc3N3b3JkIn0="

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
	digest := "sha256:18cd47570b4fd77db3c0d9be067f82b1aeb2299ff6df7e765c9d087b7549d24d"
	imageID, repoDigest, err := ResolveLocalImageAfterPull(context.Background(), lister, named, digest)
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
	digest := "sha256:18cd47570b4fd77db3c0d9be067f82b1aeb2299ff6df7e765c9d087b7549d24d"
	_, _, errActual := ResolveLocalImageAfterPull(context.Background(), lister, named, digest)
	if errActual != errExpected {
		t.Error(errActual)
	}
}

func Test_ResolveLocalImageAfterPull_SuccessFound(t *testing.T) {
	digest := "sha256:18cd47570b4fd77db3c0d9be067f82b1aeb2299ff6df7e765c9d087b7549d24d"
	imageIDExpected := "18cd47570b4fd77db3c0d9be067f82b1aeb2299ff6df7e765c9d087b7549d24d"
	imageName := "myimage"
	repoDigestExpected := imageName + "@" + digest
	lister := &testImageLister{
		imageSummaries: []dockerTypes.ImageSummary{
			dockerTypes.ImageSummary{
				ID: imageIDExpected,
				RepoDigests: []string{
					repoDigestExpected,
				},
			},
		},
	}
	named, _ := dockerRef.ParseNormalizedNamed(imageName)
	imageIDActual, repoDigestActual, err := ResolveLocalImageAfterPull(context.Background(), lister, named, digest)
	if err != nil {
		t.Error(err)
	} else if imageIDActual != imageIDExpected {
		t.Fail()
	} else if repoDigestActual != repoDigestExpected {
		t.Fail()
	}
}

// Coverage only
func Test_DefaultDomain(t *testing.T) {
	DefaultDomain()
}

// Coverage only
func Test_OfficialRepoName(t *testing.T) {
	OfficialRepoName()
}

func Test_ResolveLocalImageID_FoundRepoTag(t *testing.T) {
	imageIDExpected := "18cd47570b4fd77db3c0d9be067f82b1aeb2299ff6df7e765c9d087b7549d24d"
	familiarNameTag := "myimage:tag"
	ref, _ := dockerRef.ParseNormalizedNamed(familiarNameTag)
	localImageIDSet := digestset.NewSet()
	localImagesCache := []dockerTypes.ImageSummary{
		dockerTypes.ImageSummary{
			ID: imageIDExpected,
			RepoTags: []string{
				familiarNameTag,
			},
		},
	}
	imageIDActual := ResolveLocalImageID(ref, localImageIDSet, localImagesCache)
	if imageIDActual != imageIDExpected {
		t.Fail()
	}
}

func Test_ResolveLocalImageID_RepoDigestNotFound(t *testing.T) {
	digest := "sha256:18cd47570b4fd77db3c0d9be067f82b1aeb2299ff6df7e765c9d087b7549d24d"
	named, err := dockerRef.ParseNormalizedNamed("myimage")
	if err != nil {
		t.Error(err)
	}
	ref, err := dockerRef.WithDigest(named, digestPackage.Digest(digest))
	if err != nil {
		t.Error(err)
	}
	localImageIDSet := digestset.NewSet()
	localImagesCache := []dockerTypes.ImageSummary{
		dockerTypes.ImageSummary{
			ID: "18cd47570b4fd77db3c0d9be067f82b1aeb2299ff6df7e765c9d087b7549d24d",
		},
	}
	imageID := ResolveLocalImageID(ref, localImageIDSet, localImagesCache)
	if imageID != "" {
		t.Fail()
	}
}

func Test_ResolveLocalImageID_RepoDigestFound(t *testing.T) {
	imageIDExpected := "18cd47570b4fd77db3c0d9be067f82b1aeb2299ff6df7e765c9d087b7549d24d"
	digest := "sha256:18cd47570b4fd77db3c0d9be067f82b1aeb2299ff6df7e765c9d087b7549d24d"
	imageName := "myimage"
	named, err := dockerRef.ParseNormalizedNamed(imageName)
	if err != nil {
		t.Error(err)
	}
	ref, err := dockerRef.WithDigest(named, digestPackage.Digest(digest))
	if err != nil {
		t.Error(err)
	}
	localImageIDSet := digestset.NewSet()
	localImagesCache := []dockerTypes.ImageSummary{
		dockerTypes.ImageSummary{
			ID: imageIDExpected,
			RepoDigests: []string{
				imageName + "@" + digest,
			},
		},
	}
	imageIDActual := ResolveLocalImageID(ref, localImageIDSet, localImagesCache)
	if imageIDActual != imageIDExpected {
		t.Fail()
	}
}

func Test_ResolveLocalImageID_ImageIDFound(t *testing.T) {
	imageIDExpected := "18cd47570b4fd77db3c0d9be067f82b1aeb2299ff6df7e765c9d087b7549d24d"
	ref, err := dockerRef.ParseAnyReference(imageIDExpected)
	if err != nil {
		t.Error(err)
	}
	localImageIDSet := digestset.NewSet()

	d, err := digestPackage.Parse("sha256:" + imageIDExpected)
	if err != nil {
		t.Error(err)
	}
	_ = localImageIDSet.Add(d)
	localImagesCache := []dockerTypes.ImageSummary{
		dockerTypes.ImageSummary{
			ID: imageIDExpected,
		},
	}
	imageIDActual := ResolveLocalImageID(ref, localImageIDSet, localImagesCache)
	if imageIDActual != imageIDExpected {
		t.Fail()
	}
}
