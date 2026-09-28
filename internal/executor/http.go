package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/pyd-07/k6e/internal/model"
	"github.com/pyd-07/k6e/internal/runtime"
)

type HTTPContainerExecutor struct {
	client *http.Client
}

func NewHTTPContainerExecutor(client *http.Client) *HTTPContainerExecutor {
	if client == nil {
		client = http.DefaultClient
	}

	return &HTTPContainerExecutor{
		client: client,
	}
}

type runContainerRequest struct {
	Name    string   `json:"name"`
	Image   string   `json:"image"`
	Command []string `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
}

type runContainerResponse struct {
	ContainerID string `json:"containerID"`
}

func (e *HTTPContainerExecutor) Execute(ctx context.Context, node model.Node, spec runtime.ContainerSpec) (runtime.ContainerID, error) {
	addr := strings.TrimRight(node.Address, "/")
	if addr == "" {
		return "", fmt.Errorf("node %s has no address", node.ID)
	}

	reqBody := runContainerRequest{
		Name:    spec.Name,
		Image:   spec.Image,
		Command: spec.Command,
		Args:    spec.Args,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal container request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"http://"+addr+"/api/v1/containers",
		bytes.NewReader(body),
	)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response runContainerResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if response.ContainerID == "" {
		return "", fmt.Errorf("empty container ID in response")
	}

	return runtime.ContainerID(response.ContainerID), nil
}
