package store

import (
	"context"
	"database/sql"

	"github.com/kubeli8/k6e/internal/model"
	"github.com/kubeli8/k6e/internal/runtime"
)

func (s *SQLiteStore) CreateAssignment(ctx context.Context, assignment model.Assignment) error {
	query := `
	INSERT INTO assignments
		(id, workload_name, workload_namespace, node_id, status, container_id, template_hash)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	if assignment.ID == "" {
		return ErrMissingID
	}

	_, err := s.db.ExecContext(ctx, query,
		assignment.ID,
		assignment.Workload.Name,
		assignment.Workload.Namespace,
		assignment.NodeID,
		assignment.Status,
		assignment.ContainerID,
		assignment.TemplateHash,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrAlreadyExists
		}
		return err
	}
	return nil
}

func (s *SQLiteStore) GetAssignment(ctx context.Context, id string) (model.Assignment, error) {
	query := `
	SELECT id, workload_name, workload_namespace, node_id, status, container_id, template_hash
	FROM assignments
	WHERE id = ?
	`
	var assignment model.Assignment
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&assignment.ID,
		&assignment.Workload.Name,
		&assignment.Workload.Namespace,
		&assignment.NodeID,
		&assignment.Status,
		&assignment.ContainerID,
		&assignment.TemplateHash,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return assignment, ErrNotFound
		}
		return assignment, err
	}
	return assignment, nil
}

func (s *SQLiteStore) ListAssignments(ctx context.Context, namespace string) ([]model.Assignment, error) {
	query := `
	SELECT id, workload_name, workload_namespace, node_id, status, container_id, template_hash
	FROM assignments
	WHERE workload_namespace = ?
	`
	rows, err := s.db.QueryContext(ctx, query, namespace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assignments []model.Assignment
	for rows.Next() {
		var assignment model.Assignment
		err := rows.Scan(
			&assignment.ID,
			&assignment.Workload.Name,
			&assignment.Workload.Namespace,
			&assignment.NodeID,
			&assignment.Status,
			&assignment.ContainerID,
			&assignment.TemplateHash,
		)
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, assignment)
	}

	return assignments, nil
}

func (s *SQLiteStore) ListAssignmentsForWorkload(ctx context.Context, ref model.WorkloadRef) ([]model.Assignment, error) {
	query := `
	SELECT id, workload_name, workload_namespace, node_id, status, container_id, template_hash
	FROM assignments
	WHERE workload_name = ? AND workload_namespace = ?
	`
	rows, err := s.db.QueryContext(ctx, query, ref.Name, ref.Namespace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assignments []model.Assignment
	for rows.Next() {
		var assignment model.Assignment
		err := rows.Scan(
			&assignment.ID,
			&assignment.Workload.Name,
			&assignment.Workload.Namespace,
			&assignment.NodeID,
			&assignment.Status,
			&assignment.ContainerID,
			&assignment.TemplateHash,
		)
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, assignment)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return assignments, nil
}

func (s *SQLiteStore) UpdateStatusAssignment(ctx context.Context, id string, status model.AssignmentStatus) error {
	query := `
	UPDATE assignments
	SET status = ?
	WHERE id = ?
	`
	result, err := s.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return err
}

func (s *SQLiteStore) UpdateAssignmentExecution(ctx context.Context, id string, containerID runtime.ContainerID, status model.AssignmentStatus) error {
	query := `
	UPDATE assignments
	SET container_id = ?, status = ?
	WHERE id = ?
	`
	result, err := s.db.ExecContext(ctx, query, string(containerID), status, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SQLiteStore) DeleteAssignment(ctx context.Context, id string) error {
	query := `
	DELETE FROM assignments
	WHERE id = ?
	`
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}
