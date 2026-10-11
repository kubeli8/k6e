package controller

import (
	"context"
	"errors"
	"fmt"

	"github.com/kubeli8/k6e/internal/agentclient"
	"github.com/kubeli8/k6e/internal/model"
	"github.com/kubeli8/k6e/internal/runtime"
	"github.com/kubeli8/k6e/internal/store"
)

// ErrNodeUnavailable is returned when a node is not available / lookup fails.
var ErrNodeUnavailable = errors.New("node unavailable")

// RuntimeObserver queries the live state of a container associated with an
// assignment. It is used during reconciliation to detect containers that have
// stopped outside k6e's control.
type RuntimeObserver interface {
	Inspect(ctx context.Context, assignment model.Assignment) (runtime.ContainerInfo, error)
}

// HTTPRuntimeObserver implements RuntimeObserver by looking up the node address
// from the NodeStore and forwarding an inspect request to the agent running on
// that node.
type HTTPRuntimeObserver struct {
	nodes  store.NodeStore
	client *agentclient.Client
}

// NewHTTPRuntimeObserver constructs an HTTPRuntimeObserver that uses nodes to
// resolve node addresses and client to communicate with agents.
func NewHTTPRuntimeObserver(nodes store.NodeStore, client *agentclient.Client) *HTTPRuntimeObserver {
	return &HTTPRuntimeObserver{nodes: nodes, client: client}
}

func (o *HTTPRuntimeObserver) Inspect(ctx context.Context, assignment model.Assignment) (runtime.ContainerInfo, error) {
	node, err := o.nodes.GetNode(ctx, assignment.NodeID)
	if err != nil {
		return runtime.ContainerInfo{}, err
	}

	if node.Status != model.NodeStatusReady {
		return runtime.ContainerInfo{}, fmt.Errorf(
			"%w: node %s has status %s",
			ErrNodeUnavailable,
			node.ID,
			node.Status,
		)
	}

	return o.client.InspectContainer(ctx, node, runtime.ContainerID(assignment.ContainerID))
}
