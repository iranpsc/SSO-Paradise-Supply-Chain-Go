package cache

import (
	"context"
	"time"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
)

// --- application.Wallets ---
//
// Wallet data bypasses Redis: wallet reads happen at login frequency and
// wallet writes always invalidate the affected user object (whose cached
// JSON now carries wallet_address), so no wallet key can go stale.

func (s *Store) UserByWallet(ctx context.Context, address string) (domain.User, error) {
	return s.DB.UserByWallet(ctx, address)
}

func (s *Store) UserByCode(ctx context.Context, code string) (domain.User, error) {
	return s.DB.UserByCode(ctx, code)
}

func (s *Store) WalletOf(ctx context.Context, userID int64) (string, error) {
	return s.DB.WalletOf(ctx, userID)
}

func (s *Store) WalletTaken(ctx context.Context, address string) (bool, error) {
	return s.DB.WalletTaken(ctx, address)
}

func (s *Store) AttachWallet(ctx context.Context, userID int64, address string) (string, error) {
	result, err := s.DB.AttachWallet(ctx, userID, address)
	if err != nil {
		return result, err
	}
	if result == domain.WalletLinkSuccess {
		s.invalidateUser(ctx, userID)
	}
	return result, nil
}

func (s *Store) CreateWalletUser(ctx context.Context, address string, now time.Time) (domain.User, error) {
	created, err := s.DB.CreateWalletUser(ctx, address, now)
	if err != nil {
		return created, err
	}
	s.primeUser(ctx, created)
	return created, nil
}

func (s *Store) AttachWalletByCode(ctx context.Context, address, code string) (domain.User, error) {
	updated, err := s.DB.AttachWalletByCode(ctx, address, code)
	if err != nil {
		return updated, err
	}
	s.invalidateUser(ctx, updated.ID)
	s.primeUser(ctx, updated)
	return updated, nil
}

// --- application.Challenges ---

func (s *Store) SaveChallenge(ctx context.Context, key, message string, expiresAt time.Time) error {
	return s.DB.SaveChallenge(ctx, key, message, expiresAt)
}

func (s *Store) PullChallenge(ctx context.Context, key string, now time.Time) (string, error) {
	return s.DB.PullChallenge(ctx, key, now)
}

// --- application.SessionAttributes ---

func (s *Store) SetSessionAttribute(ctx context.Context, tokenHash, name, value string, expiresAt time.Time) error {
	return s.DB.SetSessionAttribute(ctx, tokenHash, name, value, expiresAt)
}

func (s *Store) SessionAttribute(ctx context.Context, tokenHash, name string, now time.Time) (string, error) {
	return s.DB.SessionAttribute(ctx, tokenHash, name, now)
}

func (s *Store) PullSessionAttribute(ctx context.Context, hash, name string, now time.Time) (string, error) {
	return s.DB.PullSessionAttribute(ctx, hash, name, now)
}
