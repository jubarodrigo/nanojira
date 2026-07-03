package service

import (
	"context"
	"errors"

	"github.com/rodrigocavalhero/nanojira/internal/domain"
)

type ListTasksInput struct {
	Status *domain.TaskStatus
	Limit  int
	Offset int
}

type ListTasksResult struct {
	Tasks  []domain.Task `json:"tasks"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}

func (s *Service) ListTasks(ctx context.Context, actorID string, input ListTasksInput) (*ListTasksResult, error) {
	actor, err := s.users.GetByID(ctx, actorID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NotFound("user")
		}
		return nil, err
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := input.Offset
	if offset < 0 {
		offset = 0
	}

	filter := domain.TaskFilter{
		Status: input.Status,
		Limit:  limit,
		Offset: offset,
	}

	if actor.IsWorker() {
		filter.AssigneeID = &actor.ID
	}

	total, err := s.tasks.Count(ctx, filter)
	if err != nil {
		return nil, err
	}

	tasks, err := s.tasks.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	enriched, err := s.enrichTasksWithPending(ctx, tasks)
	if err != nil {
		return nil, err
	}

	return &ListTasksResult{
		Tasks:  enriched,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *Service) GetTask(ctx context.Context, actorID, taskID string) (*domain.Task, error) {
	actor, err := s.users.GetByID(ctx, actorID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NotFound("user")
		}
		return nil, err
	}

	task, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NotFound("task")
		}
		return nil, err
	}

	if actor.IsWorker() {
		if task.AssigneeID == nil || *task.AssigneeID != actor.ID {
			return nil, domain.Forbidden("workers can only view their assigned tasks")
		}
	}

	return s.enrichTaskWithPending(ctx, task)
}

func (s *Service) GetAssignmentNotifications(ctx context.Context, actorID, taskID string) ([]domain.AssignmentNotification, error) {
	if _, err := s.GetTask(ctx, actorID, taskID); err != nil {
		return nil, err
	}
	return s.tasks.ListAssignmentNotifications(ctx, taskID)
}
