package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kubeli8/k6e/internal/agentclient"
	"github.com/kubeli8/k6e/internal/model"
	"github.com/kubeli8/k6e/internal/store"
)

func TestHTTPRuntimeObserver(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":"container","running":true,"state":"running"}`))
	}))
	defer server.Close()
	nodes := store.NewMemoryNodeStore()
	if err := nodes.RegisterNode(context.Background(), model.Node{ID: "node", Address: server.URL, Status: model.NodeStatusReady}); err != nil {
		t.Fatal(err)
	}
	info, err := NewHTTPRuntimeObserver(nodes, agentclient.New(server.Client())).Inspect(context.Background(), model.Assignment{NodeID: "node", ContainerID: "container"})
	if err != nil || !info.Running {
		t.Fatalf("info=%+v err=%v", info, err)
	}
}
