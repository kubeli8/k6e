package api

import (
	"net/http"

	"github.com/pyd-07/k6e/internal/store"
)

type Server struct {
	workloadStore store.WorkloadStore
	nodeStore     store.NodeStore
}

func NewServer(workloadStore store.WorkloadStore, nodeStore store.NodeStore) *Server {
	return &Server{
		workloadStore: workloadStore,
		nodeStore:     nodeStore,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Workload endpoints
	mux.HandleFunc("POST /api/v1/workloads", s.createWorkload)
	mux.HandleFunc("GET /api/v1/workloads", s.listWorkloads)
	mux.HandleFunc("GET /api/v1/workloads/{namespace}/{name}", s.getWorkload)
	mux.HandleFunc("DELETE /api/v1/workloads/{namespace}/{name}", s.deleteWorkload)

	// Node endpoints
	mux.HandleFunc("POST /api/v1/nodes", s.registerNode)
	mux.HandleFunc("GET /api/v1/nodes", s.listNodes)
	mux.HandleFunc("GET /api/v1/nodes/{id}", s.getNode)
	mux.HandleFunc("DELETE /api/v1/nodes/{id}", s.removeNode)

	return mux
}
