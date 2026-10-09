# Upstream changes ported on 2026-10-09

Source: Laravel `368600f..6c0e18c` (36 changed files). Target: Go backend and independent Next frontend.

| Upstream change | Go / Next result |
| --- | --- |
| Stop resending verification email on page load | Next already used manual resend. Added the 60-second persistent cooldown; mounting/reloading never sends mail. |
| Verification callback query handling and domain validation | Already implemented with URL parsing, `verified=1`, exact HTTPS metarang.com validation and one-use callbacks. |
| Shared numeric member-code allocation and unique index | Go already had a transactional shared sequence and unique codes. Fresh allocation now starts at `hm-2000000`; migration 11 adjusts only the unused old floor and preserves allocated codes. |
| Drop users.nonce | Go stores temporary challenges separately and has no users.nonce column. The synthetic Laravel contract probe now reflects the removed column. |
| Remove GET /api/users/{user} | Removed from Go; returns 404. Authenticated `/api/me` and public avatar files remain available. |
| Skip OAuth consent only for confidential first-party clients | Added ownership classification, source-owner import, session-bound expiring one-use consent requests, approval/denial handlers and the Next `/authorize` page. |
| Disable Passport password grant | `/oauth/token` rejects password grants with `unsupported_grant_type`; authorization-code and refresh grants remain enabled. |
| Security headers | Applied DENY, nosniff, referrer policy, cross-domain policy and the upstream CSP in Go and Next. HSTS is emitted for HTTPS deployments. Next requires PUBLIC_URL at build time to determine HTTPS. |
| Restricted CORS | Exact accounts.irpsc.com origin and HTTPS metarang.com subdomains. External API preflights permit GET/POST and specified headers; browser session mutations still require the configured same origin. Go's own frontend retains PUT/PATCH/DELETE support. |
| Trusted proxies | Go already trusts only configured TRUSTED_PROXY_CIDRS, never a wildcard. Compose retains its controlled private subnet. |
| Secure, encrypted sessions / SameSite=Lax | Go already uses opaque revocable cookies with hashed tokens, HTTPS Secure and Lax. Source PHP and AES-256-CBC encrypted file sessions now both import with authenticated decoding. |
| Secure document uploads up to 2 MiB | Backend validation, multipart reads, imports and frontend validation/help text use 2 MiB for documents. Avatar limit remains 1 MiB. Original filenames are validated. |
| Production log defaults and Redis null-password handling | API and worker use warning in production/staging, debug in development, LOG_LEVEL override and stderr JSON. Literal `null` Redis passwords normalize to empty. |
| Lean Docker builds and migration on startup | BuildKit dependency/build caches, stripped Go binaries, shared image and a one-shot migration service before API/worker startup. Redis requires a password and has an authenticated healthcheck. |
| Dokploy networking | Optional compose.dokploy.yaml attaches only the public Next service to dokploy-network and removes its host port. Internal API/MySQL/Redis stay on the private network. |
| Serve fonts from public document root | Next already uses `/style/fonts/...` and includes the public font assets. |
| Composer / reCAPTCHA / Sentry configuration and PHP CI | No corresponding runtime dependencies in Go. Existing Go dependency locks and MySQL integration tests remain in use. |

## Deployment details

Run `docker compose --env-file deploy/production.env up --build -d`; add `-f compose.yaml -f compose.dokploy.yaml` for Dokploy. Supply REDIS_PASSWORD and PUBLIC_URL in the environment file; PUBLIC_URL is also passed into the frontend build.

Migration 10 records `oauth_clients.first_party`. Ownership was not retained by older Go imports, so existing clients initially require consent. After verifying source ownership, operators can set first_party=1 only for internal clients. New administrator-created clients are internal; new imports preserve the source user_id/owner_id distinction. Public clients always require consent, even when internal.

No production database or deployment was changed. Verification uses disposable MySQL schemas, synthetic mail and browser sessions.

Compose files were validated as YAML. Docker is unavailable on this machine, so container image builds were not executed.
