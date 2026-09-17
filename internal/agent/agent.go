package agent

import (
	"context"
	"errors"

	"github.com/pyd-07/k6e/internal/model"
	"github.com/pyd-07/k6e/internal/runtime"
)

type Agent struct {
	runtime   runtime.ContainerRuntime
	registrar NodeRegistrar
}

func New(rt runtime.ContainerRuntime, registrar ...NodeRegistrar) *Agent {
	ag := &Agent{
		runtime: rt,
	}
	if len(registrar) > 0 {
		ag.registrar = registrar[0]
	}
	return ag
}

func (a *Agent) Register(ctx context.Context, node model.Node) error {
	if a.registrar == nil {
		return errors.New("node registrar not configured")
	}

	return a.registrar.Register(ctx, node)
}

func (a *Agent) Run(ctx context.Context, spec runtime.ContainerSpec) (runtime.ContainerID, error) {
	id, err := a.runtime.Create(ctx, spec)
	if err != nil {
		return "", err
	}

	if err := a.runtime.Start(ctx, id); err != nil {
		// Cleanup: Remove the container if starting fails
		_ = a.runtime.Remove(ctx, id)
		return "", err
	}

	return id, nil
}

func (a *Agent) Stop(ctx context.Context, id runtime.ContainerID) error {
	if err := a.runtime.Stop(ctx, id); err != nil {
		return err
	}
	return nil
}

func (a *Agent) Remove(ctx context.Context, id runtime.ContainerID) error {
	if err := a.runtime.Remove(ctx, id); err != nil {
		return err
	}
	return nil
}

func (a *Agent) Inspect(ctx context.Context, id runtime.ContainerID) (runtime.ContainerInfo, error) {
	info, err := a.runtime.Inspect(ctx, id)
	if err != nil {
		return runtime.ContainerInfo{}, err
	}
	return info, nil
}
