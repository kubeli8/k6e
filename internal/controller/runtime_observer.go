package controller

import (
	"context"

	"github.com/pyd-07/k6e/internal/model"
	"github.com/pyd-07/k6e/internal/runtime"
)

type RuntimeObserver interface {
	Inspect(ctx context.Context, assignment model.Assignment) (runtime.ContainerInfo, error)
}
