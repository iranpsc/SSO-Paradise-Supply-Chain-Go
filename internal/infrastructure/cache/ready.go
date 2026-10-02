package cache

import "context"

func (s *Store) Ready(ctx context.Context) error { return s.DB.Ready(ctx) }
