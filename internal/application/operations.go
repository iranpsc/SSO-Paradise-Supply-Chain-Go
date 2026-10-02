package application

import (
	"context"
	"time"
)

type Notification struct {
	Name, Link, Kind, AppName, Locale string
	Minutes, Year                     int
}
type NotificationMailer interface {
	SendNotification(context.Context, string, string, Notification) error
}
type MailMessage struct {
	ID                      int64
	To, Subject, Text, HTML string
	Attempts                int
}
type MultipartMailer interface {
	SendMultipart(context.Context, string, string, string, string) error
}
type MailQueue interface {
	EnqueueMail(context.Context, MailMessage, time.Time) error
	ClaimMail(context.Context, time.Time) (MailMessage, error)
	FinishMail(context.Context, MailMessage, time.Time, bool) error
}
type RateDecision struct {
	Remaining int
	Retry     time.Duration
}
type RateLimiter interface {
	CheckRate(context.Context, string, time.Time) (RateDecision, error)
}
type NamedRateLimiter interface {
	CheckBudget(context.Context, string, int, time.Time) (RateDecision, error)
}
type LoginAttemptLimiter interface {
	LoginRetry(context.Context, string, time.Time) (time.Duration, error)
	FailedLogin(context.Context, string, time.Time) error
	ClearFailedLogins(context.Context, string) error
}
type LegacyPasswordResetActions interface {
	ResetLegacyPassword(context.Context, string, string, string, time.Time) error
}
