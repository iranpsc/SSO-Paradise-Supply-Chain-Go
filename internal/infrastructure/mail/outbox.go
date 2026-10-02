package mail

import (
	"context"
	"errors"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"time"
)

type Outbox struct {
	Queue application.MailQueue
	Now   func() time.Time
}

func (o Outbox) Send(ctx context.Context, to, subject, text string) error {
	return o.SendMultipart(ctx, to, subject, text, "")
}
func (o Outbox) SendMultipart(ctx context.Context, to, subject, text, html string) error {
	return o.Queue.EnqueueMail(ctx, application.MailMessage{To: to, Subject: subject, Text: text, HTML: html}, o.Now())
}

// DeliverOne claims a durable lease. Multiple workers never deliver the same
// message simultaneously; a crash after SMTP delivery may lead to a retry.
func DeliverOne(ctx context.Context, q application.MailQueue, sender application.Mailer, now time.Time) (bool, error) {
	m, err := q.ClaimMail(ctx, now)
	if errors.Is(err, domain.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if rich, ok := sender.(application.MultipartMailer); ok {
		err = rich.SendMultipart(ctx, m.To, m.Subject, m.Text, m.HTML)
	} else {
		err = sender.Send(ctx, m.To, m.Subject, m.Text)
	}
	finishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	finishErr := q.FinishMail(finishCtx, m, now, err == nil)
	return true, errors.Join(err, finishErr)
}
