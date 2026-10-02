package application

import (
	"context"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
)

func (a *Auth) PublicUser(ctx context.Context, id int64) (domain.User, error) {
	return a.Accounts.ByID(ctx, id)
}
