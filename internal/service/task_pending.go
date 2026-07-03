package service

import (
	"context"

	"github.com/rodrigocavalhero/nanojira/internal/domain"
)

func (s *Service) enrichTaskWithPending(ctx context.Context, task *domain.Task) (*domain.Task, error) {
	pending, err := s.stepback.GetPendingByTaskID(ctx, task.ID)
	if err != nil {
		return nil, err
	}

	result := *task
	result.PendingStatusChange = domain.PendingStatusChangeFromRequest(pending)
	return &result, nil
}

func (s *Service) enrichTasksWithPending(ctx context.Context, tasks []domain.Task) ([]domain.Task, error) {
	enriched := make([]domain.Task, len(tasks))
	for i := range tasks {
		task, err := s.enrichTaskWithPending(ctx, &tasks[i])
		if err != nil {
			return nil, err
		}
		enriched[i] = *task
	}
	return enriched, nil
}
