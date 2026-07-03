package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/rodrigocavalhero/nanojira/internal/domain"
)

type TaskRepo struct {
	db *DB
}

func NewTaskRepo(db *DB) *TaskRepo {
	return &TaskRepo{db: db}
}

func (r *TaskRepo) Create(ctx context.Context, task *domain.Task) error {
	const query = `
		INSERT INTO tasks (id, title, description, status, reporter_id, assignee_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.db.pool.Exec(ctx, query,
		task.ID, task.Title, task.Description, task.Status,
		task.ReporterID, task.AssigneeID, task.CreatedAt, task.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	return nil
}

func (r *TaskRepo) GetByID(ctx context.Context, id string) (*domain.Task, error) {
	const query = `
		SELECT id, title, description, status, reporter_id, assignee_id, created_at, updated_at
		FROM tasks
		WHERE id = $1`

	task, err := scanTask(r.db.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get task by id: %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("get task by id: %w", err)
	}
	return task, nil
}

func (r *TaskRepo) List(ctx context.Context, filter domain.TaskFilter) ([]domain.Task, error) {
	query, args := buildTaskListQuery(filter, false)

	rows, err := r.db.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []domain.Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, *task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}
	return tasks, nil
}

func (r *TaskRepo) Count(ctx context.Context, filter domain.TaskFilter) (int, error) {
	query, args := buildTaskListQuery(filter, true)

	var count int
	if err := r.db.pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count tasks: %w", err)
	}
	return count, nil
}

func (r *TaskRepo) Update(ctx context.Context, task *domain.Task) error {
	const query = `
		UPDATE tasks
		SET title = $2, description = $3, status = $4, assignee_id = $5, updated_at = $6
		WHERE id = $1`

	tag, err := r.db.pool.Exec(ctx, query,
		task.ID, task.Title, task.Description, task.Status,
		task.AssigneeID, task.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update task: %w", domain.ErrNotFound)
	}
	return nil
}

func (r *TaskRepo) CreateAssignmentNotification(ctx context.Context, n *domain.AssignmentNotification) error {
	const query = `
		INSERT INTO assignment_notifications (id, task_id, assignee_id, email, sent_at)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := r.db.pool.Exec(ctx, query, n.ID, n.TaskID, n.AssigneeID, n.Email, n.SentAt)
	if err != nil {
		return fmt.Errorf("create assignment notification: %w", err)
	}
	return nil
}

func (r *TaskRepo) ListAssignmentNotifications(ctx context.Context, taskID string) ([]domain.AssignmentNotification, error) {
	const query = `
		SELECT id, task_id, assignee_id, email, sent_at
		FROM assignment_notifications
		WHERE task_id = $1
		ORDER BY sent_at DESC`

	rows, err := r.db.pool.Query(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("list assignment notifications: %w", err)
	}
	defer rows.Close()

	var notifications []domain.AssignmentNotification
	for rows.Next() {
		var n domain.AssignmentNotification
		if err := rows.Scan(&n.ID, &n.TaskID, &n.AssigneeID, &n.Email, &n.SentAt); err != nil {
			return nil, fmt.Errorf("scan assignment notification: %w", err)
		}
		notifications = append(notifications, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate assignment notifications: %w", err)
	}
	return notifications, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanTask(row scannable) (*domain.Task, error) {
	var task domain.Task
	err := row.Scan(
		&task.ID, &task.Title, &task.Description, &task.Status,
		&task.ReporterID, &task.AssigneeID, &task.CreatedAt, &task.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func buildTaskListQuery(filter domain.TaskFilter, countOnly bool) (string, []any) {
	args := make([]any, 0, 4)
	where := "WHERE 1=1"
	argPos := 1

	if filter.AssigneeID != nil {
		where += fmt.Sprintf(" AND assignee_id = $%d", argPos)
		args = append(args, *filter.AssigneeID)
		argPos++
	}
	if filter.Status != nil {
		where += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, *filter.Status)
		argPos++
	}

	if countOnly {
		return "SELECT COUNT(*) FROM tasks " + where, args
	}

	query := fmt.Sprintf(`
		SELECT id, title, description, status, reporter_id, assignee_id, created_at, updated_at
		FROM tasks %s
		ORDER BY created_at DESC`, where)

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, filter.Limit)
		argPos++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, filter.Offset)
	}

	return query, args
}
