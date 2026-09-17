package store

import (
	"context"

	"github.com/pyd-07/k6e/internal/model"
)

type NodeStore interface {
	RegisterNode(ctx context.Context, node model.Node) error
	GetNode(ctx context.Context, id string) (model.Node, error)
	ListNodes(ctx context.Context) ([]model.Node, error)
	RemoveNode(ctx context.Context, id string) error
}
