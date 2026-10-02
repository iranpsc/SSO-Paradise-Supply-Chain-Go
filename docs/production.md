# Production runbook

## Configuration

The API now accepts APP_ENV=production or staging. Startup rejects a non-HTTPS PUBLIC_URL, absent/invalid 32-byte APP_KEY, missing OAUTH_PRIVATE_KEY_PATH, plaintext SMTP, or synchronous production mail. It never creates production signing keys. Development still generates persistent local keys and can write notices to files.

Copy deploy/production.env.example to a private deploy/production.env and replace placeholders. Reuse the **literal Laravel APP_KEY** and a securely provisioned copy of Laravel's RSA private key for compatible existing tokens, signed links and cookies. Do not rotate those keys during cutover. Provision SMTP credentials, the public WalletConnect project ID, and the public HTTPS origin. A copied key should be readable only by the application OS user (container uid 10001). Never commit production.env or deploy/secrets.

Go binaries load .env from the working directory without replacing existing process variables. ENV_FILE selects an explicit configuration file; a missing explicit file fails startup. Container env_file and CI/service-manager variables take precedence over the local file. On Windows/Linux without containers, inject the same variables through the service manager and supervise all three processes:

```powershell
go build -o bin/api.exe ./cmd/api
go build -o bin/worker.exe ./cmd/worker
cd web
npm ci
npm run build
npm run start
```

Run bin/api.exe and bin/worker.exe from the Go project root in separate managed services. Set API_ORIGIN for Next **before building**, because rewrites are compiled into its build. Backend default listen is loopback 127.0.0.1:8080. Next standalone output is also available; include .next/static and public with .next/standalone when deploying it manually.

## Container files

Dockerfile builds the API, worker, migrate/import/manage commands. web/Dockerfile packages Next standalone. compose.yaml starts isolated MySQL/Redis, API, worker and web, with durable database storage and restart policies. Only web's port is exposed on host loopback; use your existing HTTPS reverse proxy. deploy/nginx.conf.example overwrites forwarded IP headers and permits the validated image-upload size. Adapt its domain/certificate paths to your server.

Docker is unavailable in the current workspace; image builds/Compose startup have **not** been verified here. Native Go and Next builds and browser/database tests are the verified path. Validate these container definitions on the deployment host before using them.

```powershell
docker compose --env-file deploy/production.env config --quiet
docker compose --env-file deploy/production.env build
docker compose --env-file deploy/production.env up -d mysql redis
```

Then perform schema/import/cutover below before starting the API, which lazily creates its personal-access client. Compose uses subnet 172.30.42.0/24; the example trusted-proxy range assumes that private network and the supplied proxy header policy. If the subnet conflicts on your host, change both the Compose network and TRUSTED_PROXY_CIDRS. Without a correctly configured trusted proxy, unauthenticated proxied users share the peer's rate budget.

## Cutover

1. Back up the Laravel database, media and session files and the Go target. Stop Laravel writes, uploads and session-file writes for the final snapshot. Keep the previous deployment and its keys for rollback.
2. Create/migrate a separate empty Go target with go run ./cmd/migrate. Do not point this at the Laravel database. Source database privileges should be read-only.
3. Run the import dry run with LARAVEL_MYSQL_DSN and the source media root. If retaining browser sessions, also provide --sessions-root, the exact APP_KEY, and source --session-lifetime (default 2h). Review counts, rejected values and skipped expired/guest/stale-password sessions.
4. Apply using the same flags plus --apply. The target data commits atomically; source tables/files are never changed. Source signing/cookie keys and original public origin must match.
5. Start API, worker and the built web server; switch the HTTPS reverse proxy when the checks below succeed. In Compose: docker compose --env-file deploy/production.env up -d api worker web.
6. Exercise registration/verification, real SMTP delivery, login/logout, wallet signing and authorization/refresh/revocation with each actual client. Confirm the client still validates JWTs using the same RSA public key. Check /readyz directly on the private API and worker logs.

For import/manage inside the container, run one-off API-image commands with the required read-only source filesystem mounts and source DSN. Compose's default MYSQL_HOST=mysql is the **target**; configure LARAVEL_MYSQL_DSN to reach the separate source. Do not copy secrets into a built image.

## Mail, cleanup and monitoring

SMTP notices are durably queued in MySQL. Keep cmd/worker running; queued acceptance is not confirmed external delivery. Workers claim separate leases, retry with exponential backoff (maximum ten attempts), and never send notices after their one-hour deadline. A crash after SMTP accepts a message but before acknowledgment can cause duplicate delivery; this is at-least-once delivery.

The worker applies expired/unverified cleanup on startup and subsequently at midnight in APP_TIMEZONE (default Asia/Tehran), matching Laravel's daily schedule. --cleanup-every overrides that schedule; --once supports a supervised one-off/cron invocation. Cleanup invalidates only affected users in Redis. It never flushes the entire cache.

Runtime uses a 60/minute global budget keyed by authenticated user (otherwise peer IP) (RATE_LIMIT_PER_MINUTE), a separate 10/minute Web3 budget and 6/minute verification budget, following the inspected Laravel limits. Login locks an identifier/IP pair after five failed credentials within one minute; successful login clears its failure counter. MySQL stores these shared windows and the additional escalating request penalty history. This persists across API restarts and instances; database failure does not silently bypass throttling. /readyz bypasses this budget and checks database reachability; /healthz is the existing lightweight route.

Monitor mail_outbox state counts, worker delivery/cleanup errors, 429s and database capacity. Sent/dead mail contains account URLs: restrict database access and establish a retention policy. Run backups for MySQL; media is stored there as MEDIUMBLOB. Tests used a local fake SMTP endpoint; external deliverability, certificates and real client behavior require deployment-host validation.

## Supported legacy state

Existing numeric-ID Laravel clients, RSA access tokens, Defuse V2 encrypted refresh tokens and authorization codes, bcrypt password-reset records, default unencrypted PHP **file sessions**, and AES-256-CBC encrypted session/remember cookies are supported. Enable LARAVEL_SESSION_COOKIE with the exact source cookie name; LARAVEL_REMEMBER_COOKIE defaults to the framework's web-guard name when the session bridge is enabled. Imported source sessions/remember records are revoked together with Go sessions on logout/password reset/change.

Source guest/expired/stale-password sessions are skipped and counted. PHP objects/references, encrypted session-file payloads, non-file session drivers, custom guards, AES-GCM cookie encryption, previous-key rotation and custom Passport encryption callbacks are not inferred automatically. They require explicit adaptation if present in the actual source deployment; the importer rejects unsupported file data rather than interpreting it unsafely. Ordinary application behavior is supported independently of importing old sessions.

JWT lifetimes now follow Passport's default calendar year (P1Y) for access, refresh and personal-access tokens. OAUTH_ACCESS_TTL/OAUTH_REFRESH_TTL/OAUTH_PERSONAL_TTL accept positive Go durations to override; blank keeps calendar-year semantics. Normal Go sessions default to a 2h idle window; REMEMBER_SESSION_TTL defaults to Laravel's 576000-minute recaller duration (400 days). SESSION_TTL can be adjusted for the deployment. Activity extends live normal sessions; expired or revoked sessions are never recreated. Remember credentials keep a fixed lifetime. Existing Go sessions retain their previous fixed deadline after upgrade.

This runbook prepares deployment but does not certify an untouched live environment as bug-free. Real data/import, actual SMTP and connected clients must be verified before switching traffic.
