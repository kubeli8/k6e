package agent

import (
	"context"
	"errors"
	"testing"

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

func TestAgentRun(t *testing.T) {
	fake := &FakeRuntime{}

	ag := New(fake)

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

	if !fake.createCalled {
		t.Error("expected Create() to be called")
	}

	if !fake.startCalled {
		t.Error("expected Start() to be called")
	}
}

func TestAgentRunCleansUpOnStartError(t *testing.T) {
	fake := &FakeRuntime{
		startError: errors.New("start failed"),
	}

	ag := New(fake)

	_, err := ag.Run(context.Background(), runtime.ContainerSpec{
		Name:    "test",
		Image:   "alpine",
		Command: []string{"sleep", "60"},
	})

	if err == nil {
		t.Fatal("expected Run() to return an error")
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

func TestAgentContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fake := &FakeRuntime{}
	ag := New(fake)

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

	ag := New(fakeRuntime, fakeRegistrar)

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
