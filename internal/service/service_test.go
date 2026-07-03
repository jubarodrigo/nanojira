package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rodrigocavalhero/nanojira/internal/domain"
	"github.com/rodrigocavalhero/nanojira/internal/email"
	"github.com/rodrigocavalhero/nanojira/internal/service"
	"github.com/rodrigocavalhero/nanojira/mocks"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func newTestService(t *testing.T) (*service.Service, *mocks.MockUserRepository, *mocks.MockTaskRepository, *mocks.MockStepBackRepository, *mocks.MockSender) {
	t.Helper()
	ctrl := gomock.NewController(t)
	users := mocks.NewMockUserRepository(ctrl)
	tasks := mocks.NewMockTaskRepository(ctrl)
	stepback := mocks.NewMockStepBackRepository(ctrl)
	sender := mocks.NewMockSender(ctrl)
	log := zap.NewNop()
	return service.New(users, tasks, stepback, sender, log), users, tasks, stepback, sender
}

func TestCreateTask_ManagerOnly(t *testing.T) {
	svc, users, _, _, _ := newTestService(t)
	ctx := context.Background()

	users.EXPECT().GetByID(ctx, "worker-1").Return(&domain.User{
		ID: "worker-1", Role: domain.RoleWorker,
	}, nil)

	_, err := svc.CreateTask(ctx, "worker-1", service.CreateTaskInput{Title: "New task"})
	if err == nil {
		t.Fatal("expected error")
	}
	var appErr *domain.AppError
	if !errors.As(err, &appErr) || !errors.Is(appErr.Err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestCreateTask_WithAssigneeSendsEmail(t *testing.T) {
	svc, users, tasks, _, sender := newTestService(t)
	ctx := context.Background()

	manager := &domain.User{ID: "manager-1", Name: "Alice", Role: domain.RoleManager}
	worker := &domain.User{ID: "worker-1", Name: "Carol", Email: "carol@example.com", Role: domain.RoleWorker}
	assigneeID := worker.ID

	users.EXPECT().GetByID(ctx, manager.ID).Return(manager, nil)
	users.EXPECT().GetByID(ctx, worker.ID).Return(worker, nil).Times(2)
	tasks.EXPECT().Create(ctx, gomock.Any()).Return(nil)
	sender.EXPECT().SendAssignmentNotification(ctx, gomock.AssignableToTypeOf(email.AssignmentEmail{})).Return(nil)
	tasks.EXPECT().CreateAssignmentNotification(ctx, gomock.Any()).Return(nil)

	task, err := svc.CreateTask(ctx, manager.ID, service.CreateTaskInput{
		Title:      "Deploy",
		AssigneeID: &assigneeID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.AssigneeID == nil || *task.AssigneeID != worker.ID {
		t.Fatalf("expected assignee %s, got %v", worker.ID, task.AssigneeID)
	}
}

func TestUpdateTaskStatus_BackwardCreatesPendingWithoutChangingStatus(t *testing.T) {
	svc, users, tasks, stepback, _ := newTestService(t)
	ctx := context.Background()

	worker := &domain.User{ID: "worker-1", Role: domain.RoleWorker}
	task := &domain.Task{
		ID: "task-1", Status: domain.StatusTesting,
		AssigneeID: &worker.ID,
	}

	users.EXPECT().GetByID(ctx, worker.ID).Return(worker, nil)
	tasks.EXPECT().GetByID(ctx, task.ID).Return(task, nil)
	stepback.EXPECT().GetPendingByTaskID(ctx, task.ID).Return(nil, nil)
	stepback.EXPECT().Create(ctx, gomock.Any()).Return(nil)

	updated, err := svc.UpdateTaskStatus(ctx, worker.ID, task.ID, service.UpdateStatusInput{
		Status: domain.StatusDoing,
		Reason: "Found regression in auth module",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Status != domain.StatusTesting {
		t.Fatalf("expected status to remain testing, got %s", updated.Status)
	}
	if updated.PendingStatusChange == nil {
		t.Fatal("expected pending status change")
	}
	if updated.PendingStatusChange.RequestedStatus != domain.StatusDoing {
		t.Fatalf("expected requested status doing, got %s", updated.PendingStatusChange.RequestedStatus)
	}
}

func TestUpdateTaskStatus_BackwardRequiresReason(t *testing.T) {
	svc, users, tasks, stepback, _ := newTestService(t)
	ctx := context.Background()

	worker := &domain.User{ID: "worker-1", Role: domain.RoleWorker}
	task := &domain.Task{
		ID: "task-1", Status: domain.StatusTesting,
		AssigneeID: &worker.ID,
	}

	users.EXPECT().GetByID(ctx, worker.ID).Return(worker, nil)
	tasks.EXPECT().GetByID(ctx, task.ID).Return(task, nil)
	stepback.EXPECT().GetPendingByTaskID(ctx, task.ID).Return(nil, nil)

	_, err := svc.UpdateTaskStatus(ctx, worker.ID, task.ID, service.UpdateStatusInput{
		Status: domain.StatusDoing,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	var appErr *domain.AppError
	if !errors.As(err, &appErr) || !errors.Is(appErr.Err, domain.ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestUpdateTaskStatus_ForwardTransition(t *testing.T) {
	svc, users, tasks, stepback, _ := newTestService(t)
	ctx := context.Background()

	worker := &domain.User{ID: "worker-1", Role: domain.RoleWorker}
	task := &domain.Task{
		ID: "task-1", Status: domain.StatusTodo,
		AssigneeID: &worker.ID,
	}

	users.EXPECT().GetByID(ctx, worker.ID).Return(worker, nil)
	tasks.EXPECT().GetByID(ctx, task.ID).Return(task, nil)
	stepback.EXPECT().GetPendingByTaskID(ctx, task.ID).Return(nil, nil).Times(2)
	tasks.EXPECT().Update(ctx, gomock.Any()).Return(nil)

	updated, err := svc.UpdateTaskStatus(ctx, worker.ID, task.ID, service.UpdateStatusInput{
		Status: domain.StatusDoing,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Status != domain.StatusDoing {
		t.Fatalf("expected doing, got %s", updated.Status)
	}
}

func TestUpdateTaskStatus_ManagerApproveUpdatesStatus(t *testing.T) {
	svc, users, tasks, stepback, _ := newTestService(t)
	ctx := context.Background()

	manager := &domain.User{ID: "manager-1", Role: domain.RoleManager}
	task := &domain.Task{ID: "task-1", Status: domain.StatusTesting}
	pending := &domain.StepBackRequest{
		ID: "req-1", TaskID: task.ID,
		FromStatus: domain.StatusTesting, ToStatus: domain.StatusDoing,
		Status: domain.StepBackPending,
	}
	approve := true

	users.EXPECT().GetByID(ctx, manager.ID).Return(manager, nil)
	tasks.EXPECT().GetByID(ctx, task.ID).Return(task, nil)
	stepback.EXPECT().GetPendingByTaskID(ctx, task.ID).Return(pending, nil)
	tasks.EXPECT().Update(ctx, gomock.Any()).Return(nil)
	stepback.EXPECT().Update(ctx, gomock.Any()).Return(nil)
	stepback.EXPECT().GetPendingByTaskID(ctx, task.ID).Return(nil, nil)

	updated, err := svc.UpdateTaskStatus(ctx, manager.ID, task.ID, service.UpdateStatusInput{
		ApproveStatusChange: &approve,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Status != domain.StatusDoing {
		t.Fatalf("expected doing after approval, got %s", updated.Status)
	}
	if updated.PendingStatusChange != nil {
		t.Fatal("expected no pending status change after approval")
	}
}

func TestAssignTask_SendsEmail(t *testing.T) {
	svc, users, tasks, _, sender := newTestService(t)
	ctx := context.Background()

	manager := &domain.User{ID: "manager-1", Name: "Alice", Role: domain.RoleManager}
	worker := &domain.User{ID: "worker-1", Name: "Carol", Email: "carol@example.com", Role: domain.RoleWorker}
	task := &domain.Task{ID: "task-1", Title: "Bugfix", Status: domain.StatusTodo}

	users.EXPECT().GetByID(ctx, manager.ID).Return(manager, nil)
	users.EXPECT().GetByID(ctx, worker.ID).Return(worker, nil)
	tasks.EXPECT().GetByID(ctx, task.ID).Return(task, nil)
	tasks.EXPECT().Update(ctx, gomock.Any()).Return(nil)
	users.EXPECT().GetByID(ctx, worker.ID).Return(worker, nil)
	sender.EXPECT().SendAssignmentNotification(ctx, gomock.Any()).Return(nil)
	tasks.EXPECT().CreateAssignmentNotification(ctx, gomock.Any()).Return(nil)

	result, err := svc.AssignTask(ctx, manager.ID, task.ID, service.AssignTaskInput{AssigneeID: worker.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AssigneeID == nil || *result.AssigneeID != worker.ID {
		t.Fatal("expected assignee to be set")
	}
}
