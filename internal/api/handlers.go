package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/pyd-07/k6e/internal/model"
	"github.com/pyd-07/k6e/internal/store"
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

	if err := s.store.Create(r.Context(), workload); err != nil {
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

	workloads, err := s.store.List(r.Context(), namespace)
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

	workload, err := s.store.Get(r.Context(), namespace, name)
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

func (s *Server) deleteWorkload(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	name := r.PathValue("name")

	if strings.TrimSpace(namespace) == "" || strings.TrimSpace(name) == "" {
		writeError(w, http.StatusBadRequest, "Namespace and name are required")
		return
	}

	if err := s.store.Delete(r.Context(), namespace, name); err != nil {
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
