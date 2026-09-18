package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pyd-07/k6e/internal/model"
)

func testNode(id, address string, status model.NodeStatus) model.Node {
	return model.Node{
		ID:            id,
		Address:       address,
		Status:        status,
		LastHeartbeat: time.Now(),
	}
}

func TestHTTPNodeRegistrarRegister(t *testing.T) {
	var recieved model.Node

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		if r.URL.Path != "/api/v1/nodes" {
			t.Errorf("expected /api/v1/nodes, got %s", r.URL.Path)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json content type")
		}

		if err := json.NewDecoder(r.Body).Decode(&recieved); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	registrar := NewHTTPNodeRegistrar(server.URL)
	node := testNode("test-node", "127.0.0.1:8080", model.NodeStatusReady)

	err := registrar.Register(context.Background(), node)
	if err != nil {
		t.Fatalf("failed to register node: %v", err)
	}

	if recieved.ID != node.ID ||
		recieved.Address != node.Address ||
		recieved.Status != node.Status {
		t.Errorf("want %+v, got %+v", node, recieved)
	}
}

func TestHTTPNodeRegistrarRegisterServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	registrar := NewHTTPNodeRegistrar(server.URL)
	node := testNode("test-node", "127.0.0.1:8080", model.NodeStatusReady)

	err := registrar.Register(context.Background(), node)
	if err == nil {
		t.Fatalf("expected Register() to return an error, got nil")
	}
}
