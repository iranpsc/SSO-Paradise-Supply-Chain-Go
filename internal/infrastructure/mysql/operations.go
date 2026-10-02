package mysql

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"strings"
	"time"
)

//go:embed operations.sql
var operationsSchema string

func migrateOperations(ctx context.Context, conn *sql.Conn) error {
	var count int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version=4`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return migrateLegacySessions(ctx, conn)
	}
	for _, statement := range strings.Split(operationsSchema, ";") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if _, err := conn.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	var legacyColumn int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='oauth_codes' AND column_name='legacy'`).Scan(&legacyColumn); err != nil {
		return err
	}
	if legacyColumn == 0 {
		if _, err := conn.ExecContext(ctx, `ALTER TABLE oauth_codes ADD COLUMN legacy BOOLEAN NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}
	_, err := conn.ExecContext(ctx, `INSERT INTO schema_migrations(version,applied_at) VALUES(4,?)`, stamp(time.Now()))
	if err != nil {
		return err
	}
	return migrateLegacySessions(ctx, conn)
}

func (s *Store) EnqueueMail(ctx context.Context, m application.MailMessage, now time.Time) error {
	if len(m.To) > 255 || len([]rune(m.Subject)) > 255 || len(m.Text)+len(m.HTML) > 1<<20 {
		return errors.New("mail message exceeds queue limits")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO mail_outbox(recipient,subject,text_body,html_body,available_at,created_at,expires_at) VALUES(?,?,?,?,?,?,?)`, m.To, m.Subject, m.Text, m.HTML, stamp(now), stamp(now), stamp(now.Add(time.Hour)))
	return err
}
func (s *Store) ClaimMail(ctx context.Context, now time.Time) (application.MailMessage, error) {
	var m application.MailMessage
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return m, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE mail_outbox SET state='dead' WHERE expires_at<=? AND state IN ('pending','processing')`, stamp(now)); err != nil {
		return m, err
	}
	err = tx.QueryRowContext(ctx, `SELECT id,recipient,subject,text_body,html_body,attempts FROM mail_outbox WHERE state IN ('pending','processing') AND available_at<=? AND expires_at>? ORDER BY available_at,id LIMIT 1 FOR UPDATE SKIP LOCKED`, stamp(now), stamp(now)).Scan(&m.ID, &m.To, &m.Subject, &m.Text, &m.HTML, &m.Attempts)
	if errors.Is(err, sql.ErrNoRows) {
		if err = tx.Commit(); err != nil {
			return m, err
		}
		return m, domain.ErrNotFound
	}
	if err != nil {
		return m, err
	}
	m.Attempts++
	// A crashed worker's lease becomes available again. SMTP is at-least-once.
	_, err = tx.ExecContext(ctx, `UPDATE mail_outbox SET state='processing',attempts=?,available_at=? WHERE id=?`, m.Attempts, stamp(now.Add(5*time.Minute)), m.ID)
	if err != nil {
		return m, err
	}
	return m, tx.Commit()
}
func (s *Store) FinishMail(ctx context.Context, m application.MailMessage, now time.Time, success bool) error {
	state := "pending"
	var sent any
	if success {
		state = "sent"
		sent = stamp(now)
	} else if m.Attempts >= 10 {
		state = "dead"
	}
	delay := time.Minute * time.Duration(1<<min(m.Attempts-1, 10))
	_, err := s.db.ExecContext(ctx, `UPDATE mail_outbox SET state=?,sent_at=?,available_at=? WHERE id=? AND attempts=? AND state='processing'`, state, sent, stamp(now.Add(delay)), m.ID, m.Attempts)
	return err
}

func (s *Store) CheckRate(ctx context.Context, peer string, now time.Time) (application.RateDecision, error) {
	limit := s.peerLimit
	if limit <= 0 {
		limit = 10
	}
	return s.CheckBudget(ctx, peer, limit, now)
}
func (s *Store) ConfigureRateLimit(limit int) { s.peerLimit = limit }
func (s *Store) CheckBudget(ctx context.Context, peer string, limit int, now time.Time) (application.RateDecision, error) {
	var d application.RateDecision
	if limit < 1 || limit > 1000 {
		return d, errors.New("invalid rate budget")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return d, err
	}
	defer tx.Rollback()
	hash := application.Digest(peer)
	_, err = tx.ExecContext(ctx, `INSERT INTO rate_limits(peer_hash,window_start) VALUES(?,?) ON DUPLICATE KEY UPDATE peer_hash=peer_hash`, hash, stamp(now))
	if err != nil {
		return d, err
	}
	var start time.Time
	var blocked sql.NullTime
	var count, strikes int
	err = tx.QueryRowContext(ctx, `SELECT window_start,blocked_until,request_count,strikes FROM rate_limits WHERE peer_hash=? FOR UPDATE`, hash).Scan(&start, &blocked, &count, &strikes)
	if err != nil {
		return d, err
	}
	if blocked.Valid && now.Before(blocked.Time) {
		d.Retry = blocked.Time.Sub(now)
		return d, tx.Commit()
	}
	if now.Sub(start) >= time.Minute {
		start = now
		count = 0
	}
	if count >= limit {
		penalties := []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute, 30 * time.Minute, time.Hour, 3 * time.Hour, 24 * time.Hour}
		until := now.Add(penalties[min(strikes, 6)])
		blocked = sql.NullTime{Time: until, Valid: true}
		d.Retry = until.Sub(now)
		strikes = min(strikes+1, 6)
	} else {
		count++
		d.Remaining = limit - count
	}
	var until any
	if blocked.Valid {
		until = stamp(blocked.Time)
	}
	_, err = tx.ExecContext(ctx, `UPDATE rate_limits SET window_start=?,blocked_until=?,request_count=?,strikes=? WHERE peer_hash=?`, stamp(start), until, count, strikes, hash)
	if err != nil {
		return d, err
	}
	return d, tx.Commit()
}
