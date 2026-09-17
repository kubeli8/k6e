package agent

import (
	"encoding/json"
	"net/http"

	"github.com/pyd-07/k6e/internal/runtime"
)

type Server struct {
	agent *Agent
}

func NewServer(ag *Agent) *Server {
	return &Server{
		agent: ag,
	}
}

type RunContainerRequest struct {
	Name    string   `json:"name"`
	Image   string   `json:"image"`
	Command []string `json:"command,omitempty"`
}
type RunContainerResponse struct {
	ContainerID string `json:"containerId"`
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/containers", s.runContainer)

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
