package service

import (
	"github.com/rodrigocavalhero/nanojira/internal/email"
	"github.com/rodrigocavalhero/nanojira/internal/repository"
	"go.uber.org/zap"
)

type Service struct {
	users    repository.UserRepository
	tasks    repository.TaskRepository
	stepback repository.StepBackRepository
	email    email.Sender
	log      *zap.Logger
}

func New(
	users repository.UserRepository,
	tasks repository.TaskRepository,
	stepback repository.StepBackRepository,
	emailSender email.Sender,
	log *zap.Logger,
) *Service {
	return &Service{
		users:    users,
		tasks:    tasks,
		stepback: stepback,
		email:    emailSender,
		log:      log,
	}
}
