package application

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
)

type LegacyTokenCipher interface{ Decode(string) ([]byte, error) }
type LegacyCodeBinder interface {
	BindLegacyCode(context.Context, string, int64, string, string) error
}
type legacyPayload struct {
	ClientID  string  `json:"client_id"`
	CodeID    string  `json:"auth_code_id"`
	RefreshID string  `json:"refresh_token_id"`
	Expires   float64 `json:"expire_time"`
	Redirect  string  `json:"redirect_uri"`
	Challenge string  `json:"code_challenge"`
	Method    string  `json:"code_challenge_method"`
}

func (o *OAuth) decodeLegacy(token string, client int64) (legacyPayload, error) {
	var p legacyPayload
	if o.LegacyCipher == nil {
		return p, OAuthInvalidGrant
	}
	raw, err := o.LegacyCipher.Decode(token)
	if err != nil {
		return p, OAuthInvalidGrant
	}
	if err = json.Unmarshal(raw, &p); err != nil || p.ClientID != strconv.FormatInt(client, 10) || p.Expires <= float64(o.Now().Unix()) {
		return p, OAuthInvalidGrant
	}
	id := p.CodeID
	if id == "" {
		id = p.RefreshID
	}
	if len(id) != 80 || strings.Trim(id, "0123456789abcdef") != "" {
		return p, OAuthInvalidGrant
	}
	return p, nil
}
