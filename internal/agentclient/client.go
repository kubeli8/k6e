// Package agentclient provides the control-plane client for worker agents.
package agentclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pyd-07/k6e/internal/model"
	"github.com/pyd-07/k6e/internal/runtime"
)

// Client is the control-plane-side HTTP client for a node agent. The
// executor uses it to run and delete containers, and the runtime observer
// uses it to inspect container state on remote nodes.
type Client struct{ client *http.Client }

// New constructs a Client. If client is nil, a default client with a
// 5-minute timeout is used, accommodating slow image pulls on RunContainer.
func New(client *http.Client) *Client {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Minute}
	}
	return &Client{client: client}
}

type runContainerRequest struct {
	Name    string   `json:"name"`
	Image   string   `json:"image"`
	Command []string `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
}

type runContainerResponse struct {
	ContainerID string `json:"containerId"`
}

// RunContainer asks node's agent to create and start a container and returns
// its ID. It expects a 201 Created response and rejects empty container IDs.
func (c *Client) RunContainer(ctx context.Context, node model.Node, spec runtime.ContainerSpec) (runtime.ContainerID, error) {
	body, err := json.Marshal(runContainerRequest{Name: spec.Name, Image: spec.Image, Command: spec.Command, Args: spec.Args})
	if err != nil {
		return "", fmt.Errorf("marshal container request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, nodeURL(node, "/api/v1/containers"), bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	var response runContainerResponse
	if err := c.doJSON(req, http.StatusCreated, &response); err != nil {
		return "", err
	}

	if response.ContainerID == "" {
		return "", fmt.Errorf("empty container ID in response")
	}
	return runtime.ContainerID(response.ContainerID), nil
}

// InspectContainer fetches the current state of a container from node's
// agent. It returns an error when the container ID is empty.
func (c *Client) InspectContainer(ctx context.Context, node model.Node, id runtime.ContainerID) (runtime.ContainerInfo, error) {
	if id == "" {
		return runtime.ContainerInfo{}, fmt.Errorf("container ID is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, nodeURL(node, "/api/v1/containers/"+string(id)), nil)
	if err != nil {
		return runtime.ContainerInfo{}, fmt.Errorf("create request: %w", err)
	}

	var info runtime.ContainerInfo
	if err := c.doJSON(req, http.StatusOK, &info); err != nil {
		return runtime.ContainerInfo{}, err
	}

	return info, nil
}

// DeleteContainer asks node's agent to stop and remove the container, and
// expects a 204 No Content response.
func (c *Client) DeleteContainer(ctx context.Context, node model.Node, id runtime.ContainerID) error {
	if id == "" {
		return fmt.Errorf("container ID is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, nodeURL(node, "/api/v1/containers/"+string(id)), nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	return c.doJSON(req, http.StatusNoContent, nil)
}

// doJSON sends req and enforces that the response status equals
// expectedStatus exactly; on mismatch it fails with the first 4 KiB of the
// response body. When target is non-nil the response body is decoded into it.
func (c *Client) doJSON(req *http.Request, expectedStatus int, target any) error {
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return fmt.Errorf("unexpected status code: %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if target == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// nodeURL builds the absolute URL for a request to node's agent, prefixing
// the node address with http:// when it carries no scheme.
func nodeURL(node model.Node, path string) string {
	address := strings.TrimRight(node.Address, "/")
	if !strings.HasPrefix(address, "http://") && !strings.HasPrefix(address, "https://") {
		address = "http://" + address
	}
	return address + path
}
