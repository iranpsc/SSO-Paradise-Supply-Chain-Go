package application

import (
	"context"
	"time"
)

func (o *OAuth) PasswordAccess(ctx context.Context, userID, clientID int64) (OAuthTokens, error) {
	store, ok := o.Store.(interface {
		IssuePasswordOAuthToken(context.Context, int64, int64, TokenPair) (GrantIdentity, error)
	})
	if !ok {
		return OAuthTokens{}, OAuthError("unsupported_grant_type")
	}
	access, err := newOAuthID()
	if err != nil {
		return OAuthTokens{}, err
	}
	refresh, err := newToken()
	if err != nil {
		return OAuthTokens{}, err
	}
	now := o.Now()
	pair := TokenPair{Digest(access), Digest(refresh), oauthExpiry(now, o.AccessTTL), oauthExpiry(now, o.RefreshTTL)}
	if _, err = store.IssuePasswordOAuthToken(ctx, userID, clientID, pair); err != nil {
		return OAuthTokens{}, err
	}
	if o.Signer != nil {
		access, err = o.Signer.SignAccess(access, clientID, userID, []string{}, now, pair.AccessExpiry)
		if err != nil {
			return OAuthTokens{}, err
		}
	}
	return OAuthTokens{AccessToken: access, RefreshToken: refresh, TokenType: "Bearer", ExpiresIn: int(pair.AccessExpiry.Sub(now) / time.Second)}, nil
}
