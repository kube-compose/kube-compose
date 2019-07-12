package service

import (
	sha256 "crypto/sha256"
	"testing"

	"github.com/docker/distribution/digestset"
	dockerRef "github.com/docker/distribution/reference"
	dockerTypes "github.com/docker/docker/api/types"
	digestPackage "github.com/opencontainers/go-digest"
)

// The encoded form of user:password to be used as docker registry authentication header value
// This is a test password, so ignore linting issue.
// nolint
const (
	testToken   = "eyJ1c2VybmFtZSI6InVzZXIiLCJwYXNzd29yZCI6InBhc3N3b3JkIn0="
	testDigest  = "sha256:" + testImageID
	testImageID = "18cd47570b4fd77db3c0d9be067f82b1aeb2299ff6df7e765c9d087b7549d24d"
)

func Test_ResolveLocalImageID_FoundRepoTag(t *testing.T) {
	imageIDExpected := testImageID
	familiarNameTag := "myimage:tag"
	ref, _ := dockerRef.ParseNormalizedNamed(familiarNameTag)
	localImageIDSet := digestset.NewSet()
	localImagesCache := []dockerTypes.ImageSummary{
		{
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
	named, err := dockerRef.ParseNormalizedNamed("imgrepodigestnotfound")
	if err != nil {
		t.Error(err)
	}
	ref, err := dockerRef.WithDigest(named, digestPackage.Digest(testDigest))
	if err != nil {
		t.Error(err)
	}
	localImageIDSet := digestset.NewSet()
	localImagesCache := []dockerTypes.ImageSummary{
		{
			ID: testImageID,
		},
	}
	imageID := ResolveLocalImageID(ref, localImageIDSet, localImagesCache)
	if imageID != "" {
		t.Fail()
	}
}

func Test_ResolveLocalImageID_RepoDigestFound(t *testing.T) {
	imageIDExpected := testImageID
	digest := testDigest
	imageName := "imgrepodigestfound"
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
		{
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
	imageIDExpected := testImageID
	ref, err := dockerRef.ParseAnyReference(imageIDExpected)
	if err != nil {
		t.Error(err)
	}
	localImageIDSet := digestset.NewSet()

	// Since go lazily registers Hash algorithms in the "crypto" module, digestPackage.Parse will fail unless we load the sha256 package.
	_ = sha256.New()
	d, err := digestPackage.Parse(testDigest)
	if err != nil {
		t.Error(err)
	}
	_ = localImageIDSet.Add(d)
	localImagesCache := []dockerTypes.ImageSummary{
		{
			ID: imageIDExpected,
		},
	}
	imageIDActual := ResolveLocalImageID(ref, localImageIDSet, localImagesCache)
	if imageIDActual != "sha256:"+imageIDExpected {
		t.Fail()
	}
}

func Test_ResolveLocalImageID_ImageIDNotFound(t *testing.T) {
	ref, err := dockerRef.ParseAnyReference(testDigest)
	if err != nil {
		t.Error(err)
	}
	localImageIDSet := digestset.NewSet()
	localImagesCache := []dockerTypes.ImageSummary{}
	imageID := ResolveLocalImageID(ref, localImageIDSet, localImagesCache)
	if imageID != "" {
		t.Fail()
	}
}
