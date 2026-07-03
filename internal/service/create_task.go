package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rodrigocavalhero/nanojira/internal/domain"
	"go.uber.org/zap"
)

type CreateTaskInput struct {
	Title       string
	Description string
	AssigneeID  *string
}

func (s *Service) CreateTask(ctx context.Context, actorID string, input CreateTaskInput) (*domain.Task, error) {
	actor, err := s.users.GetByID(ctx, actorID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NotFound("user")
		}
		return nil, err
	}
	if !actor.IsManager() {
		return nil, domain.Forbidden("only managers can create tasks")
	}

	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, domain.InvalidInput("title is required")
	}

	if input.AssigneeID != nil {
		assignee, err := s.users.GetByID(ctx, *input.AssigneeID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, domain.NotFound("assignee")
			}
			return nil, err
		}
		if !assignee.IsWorker() {
			return nil, domain.InvalidInput("assignee must be a worker")
		}
	}

	now := time.Now().UTC()
	task := &domain.Task{
		ID:          uuid.New().String(),
		Title:       title,
		Description: strings.TrimSpace(input.Description),
		Status:      domain.StatusTodo,
		ReporterID:  actor.ID,
		AssigneeID:  input.AssigneeID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.tasks.Create(ctx, task); err != nil {
		return nil, err
	}

	if task.AssigneeID != nil {
		if err := s.sendAssignmentEmail(ctx, task, actor); err != nil {
			return nil, err
		}
	}

	s.log.Info("task created", zap.String("task_id", task.ID), zap.String("reporter_id", actor.ID))
	return task, nil
}
