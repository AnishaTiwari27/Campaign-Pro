package service

import (
	"context"
	"log/slog"
)

// Mailer is behind an interface so a real provider can be swapped in later
// without touching the reports service; LogMailer is the dev implementation
// MAIL_MODE=log selects.
type Mailer interface {
	Send(ctx context.Context, to []string, subject, attachmentName string, attachment []byte) error
}

type LogMailer struct {
	Logger *slog.Logger
}

func (m LogMailer) Send(ctx context.Context, to []string, subject, attachmentName string, attachment []byte) error {
	m.Logger.Info("mail sent (log mode)", "to", to, "subject", subject, "attachment", attachmentName, "bytes", len(attachment))
	return nil
}
