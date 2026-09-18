package agent

import (
	"context"
	"fmt"
	"net/http"
)

type HTTPNodeHeartbeater struct {
	baseURL string
	client  *http.Client
}

type NodeHeartbeater interface {
	Heartbeat(ctx context.Context, nodeID string) error
}

func NewHTTPNodeHeartbeater(baseURL string) *HTTPNodeHeartbeater {
	return &HTTPNodeHeartbeater{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

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
