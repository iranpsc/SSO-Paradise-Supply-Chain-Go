# SSO Paradise Supply Chain — Go / Next.js

Go implementation of the sibling Laravel SSO, with a Persian RTL Next.js interface, MySQL persistence and optional Redis caching.

Implemented flows include registration and email/username login, remember-me, password confirmation/reset/change, Laravel-compatible signed email verification, account/profile editing, private documents and avatars, MetaMask/WalletConnect login and wallet linking, OAuth authorization code/PKCE, rotating refresh tokens, revocation, registration callbacks and unverified-account cleanup.

OAuth follows the custom Laravel client policy: authenticated authorization requests are approved automatically. RS256 JWTs were tested against the actual sibling Passport dependencies in both directions. Signed verification URLs were checked using Laravel's actual URL validator. Import now supports Passport access/refresh/code records, old password-reset links, default PHP file sessions and Laravel session/remember cookies; see [migration details](docs/migration.md) for exact supported formats.

## Run locally

Requirements: Go (version in go.mod), Node.js 22.12+, MySQL 8.0.13+ or 9.x; Redis is optional. Integration tests have been run against local MySQL 9.1. Go automatically loads .env from the working directory. Existing process/CI variables take precedence; ENV_FILE selects an explicit file and fails if it is unavailable. Use [.env.example](.env.example) and [web/.env.example](web/.env.example) as configuration references.

From this project root, select a **separate Go database**, then create/migrate it:

```powershell
$env:MYSQL_HOST = '127.0.0.1'
$env:MYSQL_DATABASE = 'paradise_go'
go run ./cmd/migrate
go run ./cmd/api
```

In a second terminal:

```powershell
cd web
npm ci
npm run dev
```

Open http://localhost:3000/login. API health: http://127.0.0.1:8080/healthz. PUBLIC_URL must match the browser origin. Copy web/.env.example to web/.env.local for WalletConnect configuration.

Development email is written to var/mail. MAIL_MAILER=smtp selects SMTP. Without supplied keys, development creates persistent var/app.key and var/oauth-private.key. Keep these stable between restarts. Production/staging startup validates HTTPS, stable keys and TLS SMTP. SMTP is durably queued by default; run `go run ./cmd/worker` alongside the API. See the [production runbook](docs/production.md).

## Database and import

[Schema dictionary](docs/database.md) describes atomic fields, exact lengths, foreign keys and indexes. Migration is versioned and does not drop databases. Legacy Go profile JSON is backfilled into columns and retained only as a read-only archive.

[Feature audit](docs/feature-parity.md) maps the inspected Laravel features to implementation/tests. [Migration guide](docs/migration.md) covers dry-run/apply import from Laravel, client creation, keys and compatibility limits. Import requires an empty target and leaves the source read-only. No live Laravel data has been changed as part of this implementation.

## Validation

```powershell
$env:MYSQL_HOST = '127.0.0.1'
go test ./...
go vet ./...
go build -o bin/sso-paradise-supply-chain.exe ./cmd/api
cd web
npm run typecheck
npm run build
$env:PLAYWRIGHT_CHANNEL = 'chrome'
npm test
```

Go database tests use random disposable paradise_test_* databases. Browser tests create a dedicated paradise_e2e_* database and synthetic mail/key files; teardown deletes only that test database. Tests cover migration/import preservation, concurrent nonce consumption, OAuth/PKCE/refresh replay, JWT/Defuse/cookie interoperability against actual Laravel dependencies, cache revocation/password freshness, signed verification, queue leases/retries, shared throttling and account/wallet browser flows. Browser layout checks cover 390, 768 and 1440 pixels.

## Layout

- cmd/api: dependency wiring and API runtime.
- cmd/migrate: safe versioned schema initialization/upgrade.
- cmd/import: Laravel snapshot and local-media import.
- cmd/manage: OAuth client creation and account/expiry cleanup.
- cmd/worker: durable SMTP delivery/retries and daily scheduled cleanup.
- internal/domain, application: business rules and interfaces.
- internal/infrastructure: MySQL, optional Redis, crypto, SMTP/file mail and Metarang.
- internal/transport/httpapi: HTTP contracts, cookies and rate limits.
- web/src: Next.js pages and Persian forms, wallet providers and same-origin API rewrites.

MySQL is authoritative for session expiry, revocation and credentials even when Redis is available. Cookie-based browser mutations require the configured origin. Tokens are not persisted in browser localStorage.

[Local WAMP and CI setup](docs/local-wamp.md) documents dotenv precedence, the separate `sso_go` target, local mail files and production configuration during automatic pulls.

[Frontend test data](docs/frontend-test-data.md) explains `cmd/seed-local`, the five demo accounts and five importable wallet scenarios. Populated credentials are stored only under Git-ignored `var/test-data`.
