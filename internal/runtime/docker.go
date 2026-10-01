package runtime

import (
	"context"
	"fmt"
	"io"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

// DockerRuntime is a ContainerRuntime backed by the Docker Engine API. It
// talks to the local Docker daemon, whose connection settings are taken from
// the environment (DOCKER_HOST and related variables).
type DockerRuntime struct {
	client *client.Client
}

// NewDockerRuntime creates a Docker client configured from the environment
// and negotiates the API version with the daemon.
func NewDockerRuntime() (*DockerRuntime, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &DockerRuntime{
		client: cli,
	}, nil
}

// Create creates the container without starting it. If the image is not
// present locally, it is pulled first and the creation is retried.
func (r *DockerRuntime) Create(ctx context.Context, spec ContainerSpec) (ContainerID, error) {
	cmd := append(append([]string{}, spec.Command...), spec.Args...)
	resp, err := r.client.ContainerCreate(
		ctx,
		&container.Config{
			Image: spec.Image,
			Cmd:   cmd,
		},
		nil,
		nil,
		nil,
		spec.Name,
	)
	if err != nil {
		if errdefs.IsNotFound(err) {
			reader, err := r.client.ImagePull(ctx, spec.Image, image.PullOptions{})
			if err != nil {
				return "", fmt.Errorf("failed to pull image %q: %w", spec.Image, err)
			}

			_, err = io.Copy(io.Discard, reader)
			reader.Close()

			if err != nil {
				return "", fmt.Errorf("failed while pulling image %q: %w", spec.Image, err)
			}
			resp, err = r.client.ContainerCreate(
				ctx,
				&container.Config{
					Image: spec.Image,
					Cmd:   cmd,
				},
				nil,
				nil,
				nil,
				spec.Name,
			)

			if err != nil {
				return "", fmt.Errorf("failed to create container after pulling image %q: %w", spec.Image, err)
			}
		} else {
			return "", fmt.Errorf("failed to create container: %w", err)
		}
	}

	return ContainerID(resp.ID), nil
}

func (r *DockerRuntime) Start(ctx context.Context, id ContainerID) error {
	return r.client.ContainerStart(
		ctx,
		string(id),
		container.StartOptions{},
	)
}

func (r *DockerRuntime) Inspect(ctx context.Context, id ContainerID) (ContainerInfo, error) {
	resp, err := r.client.ContainerInspect(
		ctx,
		string(id),
	)
	if err != nil {
		return ContainerInfo{}, fmt.Errorf("failed to inspect container: %w", err)
	}

	return ContainerInfo{
		ID:      ContainerID(resp.ID),
		Name:    resp.Name,
		Image:   resp.Image,
		State:   resp.State.Status,
		Running: resp.State.Running,
	}, nil
}

func (r *DockerRuntime) Stop(ctx context.Context, id ContainerID) error {
	err := r.client.ContainerStop(
		ctx,
		string(id),
		container.StopOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}
	return nil
}

func (r *DockerRuntime) Remove(ctx context.Context, id ContainerID) error {
	err := r.client.ContainerRemove(
		ctx,
		string(id),
		container.RemoveOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}
	return nil
}
