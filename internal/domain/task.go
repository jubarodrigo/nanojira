package domain

import "time"

type TaskStatus string

const (
	StatusTodo    TaskStatus = "todo"
	StatusDoing   TaskStatus = "doing"
	StatusOnHold  TaskStatus = "on_hold"
	StatusTesting TaskStatus = "testing"
	StatusDone    TaskStatus = "done"
)

var validStatuses = map[TaskStatus]struct{}{
	StatusTodo:    {},
	StatusDoing:   {},
	StatusOnHold:  {},
	StatusTesting: {},
	StatusDone:    {},
}

func (s TaskStatus) Valid() bool {
	_, ok := validStatuses[s]
	return ok
}

// ForwardTransitions define allowed forward moves without manager approval.
var ForwardTransitions = map[TaskStatus][]TaskStatus{
	StatusTodo:    {StatusDoing},
	StatusDoing:   {StatusTesting, StatusOnHold},
	StatusTesting: {StatusDone, StatusOnHold},
	StatusOnHold:  {StatusDoing},
	StatusDone:    {},
}

// BackwardTransitions require a step-back request approved by a manager.
var BackwardTransitions = map[TaskStatus][]TaskStatus{
	StatusDoing:   {StatusTodo},
	StatusTesting: {StatusDoing},
	StatusDone:    {StatusTesting},
}

func IsForwardTransition(from, to TaskStatus) bool {
	for _, allowed := range ForwardTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

func IsBackwardTransition(from, to TaskStatus) bool {
	for _, allowed := range BackwardTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

type Task struct {
	ID                  string               `json:"id"`
	Title               string               `json:"title"`
	Description         string               `json:"description,omitempty"`
	Status              TaskStatus           `json:"status"`
	ReporterID          string               `json:"reporter_id"`
	AssigneeID          *string              `json:"assignee_id,omitempty"`
	PendingStatusChange *PendingStatusChange `json:"pending_status_change,omitempty"`
	CreatedAt           time.Time            `json:"created_at"`
	UpdatedAt           time.Time            `json:"updated_at"`
}

type AssignmentNotification struct {
	ID         string    `json:"id"`
	TaskID     string    `json:"task_id"`
	AssigneeID string    `json:"assignee_id"`
	Email      string    `json:"email"`
	SentAt     time.Time `json:"sent_at"`
}

type TaskFilter struct {
	AssigneeID *string
	Status     *TaskStatus
	Limit      int
	Offset     int
}
