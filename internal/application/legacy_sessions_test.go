package application_test

import (
	"context"
	"database/sql"
	"errors"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/mysql/mysqltest"
	"strings"
	"testing"
	"time"
)

type plainCookie struct{}

func (plainCookie) DecodeCookie(_, value string) (string, error) { return value, nil }

type logoutDuringLookup struct{ *mysql.Store }

func (s logoutDuringLookup) SessionUser(ctx context.Context, hash string, now time.Time) (domain.User, error) {
	u, err := s.Store.SessionUser(ctx, hash, now)
	if err != nil {
		return u, err
	}
	if err = s.Store.RevokeSessions(ctx, u.ID); err != nil {
		return u, err
	}
	return u, nil
}
func TestLegacySessionCannotResurrectConcurrentLogout(t *testing.T) {
	db := mysqltest.Open(t)
	ctx := context.Background()
	now := time.Now()
	u, err := db.Create(ctx, domain.User{Name: "Legacy", Email: "legacy@example.com", CreatedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	id := strings.Repeat("s", 40)
	if err = db.CreateSession(ctx, u.ID, application.Digest(id), now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	a := application.Auth{Accounts: db, Sessions: logoutDuringLookup{db}, Now: time.Now, SessionTTL: time.Hour, LegacySessionCookie: "legacy", LegacyCookies: plainCookie{}}
	if _, _, err = a.RestoreLegacyCookie(ctx, "legacy", id); !errors.Is(err, domain.ErrCredentials) {
		t.Fatal("logout between validation and restoration was ignored", err)
	}
	raw, err := sql.Open("mysql", db.DSN())
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	var count int
	if err = raw.QueryRow(`SELECT COUNT(*) FROM sessions WHERE user_id=?`, u.ID).Scan(&count); err != nil || count != 0 {
		t.Fatal("restoration recreated a revoked session", err)
	}
}
