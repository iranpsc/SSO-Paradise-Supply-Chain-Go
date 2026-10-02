package mysql

import "context"

func (s *Store) Ready(ctx context.Context) error { return s.db.PingContext(ctx) }
