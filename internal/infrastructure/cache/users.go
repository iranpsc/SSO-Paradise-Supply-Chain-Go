package cache

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
)

// --- application.Accounts ---

func (s *Store) Create(ctx context.Context, u domain.User) (domain.User, error) {
	created, err := s.DB.Create(ctx, u)
	if err != nil {
		return created, err
	}
	s.primeUser(ctx, created)
	return created, nil
}

func (s *Store) ByID(ctx context.Context, id int64) (domain.User, error) {
	var c cachedUser
	if s.getJSON(ctx, userKey(id), &c) {
		return c.user(), nil
	}
	u, err := s.DB.ByID(ctx, id)
	if err != nil {
		return u, err
	}
	s.primeUser(ctx, u)
	return u, nil
}

func (s *Store) ByEmail(ctx context.Context, email string) (domain.User, error) {
	if id, ok := s.cachedID(ctx, emailKey(email)); ok {
		if u, err := s.ByID(ctx, id); err == nil && strings.EqualFold(u.Email, email) {
			return u, nil
		}
		s.del(ctx, emailKey(email))
	}
	u, err := s.DB.ByEmail(ctx, email)
	if err != nil {
		return u, err
	}
	s.primeUser(ctx, u)
	return u, nil
}

func (s *Store) ByLogin(ctx context.Context, identifier string) (domain.User, error) {
	if !s.enabled() {
		return s.DB.ByLogin(ctx, identifier)
	}
	if strings.Contains(identifier, "@") {
		return s.ByEmail(ctx, identifier)
	}
	if id, ok := s.cachedID(ctx, usernameKey(identifier)); ok {
		if u, err := s.ByID(ctx, id); err == nil {
			return u, nil
		}
		s.del(ctx, usernameKey(identifier))
	}
	u, err := s.DB.ByLogin(ctx, identifier)
	if err != nil {
		return u, err
	}
	s.primeUser(ctx, u)
	return u, nil
}

func (s *Store) CodeExists(ctx context.Context, code string) (bool, error) {
	return s.DB.CodeExists(ctx, code)
}

func (s *Store) UpdateIdentity(ctx context.Context, id int64, name, email string) (domain.User, error) {
	old, oldErr := s.DB.ByID(ctx, id)
	updated, err := s.DB.UpdateIdentity(ctx, id, name, email)
	if err != nil {
		return updated, err
	}
	s.invalidateUser(ctx, id)
	if oldErr == nil && old.Email != "" {
		s.del(ctx, emailKey(old.Email))
	}
	if updated.Email != "" {
		s.del(ctx, emailKey(updated.Email))
	}
	s.primeUser(ctx, updated)
	return updated, nil
}

func (s *Store) ChangePassword(ctx context.Context, id int64, hash, keepSession string) error {
	if err := s.DB.ChangePassword(ctx, id, hash, keepSession); err != nil {
		return err
	}
	s.invalidateUser(ctx, id)
	s.dropSessions(ctx, id)
	// Re-track the kept session so the current device stays fast; it will be
	// re-primed from the database on its next request.
	return nil
}

// --- application.Sessions ---

func (s *Store) CreateSession(ctx context.Context, id int64, hash string, until time.Time) error {
	if err := s.DB.CreateSession(ctx, id, hash, until); err != nil {
		return err
	}
	s.trackSession(ctx, id, hash, time.Until(until))
	return nil
}

// Session validity and current identity always come from MySQL. Redis must
// never extend a session or resurrect a revoked token after an outage.
func (s *Store) SessionUser(ctx context.Context, hash string, now time.Time) (domain.User, error) {
	u, err := s.DB.SessionUser(ctx, hash, now)
	if err != nil {
		s.del(ctx, sessionKey(hash))
		return u, err
	}
	_, until, err := s.DB.SessionLookup(ctx, hash, now)
	if err != nil {
		return domain.User{}, err
	}
	s.primeUser(ctx, u)
	s.trackSession(ctx, u.ID, hash, until.Sub(now))
	return u, nil
}

func (s *Store) RevokeSessions(ctx context.Context, id int64) error {
	if err := s.DB.RevokeSessions(ctx, id); err != nil {
		return err
	}
	s.dropSessions(ctx, id)
	return nil
}

// --- application.Actions ---

func (s *Store) SaveAction(ctx context.Context, id int64, kind, hash, email string, until time.Time) error {
	return s.DB.SaveAction(ctx, id, kind, hash, email, until)
}

func (s *Store) VerifyEmail(ctx context.Context, id int64, hash string, now time.Time) error {
	if err := s.DB.VerifyEmail(ctx, id, hash, now); err != nil {
		return err
	}
	// Verification assigns the member code, so the user object and the public
	// resource of this user only are dropped and re-primed on next read.
	s.invalidateUser(ctx, id)
	if u, err := s.DB.ByID(ctx, id); err == nil {
		s.primeUser(ctx, u)
	}
	return nil
}

func (s *Store) ResetPassword(ctx context.Context, email, tokenHash, passwordHash string, now time.Time) error {
	var id int64
	if u, err := s.DB.ByEmail(ctx, email); err == nil {
		id = u.ID
	} else if !errors.Is(err, domain.ErrNotFound) {
		return err
	}
	if err := s.DB.ResetPassword(ctx, email, tokenHash, passwordHash, now); err != nil {
		return err
	}
	if id != 0 {
		s.invalidateUser(ctx, id)
		s.dropSessions(ctx, id)
	}
	return nil
}

func (s *Store) CreateRegistration(ctx context.Context, u domain.User, backURL string, until time.Time) (domain.User, error) {
	created, err := s.DB.CreateRegistration(ctx, u, backURL, until)
	if err == nil {
		s.primeUser(ctx, created)
	}
	return created, err
}
func (s *Store) PullRegistrationCallback(ctx context.Context, id int64, now time.Time) (string, error) {
	return s.DB.PullRegistrationCallback(ctx, id, now)
}

func (s *Store) VerifySignedEmail(ctx context.Context, id int64, email string, now time.Time) error {
	if err := s.DB.VerifySignedEmail(ctx, id, email, now); err != nil {
		return err
	}
	s.invalidateUser(ctx, id)
	return nil
}

// Password checks bypass cached identity, even when Redis invalidation failed.
func (s *Store) CredentialsByLogin(ctx context.Context, identifier string) (domain.User, error) {
	return s.DB.ByLogin(ctx, identifier)
}

func (s *Store) ResetLegacyPassword(ctx context.Context, email, tokenHash, passwordHash string, now time.Time) error {
	var id int64
	if u, err := s.DB.ByEmail(ctx, email); err == nil {
		id = u.ID
	} else if !errors.Is(err, domain.ErrNotFound) {
		return err
	}
	if err := s.DB.ResetLegacyPassword(ctx, email, tokenHash, passwordHash, now); err != nil {
		return err
	}
	if id != 0 {
		s.invalidateUser(ctx, id)
		s.dropSessions(ctx, id)
	}
	return nil
}
