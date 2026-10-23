# Paradise Supply Chain SSO — Go Backend

Go-based identity and single sign-on service, designed to interoperate with the sibling Laravel SSO and its Passport conventions. The Persian RTL user interface is maintained separately in [SSO-Paradise-Supply-Chain-Go-Front](https://github.com/iranpsc/SSO-Paradise-Supply-Chain-Go-Front). No Node.js installation or frontend checkout is needed to build, test or deploy this backend.

> **Security:** This is a public source repository, not a place to publish credentials, production configuration, customer records, private signing keys, or vulnerability details. See [SECURITY.md](SECURITY.md).

## Contents

- [Capabilities](#capabilities)
- [Architecture and repository layout](#architecture-and-repository-layout)
- [Requirements](#requirements)
- [Local setup](#local-setup)
- [Configuration and secrets](#configuration-and-secrets)
- [Testing and quality checks](#testing-and-quality-checks)
- [Laravel migration](#laravel-migration)
- [Deployment and operations](#deployment-and-operations)
- [Documentation](#documentation)
- [Contributing and security](#contributing-and-security)

## Capabilities

- Registration; email/username login; remember-me; password confirmation, reset, and change; signed email verification; account and profile management; private avatars and documents.
- MetaMask/WalletConnect sign-in and wallet linking.
- OAuth authorization-code flow with PKCE, consent handling, RS256 JWT interoperability, refresh-token rotation, and revocation. The password grant is disabled. Only confidential first-party clients bypass consent under the project's custom client policy.
- MySQL-backed credentials, sessions, expiration, and revocation, with optional Redis caching.
- Laravel compatibility and controlled import of supported Passport, password-reset, session, and remember-cookie records; consult the migration guide for exact limitations.
- File-based development mail or SMTP with a durable worker and retries.

These describe the repository's documented implementation, not a claim of third-party certification or a guarantee of deployment security.

## Architecture and repository layout

| Path | Responsibility |
| --- | --- |
| `cmd/api` | HTTP API runtime and dependency wiring |
| `cmd/migrate` | Versioned schema initialization and upgrades |
| `cmd/import` | Controlled Laravel snapshot/local-media import |
| `cmd/manage` | OAuth client management and maintenance commands |
| `cmd/worker` | Queued SMTP delivery, retries, and scheduled cleanup |
| `cmd/seed-local` | Local synthetic demonstration data; never use against production |
| `internal/domain`, `internal/application` | Business rules and application interfaces |
| `internal/infrastructure` | MySQL, optional Redis, cryptography, mail, integrations |
| `internal/transport/httpapi` | HTTP contracts, cookies, throttling, and transport |
| `docs/` | Operational, database, parity, and migration references |
| `deploy/`, `compose*.yaml`, `Dockerfile` | Deployment assets; review before using in an environment |

The Go backend owns authentication, token issuance, revocation, persistence, and server-side policy. The separate Next.js frontend owns the Persian RTL interface and browser flows. MySQL remains authoritative for session expiration, revocation, and credentials even with Redis enabled. Browser tokens must not be stored in `localStorage`.

## Requirements

- Go: use the version specified in [`go.mod`](go.mod).
- MySQL 8.0.13+ or 9.x; use a dedicated database for this service.
- Redis is optional.
- The independent frontend has its own Node.js requirements and browser test suite; see its repository for setup.
- An SMTP service is required when configuring SMTP delivery; development can use file mail.

## Local setup

Clone the repositories as siblings:

```bash
git clone https://github.com/iranpsc/SSO-Paradise-Supply-Chain-Go.git
git clone https://github.com/iranpsc/SSO-Paradise-Supply-Chain-Go-Front.git
cd SSO-Paradise-Supply-Chain-Go
```

Copy `.env.example` to `.env` and configure **local-only** MySQL credentials, a dedicated database, and the appropriate browser origin. Existing process/CI variables take precedence over `.env`; `ENV_FILE` selects an explicit configuration file and fails if it is unavailable. Check [`docs/local-wamp.md`](docs/local-wamp.md) for the documented local setup.

PowerShell example (after configuring the remaining values in `.env`):

```powershell
Copy-Item .env.example .env
$env:MYSQL_HOST = '127.0.0.1'
$env:MYSQL_DATABASE = 'paradise_go'
go run ./cmd/migrate
go run ./cmd/api
```

POSIX shell equivalent:

```bash
cp .env.example .env
export MYSQL_HOST=127.0.0.1 MYSQL_DATABASE=paradise_go
go run ./cmd/migrate
go run ./cmd/api
```

Set up and deploy the frontend separately using its repository's instructions.

The documented development defaults are `http://localhost:3000/login` for the frontend and `http://127.0.0.1:8080/healthz` for the API. `PUBLIC_URL` must match the browser origin; use the examples in both repositories as references and resolve any conflicting sample values before testing cookies and callbacks. Configure WalletConnect in the frontend environment when needed.

For local development, mail is written to `var/mail` by default. If keys are not supplied, development generates persistent `var/app.key` and `var/oauth-private.key`. Preserve them between restarts, never commit them, and never rely on automatically generated development keys in production.

## Configuration and secrets

Use [`.env.example`](.env.example) as the authoritative list of supported settings; do not copy real values into documentation or PRs. Configure distinct database identities and signing material for local, test, staging, and production environments. Limit database permissions, protect backups, and provide production secrets through the deployment platform's secret-management mechanism. Cookie-based browser mutations require the configured origin.

Production/staging startup validates HTTPS by default (HTTP requires explicit `ALLOW_INSECURE_HTTP=true`), stable keys, and TLS SMTP. `PUBLIC_URL` must match the browser-facing frontend origin. When `MAIL_MAILER=smtp`, run the worker alongside the API:

```bash
go run ./cmd/worker
```

Do not log passwords, session cookies, bearer tokens, refresh tokens, private keys, wallet signatures, personal documents, or full connection strings.

## Testing and quality checks

Use an isolated, disposable MySQL instance and **verify its host/database before running integration or browser tests**. The documented Go tests create random `paradise_test_*` databases; browser tests use a dedicated `paradise_e2e_*` database with synthetic mail and keys. Never point test commands at a production database.

```bash
go test ./...
go vet ./...
go build ./cmd/api
```

Frontend/browser validation is maintained in the [independent frontend repository](https://github.com/iranpsc/SSO-Paradise-Supply-Chain-Go-Front); follow its setup and Playwright instructions. Existing documented coverage includes migration/import preservation, concurrent nonce use, PKCE/refresh replay, JWT and Laravel compatibility, cookie interoperability, revocation, password freshness, signed verification, mail queue retries, throttling, and account/wallet browser flows. Add regression tests for every changed security-sensitive behavior. A passing test suite does not replace a security review.

## Laravel migration

Read [`docs/migration.md`](docs/migration.md) **before** importing. The documented import expects an empty target database and leaves the Laravel source read-only. Perform a dry run, verify backups and compatibility limits, rehearse rollback, and explicitly authorize any production migration. Never upload production dumps, customer data, old sessions, or signing keys to issues or PRs.

## Deployment and operations

Deployment assets include `Dockerfile`, `compose.yaml`, `docker-compose.yml`, `compose.dokploy.yaml`, and `deploy/`. They are alternatives/configurations to inspect, not instructions to run every Compose file together. All Compose files here deploy backend services only. Use `docker-compose.yml` for an independent Dokploy backend with external MySQL/Redis, or `compose.yaml` for a backend stack with managed MySQL/Redis. Its optional `compose.dokploy.yaml` override exposes the API on port 8080 to Dokploy. Build and deploy the frontend from its own repository. Use [`docs/production.md`](docs/production.md) as the deployment runbook. Ensure that API and worker processes receive the required configuration, migrations are controlled, the selected HTTPS or explicitly enabled HTTP mode matches the proxy configuration, SMTP/TLS is configured, persistent keys are securely provisioned, and health checks, backups, log redaction, monitoring, and rollback are addressed. Test the deployment in staging before production.

## Documentation

- [Production runbook](docs/production.md)
- [Database schema dictionary](docs/database.md)
- [Laravel migration and compatibility](docs/migration.md)
- [Feature parity audit](docs/feature-parity.md)
- [Local WAMP and CI setup](docs/local-wamp.md)
- [Local backend seed data](docs/local-test-data.md)
- [Postman collection](postman_collection.json) — inspect its environment/credentials before use
- [Frontend repository](https://github.com/iranpsc/SSO-Paradise-Supply-Chain-Go-Front)

## Contributing and security

Read [CONTRIBUTING.md](CONTRIBUTING.md) for branch, test, review, and PR requirements. Report suspected vulnerabilities **privately**, following [SECURITY.md](SECURITY.md); do not open a public security issue. Public repository visibility does not imply an open-source license: check for a separately published license before assuming reuse rights.
