package controller

import (
	"context"
	"errors"
	"time"

	"github.com/pyd-07/k6e/internal/model"
	"github.com/pyd-07/k6e/internal/store"
)

type LivenessChecker struct {
	store   store.NodeStore
	timeout time.Duration
}

func NewLivenessChecker(nodeStore store.NodeStore, timeout time.Duration) *LivenessChecker {
	return &LivenessChecker{
		store:   nodeStore,
		timeout: timeout,
	}
}

func (c *LivenessChecker) CheckLiveness(ctx context.Context) error {
	if c.store == nil {
		return errors.New("node store not configurd")
	}

	if c.timeout <= 0 {
		return errors.New("invalid timeout duration")
	}

	nodes, err := c.store.ListNodes(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, node := range nodes {
		if now.Sub(node.LastHeartbeat) > c.timeout {
			if node.Status == model.NodeStatusNotReady {
				continue
			}
			if err := c.store.UpdateNodeStatus(ctx, node.ID, model.NodeStatusNotReady); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *LivenessChecker) Start(ctx context.Context, interval time.Duration) error {
	if c.store == nil {
		return errors.New("node store not configured")
	}

	if c.timeout <= 0 {
		return errors.New("liveness timeout must be greater than zero")
	}

	if interval <= 0 {
		return errors.New("liveness check interval must be greater than zero")
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.C:
			if err := c.CheckLiveness(ctx); err != nil {
				return err
			}
		}
	}
}
