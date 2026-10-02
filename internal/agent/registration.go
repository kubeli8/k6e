package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kubeli8/k6e/internal/model"
)

// NodeRegistrar registers a node with the control plane so that the
// scheduler can place assignments on it.
type NodeRegistrar interface {
	Register(ctx context.Context, node model.Node) error
}

// HTTPNodeRegistrar registers nodes by POSTing the node record to the
// control plane's node endpoint. Registration is an upsert on the control
// plane, so it is safe to re-register after an agent restart.
type HTTPNodeRegistrar struct {
	baseURL string
	client  *http.Client
}

// NewHTTPNodeRegistrar constructs a registrar targeting the control plane
// at baseURL.
func NewHTTPNodeRegistrar(baseURL string) *HTTPNodeRegistrar {
	return &HTTPNodeRegistrar{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

// Register marshals the node record and POSTs it to the control plane. It
// expects a 201 Created response and treats any other status as failure.
func (r *HTTPNodeRegistrar) Register(ctx context.Context, node model.Node) error {
	body, err := json.Marshal(node)
	if err != nil {
		return fmt.Errorf("marshal node: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		r.baseURL+"/api/v1/nodes",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("create registration request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("send registration request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("node resgistration failed: status %s", resp.Status)
	}

	return nil
}
