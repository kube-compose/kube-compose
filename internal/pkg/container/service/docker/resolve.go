package docker

import (
	"context"

	dockerRef "github.com/docker/distribution/reference"
	dockerTypes "github.com/docker/docker/api/types"
	dockerFilters "github.com/docker/docker/api/types/filters"
)

type ImageLister interface {
	ImageList(ctx context.Context, listOptions dockerTypes.ImageListOptions) ([]dockerTypes.ImageSummary, error)
}

// imagePullResolve resolves an image based on a repository and digest by querying the docker daemon.
// This is exactly the information we have available after pulling an image.
// Returns the image ID, repo digest and optionally an error. A repo digest is a familiar name with an "@" and digest.
func imagePullResolve(
	ctx context.Context,
	lister ImageLister,
	named dockerRef.Named,
	digest string) (imageID string, err error) {
	filters := dockerFilters.NewArgs()
	familiarName := dockerRef.FamiliarName(named)
	filters.Add("reference", familiarName)
	imageSummaries, err := lister.ImageList(ctx, dockerTypes.ImageListOptions{
		All:     false,
		Filters: filters,
	})
	if err != nil {
		return "", err
	}
	repoDigest = familiarName + "@" + digest
	for i := 0; i < len(imageSummaries); i++ {
		for _, repoDigest2 := range imageSummaries[i].RepoDigests {
			if repoDigest == repoDigest2 {
				return imageSummaries[i].ID, nil
			}
		}
	}
	return "", nil
}
