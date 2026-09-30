package executor

import (
	"context"

	"github.com/pyd-07/k6e/internal/model"
	"github.com/pyd-07/k6e/internal/runtime"
)

type ContainerExecutor interface {
	RunContainer(ctx context.Context, node model.Node, spec runtime.ContainerSpec) (runtime.ContainerID, error)
	DeleteContainer(ctx context.Context, node model.Node, containerID runtime.ContainerID) error
}
