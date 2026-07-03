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

type UpdateStatusInput struct {
	Status              domain.TaskStatus
	Reason              string
	ApproveStatusChange *bool
}

func (s *Service) UpdateTaskStatus(ctx context.Context, actorID, taskID string, input UpdateStatusInput) (*domain.Task, error) {
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

	if actor.IsManager() {
		return s.managerUpdateStatus(ctx, actor, task, input)
	}

	return s.workerUpdateStatus(ctx, actor, task, input)
}

func (s *Service) managerUpdateStatus(ctx context.Context, actor *domain.User, task *domain.Task, input UpdateStatusInput) (*domain.Task, error) {
	if input.ApproveStatusChange == nil {
		return nil, domain.Forbidden("managers can only approve or reject pending status changes")
	}

	pending, err := s.stepback.GetPendingByTaskID(ctx, task.ID)
	if err != nil {
		return nil, err
	}
	if pending == nil {
		return nil, domain.NewAppError(
			"NO_PENDING_STATUS_CHANGE",
			"task has no pending status change to review",
			domain.ErrInvalidStepBack,
		)
	}

	now := time.Now().UTC()
	reviewerID := actor.ID
	pending.ReviewedByID = &reviewerID
	pending.ReviewedAt = &now

	if *input.ApproveStatusChange {
		pending.Status = domain.StepBackApproved
		task.Status = pending.ToStatus
		task.UpdatedAt = now
		if err := s.tasks.Update(ctx, task); err != nil {
			return nil, err
		}
	} else {
		pending.Status = domain.StepBackRejected
	}

	if err := s.stepback.Update(ctx, pending); err != nil {
		return nil, err
	}

	s.log.Info("pending status change reviewed",
		zap.String("task_id", task.ID),
		zap.Bool("approved", *input.ApproveStatusChange),
		zap.String("manager_id", actor.ID),
	)

	return s.enrichTaskWithPending(ctx, task)
}

func (s *Service) workerUpdateStatus(ctx context.Context, actor *domain.User, task *domain.Task, input UpdateStatusInput) (*domain.Task, error) {
	if input.ApproveStatusChange != nil {
		return nil, domain.Forbidden("workers cannot approve status changes")
	}

	if !input.Status.Valid() {
		return nil, domain.InvalidInput("invalid status value")
	}

	if task.AssigneeID == nil || *task.AssigneeID != actor.ID {
		return nil, domain.Forbidden("workers can only update their assigned tasks")
	}

	if task.Status == input.Status {
		return s.enrichTaskWithPending(ctx, task)
	}

	existingPending, err := s.stepback.GetPendingByTaskID(ctx, task.ID)
	if err != nil {
		return nil, err
	}
	if existingPending != nil {
		return nil, domain.NewAppError(
			"PENDING_STATUS_CHANGE",
			"task already has a pending status change awaiting manager approval",
			domain.ErrPendingStepBack,
		)
	}

	if domain.IsBackwardTransition(task.Status, input.Status) {
		return s.requestBackwardStatus(ctx, actor, task, input)
	}

	if !domain.IsForwardTransition(task.Status, input.Status) {
		return nil, domain.InvalidTransition(task.Status, input.Status)
	}

	task.Status = input.Status
	task.UpdatedAt = time.Now().UTC()

	if err := s.tasks.Update(ctx, task); err != nil {
		return nil, err
	}

	s.log.Info("task status updated",
		zap.String("task_id", task.ID),
		zap.String("status", string(task.Status)),
		zap.String("actor_id", actor.ID),
	)

	return s.enrichTaskWithPending(ctx, task)
}

func (s *Service) requestBackwardStatus(ctx context.Context, actor *domain.User, task *domain.Task, input UpdateStatusInput) (*domain.Task, error) {
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		return nil, domain.InvalidInput("reason is required when requesting a backward status change")
	}

	now := time.Now().UTC()
	req := &domain.StepBackRequest{
		ID:            uuid.New().String(),
		TaskID:        task.ID,
		RequestedByID: actor.ID,
		FromStatus:    task.Status,
		ToStatus:      input.Status,
		Reason:        reason,
		Status:        domain.StepBackPending,
		CreatedAt:     now,
	}

	if err := s.stepback.Create(ctx, req); err != nil {
		return nil, err
	}

	s.log.Info("backward status change requested",
		zap.String("task_id", task.ID),
		zap.String("current_status", string(task.Status)),
		zap.String("requested_status", string(input.Status)),
		zap.String("worker_id", actor.ID),
	)

	result := *task
	result.PendingStatusChange = domain.PendingStatusChangeFromRequest(req)
	return &result, nil
}
