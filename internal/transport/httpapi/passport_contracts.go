package httpapi

import (
	"fmt"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"net/http"
	"strings"
)

type passportError struct {
	kind      application.OAuthError
	parameter string
}

func (e passportError) Error() string { return string(e.kind) }
func passportPayload(kind application.OAuthError, parameter string) (int, map[string]string) {
	status := 400
	body := map[string]string{"error": string(kind)}
	switch kind {
	case "unauthorized_client":
		body["error_description"] = "The authenticated client is not authorized to use this authorization grant type."
	case "invalid_credentials":
		body["error"] = "invalid_grant"
		body["error_description"] = "The user credentials were incorrect."
	case application.OAuthInvalidClient:
		status = 401
		body["error_description"] = "Client authentication failed"
	case application.OAuthInvalidRequest:
		body["error_description"] = "The request is missing a required parameter, includes an invalid parameter value, includes a parameter more than once, or is otherwise malformed."
		if parameter == "" {
			parameter = "grant_type"
		}
		body["hint"] = fmt.Sprintf("Check the `%s` parameter", parameter)
	case application.OAuthInvalidGrant:
		body["error_description"] = "The provided authorization grant (e.g., authorization code, resource owner credentials) or refresh token is invalid, expired, revoked, does not match the redirection URI used in the authorization request, or was issued to another client."
		body["hint"] = ""
	case application.OAuthInvalidScope:
		body["error_description"] = "The requested scope is invalid, unknown, or malformed"
		body["hint"] = fmt.Sprintf("Check the `%s` scope", parameter)
	case "unsupported_grant_type":
		body["error_description"] = "The authorization grant type is not supported by the authorization server."
		body["hint"] = "Check that all required parameters have been provided"
	case "access_denied":
		status = 401
		body["error_description"] = "The resource owner or authorization server denied the request."
	case "unsupported_response_type":
		return passportPayload(application.OAuthInvalidRequest, "response_type")
	default:
		body["error_description"] = "The authorization server encountered an unexpected condition which prevented it from fulfilling the request."
	}
	return status, body
}

func passportRespond(w http.ResponseWriter, r *http.Request, kind application.OAuthError, parameter string) {
	status, body := passportPayload(kind, parameter)
	if kind == application.OAuthInvalidClient && r != nil {
		scheme := strings.Fields(r.Header.Get("Authorization"))
		if len(scheme) > 0 {
			w.Header().Set("WWW-Authenticate", scheme[0]+` realm="OAuth"`)
		}
	}
	respond(w, status, body)
}
