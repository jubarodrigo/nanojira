package email

import "context"

//go:generate mockgen -destination=../../mocks/mock_email.go -package=mocks github.com/rodrigocavalhero/nanojira/internal/email Sender

type AssignmentEmail struct {
	To           string
	AssigneeName string
	TaskTitle    string
	TaskID       string
	AssignedBy   string
}

type Sender interface {
	SendAssignmentNotification(ctx context.Context, email AssignmentEmail) error
}
