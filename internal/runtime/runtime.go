// Package runtime abstracts the container runtime used to create, start, and
// inspect containers on a node. The control plane's executor and the node
// agent both operate on these interfaces, keeping them independent of Docker
// itself.
package runtime

import "context"

// ContainerID identifies a container within the runtime.
type ContainerID string

// ContainerSpec describes the container to create: its name, image, and
// optional command and arguments.
type ContainerSpec struct {
	Name    string
	Image   string
	Command []string
	Args    []string
}

// ContainerInfo is the observed state of a container as reported by the
// runtime. Running is what the controller relies on to detect containers
// that have stopped outside k6e's control.
type ContainerInfo struct {
	ID      ContainerID
	Name    string
	Image   string
	State   string
	Running bool
}

// ContainerRuntime is the lifecycle interface for containers on a node.
// Create allocates a container without starting it; Start runs it. Inspect
// reports the current state; Stop and Remove terminate and clean up.
type ContainerRuntime interface {
	Create(ctx context.Context, spec ContainerSpec) (ContainerID, error)
	Start(ctx context.Context, id ContainerID) error
	Inspect(ctx context.Context, id ContainerID) (ContainerInfo, error)
	Stop(ctx context.Context, id ContainerID) error
	Remove(ctx context.Context, id ContainerID) error
}
