package store

import (
	"context"
	"sync"

	"github.com/pyd-07/k6e/internal/model"
)

type MemoryAssignmentStore struct {
	mu          sync.RWMutex
	assignments map[string]model.Assignment
}

func NewMemoryAssignmentStore() *MemoryAssignmentStore {
	return &MemoryAssignmentStore{
		assignments: make(map[string]model.Assignment),
	}
}

func (s *MemoryAssignmentStore) CreateAssignment(ctx context.Context, assignment model.Assignment) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if assignment.ID == "" {
		return ErrMissingID
	}
	if _, exists := s.assignments[assignment.ID]; exists {
		return ErrAlreadyExists
	}

	s.assignments[assignment.ID] = assignment
	return nil
}

func (s *MemoryAssignmentStore) GetAssignment(ctx context.Context, id string) (model.Assignment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	assignment, exists := s.assignments[id]
	if !exists {
		return model.Assignment{}, ErrNotFound
	}
	return assignment, nil
}

func (s *MemoryAssignmentStore) ListAssignments(ctx context.Context, namespace string) ([]model.Assignment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	assignments := make([]model.Assignment, 0)
	for _, assignment := range s.assignments {
		if assignment.Workload.Namespace == namespace {
			assignments = append(assignments, assignment)
		}
	}
	return assignments, nil
}

func (s *MemoryAssignmentStore) UpdateStatusAssignment(ctx context.Context, id string, status model.AssignmentStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	assignment, exists := s.assignments[id]
	if !exists {
		return ErrNotFound
	}
	assignment.Status = status
	s.assignments[id] = assignment
	return nil
}

func (s *MemoryAssignmentStore) DeleteAssignment(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.assignments[id]
	if !exists {
		return ErrNotFound
	}
	delete(s.assignments, id)
	return nil
}
