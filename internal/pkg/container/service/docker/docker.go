package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	dockerRef "github.com/docker/distribution/reference"
	dockerTypes "github.com/docker/docker/api/types"
	dockerContainers "github.com/docker/docker/api/types/container"
	dockerClient "github.com/docker/docker/client"
	dockerArchive "github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/kube-compose/kube-compose/internal/pkg/container/service"
	"github.com/kube-compose/kube-compose/internal/pkg/util"
	"github.com/pkg/errors"
)

type dockerContainerService struct {
	dc *dockerClient.Client
}

// New creates a new service.ContainerService backed by a docker daemon.
func New(dc *dockerClient.Client) service.ContainerService {
	d := &dockerContainerService{
		dc: dc,
	}
	return d
}

func (d *dockerContainerService) Close() error {
	return d.dc.Close()
}

func (d *dockerContainerService) ContainerCreateForCopyFromContainer(ctx context.Context, image string) (string, error) {
	containerConfig := &dockerContainers.Config{
		Entrypoint: []string{"sh"},
		Image:      image,
		WorkingDir: "/",
	}
	resp, err := d.dc.ContainerCreate(ctx, containerConfig, nil, nil, "")
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

func (d *dockerContainerService) CopyFromContainerToFile(ctx context.Context, containerID, srcFile, dstFile string) error {
	readCloser, stat, err := d.dc.CopyFromContainer(ctx, containerID, srcFile)
	if err != nil {
		return err
	}
	defer util.CloseAndLogError(readCloser)
	if (stat.Mode & os.ModeType) != 0 {
		// TODO https://github.com/kube-compose/kube-compose/issues/70 we should follow symlinks
		return fmt.Errorf("could not copy %#v because it is not a regular file", srcFile)
	}
	srcInfo := dockerArchive.CopyInfo{
		Path:       srcFile,
		Exists:     true,
		IsDir:      false,
		RebaseName: "",
	}
	err = dockerArchive.CopyTo(readCloser, srcInfo, dstFile)
	if err != nil {
		return errors.Wrapf(err, "error while copying image file %#v to local file %#v", srcFile, dstFile)
	}
	return nil
}

func (d *dockerContainerService) ContainerRemove(ctx context.Context, containerID string) error {
	return d.dc.ContainerRemove(ctx, containerID, dockerTypes.ContainerRemoveOptions{})
}

func (d *dockerContainerService) ImageBuild(opts *service.ImageBuildOptions) (string, error) {
	response, err := d.dc.ImageBuild(
		opts.Context,
		opts.BuildContext,
		dockerTypes.ImageBuildOptions{
			BuildArgs: opts.BuildArgs,
			// Only the image ID is output when SupressOutput is true.
			SuppressOutput: true,
			Remove:         true,
		},
	)
	if err != nil {
		return "", err
	}
	decoder := json.NewDecoder(response.Body)
	var lastImageID string
	for {
		var msg jsonmessage.JSONMessage
		err = decoder.Decode(&msg)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}

		if imageID := FindDigest(msg.Stream); imageID != "" {
			lastImageID = imageID
		}
	}
	if lastImageID == "" {
		return "", fmt.Errorf("could not parse image ID from docker build output stream")
	}
	return lastImageID, nil
}

func (d *dockerContainerService) ImageInspectWithRaw(ctx context.Context, imageID string) (dockerTypes.ImageInspect, []byte, error) {
	return d.dc.ImageInspectWithRaw(ctx, imageID)
}

func (d *dockerContainerService) ImageList(
	ctx context.Context,
	listOptions dockerTypes.ImageListOptions) (
	[]dockerTypes.ImageSummary,
	error) {
	return d.dc.ImageList(ctx, listOptions)
}

func (d *dockerContainerService) ImageTag(ctx context.Context, source, target string) error {
	return d.dc.ImageTag(ctx, source, target)
}

func (d *dockerContainerService) ImagePull(
	ctx context.Context,
	image, registryAuth string,
	onUpdate func(service.Progress),
) (imageID string, err error) {
	digest, err := imagePull(ctx, d.dc, image, registryAuth, func(pull *pullOrPush) {
		onUpdate(pull)
	})
	if err != nil {
		return
	}
	var named dockerRef.Named
	named, err = dockerRef.ParseNormalizedNamed(image)
	if err != nil {
		panic(errors.Wrapf(err, "could not image %#v as named image but docker pull succeeded, please report this as a bug at " +
			"https://github.com/kube-compose/kube-compose", image))
	}
	imageID, err = imagePullResolve(ctx, d.dc, named, digest)
	return
}

func (d *dockerContainerService) ImagePush(
	ctx context.Context,
	image, registryAuth string,
	onUpdate func(service.Progress),
) (digest string, err error) {
	return imagePush(ctx, d.dc, image, registryAuth, func(push *pullOrPush) {
		onUpdate(push)
	})
}

func (d *dockerContainerService) ImagePullResolve(
	ctx context.Context,
	named dockerRef.Named,
	digest string,
) (imageID, repoDigest string, err error) {
	return imagePullResolve(ctx, d.dc, named, digest)
}
