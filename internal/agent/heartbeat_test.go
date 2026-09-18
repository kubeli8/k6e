package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPNodeHeartbeaterHeartbeat(t *testing.T) {
	var receivedNodeID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/nodes/test-node/heartbeat" {
			t.Errorf("expected /api/v1/nodes/test-node/heartbeat, got %s", r.URL.Path)
		}
		receivedNodeID = "test-node"
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	heartbeater := NewHTTPNodeHeartbeater(server.URL)
	err := heartbeater.Heartbeat(context.Background(), "test-node")
	if err != nil {
		t.Fatalf("failed to send heartbeat: %v", err)
	}

	if receivedNodeID != "test-node" {
		t.Errorf("expected node ID 'test-node', got '%s'", receivedNodeID)
	}
}

func TestHTTPNodeHeartbeaterHeartbeatFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	heartbeater := NewHTTPNodeHeartbeater(server.URL)
	err := heartbeater.Heartbeat(context.Background(), "test-node")
	if err == nil {
		t.Fatalf("expected error on heartbeat failure, got nil")
	}
}
