package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/rodrigocavalhero/nanojira/internal/domain"
)

type StepBackRepo struct {
	db *DB
}

func NewStepBackRepo(db *DB) *StepBackRepo {
	return &StepBackRepo{db: db}
}

func (r *StepBackRepo) Create(ctx context.Context, req *domain.StepBackRequest) error {
	const query = `
		INSERT INTO stepback_requests (id, task_id, requested_by_id, from_status, to_status, reason, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.db.pool.Exec(ctx, query,
		req.ID, req.TaskID, req.RequestedByID, req.FromStatus, req.ToStatus,
		req.Reason, req.Status, req.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create stepback request: %w", err)
	}
	return nil
}

func (r *StepBackRepo) GetByID(ctx context.Context, id string) (*domain.StepBackRequest, error) {
	const query = `
		SELECT id, task_id, requested_by_id, from_status, to_status, reason, status, reviewed_by_id, reviewed_at, created_at
		FROM stepback_requests
		WHERE id = $1`

	req, err := scanStepBack(r.db.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get stepback by id: %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get stepback by id: %w", err)
	}
	return req, nil
}

func (r *StepBackRepo) GetPendingByTaskID(ctx context.Context, taskID string) (*domain.StepBackRequest, error) {
	const query = `
		SELECT id, task_id, requested_by_id, from_status, to_status, reason, status, reviewed_by_id, reviewed_at, created_at
		FROM stepback_requests
		WHERE task_id = $1 AND status = 'pending'
		ORDER BY created_at DESC
		LIMIT 1`

	req, err := scanStepBack(r.db.pool.QueryRow(ctx, query, taskID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get pending stepback: %w", err)
	}
	return req, nil
}

func (r *StepBackRepo) Update(ctx context.Context, req *domain.StepBackRequest) error {
	const query = `
		UPDATE stepback_requests
		SET status = $2, reviewed_by_id = $3, reviewed_at = $4
		WHERE id = $1`

	tag, err := r.db.pool.Exec(ctx, query, req.ID, req.Status, req.ReviewedByID, req.ReviewedAt)
	if err != nil {
		return fmt.Errorf("update stepback request: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update stepback request: %w", domain.ErrNotFound)
	}
	return nil
}

func scanStepBack(row scannable) (*domain.StepBackRequest, error) {
	var req domain.StepBackRequest
	err := row.Scan(
		&req.ID, &req.TaskID, &req.RequestedByID, &req.FromStatus, &req.ToStatus,
		&req.Reason, &req.Status, &req.ReviewedByID, &req.ReviewedAt, &req.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &req, nil
}
