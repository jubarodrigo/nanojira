package domain

import "time"

type PendingStatusChange struct {
	ID              string     `json:"id"`
	RequestedStatus TaskStatus `json:"requested_status"`
	Reason          string     `json:"reason"`
	RequestedByID   string     `json:"requested_by_id"`
	RequestedAt     time.Time  `json:"requested_at"`
}

func PendingStatusChangeFromRequest(req *StepBackRequest) *PendingStatusChange {
	if req == nil {
		return nil
	}
	return &PendingStatusChange{
		ID:              req.ID,
		RequestedStatus: req.ToStatus,
		Reason:          req.Reason,
		RequestedByID:   req.RequestedByID,
		RequestedAt:     req.CreatedAt,
	}
}
