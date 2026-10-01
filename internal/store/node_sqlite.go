package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/pyd-07/k6e/internal/model"
)

// RegisterNode upserts the node record. If a node with the same ID already
// exists, its address, status, and last_heartbeat are overwritten. If
// LastHeartbeat is zero, it is set to the current time before persisting.
func (s *SQLiteStore) RegisterNode(ctx context.Context, node model.Node) error {
	if node.LastHeartbeat.IsZero() {
		node.LastHeartbeat = time.Now()
	}
	query := `
	INSERT INTO nodes (id, address, status, last_heartbeat)
	VALUES (?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		address=excluded.address,
		status=excluded.status,
		last_heartbeat=excluded.last_heartbeat;
	`
	_, err := s.db.ExecContext(ctx, query,
		node.ID,
		node.Address,
		node.Status,
		node.LastHeartbeat,
	)
	if err != nil {
		return fmt.Errorf("register node: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetNode(ctx context.Context, id string) (model.Node, error) {
	query := `
	SELECT id, address, status, last_heartbeat
	FROM nodes
	WHERE id = ?
	`
	var node model.Node
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&node.ID,
		&node.Address,
		&node.Status,
		&node.LastHeartbeat,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Node{}, ErrNotFound
		}
		return model.Node{}, fmt.Errorf("query node: %w", err)
	}
	return node, nil
}

func (s *SQLiteStore) ListNodes(ctx context.Context) ([]model.Node, error) {
	query := `
	SELECT id, address, status, last_heartbeat
	FROM nodes
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query nodes: %w", err)
	}
	defer rows.Close()

	var nodes []model.Node
	for rows.Next() {
		var node model.Node
		if err := rows.Scan(
			&node.ID,
			&node.Address,
			&node.Status,
			&node.LastHeartbeat,
		); err != nil {
			return nil, fmt.Errorf("scan node: %w", err)
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (s *SQLiteStore) RemoveNode(ctx context.Context, id string) error {
	query := `
	DELETE FROM nodes
	WHERE id = ?
	`
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("remove node: %w", err)
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

// UpdateHeartbeat records timestamp as the node's last heartbeat and
// unconditionally sets the node's status to NodeStatusReady. Returns
// ErrNotFound if no node with the given id exists.
func (s *SQLiteStore) UpdateHeartbeat(ctx context.Context, id string, timestamp time.Time) error {
	query := `
	UPDATE nodes
	SET last_heartbeat = ?,
		status = ?
	WHERE id = ?
	`
	result, err := s.db.ExecContext(
		ctx,
		query,
		timestamp,
		model.NodeStatusReady,
		id,
	)
	if err != nil {
		return fmt.Errorf("update heartbeat: %w", err)
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

func (s *SQLiteStore) UpdateNodeStatus(ctx context.Context, id string, status model.NodeStatus) error {
	query := `
	UPDATE nodes
	SET status = ?
	WHERE id = ?
	`
	result, err := s.db.ExecContext(
		ctx,
		query,
		status,
		id,
	)
	if err != nil {
		return fmt.Errorf("update node status: %w", err)
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
