package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pyd-07/k6e/internal/runtime"
)

func TestServerRunContainer(t *testing.T) {
	fakeRuntime := &FakeRuntime{}
	ag := New(fakeRuntime, nil, nil)
	server := NewServer(ag)

	body := RunContainerRequest{
		Name:    "test-container",
		Image:   "alpine",
		Command: []string{"sleep", "60"},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/containers",
		bytes.NewReader(payload),
	)

	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, not %d", rec.Code)
	}

	var response RunContainerResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.ContainerID != "fake-container-123" {
		t.Errorf("expected container ID 'fake-container-123', got '%s'", response.ContainerID)
	}

	if !fakeRuntime.createCalled {
		t.Error("expected Create() to be called")
	}
	if !fakeRuntime.startCalled {
		t.Error("expected Start() to be called")
	}
}

func TestServerRunContainerInvalidJSON(t *testing.T) {
	fakeRuntime := &FakeRuntime{}
	ag := New(fakeRuntime, nil, nil)
	server := NewServer(ag)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/containers",
		bytes.NewBufferString(`{"name":}`),
	)

	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, not %d", rec.Code)
	}
	if fakeRuntime.createCalled {
		t.Error("expected Create() not to be called")
	}
}

func TestServerRunContainerRuntimeError(t *testing.T) {
	fakeRuntime := &FakeRuntime{
		startError: errors.New("start failed"),
	}
	ag := New(fakeRuntime, nil, nil)
	server := NewServer(ag)

	body := RunContainerRequest{
		Name:    "test-container",
		Image:   "alpine",
		Command: []string{"sleep", "60"},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/containers",
		bytes.NewReader(payload),
	)

	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, not %d", rec.Code)
	}

	if !fakeRuntime.createCalled {
		t.Error("expected Create() to be called")
	}
	if !fakeRuntime.startCalled {
		t.Error("expected Start() to be called")
	}
	if !fakeRuntime.removeCalled {
		t.Error("expected Remove() to be called after Start() failed")
	}
}

func TestServerInspectContainer(t *testing.T) {
	fakeRuntime := &FakeRuntime{
		inspectInfo: runtime.ContainerInfo{
			ID:      "fake-container-123",
			Name:    "test-container",
			Image:   "alpine",
			State:   "running",
			Running: true,
		},
	}

	ag := New(fakeRuntime, nil, nil)
	server := NewServer(ag)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/containers/fake-container-123",
		nil,
	)

	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, not %d", rec.Code)
	}

	var response runtime.ContainerInfo

	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.ID != "fake-container-123" {
		t.Errorf(
			"expected container ID 'fake-container-123', got '%s'",
			response.ID,
		)
	}

	if !response.Running {
		t.Error("expected container to be running")
	}

	if !fakeRuntime.inspectCalled {
		t.Error("expected Inspect() to be called")
	}

	if fakeRuntime.inspectID != "fake-container-123" {
		t.Errorf(
			"expected Inspect() ID 'fake-container-123', got '%s'",
			fakeRuntime.inspectID,
		)
	}
}

func TestServerInspectContainerRuntimeError(t *testing.T) {
	fakeRuntime := &FakeRuntime{
		inspectError: errors.New("inspect failed"),
	}

	ag := New(fakeRuntime, nil, nil)
	server := NewServer(ag)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/containers/fake-container-123",
		nil,
	)

	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status 500, not %d",
			rec.Code,
		)
	}

	if !fakeRuntime.inspectCalled {
		t.Error("expected Inspect() to be called")
	}
}
