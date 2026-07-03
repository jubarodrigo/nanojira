package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound          = errors.New("resource not found")
	ErrForbidden         = errors.New("forbidden")
	ErrInvalidInput      = errors.New("invalid input")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrPendingStepBack   = errors.New("task already has a pending status change awaiting approval")
	ErrUnassigned        = errors.New("task has no assignee")
	ErrNotAssignee       = errors.New("user is not the task assignee")
	ErrInvalidStepBack   = errors.New("invalid step-back request state")
	ErrEmailSend         = errors.New("failed to send assignment email")
)

type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewAppError(code, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}

func NotFound(resource string) *AppError {
	return NewAppError("NOT_FOUND", fmt.Sprintf("%s not found", resource), ErrNotFound)
}

func Forbidden(message string) *AppError {
	if message == "" {
		message = "you are not allowed to perform this action"
	}
	return NewAppError("FORBIDDEN", message, ErrForbidden)
}

func InvalidInput(message string) *AppError {
	return NewAppError("INVALID_INPUT", message, ErrInvalidInput)
}

func InvalidTransition(from, to TaskStatus) *AppError {
	return NewAppError(
		"INVALID_TRANSITION",
		fmt.Sprintf("cannot transition from %s to %s", from, to),
		ErrInvalidTransition,
	)
}
