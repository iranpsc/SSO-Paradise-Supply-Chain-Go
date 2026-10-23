# SSO Paradise Supply Chain — Go backend

Go implementation of the sibling Laravel SSO, with MySQL persistence and optional Redis caching. The Persian RTL Next.js interface lives in the independent [frontend repository](https://github.com/iranpsc/SSO-Paradise-Supply-Chain-Go-Front) and is built and deployed separately. No Node.js installation or frontend checkout is needed to build, test or deploy this backend.

Implemented flows include registration and email/username login, remember-me, password confirmation/reset/change, Laravel-compatible signed email verification, account/profile editing, private documents and avatars, MetaMask/WalletConnect login and wallet linking, OAuth authorization code/PKCE, rotating refresh tokens, revocation, registration callbacks and unverified-account cleanup.

OAuth follows the custom Laravel client policy: only confidential first-party clients skip consent; other clients use the /authorize approval screen. The OAuth password grant is disabled, matching upstream. RS256 JWTs were tested against the actual sibling Passport dependencies in both directions. Signed verification URLs were checked using Laravel's actual URL validator. Import now supports Passport access/refresh/code records, old password-reset links, default PHP file sessions and Laravel session/remember cookies; see [migration details](docs/migration.md) for exact supported formats.

## Run locally

Requirements: Go (version in go.mod), MySQL 8.0.13+ or 9.x; Redis is optional. Integration tests have been run against local MySQL 9.1. Go automatically loads .env from the working directory. Existing process/CI variables take precedence; ENV_FILE selects an explicit file and fails if it is unavailable. Use [.env.example](.env.example) as the backend configuration reference.

From this project root, select a **separate Go database**, then create/migrate it:

```powershell
$env:MYSQL_HOST = '127.0.0.1'
$env:MYSQL_DATABASE = 'paradise_go'
go run ./cmd/migrate
go run ./cmd/api
```

API health: http://127.0.0.1:8080/healthz; database readiness: http://127.0.0.1:8080/readyz. PUBLIC_URL must match the browser-facing origin of the independently deployed frontend. Follow the frontend repository's instructions for its setup.

Development email is written to var/mail. MAIL_MAILER=smtp selects SMTP. Without supplied keys, development creates persistent var/app.key and var/oauth-private.key. Keep these stable between restarts. Production/staging startup validates HTTPS (unless explicitly enabled with ALLOW_INSECURE_HTTP=true), stable keys and TLS SMTP. SMTP is durably queued by default; run `go run ./cmd/worker` alongside the API. See the [production runbook](docs/production.md).

## Database and import

[Schema dictionary](docs/database.md) describes atomic fields, exact lengths, foreign keys and indexes. Migration is versioned and does not drop databases. Legacy Go profile JSON is backfilled into columns and retained only as a read-only archive.

[Feature audit](docs/feature-parity.md) maps the inspected Laravel features to implementation/tests. [Migration guide](docs/migration.md) covers dry-run/apply import from Laravel, client creation, keys and compatibility limits. Import requires an empty target and leaves the source read-only. No live Laravel data has been changed as part of this implementation.

## Validation

```powershell
$env:MYSQL_HOST = '127.0.0.1'
go test ./...
go vet ./...
go build -o bin/sso-paradise-supply-chain.exe ./cmd/api
```

Go database tests use random disposable paradise_test_* databases. Tests cover migration/import preservation, concurrent nonce consumption, OAuth/PKCE/refresh replay, JWT/Defuse/cookie interoperability against actual Laravel dependencies, cache revocation/password freshness, signed verification, queue leases/retries and shared throttling. The independent frontend repository owns browser tests, including account/wallet flows and responsive layout checks.

## Layout

- cmd/api: dependency wiring and API runtime.
- cmd/migrate: safe versioned schema initialization/upgrade.
- cmd/import: Laravel snapshot and local-media import.
- cmd/manage: OAuth client creation and account/expiry cleanup.
- cmd/worker: durable SMTP delivery/retries and daily scheduled cleanup.
- internal/domain, application: business rules and interfaces.
- internal/infrastructure: MySQL, optional Redis, crypto, SMTP/file mail and Metarang.
- internal/transport/httpapi: HTTP contracts, cookies and rate limits.
- Dockerfile and Compose files: backend binaries, optional managed MySQL/Redis and API proxy configuration; no frontend service.

MySQL is authoritative for session expiry, revocation and credentials even when Redis is available. Cookie-based browser mutations require the configured origin. Tokens are not persisted in browser localStorage.

[Local WAMP and CI setup](docs/local-wamp.md) documents dotenv precedence, the separate `sso_go` target, local mail files and production configuration during automatic pulls.

[Local backend test data](docs/local-test-data.md) explains `cmd/seed-local`, the five demo accounts and five importable wallet scenarios. Populated credentials are stored only under Git-ignored `var/test-data`.
