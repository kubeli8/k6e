package controller

import (
	"context"

	"github.com/pyd-07/k6e/internal/agent"
	"github.com/pyd-07/k6e/internal/model"
	"github.com/pyd-07/k6e/internal/runtime"
)

type RuntimeObserver interface {
	Inspect(ctx context.Context, assignment model.Assignment) (runtime.ContainerInfo, error)
}

type AgentRuntimeObserver struct {
	agent *agent.Agent
}

func NewAgentRuntimeObserver(agent *agent.Agent) *AgentRuntimeObserver {
	return &AgentRuntimeObserver{
		agent: agent,
	}
}

func (o *AgentRuntimeObserver) Inspect(ctx context.Context, assignment model.Assignment) (runtime.ContainerInfo, error) {
	return o.agent.Inspect(ctx, runtime.ContainerID(assignment.ContainerID))
}
