package controller

import (
	"context"

	"github.com/pyd-07/k6e/internal/agentclient"
	"github.com/pyd-07/k6e/internal/model"
	"github.com/pyd-07/k6e/internal/runtime"
	"github.com/pyd-07/k6e/internal/store"
)

type RuntimeObserver interface {
	Inspect(ctx context.Context, assignment model.Assignment) (runtime.ContainerInfo, error)
}

type HTTPRuntimeObserver struct {
	nodes  store.NodeStore
	client *agentclient.Client
}

func NewHTTPRuntimeObserver(nodes store.NodeStore, client *agentclient.Client) *HTTPRuntimeObserver {
	return &HTTPRuntimeObserver{nodes: nodes, client: client}
}

func (o *HTTPRuntimeObserver) Inspect(ctx context.Context, assignment model.Assignment) (runtime.ContainerInfo, error) {
	node, err := o.nodes.GetNode(ctx, assignment.NodeID)
	if err != nil {
		return runtime.ContainerInfo{}, err
	}
	return o.client.InspectContainer(ctx, node, runtime.ContainerID(assignment.ContainerID))
}
