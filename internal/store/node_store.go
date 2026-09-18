package store

import (
	"context"
	"time"

	"github.com/pyd-07/k6e/internal/model"
)

type NodeStore interface {
	RegisterNode(ctx context.Context, node model.Node) error
	GetNode(ctx context.Context, id string) (model.Node, error)
	ListNodes(ctx context.Context) ([]model.Node, error)
	RemoveNode(ctx context.Context, id string) error
	UpdateHeartbeat(ctx context.Context, id string, timestamp time.Time) error
	UpdateNodeStatus(ctx context.Context, id string, status model.NodeStatus) error
}
