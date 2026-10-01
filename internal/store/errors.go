// Package store defines the persistence interfaces and their implementations
// for the three core k6e resource types: Workload, Node, and Assignment.
//
// Two backend implementations are provided:
//   - In-memory stores (MemoryStore, MemoryNodeStore, MemoryAssignmentStore)
//     suitable for tests and non-persistent deployments.
//   - SQLiteStore, which persists all resources to a local SQLite database
//     using WAL mode for concurrent read performance.
//
// All store operations accept a context and propagate cancellation.
// Callers should check errors against the sentinel values ErrNotFound,
// ErrAlreadyExists, and ErrMissingID using errors.Is.
package store

import "errors"

var (
	// ErrNotFound is returned when a requested resource does not exist in the store.
	ErrNotFound = errors.New("workload not found")
	// ErrAlreadyExists is returned when attempting to create a resource whose
	// key is already present in the store.
	ErrAlreadyExists = errors.New("workload already exists")
	// ErrMissingID is returned when an assignment is created without an ID.
	ErrMissingID = errors.New("missing resource ID")
)
