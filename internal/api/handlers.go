package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/kubeli8/k6e/internal/model"
	"github.com/kubeli8/k6e/internal/scheduler"
	"github.com/kubeli8/k6e/internal/store"
)

func (s *Server) createWorkload(w http.ResponseWriter, r *http.Request) {
	var workload model.Workload

	if err := json.NewDecoder(r.Body).Decode(&workload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if err := validateWorkload(workload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.workloadStore.Create(r.Context(), workload); err != nil {
		switch {
		case errors.Is(err, store.ErrAlreadyExists):
			writeError(w, http.StatusConflict, "Workload already exists")
		default:
			writeError(w, http.StatusInternalServerError, "Failed to create workload")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(workload)
}

func (s *Server) listWorkloads(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	namespace := queryParams.Get("namespace")
	if strings.TrimSpace(namespace) == "" {
		writeError(w, http.StatusBadRequest, "Namespace query parameter is required")
		return
	}

	workloads, err := s.workloadStore.List(r.Context(), namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list workloads")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(workloads)
}

func (s *Server) getWorkload(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	name := r.PathValue("name")

	if strings.TrimSpace(namespace) == "" || strings.TrimSpace(name) == "" {
		writeError(w, http.StatusBadRequest, "Namespace and name are required")
		return
	}

	workload, err := s.workloadStore.Get(r.Context(), model.WorkloadRef{Namespace: namespace, Name: name})
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			writeError(w, http.StatusNotFound, "Workload not found")
		default:
			writeError(w, http.StatusInternalServerError, "Failed to retrieve workload")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(workload)
}

func (s *Server) updateWorkload(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	name := r.PathValue("name")

	if strings.TrimSpace(namespace) == "" || strings.TrimSpace(name) == "" {
		writeError(w, http.StatusBadRequest, "Namespace and name are required")
		return
	}

	var workload model.Workload
	if err := json.NewDecoder(r.Body).Decode(&workload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if workload.Metadata.Namespace != namespace || workload.Metadata.Name != name {
		writeError(w, http.StatusBadRequest, "Namespace and name in the payload must match the URL")
		return
	}

	if err := validateWorkload(workload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.workloadStore.Update(r.Context(), model.WorkloadRef{Namespace: namespace, Name: name}, workload); err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			writeError(w, http.StatusNotFound, "Workload not found")
		default:
			writeError(w, http.StatusInternalServerError, "Failed to update workload")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(workload)
}

func (s *Server) deleteWorkload(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	name := r.PathValue("name")

	if strings.TrimSpace(namespace) == "" || strings.TrimSpace(name) == "" {
		writeError(w, http.StatusBadRequest, "Namespace and name are required")
		return
	}

	if err := s.workloadStore.Delete(r.Context(), model.WorkloadRef{Namespace: namespace, Name: name}); err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			writeError(w, http.StatusNotFound, "Workload not found")
		default:
			writeError(w, http.StatusInternalServerError, "Failed to delete workload")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) registerNode(w http.ResponseWriter, r *http.Request) {
	var node model.Node

	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}
	if err := validateNode(node); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.nodeStore.RegisterNode(r.Context(), node); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to register node")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(node)
}

func (s *Server) getNode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if strings.TrimSpace(id) == "" {
		writeError(w, http.StatusBadRequest, "Node ID is required")
		return
	}

	node, err := s.nodeStore.GetNode(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			writeError(w, http.StatusNotFound, "Node not found")
		default:
			writeError(w, http.StatusInternalServerError, "Failed to retrieve node")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(node)
}

func (s *Server) listNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := s.nodeStore.ListNodes(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list nodes")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(nodes)
}

func (s *Server) removeNode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if strings.TrimSpace(id) == "" {
		writeError(w, http.StatusBadRequest, "Node ID is required")
		return
	}

	if err := s.nodeStore.RemoveNode(r.Context(), id); err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			writeError(w, http.StatusNotFound, "Node not found")
		default:
			writeError(w, http.StatusInternalServerError, "Failed to delete node")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) updateNodeHeartbeat(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if strings.TrimSpace(id) == "" {
		writeError(w, http.StatusBadRequest, "Node ID is required")
		return
	}

	if err := s.nodeStore.UpdateHeartbeat(r.Context(), id, time.Now()); err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			writeError(w, http.StatusNotFound, "Node not found")
		default:
			writeError(w, http.StatusInternalServerError, "Failed to update node heartbeat")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) scheduleWorkload(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	name := r.PathValue("name")

	if strings.TrimSpace(namespace) == "" || strings.TrimSpace(name) == "" {
		writeError(w, http.StatusBadRequest, "Namespace and name are required")
		return
	}

	ref := model.WorkloadRef{Namespace: namespace, Name: name}
	assignment, err := s.scheduler.ScheduleWorkload(r.Context(), ref)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			writeError(w, http.StatusNotFound, "Workload not found")
		case errors.Is(err, scheduler.ErrNoReadyNodes):
			writeError(w, http.StatusConflict, "No available nodes to schedule the workload")
		default:
			writeError(w, http.StatusInternalServerError, "Failed to schedule workload")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(assignment)
}

func (s *Server) executeAssignment(w http.ResponseWriter, r *http.Request) {
	assignmentID := r.PathValue("id")
	if strings.TrimSpace(assignmentID) == "" {
		writeError(w, http.StatusBadRequest, "Assignment ID is required")
		return
	}

	assignment, err := s.executor.ExecuteAssignment(r.Context(), assignmentID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			writeError(w, http.StatusNotFound, "Assignment not found")
		default:
			writeError(w, http.StatusInternalServerError, "Failed to execute assignment")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(assignment)
}
