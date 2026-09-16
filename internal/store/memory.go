package store

import (
	"context"
	"sync"

	"github.com/pyd-07/k6e/internal/model"
)

type MemoryStore struct {
	mu        sync.RWMutex
	workloads map[string]model.Workload
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		workloads: make(map[string]model.Workload),
	}
}

func (s *MemoryStore) Create(ctx context.Context, workload model.Workload) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	key := workloadKey(workload.Metadata.Namespace, workload.Metadata.Name)
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.workloads[key]; exists {
		return ErrAlreadyExists
	}

	s.workloads[key] = workload
	return nil
}

func (s *MemoryStore) Get(ctx context.Context, namespace, name string) (model.Workload, error) {
	if err := ctx.Err(); err != nil {
		return model.Workload{}, err
	}

	key := workloadKey(namespace, name)
	s.mu.RLock()
	defer s.mu.RUnlock()

	workload, exists := s.workloads[key]
	if !exists {
		return model.Workload{}, ErrNotFound
	}

	return workload, nil
}

func (s *MemoryStore) List(ctx context.Context, namespace string) ([]model.Workload, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	workloads := make([]model.Workload, 0)

	for _, workload := range s.workloads {
		if workload.Metadata.Namespace == namespace {
			workloads = append(workloads, workload)
		}
	}

	return workloads, nil
}

func (s *MemoryStore) Delete(ctx context.Context, namespace, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	key := workloadKey(namespace, name)
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.workloads[key]; exists {
		delete(s.workloads, key)
		return nil
	}

	return ErrNotFound
}

func workloadKey(namespace, name string) string {
	return namespace + "/" + name
}
