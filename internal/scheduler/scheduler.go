// Package scheduler selects a node for a workload replica and persists a new
// Assignment in the Pending state. It is invoked by the controller for each
// replica that needs to be created.
package scheduler

import (
	"context"
	"errors"

	"github.com/pyd-07/k6e/internal/model"
)

// ErrNoReadyNodes is returned by a Scheduler when no node with
// NodeStatusReady is available to accept the workload.
var ErrNoReadyNodes = errors.New("no ready nodes available")

// Scheduler selects a node from the provided list on which to place a
// workload replica. Implementations may use any placement strategy.
type Scheduler interface {
	Schedule(ctx context.Context, workload model.Workload, nodes []model.Node) (model.Node, error)
}

// SimpleScheduler is a first-fit placement strategy that returns the first
// node with NodeStatusReady. It does not consider load or resource capacity.
type SimpleScheduler struct{}

// Schedule returns the first Ready node in nodes, or ErrNoReadyNodes if none
// is available.
func (s *SimpleScheduler) Schedule(ctx context.Context, workload model.Workload, nodes []model.Node) (model.Node, error) {
	for _, node := range nodes {
		if node.Status == model.NodeStatusReady {
			return node, nil
		}
	}

	return model.Node{}, ErrNoReadyNodes
}
