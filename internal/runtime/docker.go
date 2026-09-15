package runtime

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type DockerRuntime struct {
	client *client.Client
}

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

func (r *DockerRuntime) Create(ctx context.Context, spec ContainerSpec) (ContainerID, error) {
	resp, err := r.client.ContainerCreate(
		ctx,
		&container.Config{
			Image: spec.Image,
			Cmd:   spec.Command,
		},
		nil,
		nil,
		nil,
		spec.Name,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
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
