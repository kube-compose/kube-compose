package service

import (
	"context"
	"io"

	"github.com/docker/distribution/digestset"
	dockerRef "github.com/docker/distribution/reference"
	dockerTypes "github.com/docker/docker/api/types"
)

type Progress interface {
	Progress() float64
}

type ImageBuildOptions struct {
	BuildArgs    map[string]*string
	BuildContext io.Reader
	Context      context.Context
	OnUpdate     func(p Progress)
}

type ContainerService interface {
	io.Closer
	ContainerCreateForCopyFromContainer(ctx context.Context, image string) (string, error)
	ContainerRemove(ctx context.Context, containerID string) error
	CopyFromContainerToFile(ctx context.Context, containerID, srcFile, dstFile string) error
	ImageBuild(opts *ImageBuildOptions) (string, error)
	ImageInspectWithRaw(ctx context.Context, imageID string) (dockerTypes.ImageInspect, []byte, error)
	ImageList(
		ctx context.Context,
		listOptions dockerTypes.ImageListOptions,
	) ([]dockerTypes.ImageSummary, error)
	ImageTag(ctx context.Context, source, target string) error
	PullImage(
		ctx context.Context,
		image, registryAuth string,
		onUpdate func(Progress),
	) (digest string, err error)
	PushImage(
		ctx context.Context,
		image, registryAuth string,
		onUpdate func(Progress),
	) (digest string, err error)
	ResolveLocalImageAfterPull(
		ctx context.Context,
		named dockerRef.Named,
		digest string,
	) (imageID, repoDigest string, err error)
}

// ResolveLocalImageID resolves an image ID against a cached list (like the one output by the command "docker images").
// ref is assumed not to be a partial image ID.
func ResolveLocalImageID(ref dockerRef.Reference, localImageIDSet *digestset.Set, localImagesCache []dockerTypes.ImageSummary) string {
	named, isNamed := ref.(dockerRef.Named)
	digested, isDigested := ref.(dockerRef.Digested)
	// By definition of dockerRef.ParseAnyReferenceWithSet isNamed or isDigested is true
	if !isNamed {
		imageID := digested.String()
		if _, err := localImageIDSet.Lookup(imageID); err == digestset.ErrDigestNotFound {
			return ""
		}
		// The only other error returned by Lookup is a digestset.ErrDigestAmbiguous, which cannot
		// happen by our assumption that ref cannot be a partial image ID
		return imageID
	}
	familiarName := dockerRef.FamiliarName(named)
	// The source image must be named
	if isDigested {
		// docker images returns RepoDigests as a familiar name with a digest
		repoDigest := familiarName + "@" + string(digested.Digest())
		for i := 0; i < len(localImagesCache); i++ {
			for _, repoDigest2 := range localImagesCache[i].RepoDigests {
				if repoDigest == repoDigest2 {
					return localImagesCache[i].ID
				}
			}
		}
	}
	return resolveLocalImageIDTag(ref, familiarName, localImagesCache)
}

func resolveLocalImageIDTag(ref dockerRef.Reference, familiarName string, localImagesCache []dockerTypes.ImageSummary) string {
	tag := getTag(ref)
	if len(tag) > 0 {
		// docker images returns RepoTags as a familiar name with a tag
		repoTag := familiarName + ":" + tag
		for i := 0; i < len(localImagesCache); i++ {
			for _, repoTag2 := range localImagesCache[i].RepoTags {
				if repoTag == repoTag2 {
					return localImagesCache[i].ID
				}
			}
		}
	}
	return ""
}

type hasTag interface {
	Tag() string
}

func getTag(ref dockerRef.Reference) string {
	refWithTag, ok := ref.(hasTag)
	if !ok {
		return ""
	}
	return refWithTag.Tag()
}
