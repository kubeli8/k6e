package controller

import (
	"context"
	"errors"
	"time"

	"github.com/kubeli8/k6e/internal/model"
	"github.com/kubeli8/k6e/internal/store"
)

// LivenessChecker monitors registered nodes and marks them NotReady when they
// exceed the heartbeat timeout. It is intended to run as a background goroutine
// alongside the Controller.
type LivenessChecker struct {
	store   store.NodeStore
	timeout time.Duration
}

// NewLivenessChecker constructs a LivenessChecker. timeout is the maximum
// duration a node may go without a heartbeat before being marked NotReady.
func NewLivenessChecker(nodeStore store.NodeStore, timeout time.Duration) *LivenessChecker {
	return &LivenessChecker{
		store:   nodeStore,
		timeout: timeout,
	}
}

// CheckLiveness performs a single liveness pass over all registered nodes.
// Any node whose last heartbeat exceeds the configured timeout is transitioned
// to NodeStatusNotReady. Nodes that are already NotReady are skipped to avoid
// redundant store writes.
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

// Start runs CheckLiveness on every tick of interval until ctx is cancelled.
// It returns nil when the context is done and propagates any error from
// CheckLiveness immediately, stopping the loop.
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
