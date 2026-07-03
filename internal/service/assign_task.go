package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rodrigocavalhero/nanojira/internal/domain"
	"github.com/rodrigocavalhero/nanojira/internal/email"
	"go.uber.org/zap"
)

func (s *Service) sendAssignmentEmail(ctx context.Context, task *domain.Task, assignedBy *domain.User) error {
	if task.AssigneeID == nil {
		return nil
	}

	assignee, err := s.users.GetByID(ctx, *task.AssigneeID)
	if err != nil {
		return fmt.Errorf("load assignee for notification: %w", err)
	}

	mail := email.AssignmentEmail{
		To:           assignee.Email,
		AssigneeName: assignee.Name,
		TaskTitle:    task.Title,
		TaskID:       task.ID,
		AssignedBy:   assignedBy.Name,
	}

	if err := s.email.SendAssignmentNotification(ctx, mail); err != nil {
		s.log.Error("failed to send assignment email",
			zap.String("task_id", task.ID),
			zap.String("assignee_id", assignee.ID),
			zap.Error(err),
		)
		return domain.NewAppError("EMAIL_FAILED", "failed to send assignment notification", domain.ErrEmailSend)
	}

	notification := &domain.AssignmentNotification{
		ID:         uuid.New().String(),
		TaskID:     task.ID,
		AssigneeID: assignee.ID,
		Email:      assignee.Email,
		SentAt:     time.Now().UTC(),
	}
	if err := s.tasks.CreateAssignmentNotification(ctx, notification); err != nil {
		return fmt.Errorf("record assignment notification: %w", err)
	}

	s.log.Info("assignment email sent",
		zap.String("task_id", task.ID),
		zap.String("assignee_id", assignee.ID),
		zap.String("email", assignee.Email),
	)
	return nil
}

type AssignTaskInput struct {
	AssigneeID string
}

func (s *Service) AssignTask(ctx context.Context, actorID, taskID string, input AssignTaskInput) (*domain.Task, error) {
	actor, err := s.users.GetByID(ctx, actorID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NotFound("user")
		}
		return nil, err
	}
	if !actor.IsManager() {
		return nil, domain.Forbidden("only managers can assign tasks")
	}

	assignee, err := s.users.GetByID(ctx, input.AssigneeID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NotFound("assignee")
		}
		return nil, err
	}
	if !assignee.IsWorker() {
		return nil, domain.InvalidInput("assignee must be a worker")
	}

	task, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NotFound("task")
		}
		return nil, err
	}

	if task.Status == domain.StatusDone {
		return nil, domain.InvalidInput("cannot assign a completed task")
	}

	previousAssignee := task.AssigneeID
	task.AssigneeID = &assignee.ID
	task.UpdatedAt = time.Now().UTC()

	if err := s.tasks.Update(ctx, task); err != nil {
		return nil, err
	}

	if previousAssignee == nil || *previousAssignee != assignee.ID {
		if err := s.sendAssignmentEmail(ctx, task, actor); err != nil {
			return nil, err
		}
	}

	s.log.Info("task assigned",
		zap.String("task_id", task.ID),
		zap.String("assignee_id", assignee.ID),
		zap.String("manager_id", actor.ID),
	)
	return task, nil
}
