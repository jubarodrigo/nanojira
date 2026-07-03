package repository

import (
	"context"

	"github.com/rodrigocavalhero/nanojira/internal/domain"
)

//go:generate mockgen -destination=../../mocks/mock_repository.go -package=mocks github.com/rodrigocavalhero/nanojira/internal/repository UserRepository,TaskRepository,StepBackRepository

type UserRepository interface {
	GetByID(ctx context.Context, id string) (*domain.User, error)
}

type TaskRepository interface {
	Create(ctx context.Context, task *domain.Task) error
	GetByID(ctx context.Context, id string) (*domain.Task, error)
	List(ctx context.Context, filter domain.TaskFilter) ([]domain.Task, error)
	Count(ctx context.Context, filter domain.TaskFilter) (int, error)
	Update(ctx context.Context, task *domain.Task) error
	CreateAssignmentNotification(ctx context.Context, n *domain.AssignmentNotification) error
	ListAssignmentNotifications(ctx context.Context, taskID string) ([]domain.AssignmentNotification, error)
}

type StepBackRepository interface {
	Create(ctx context.Context, req *domain.StepBackRequest) error
	GetByID(ctx context.Context, id string) (*domain.StepBackRequest, error)
	GetPendingByTaskID(ctx context.Context, taskID string) (*domain.StepBackRequest, error)
	Update(ctx context.Context, req *domain.StepBackRequest) error
}
