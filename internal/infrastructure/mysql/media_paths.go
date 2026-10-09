package mysql

import (
	"context"
	"database/sql"
	"errors"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
	"time"
)

func migrateMediaPaths(ctx context.Context, conn *sql.Conn) error {
	var count int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version=8`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return migrateClientGrants(ctx, conn)
	}
	if _, err := conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS media_public_paths(user_id BIGINT NOT NULL PRIMARY KEY,path VARCHAR(1024) NOT NULL,absolute_url BOOLEAN NOT NULL DEFAULT 0,CONSTRAINT fk_public_media_user FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE) ENGINE=InnoDB`); err != nil {
		return err
	}
	_, err := conn.ExecContext(ctx, `INSERT INTO schema_migrations(version,applied_at) VALUES(8,?)`, stamp(time.Now()))
	if err != nil {
		return err
	}
	return migrateClientGrants(ctx, conn)
}

func (s *Store) LegacyPublicMedia(ctx context.Context, path string) (domain.Media, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `SELECT user_id FROM media_public_paths WHERE path=?`, path).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Media{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Media{}, err
	}
	return s.Media(ctx, id, "avatars")
}
