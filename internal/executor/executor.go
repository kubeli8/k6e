// Package executor starts and stops the containers backing Assignments.
//
// The Service resolves an assignment to its workload and node, delegates the
// container operations to a ContainerExecutor (typically the control-plane
// client for the node agent), and records the resulting status and container
// ID back in the assignment store.
package executor

import (
	"context"

	"github.com/pyd-07/k6e/internal/model"
	"github.com/pyd-07/k6e/internal/runtime"
)

// ContainerExecutor performs the actual container operations on a specific
// node, decoupling the executor service from how containers are created and
// removed (for example, an HTTP call to the node agent).
type ContainerExecutor interface {
	RunContainer(ctx context.Context, node model.Node, spec runtime.ContainerSpec) (runtime.ContainerID, error)
	DeleteContainer(ctx context.Context, node model.Node, containerID runtime.ContainerID) error
}
