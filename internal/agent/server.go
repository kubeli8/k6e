package agent

import (
	"encoding/json"
	"net/http"

	"github.com/kubeli8/k6e/internal/runtime"
)

// Server exposes the agent's container operations over HTTP. It is the
// endpoint the control plane's agent client calls to run, inspect, and
// delete containers on this node.
type Server struct {
	agent *Agent
}

// NewServer constructs a Server that delegates all container operations to
// the given agent.
func NewServer(ag *Agent) *Server {
	return &Server{
		agent: ag,
	}
}

type RunContainerRequest struct {
	Name    string   `json:"name"`
	Image   string   `json:"image"`
	Command []string `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
}
type RunContainerResponse struct {
	ContainerID string `json:"containerId"`
}

// Handler returns the agent's HTTP API:
//
//	POST   /api/v1/containers        run a container
//	GET    /api/v1/containers/{id}   inspect a container
//	DELETE /api/v1/containers/{id}   stop and remove a container
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/containers", s.runContainer)
	mux.HandleFunc("GET /api/v1/containers/{id}", s.inspectContainer)
	mux.HandleFunc("DELETE /api/v1/containers/{id}", s.deleteContainer)

	return mux
}

func (s *Server) runContainer(w http.ResponseWriter, r *http.Request) {
	var req RunContainerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	spec := runtime.ContainerSpec{
		Name:    req.Name,
		Image:   req.Image,
		Command: req.Command,
		Args:    req.Args,
	}

	id, err := s.agent.Run(r.Context(), spec)
	if err != nil {
		http.Error(w, "Failed to run container: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	_ = json.NewEncoder(w).Encode(RunContainerResponse{
		ContainerID: string(id),
	})
}

// deleteContainer stops and removes the container in one request, matching
// the executor's delete semantics.
func (s *Server) deleteContainer(w http.ResponseWriter, r *http.Request) {
	id := runtime.ContainerID(r.PathValue("id"))

	err := s.agent.Stop(r.Context(), id)
	if err != nil {
		http.Error(w, "Failed to stop container: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = s.agent.Remove(r.Context(), id)
	if err != nil {
		http.Error(w, "Failed to remove container: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) inspectContainer(w http.ResponseWriter, r *http.Request) {
	id := runtime.ContainerID(r.PathValue("id"))

	info, err := s.agent.Inspect(r.Context(), id)
	if err != nil {
		http.Error(w, "Failed to inspect container: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(info)
}
