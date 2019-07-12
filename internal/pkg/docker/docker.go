package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"strings"

	"github.com/docker/distribution/digestset"
	dockerRef "github.com/docker/distribution/reference"
	dockerTypes "github.com/docker/docker/api/types"
	dockerFilters "github.com/docker/docker/api/types/filters"
	"github.com/kube-compose/kube-compose/internal/pkg/util"
)

// Defaults useful when constructing fully qualified image refs.
// Sourced from https://github.com/moby/moby/blob/master/vendor/github.com/docker/distribution/reference/normalize.go#L14
var (
	defaultDomain    string
	officialRepoName string
)

func init() {
	named, _ := dockerRef.ParseNormalizedNamed("m")
	parts := strings.Split(named.String(), "/")
	defaultDomain = parts[0]
	officialRepoName = parts[1]
}

func DefaultDomain() string {
	return defaultDomain
}

func OfficialRepoName() string {
	return officialRepoName
}

func EncodeRegistryAuth(username, password string) string {
	authConfig := dockerTypes.AuthConfig{
		Username: username,
		Password: password,
	}
	authConfigBytes, _ := json.Marshal(&authConfig)
	return base64.StdEncoding.EncodeToString(authConfigBytes)
}

type ImagePuller interface {
	ImagePull(ctx context.Context, image string, pushOptions dockerTypes.ImagePullOptions) (io.ReadCloser, error)
}

func PullImage(ctx context.Context, puller ImagePuller, image, registryAuth string, onUpdate func(*PullOrPush)) (string, error) {
	pullOptions := dockerTypes.ImagePullOptions{
		RegistryAuth: registryAuth,
	}
	readCloser, err := puller.ImagePull(ctx, image, pullOptions)
	if err != nil {
		return "", err
	}
	defer util.CloseAndLogError(readCloser)
	pull := NewPull(readCloser)
	return pull.Wait(onUpdate)
}

type ImagePusher interface {
	ImagePush(ctx context.Context, image string, pushOptions dockerTypes.ImagePushOptions) (io.ReadCloser, error)
}

func PushImage(ctx context.Context, pusher ImagePusher, image, registryAuth string, onUpdate func(*PullOrPush)) (string, error) {
	pushOptions := dockerTypes.ImagePushOptions{
		RegistryAuth: registryAuth,
	}
	readCloser, err := pusher.ImagePush(ctx, image, pushOptions)
	if err != nil {
		return "", err
	}
	defer util.CloseAndLogError(readCloser)
	push := NewPush(readCloser)
	return push.Wait(onUpdate)
}

type ImageLister interface {
	ImageList(ctx context.Context, listOptions dockerTypes.ImageListOptions) ([]dockerTypes.ImageSummary, error)
}

// ResolveLocalImageAfterPull resolves an image based on a repository and digest by querying the docker daemon.
// This is exactly the information we have available after pulling an image.
// Returns the image ID, repo digest and optionally an error. A repo digest is a familiar name with an "@" and digest.
func ResolveLocalImageAfterPull(
	ctx context.Context,
	lister ImageLister,
	named dockerRef.Named,
	digest string) (imageID, repoDigest string, err error) {
	filters := dockerFilters.NewArgs()
	familiarName := dockerRef.FamiliarName(named)
	filters.Add("reference", familiarName)
	imageSummaries, err := lister.ImageList(ctx, dockerTypes.ImageListOptions{
		All:     false,
		Filters: filters,
	})
	if err != nil {
		return "", "", err
	}
	repoDigest = familiarName + "@" + digest
	for i := 0; i < len(imageSummaries); i++ {
		for _, repoDigest2 := range imageSummaries[i].RepoDigests {
			if repoDigest == repoDigest2 {
				return imageSummaries[i].ID, repoDigest, nil
			}
		}
	}
	return "", "", nil
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
