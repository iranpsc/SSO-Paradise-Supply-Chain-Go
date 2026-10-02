package cache

import (
	"context"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
)

// --- application.Profiles ---

func (s *Store) PersonalInfo(ctx context.Context, id int64) (domain.PersonalInfo, error) {
	var p domain.PersonalInfo
	if s.getJSON(ctx, profileKey(id), &p) {
		return p, nil
	}
	p, err := s.DB.PersonalInfo(ctx, id)
	if err != nil {
		return p, err
	}
	s.setJSON(ctx, profileKey(id), p, s.userTTL())
	return p, nil
}

func (s *Store) SavePersonalInfo(ctx context.Context, p domain.PersonalInfo, media []domain.Media) error {
	if err := s.DB.SavePersonalInfo(ctx, p, media); err != nil {
		return err
	}
	// Only this user's profile and public card are dropped; nothing else in
	// the cache is touched.
	s.del(ctx, profileKey(p.UserID), publicKey(p.UserID))
	return nil
}

func (s *Store) SaveAvatar(ctx context.Context, m domain.Media) error {
	if err := s.DB.SaveAvatar(ctx, m); err != nil {
		return err
	}
	s.del(ctx, publicKey(m.UserID))
	return nil
}

func (s *Store) Media(ctx context.Context, id int64, kind string) (domain.Media, error) {
	// Binary documents stay in the database (private, served rarely); only
	// JSON metadata and lookups live in Redis.
	return s.DB.Media(ctx, id, kind)
}

func (s *Store) PublicProfile(ctx context.Context, id int64) (domain.PublicProfile, error) {
	var p domain.PublicProfile
	if s.getJSON(ctx, publicKey(id), &p) {
		return p, nil
	}
	p, err := s.DB.PublicProfile(ctx, id)
	if err != nil {
		return p, err
	}
	s.setJSON(ctx, publicKey(id), p, s.userTTL())
	return p, nil
}
