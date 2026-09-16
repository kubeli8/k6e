package api

import (
	"net/http"

	"github.com/pyd-07/k6e/internal/store"
)

type Server struct {
	store store.WorkloadStore
}

func NewServer(workloadStore store.WorkloadStore) *Server {
	return &Server{
		store: workloadStore,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/workloads", s.createWorkload)
	mux.HandleFunc("GET /api/v1/workloads", s.listWorkloads)
	mux.HandleFunc("GET /api/v1/workloads/{namespace}/{name}", s.getWorkload)
	mux.HandleFunc("DELETE /api/v1/workloads/{namespace}/{name}", s.deleteWorkload)

	return mux
}
