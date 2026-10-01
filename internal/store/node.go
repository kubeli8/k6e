package store

import (
	"context"
	"time"

	"github.com/pyd-07/k6e/internal/model"
)

// NodeStore is the persistence interface for Node records.
type NodeStore interface {
	// RegisterNode persists a node, overwriting any existing record with the
	// same ID. This makes registration idempotent so that an agent can safely
	// re-register after a restart.
	RegisterNode(ctx context.Context, node model.Node) error
	GetNode(ctx context.Context, id string) (model.Node, error)
	ListNodes(ctx context.Context) ([]model.Node, error)
	RemoveNode(ctx context.Context, id string) error
	// UpdateHeartbeat records the latest heartbeat timestamp for a node and
	// transitions its status to NodeStatusReady.
	UpdateHeartbeat(ctx context.Context, id string, timestamp time.Time) error
	UpdateNodeStatus(ctx context.Context, id string, status model.NodeStatus) error
}
