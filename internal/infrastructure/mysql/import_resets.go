package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"strings"
	"time"
)

type importedReset struct {
	User   int64
	Hash   string
	Expiry time.Time
}

func readLaravelResets(ctx context.Context, source *sql.Tx, users []importedUser) ([]importedReset, error) {
	table := ""
	for _, name := range []string{"password_reset_tokens", "password_resets"} {
		var n int
		if err := source.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name=?`, name).Scan(&n); err != nil {
			return nil, err
		}
		if n > 0 {
			table = name
			break
		}
	}
	if table == "" {
		return nil, nil
	}
	emails := map[string]int64{}
	for _, u := range users {
		emails[u.user.Email] = u.user.ID
	}
	rows, err := source.QueryContext(ctx, `SELECT email,token,created_at FROM `+table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []importedReset
	seen := map[int64]bool{}
	for rows.Next() {
		var email, hash string
		var created sql.NullTime
		if err = rows.Scan(&email, &hash, &created); err != nil {
			return nil, err
		}
		owner := emails[strings.ToLower(strings.TrimSpace(email))]
		if owner == 0 || !created.Valid || len(hash) > 255 {
			return nil, fmt.Errorf("incompatible/orphan Laravel password-reset row")
		}
		if _, err = bcrypt.Cost([]byte(hash)); err != nil {
			return nil, fmt.Errorf("Laravel reset hash is not bcrypt")
		}
		if seen[owner] {
			return nil, fmt.Errorf("duplicate Laravel reset owner")
		}
		seen[owner] = true
		result = append(result, importedReset{User: owner, Hash: hash, Expiry: created.Time.Add(time.Hour)})
	}
	return result, rows.Err()
}
