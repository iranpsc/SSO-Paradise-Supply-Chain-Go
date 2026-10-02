# Laravel migration and compatibility

## Delivered behavior

The five Laravel API routes retain their HTTP methods: POST /api/login, POST /api/logout, POST /api/me, GET /api/user and GET /api/users/{user}. Login returns message/token; API OAuth access tokens use RS256. Browser authentication also sets an independent HttpOnly/SameSite=Lax session cookie. User resources and verified personal names/avatars are supported. Username login remains a Go extension.

Web flows include registration, remember-me, password confirmation (three hours), password change/reset, signed verification, personal/company fields, avatars/documents, Web3 login/link and registration callback paths. Native verification uses /email/verify/{id}/{sha1(email)} and requires the same authenticated user. The callback origin is restricted to https://metarang.com as in Laravel. Registration accepts client_id, redirect_uri and back_url. MetaMask and WalletConnect sign server-issued, expiring, one-use nonces; wallet ownership is transactional. Metarang_API is required for checking unknown wallets.

OAuth endpoints are /oauth/authorize, /oauth/token and /oauth/revoke; DELETE /oauth/authorize denies and POST /oauth/token/refresh renews a first-party browser session. Exact redirect matching and authenticated automatic authorization match Laravel's custom client policy. Public clients require S256 PKCE; confidential legacy clients may omit it. The Go profile scope is an extension; imported scopes are preserved, and the original API resource routes require authentication without imposing a new scope. Codes expire in five minutes and are single-use. Access/refresh/personal token lifetimes default to a calendar year (Passport P1Y); optional duration overrides are described in production.md. Refresh replay revokes the grant.


## Safe schema upgrade

MYSQL_DATABASE must identify a dedicated Go database, never the Laravel source. Run go run ./cmd/migrate. The command creates a missing database and applies pending versions; it does not drop it. Runtime opening also checks migrations. A Laravel users.password column blocks accidental schema conversion. An advisory database lock prevents concurrent migrations.

Version 2 upgrades legacy Go storage to BIGINT identifiers, native UTC DATETIME(6), atomic personal-info columns and correctly sized media/token fields; version 3 adds OAuth tables. Versions 4–6 add the mail outbox/shared rate limiter, legacy reset/remember support, code binding and queue deadlines. Version 7 adds per-session idle duration without changing existing session deadlines. Fresh databases start directly from the canonical version-2 schema. Earlier embedded scripts are immutable upgrade inputs. Read-only profile_details preserves old JSON after backfill; application reads/writes use columns.

Preflight rejects oversized values, malformed timestamps/profile data and orphan references instead of truncating or deleting them. Fix those records deliberately and rerun. MySQL DDL implicitly commits: the upgrade is resumable, but not one all-or-nothing transaction. Back up the target and stop concurrent application writes during upgrades.

## Laravel data import

The importer preserves user IDs, bcrypt hashes, mobile, timestamps, verified status, member/referral codes, wallets, personal/company data, OAuth clients/redirect URIs, access/refresh tokens, authorization codes, remember-token digests and password-reset hashes. Plaintext legacy client secrets become bcrypt hashes; already bcrypt-hashed secrets remain unchanged. Revoked clients remain revoked. Spatie avatar and the three document collections are loaded from local/public storage, validated and stored as MEDIUMBLOB. Source personal-info media ownership is mapped to the associated user. Missing files, unsupported disks/models, oversized images, duplicate collections, duplicate identities, or invalid source values stop the import.

Create/migrate an empty target first. Use a source MySQL account with read-only permissions:

```powershell
$env:MYSQL_DATABASE = 'paradise_go'
$env:LARAVEL_MYSQL_DSN = 'readonly:REPLACE_ME@tcp(127.0.0.1:3306)/laravel_source'
go run ./cmd/migrate
go run ./cmd/import --media-root ../SSO-Paradise-Supply-Chain/storage/app
```

Default import is a dry run: source records/files are checked without inserting target data. Target schema initialization may still occur. Review the reported counts and any failures; then repeat against the same empty target with --apply:

```powershell
go run ./cmd/import --media-root ../SSO-Paradise-Supply-Chain/storage/app --apply
```

The source is read through a consistent read-only transaction; applied target rows commit together. Source and target database names must differ. The source file tree is not a transactional snapshot: pause uploads and source writes for the final cutover. Code allocation resumes above the largest imported numeric hm code. Import has been tested only on synthetic Laravel-shaped data, not on the live source.

The import CLI interprets source database wall times using `--source-timezone` (default `Asia/Tehran`, matching the checked-in Laravel app configuration), and the target writes UTC instants. Use `--source-timezone UTC` only when the source actually stores Laravel times in UTC. This preserves creation/verification/reset/OAuth expiry instants instead of silently shifting them by the Tehran offset. The locally inspected WAMP source was empty; see local-wamp.md for that report.

## Signing keys and OAuth clients

For continued public-key validation, set OAUTH_PRIVATE_KEY_PATH to a securely provisioned copy of Laravel storage/oauth-private.key. Do not commit it. Set APP_KEY to the exact Laravel value, including base64: when present, and keep PUBLIC_URL aligned with the signed link's original origin. Imported token database records and the reused keys together preserve active OAuth credentials. Laravel URL signatures use the literal APP_KEY; Passport/cookies use its decoded 32-byte encryption key.

Create a new confidential client:

```powershell
go run ./cmd/manage client:create --name 'Connected app' --redirect-uri 'https://client.example/callback'
```

Add --public for a browser/native PKCE client. The confidential secret is printed once; only its bcrypt hash is stored. Multiple exact redirect URIs can be comma-separated. Imported client IDs remain stable.

## Cleanup and mail

```powershell
go run ./cmd/manage cleanup
go run ./cmd/manage cleanup --apply
```

Cleanup defaults to dry-run, and with --apply removes unverified accounts older than 24 hours and expired records. Foreign keys cascade associated records; Redis invalidation is per user. The worker schedules this at local midnight (APP_TIMEZONE, default Asia/Tehran); the command remains available for supervised one-off runs. MAIL_MAILER=smtp uses MAIL_HOST, MAIL_PORT, MAIL_USERNAME, MAIL_PASSWORD, MAIL_FROM_ADDRESS and MAIL_ENCRYPTION (tls/ssl/none; default tls). Default file delivery is safe for local testing. Multipart SMTP delivery was tested against a local fake server; external delivery remains a deployment check. Original reset/verification templates were ported from Blade with Persian/English text fallbacks. Run cmd/worker for queued SMTP.

## Legacy browser state and actual-environment validation

Optional file-session import:

```powershell
# APP_KEY must be the exact source value, kept private.
go run ./cmd/import --media-root ../SSO-Paradise-Supply-Chain/storage/app --sessions-root ../SSO-Paradise-Supply-Chain/storage/framework/sessions --session-lifetime 2h
```

Add --apply only after reviewing the dry-run report. Set LARAVEL_SESSION_COOKIE to the exact source name in the new API. LARAVEL_REMEMBER_COOKIE defaults to the Laravel web-guard cookie name when that bridge is enabled. Files are parsed as bounded primitive PHP arrays; no PHP objects/code are deserialized. Expired/guest/stale-password sessions are skipped and counted. Active sessions are restored to HttpOnly Go cookies; wallet callback flags are consumed once and password-confirmation expiry is preserved. Remember credentials require both the imported digest and current password-hash MAC; logout/password reset/change revoke the imported credentials too.

- The inspected Laravel configuration uses file sessions and AES-256-CBC cookies. Encrypted session-file payloads, Redis/database session drivers, custom guards, AES-GCM/previous-key cookies and custom Passport encryption callbacks require explicit adaptation if the actual deployment uses them. Unsupported serialized file data blocks import.
- Old reset records are accepted from password_reset_tokens (or the older password_resets table if the newer table is absent), with bcrypt verification, one-hour expiry and one-use consumption. A new reset request invalidates the imported old link. Reset now logs the browser in automatically after revoking old sessions.
- Password/device grants and Passport JSON client-management routes were not enabled by the inspected AuthServiceProvider. The custom legacy client model does not infer client_credentials; machine-to-machine/ownerless tokens are rejected for explicit review rather than silently converted. Numeric legacy client IDs are supported; UUID client IDs require a separate schema/contract change if present in actual data.
- Media is stored in MySQL instead of Spatie disk collections. Text-only profile edits keep existing documents; this extends the Laravel re-upload flow. REST/JSON forms replace Blade flashes and redirects; unknown fields are rejected. Company identifiers are deliberately limited to 32 characters; longer source values block import for review.
- Registration retains the 8–40 character rule; reset requires at least eight, change uses complexity/breach checks, and bcrypt's 72-byte maximum is enforced. Runtime breach lookup now follows Laravel's fail-open behavior on service failure and logs the failure.
- Normal runtime Go sessions refresh their idle deadline on activity using SESSION_TTL (default 2h), matching Laravel. Expired/revoked sessions remain invalid; older Go sessions keep their fixed deadlines after migration. Remember lifetime defaults to Laravel's 576000 minutes (400 days). These are explicit configurable expiry policies.
- MySQL owns shared rate windows/penalties, queue leases and daily cleanup. Redis is optional and database session/credential checks remain authoritative. Production/staging startup is enabled with strict configuration checks, not with generated keys or file-mail defaults.

[Production runbook](production.md) includes native/container deployment, proxy settings and cutover steps. Docker is unavailable here and container startup has not been tested. All source compatibility tests and import tests use synthetic data; actual client integration, SMTP deliverability and a backed-up live snapshot still need deployment validation. No live source data or traffic has been changed.
