# Local WAMP and CI configuration

The local Go `.env` is mapped from the sibling Laravel `.env`. WAMP's MySQL connection is retained; the source database is `sso`, and the dedicated target is `sso_go`. All runtime application data, media, OAuth state, queue records and rate limits are stored in MySQL. Redis is an optional cache, and is not required for local startup.

API, worker, import, migrate and manage commands automatically load `.env` from their working directory. `ENV_FILE` selects a different file. Existing environment variables take precedence, including explicitly empty values. Malformed or missing explicitly selected files stop startup without printing credentials.

Local settings use `APP_ENV=development` and `PUBLIC_URL=http://localhost:3000`. Laravel's local `MAIL_MAILER=log` is mapped to file delivery under `var/mail`, so registration/verification/reset messages can be inspected without sending production mail. The original local RSA key is read from the sibling Laravel `storage/oauth-private.key`; it is not committed or regenerated. The source APP_KEY and cookie name are retained. Normal sessions use the source 120-minute idle lifetime.

`../SSO-Paradise-Supply-Chain-Go-Front/.env.local` contains `API_ORIGIN=http://127.0.0.1:8080` and the public WalletConnect project ID from Laravel's JavaScript. All three populated env files (`.env`, `../SSO-Paradise-Supply-Chain-Go-Front/.env.local`, `deploy/production.env`) are Git-ignored. The production file is prepared from the supplied server configuration; it must not be loaded on the local workstation.

Start from the Go project root:

```powershell
go run ./cmd/migrate
go run ./cmd/api
# In a second terminal, from the same directory:
go run ./cmd/worker
```

The frontend is at `http://localhost:3000`; API database readiness is at `http://127.0.0.1:8080/readyz`. Background process IDs from the assisted local launch are recorded in `var/local-processes.json`; logs are in `var/*.log`.

The local source snapshot inspected during setup contained zero users, profiles, wallets and OAuth clients. Import completed without creating artificial account data; 17 guest/expired/stale session files were skipped. This does not describe production data. Register a local account to exercise the actual account flow, and open the generated verification/reset files under `var/mail`.

The backend CI workflow provisions MySQL 8.4 for Go tests. `MYSQL_HOST` is set explicitly so database tests execute instead of skipping. Go race tests, vet and runtime command builds run in backend CI against temporary databases with test-only credentials. The independent frontend repository owns browser jobs.

For automatic production pulls, keep the server's populated Go `.env` outside Git, or inject equivalent service/CI variables. Set `PUBLIC_URL` to the public frontend origin, the exact source APP_KEY, a readable Laravel RSA key path, target MySQL credentials and queued TLS SMTP settings. Do not copy `sso` into `MYSQL_DATABASE`: its Laravel tables are the import source. Deploy the frontend from its own repository. Restart API and worker after the checked backend build. Git pull alone does not migrate data, build binaries or restart processes; use the production runbook for the final cutover. The workflow performs checks and does not modify the production server.
