package docker

import (
	"context"
	"fmt"
	"testing"

	dockerRef "github.com/docker/distribution/reference"
	dockerTypes "github.com/docker/docker/api/types"
)

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

func Test_ImagePullResolve_SuccessNotFound(t *testing.T) {
	lister := &testImageLister{}
	named, _ := dockerRef.ParseNormalizedNamed("myimage")
	imageID, repoDigest, err := imagePullResolve(context.Background(), lister, named, testDigest)
	if err != nil {
		t.Error(err)
	} else if imageID != "" || repoDigest != "" {
		t.Fail()
	}
}

func Test_ImagePullResolve_Error(t *testing.T) {
	errExpected := fmt.Errorf("resolveLocalImageAfterPullError")
	lister := &testImageLister{
		err: errExpected,
	}
	named, _ := dockerRef.ParseNormalizedNamed("myimage")
	_, _, errActual := imagePullResolve(context.Background(), lister, named, testDigest)
	if errActual != errExpected {
		t.Error(errActual)
	}
}

func Test_ImagePullResolve_SuccessFound(t *testing.T) {
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
	imageIDActual, repoDigestActual, err := imagePullResolve(context.Background(), lister, named, testDigest)
	switch {
	case err != nil:
		t.Error(err)
	case imageIDActual != imageIDExpected:
		t.Fail()
	case repoDigestActual != repoDigestExpected:
		t.Fail()
	}
}
