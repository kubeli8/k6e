package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServerRunContainer(t *testing.T) {
	fake := &FakeRuntime{}
	ag := New(fake)
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

	if !fake.createCalled {
		t.Error("expected Create() to be called")
	}
	if !fake.startCalled {
		t.Error("expected Start() to be called")
	}
}

func TestServerRunContainerInvalidJSON(t *testing.T) {
	fake := &FakeRuntime{}
	ag := New(fake)
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
	if fake.createCalled {
		t.Error("expected Create() not to be called")
	}
}

func TestServerRunContainerRuntimeError(t *testing.T) {
	fake := &FakeRuntime{
		startError: errors.New("start failed"),
	}
	ag := New(fake)
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

	if !fake.createCalled {
		t.Error("expected Create() to be called")
	}
	if !fake.startCalled {
		t.Error("expected Start() to be called")
	}
	if !fake.removeCalled {
		t.Error("expected Remove() to be called after Start() failed")
	}
}
