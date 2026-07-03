package email

import (
	"context"
	"fmt"
	"net/smtp"
)

type SMTPConfig struct {
	Host string
	Port int
	From string
}

type SMTPSender struct {
	cfg SMTPConfig
}

func NewSMTPSender(cfg SMTPConfig) *SMTPSender {
	return &SMTPSender{cfg: cfg}
}

func (s *SMTPSender) SendAssignmentNotification(ctx context.Context, mail AssignmentEmail) error {
	_ = ctx

	subject := fmt.Sprintf("New task assigned: %s", mail.TaskTitle)
	body := fmt.Sprintf(
		"Hello %s,\n\nYou have been assigned a new task.\n\nTask: %s\nID: %s\nAssigned by: %s\n\nPlease log in to review your queue.\n",
		mail.AssigneeName, mail.TaskTitle, mail.TaskID, mail.AssignedBy,
	)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		s.cfg.From, mail.To, subject, body,
	)

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	if err := smtp.SendMail(addr, nil, s.cfg.From, []string{mail.To}, []byte(msg)); err != nil {
		return fmt.Errorf("send assignment email: %w", err)
	}
	return nil
}
