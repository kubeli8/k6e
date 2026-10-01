// Package api implements the control plane's HTTP API for workloads, nodes,
// scheduling, and assignment execution. Handlers map store sentinel errors
// (ErrNotFound, ErrAlreadyExists) and scheduler.ErrNoReadyNodes onto HTTP
// status codes.
package api

import (
	"net/http"

	"github.com/pyd-07/k6e/internal/executor"
	"github.com/pyd-07/k6e/internal/scheduler"
	"github.com/pyd-07/k6e/internal/store"
)

// Server hosts the control plane's REST endpoints, backed by the stores and
// delegating scheduling and execution to their services.
type Server struct {
	workloadStore   store.WorkloadStore
	nodeStore       store.NodeStore
	assignmentStore store.AssignmentStore
	scheduler       *scheduler.Service
	executor        *executor.Service
}

// NewServer constructs the control-plane API server over the given stores
// and services.
func NewServer(
	workloadStore store.WorkloadStore,
	nodeStore store.NodeStore,
	assignmentStore store.AssignmentStore,
	scheduler *scheduler.Service,
	executor *executor.Service,
) *Server {
	return &Server{
		workloadStore:   workloadStore,
		nodeStore:       nodeStore,
		assignmentStore: assignmentStore,
		scheduler:       scheduler,
		executor:        executor,
	}
}

// Handler returns the control plane's HTTP API: CRUD for Workloads and
// Nodes, node heartbeat handling, workload scheduling, and assignment
// execution.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Workload endpoints
	mux.HandleFunc("POST /api/v1/workloads", s.createWorkload)
	mux.HandleFunc("GET /api/v1/workloads", s.listWorkloads)
	mux.HandleFunc("GET /api/v1/workloads/{namespace}/{name}", s.getWorkload)
	mux.HandleFunc("PUT /api/v1/workloads/{namespace}/{name}", s.updateWorkload)
	mux.HandleFunc("DELETE /api/v1/workloads/{namespace}/{name}", s.deleteWorkload)

	// Node endpoints
	mux.HandleFunc("POST /api/v1/nodes", s.registerNode)
	mux.HandleFunc("GET /api/v1/nodes", s.listNodes)
	mux.HandleFunc("GET /api/v1/nodes/{id}", s.getNode)
	mux.HandleFunc("DELETE /api/v1/nodes/{id}", s.removeNode)
	mux.HandleFunc("POST /api/v1/nodes/{id}/heartbeat", s.updateNodeHeartbeat)

	// Scheduler endpoints
	mux.HandleFunc("POST /api/v1/workloads/{namespace}/{name}/scheduler", s.scheduleWorkload)

	// Executor endpoints
	mux.HandleFunc("POST /api/v1/assignments/{id}/execute", s.executeAssignment)

	return mux
}
