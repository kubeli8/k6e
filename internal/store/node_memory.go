package store

import (
	"context"
	"sync"

	"github.com/pyd-07/k6e/internal/model"
)

type MemoryNodeStore struct {
	mu    sync.RWMutex
	nodes map[string]model.Node
}

func NewMemoryNodeStore() *MemoryNodeStore {
	return &MemoryNodeStore{
		nodes: make(map[string]model.Node),
	}
}

func (s *MemoryNodeStore) RegisterNode(ctx context.Context, node model.Node) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nodes[node.ID] = node
	return nil
}

func (s *MemoryNodeStore) GetNode(ctx context.Context, id string) (model.Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	node, exists := s.nodes[id]
	if !exists {
		return model.Node{}, ErrNotFound
	}
	return node, nil
}

func (s *MemoryNodeStore) ListNodes(ctx context.Context) ([]model.Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nodes := make([]model.Node, 0, len(s.nodes))
	for _, node := range s.nodes {
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (s *MemoryNodeStore) RemoveNode(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.nodes[id]; !exists {
		return ErrNotFound
	}
	delete(s.nodes, id)
	return nil
}
