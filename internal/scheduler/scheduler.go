package scheduler

import (
	"context"
	"errors"

	"github.com/pyd-07/k6e/internal/model"
)

var ErrNoReadyNodes = errors.New("no ready nodes available")

type Scheduler interface {
	Schedule(ctx context.Context, workload model.Workload, nodes []model.Node) (model.Node, error)
}

type SimpleScheduler struct{}

func (s *SimpleScheduler) Schedule(ctx context.Context, workload model.Workload, nodes []model.Node) (model.Node, error) {
	for _, node := range nodes {
		if node.Status == model.NodeStatusReady {
			return node, nil
		}
	}

	return model.Node{}, ErrNoReadyNodes
}
