package domain

import "time"

type StepBackStatus string

const (
	StepBackPending  StepBackStatus = "pending"
	StepBackApproved StepBackStatus = "approved"
	StepBackRejected StepBackStatus = "rejected"
)

type StepBackRequest struct {
	ID            string         `json:"id"`
	TaskID        string         `json:"task_id"`
	RequestedByID string         `json:"requested_by_id"`
	FromStatus    TaskStatus     `json:"from_status"`
	ToStatus      TaskStatus     `json:"to_status"`
	Reason        string         `json:"reason"`
	Status        StepBackStatus `json:"status"`
	ReviewedByID  *string        `json:"reviewed_by_id,omitempty"`
	ReviewedAt    *time.Time     `json:"reviewed_at,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
}
