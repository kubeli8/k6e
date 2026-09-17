package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pyd-07/k6e/internal/model"
)

type NodeRegistrar interface {
	Register(ctx context.Context, node model.Node) error
}

type HTTPNodeRegistrar struct {
	baseURL string
	client *http.Client
}

func NewHTTPNodeRegistrar(baseURL string) *HTTPNodeRegistrar {
	return &HTTPNodeRegistrar{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

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
