package agentclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pyd-07/k6e/internal/model"
	"github.com/pyd-07/k6e/internal/runtime"
)

func TestRunContainer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/containers" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"containerId":"container-1"}`))
	}))
	defer server.Close()
	client := New(server.Client())
	id, err := client.RunContainer(context.Background(), model.Node{Address: server.URL}, runtime.ContainerSpec{Name: "nginx", Image: "nginx"})
	if err != nil || id != "container-1" {
		t.Fatalf("id=%q err=%v", id, err)
	}
}

func TestClientRejectsErrorMalformedAndEmptyResponses(t *testing.T) {
	for _, body := range []string{"failure", "not-json", `{}`} {
		t.Run(body, func(t *testing.T) {
			status := http.StatusCreated
			if body == "failure" {
				status = http.StatusBadGateway
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status); _, _ = w.Write([]byte(body)) }))
			defer server.Close()
			_, err := New(server.Client()).RunContainer(context.Background(), model.Node{Address: server.URL}, runtime.ContainerSpec{})
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestInspectContainer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/containers/container-1" {
			t.Fatal(r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":"container-1","running":false,"state":"exited"}`))
	}))
	defer server.Close()
	info, err := New(server.Client()).InspectContainer(context.Background(), model.Node{Address: server.URL}, "container-1")
	if err != nil || info.Running || info.State != "exited" {
		t.Fatalf("info=%+v err=%v", info, err)
	}
}

func TestClientHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := New(http.DefaultClient).RunContainer(ctx, model.Node{Address: "localhost:1"}, runtime.ContainerSpec{})
	if err == nil {
		t.Fatal("expected cancellation error")
	}
}

func TestDeleteContainer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v1/containers/container-1" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	err := New(server.Client()).DeleteContainer(context.Background(), model.Node{Address: server.URL}, "container-1")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
}

func TestDeleteContainerEmptyID(t *testing.T) {
	err := New(http.DefaultClient).DeleteContainer(context.Background(), model.Node{Address: "localhost:1"}, "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDeleteContainerUnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()
	err := New(server.Client()).DeleteContainer(context.Background(), model.Node{Address: server.URL}, "container-1")
	if err == nil {
		t.Fatal("expected error")
	}
}
