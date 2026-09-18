package agent

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/pyd-07/k6e/internal/model"
	"github.com/pyd-07/k6e/internal/runtime"
)

type FakeRuntime struct {
	createCalled bool
	startCalled  bool
	stopCalled   bool
	removeCalled bool

	startError error
}

type FakeRegistrar struct {
	registerCalled bool
	registeredNode model.Node
	registerError  error
}

type FakeHeartbeater struct {
	mu sync.Mutex

	heartbeatCalled bool
	heartbeatNodeID string
	heartbeatError  error
}

func (f *FakeRuntime) Create(ctx context.Context, spec runtime.ContainerSpec) (runtime.ContainerID, error) {
	f.createCalled = true
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return "fake-container-123", nil
}

func (f *FakeRuntime) Start(ctx context.Context, id runtime.ContainerID) error {
	f.startCalled = true
	return f.startError
}

func (f *FakeRuntime) Inspect(ctx context.Context, id runtime.ContainerID) (runtime.ContainerInfo, error) {
	return runtime.ContainerInfo{}, nil
}

func (f *FakeRuntime) Stop(ctx context.Context, id runtime.ContainerID) error {
	f.stopCalled = true
	return nil
}

func (f *FakeRuntime) Remove(ctx context.Context, id runtime.ContainerID) error {
	f.removeCalled = true
	return nil
}

func (f *FakeRegistrar) Register(ctx context.Context, node model.Node) error {
	f.registerCalled = true
	f.registeredNode = node
	return f.registerError
}

func (f *FakeHeartbeater) Heartbeat(ctx context.Context, nodeID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.heartbeatCalled = true
	f.heartbeatNodeID = nodeID
	return f.heartbeatError
}

func TestAgentRun(t *testing.T) {
	fakeRuntime := &FakeRuntime{}

	ag := New(fakeRuntime, nil, nil)

	id, err := ag.Run(context.Background(), runtime.ContainerSpec{
		Name:    "test",
		Image:   "alpine",
		Command: []string{"sleep", "60"},
	})

	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}

	if id != "fake-container-123" {
		t.Fatalf("expected container ID fake-container-123, got %s", id)
	}

	if !fakeRuntime.createCalled {
		t.Error("expected Create() to be called")
	}

	if !fakeRuntime.startCalled {
		t.Error("expected Start() to be called")
	}
}

func TestAgentRunCleansUpOnStartError(t *testing.T) {
	fakeRuntime := &FakeRuntime{
		startError: errors.New("start failed"),
	}

	ag := New(fakeRuntime, nil, nil)

	_, err := ag.Run(context.Background(), runtime.ContainerSpec{
		Name:    "test",
		Image:   "alpine",
		Command: []string{"sleep", "60"},
	})

	if err == nil {
		t.Fatal("expected Run() to return an error")
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

func TestAgentContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fakeRuntime := &FakeRuntime{}
	ag := New(fakeRuntime, nil, nil)

	_, err := ag.Run(ctx, runtime.ContainerSpec{
		Name:    "test",
		Image:   "alpine",
		Command: []string{"sleep", "60"},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled error, got %v", err)
	}
}

func TestAgentRegister(t *testing.T) {
	fakeRuntime := &FakeRuntime{}
	fakeRegistrar := &FakeRegistrar{}

	ag := New(fakeRuntime, fakeRegistrar, nil)

	node := testNode("test-node", "127.0.0.1:8080", model.NodeStatusReady)

	err := ag.Register(context.Background(), node)
	if err != nil {
		t.Fatalf("Register() returned unexpected error: %v", err)
	}

	if !fakeRegistrar.registerCalled {
		t.Error("expected Register() to be called on the registrar")
	}

	if fakeRegistrar.registeredNode != node {
		t.Errorf("expected registered node to be %+v, got %+v", node, fakeRegistrar.registeredNode)
	}
}

func TestAgentRegisterWithoutRegistrar(t *testing.T) {
	fakeRuntime := &FakeRuntime{}
	ag := New(fakeRuntime, nil, nil)

	node := testNode("test-node", "127.0.0.1:8080", model.NodeStatusReady)

	err := ag.Register(context.Background(), node)
	if err == nil {
		t.Fatal("expected error when registering without a registrar, got nil")
	}
}

func TestAgentHeartbeat(t *testing.T) {
	fakeRuntime := &FakeRuntime{}
	fakeHeartbeater := &FakeHeartbeater{}

	ag := New(fakeRuntime, nil, fakeHeartbeater)

	err := ag.Heartbeat(context.Background(), "test-node")
	if err != nil {
		t.Fatalf("Heartbeat() returned unexpected error: %v", err)
	}

	fakeHeartbeater.mu.Lock()
	heartbeatCalled := fakeHeartbeater.heartbeatCalled
	heartbeatNodeID := fakeHeartbeater.heartbeatNodeID
	fakeHeartbeater.mu.Unlock()

	if !heartbeatCalled {
		t.Error("expected Heartbeat() to be called on the heartbeater")
	}

	if heartbeatNodeID != "test-node" {
		t.Errorf(
			"expected heartbeat node ID to be 'test-node', got '%s'",
			heartbeatNodeID,
		)
	}
}

func TestAgentHeartbeatWithoutHeartbeater(t *testing.T) {
	fakeRuntime := &FakeRuntime{}
	ag := New(fakeRuntime, nil, nil)

	err := ag.Heartbeat(context.Background(), "test-node")
	if err == nil {
		t.Fatal("expected error when heartbeating without a heartbeater, got nil")
	}
}

func TestAgentStartHeartbeat(t *testing.T) {
	fakeRuntime := &FakeRuntime{}
	fakeHeartbeater := &FakeHeartbeater{}

	ag := New(fakeRuntime, nil, fakeHeartbeater)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)

	go func() {
		done <- ag.StartHeartbeat(ctx, "test-node", 10*time.Millisecond)
	}()

	time.Sleep(50 * time.Millisecond)

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("StartHeartbeat() returned unexpected error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("StartHeartbeat() did not stop after context cancellation")
	}

	fakeHeartbeater.mu.Lock()
	heartbeatCalled := fakeHeartbeater.heartbeatCalled
	heartbeatNodeID := fakeHeartbeater.heartbeatNodeID
	fakeHeartbeater.mu.Unlock()

	if !heartbeatCalled {
		t.Error("expected Heartbeat() to be called on the heartbeater")
	}

	if heartbeatNodeID != "test-node" {
		t.Errorf(
			"expected heartbeat node ID to be 'test-node', got '%s'",
			heartbeatNodeID,
		)
	}
}
