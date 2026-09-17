package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/pyd-07/k6e/internal/model"
	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite database: %w", err)
	}

	store := &SQLiteStore{db: db}
	if err := store.initializeSchema(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize schema: %w", err)
	}
	return store, nil
}

func (s *SQLiteStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *SQLiteStore) Create(ctx context.Context, workload model.Workload) error {
	specJSON, err := json.Marshal(workload.Spec)
	if err != nil {
		return fmt.Errorf("marshal workload spec: %w", err)
	}

	query := `
	INSERT INTO workloads (namespace, name, api_version, kind, spec_json)
	VALUES (?, ?, ?, ?, ?);
	`

	_, err = s.db.ExecContext(ctx, query,
		workload.Metadata.Namespace,
		workload.Metadata.Name,
		workload.APIVersion,
		workload.Kind,
		string(specJSON),
	)

	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrAlreadyExists
		}
		return fmt.Errorf("insert workload: %w", err)
	}

	return nil
}

func (s *SQLiteStore) Get(ctx context.Context, namespace, name string) (model.Workload, error) {
	query := `
		SELECT namespace, name, api_version, kind, spec_json
		FROM workloads
		WHERE namespace = ? AND name = ?
	`
	var (
		workload model.Workload
		specJSON string
	)
	err := s.db.QueryRowContext(ctx, query, namespace, name).Scan(
		&workload.Metadata.Namespace,
		&workload.Metadata.Name,
		&workload.APIVersion,
		&workload.Kind,
		&specJSON,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Workload{}, ErrNotFound
		}
		return model.Workload{}, fmt.Errorf("query workload: %w", err)
	}

	if err := json.Unmarshal([]byte(specJSON), &workload.Spec); err != nil {
		return model.Workload{}, fmt.Errorf("unmarshal workload spec: %w", err)
	}

	return workload, nil
}

func (s *SQLiteStore) List(ctx context.Context, namespace string) ([]model.Workload, error) {
	query := `
		SELECT namespace, name, api_version, kind, spec_json
		FROM workloads
		WHERE namespace = ?
	`
	rows, err := s.db.QueryContext(ctx, query, namespace)
	if err != nil {
		return nil, fmt.Errorf("query workloads: %w", err)
	}
	defer rows.Close()

	var workloads []model.Workload
	for rows.Next() {
		var workload model.Workload
		var specJSON string
		if err := rows.Scan(
			&workload.Metadata.Namespace,
			&workload.Metadata.Name,
			&workload.APIVersion,
			&workload.Kind,
			&specJSON,
		); err != nil {
			return nil, fmt.Errorf("scan workload: %w", err)
		}

		if err := json.Unmarshal([]byte(specJSON), &workload.Spec); err != nil {
			return nil, fmt.Errorf("unmarshal workload spec: %w", err)
		}

		workloads = append(workloads, workload)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workloads: %w", err)
	}

	return workloads, nil
}

func (s *SQLiteStore) Delete(ctx context.Context, namespace, name string) error {
	query := `
	DELETE FROM workloads
	WHERE namespace = ? AND name = ?
	`
	result, err := s.db.ExecContext(ctx, query, namespace, name)
	if err != nil {
		return fmt.Errorf("delete workload: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *SQLiteStore) initializeSchema() error {
	createWorkloadsTable := `
	CREATE TABLE IF NOT EXISTS workloads (
		namespace TEXT NOT NULL,
		name TEXT NOT NULL,
		api_version TEXT NOT NULL,
		kind TEXT NOT NULL,
		spec_json TEXT NOT NULL,
		PRIMARY KEY (namespace, name)
	);
	`
	createNodesTable := `
	CREATE TABLE IF NOT EXISTS nodes(
		id TEXT PRIMARY KEY,
		address TEXT NOT NULL,
		status TEXT NOT NULL
	);
	`

	_, err := s.db.Exec(createWorkloadsTable)
	if err != nil {
		return fmt.Errorf("create workloads table: %w", err)
	}
	_, err = s.db.Exec(createNodesTable)
	if err != nil {
		return fmt.Errorf("create nodes table: %w", err)
	}
	return nil
}

func isUniqueConstraintError(err error) bool {
	return strings.Contains(
		strings.ToLower(err.Error()),
		"unique constraint",
	)
}
