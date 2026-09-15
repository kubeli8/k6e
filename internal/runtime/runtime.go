package runtime

import "context"

type ContainerID string

type ContainerSpec struct {
	Name    string
	Image   string
	Command []string
}

type ContainerInfo struct {
	ID      ContainerID
	Name    string
	Image   string
	State   string
	Running bool
}

type ContainerRuntime interface {
	Create(ctx context.Context, spec ContainerSpec) (ContainerID, error)
	Start(ctx context.Context, id ContainerID) error
	Inspect(ctx context.Context, id ContainerID) (ContainerInfo, error)
	Stop(ctx context.Context, id ContainerID) error
	Remove(ctx context.Context, id ContainerID) error
}
