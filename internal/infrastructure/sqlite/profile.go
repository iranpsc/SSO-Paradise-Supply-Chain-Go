package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
)

//go:embed features.sql
var featuresSchema string

func (s *Store) PersonalInfo(ctx context.Context, id int64) (domain.PersonalInfo, error) {
	var p domain.PersonalInfo
	var payload string
	err := s.db.QueryRowContext(ctx, "SELECT payload FROM profile_details WHERE user_id=?", id).Scan(&payload)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return p, err
	}
	if err == nil {
		if err = json.Unmarshal([]byte(payload), &p); err != nil {
			return p, err
		}
	}
	err = s.db.QueryRowContext(ctx, "SELECT user_id,first_name,last_name,is_verified FROM personal_infos WHERE user_id=?", id).Scan(&p.UserID, &p.FirstName, &p.LastName, &p.IsVerified)
	if errors.Is(err, sql.ErrNoRows) {
		err = domain.ErrNotFound
	}
	return p, err
}

func saveMedia(ctx context.Context, tx *sql.Tx, m domain.Media) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO media(user_id,kind,content_type,data) VALUES(?,?,?,?) ON CONFLICT(user_id,kind) DO UPDATE SET content_type=excluded.content_type,data=excluded.data`, m.UserID, m.Kind, m.ContentType, m.Data)
	return err
}

func (s *Store) SavePersonalInfo(ctx context.Context, p domain.PersonalInfo, media []domain.Media) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Review status can only be changed by the review workflow, never by submitted JSON.
	var verified bool
	if err = tx.QueryRowContext(ctx, "SELECT is_verified FROM personal_infos WHERE user_id=?", p.UserID).Scan(&verified); err != nil {
		return err
	}
	p.IsVerified = verified
	p.VerificationMessages = ""
	payload, err := json.Marshal(p)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE personal_infos SET first_name=?,last_name=? WHERE user_id=?", p.FirstName, p.LastName, p.UserID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO profile_details(user_id,payload) VALUES(?,?) ON CONFLICT(user_id) DO UPDATE SET payload=excluded.payload`, p.UserID, string(payload)); err != nil {
		return err
	}
	for _, m := range media {
		if err = saveMedia(ctx, tx, m); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (s *Store) SaveAvatar(ctx context.Context, m domain.Media) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	m.Kind = "avatars"
	if err = saveMedia(ctx, tx, m); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) Media(ctx context.Context, id int64, kind string) (domain.Media, error) {
	m := domain.Media{UserID: id, Kind: kind}
	err := s.db.QueryRowContext(ctx, "SELECT content_type,data FROM media WHERE user_id=? AND kind=?", id, kind).Scan(&m.ContentType, &m.Data)
	if errors.Is(err, sql.ErrNoRows) {
		err = domain.ErrNotFound
	}
	return m, err
}
func (s *Store) PublicProfile(ctx context.Context, id int64) (domain.PublicProfile, error) {
	var p domain.PublicProfile
	var avatar bool
	err := s.db.QueryRowContext(ctx, `SELECT u.id,CASE WHEN p.is_verified THEN p.first_name||' '||p.last_name ELSE u.name END,u.code,EXISTS(SELECT 1 FROM media WHERE user_id=u.id AND kind='avatars') FROM users u JOIN personal_infos p ON p.user_id=u.id WHERE u.id=?`, id).Scan(&p.ID, &p.Name, &p.Code, &avatar)
	if errors.Is(err, sql.ErrNoRows) {
		err = domain.ErrNotFound
	}
	if avatar {
		p.Avatar = fmt.Sprintf("/api/users/%d/avatar", id)
	}
	return p, err
}
