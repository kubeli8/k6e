package agent

import (
	"context"
	"fmt"
	"net/http"
)

// HTTPNodeHeartbeater reports node liveness to the control plane by POSTing
// to its heartbeat endpoint.
type HTTPNodeHeartbeater struct {
	baseURL string
	client  *http.Client
}

// NodeHeartbeater sends heartbeats for a node to the control plane. A node
// that stops heartbeating past the control plane's timeout is marked
// NotReady by the liveness checker.
type NodeHeartbeater interface {
	Heartbeat(ctx context.Context, nodeID string) error
}

// NewHTTPNodeHeartbeater constructs a heartbeater targeting the control
// plane at baseURL.
func NewHTTPNodeHeartbeater(baseURL string) *HTTPNodeHeartbeater {
	return &HTTPNodeHeartbeater{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

// Heartbeat POSTs to the control plane's heartbeat endpoint for nodeID. It
// expects a 204 No Content response and treats any other status as failure.
func (h *HTTPNodeHeartbeater) Heartbeat(ctx context.Context, nodeID string) error {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		h.baseURL+"/api/v1/nodes/"+nodeID+"/heartbeat",
		nil,
	)
	if err != nil {
		return fmt.Errorf("create heartbeat request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("send heartbeat request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("heartbeat failed: status %s", resp.Status)
	}
	return nil
}
