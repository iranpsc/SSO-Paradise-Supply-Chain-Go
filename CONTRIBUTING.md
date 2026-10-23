# Contributing to Paradise Supply Chain SSO (Go)

Thank you for helping improve the project. Authentication infrastructure has an unusually high security and compatibility impact: keep changes small, reviewable, tested, and documented. Please read [README.md](README.md) and [SECURITY.md](SECURITY.md) first.

## Before contributing

Search existing issues and pull requests; discuss large or breaking changes before implementation. Do not submit real user data, production dumps, credentials, wallet secrets, private signing keys, access tokens, session cookies, or unredacted logs. If you discover a vulnerability, use the private reporting process in [SECURITY.md](SECURITY.md), **not** a public issue or PR.

## Branch and collaboration policy

Create a branch from the current `main`; do not develop directly on `main`. Branch names are flat Git refs using slashes for readability, not actual nested branches:

- `feature/oauth/pkce-validation`
- `feature/auth/session-management`
- `feature/wallet/linking`
- `fix/oauth/refresh-replay`
- `test/migration/compatibility`
- `docs/security/reporting`
- `refactor/infrastructure/mysql`

Use one coherent purpose per branch and PR. Never force-push shared branches or `main`; do not rewrite other contributors' history. Maintainers should configure GitHub branch protection/rulesets to require pull requests, successful checks, resolved conversations, and appropriate review before merging. These are contribution expectations, not a claim that repository settings are already enabled.

## Local development

Follow the [README local setup](README.md#local-setup) and the documented `.env.example`. Use separate databases for development, integration tests, browser tests, and any Laravel import rehearsal. Do not run destructive tests against shared or production infrastructure. Preserve development signing keys between restarts; keep all generated credentials and local mail files untracked.

Keep the architecture boundaries intact: `domain` for business rules, `application` for use cases/interfaces, `infrastructure` for external adapters, `transport/httpapi` for HTTP-specific concerns, and `cmd` for entrypoints and wiring. Avoid introducing frontend-specific behavior into core identity policy; coordinate cross-repository contract changes with the sibling frontend.

## Coding expectations

- Follow idiomatic Go and format modified code with `gofmt`; run `go vet ./...`.
- Prefer explicit error handling, context propagation, bounded timeouts, and narrow interfaces; do not suppress errors silently.
- Validate input and authorization at trust boundaries; enforce origin/cookie policy for browser mutations.
- Keep MySQL authoritative for credentials, revocation, and session expiry; do not make cache availability a security bypass.
- Do not weaken PKCE, consent, redirect URI validation, token rotation/replay detection, JWT verification, wallet nonce consumption, or rate limiting for convenience.
- Do not introduce the OAuth password grant without a separately approved architectural/security decision.
- Do not expose secrets or sensitive personal information in logs, traces, error messages, test snapshots, fixtures, or screenshots.
- Make migrations versioned and reversible where feasible; never silently drop production data. Document compatibility and rollback impacts.
- Keep dependencies and documentation aligned with actual code. Avoid unrelated formatting or broad refactors in a focused PR.

## Tests and verification

Run the applicable checks before opening a PR:

```bash
gofmt -w <changed-go-files>
go test ./...
go vet ./...
go build ./cmd/api
```

The `gofmt` placeholder means your actual changed Go files; do not paste the angle-bracket placeholder into a shell. Tests require the documented isolated MySQL setup. For changes affecting HTTP/browser flows, also run the sibling frontend's typecheck, build, and Playwright tests in its dedicated test environment. Include positive, negative, expiry, concurrency, and replay cases as applicable. For migrations, test fresh setup and upgrade/import preservation. For mail/worker changes, cover retries, idempotency, and failure behavior. Record commands executed and any limitations in the PR.

## Commits and pull requests

Use descriptive Conventional Commit-style messages, e.g. `fix(oauth): reject reused refresh tokens`, `test(auth): cover expired verification links`, or `docs: clarify import prerequisites`. Keep commits focused. A PR should explain the motivation, scope, linked issue (if any), API/schema/environment changes, tests and results, security implications, compatibility with Laravel and the frontend, and deployment/rollback notes. Provide sanitized examples only.

Suggested PR checklist:

- [ ] Branch is based on current `main`; no unrelated changes.
- [ ] `gofmt`, Go tests, vet, and build pass (or limitations are explained).
- [ ] Security-sensitive paths have regression tests and review notes.
- [ ] Schema, API, environment, and frontend compatibility changes are documented.
- [ ] No credentials, personal data, generated keys, database dumps, or sensitive logs are included.
- [ ] Migration/deployment and rollback implications are described where relevant.
- [ ] Documentation and examples match the implemented behavior.

## Review and merging

Be constructive and specific. Authors should address review comments and rerun relevant checks after changes. Security-sensitive changes (authentication, OAuth, cryptography, sessions, wallet signatures, data import, secrets, and authorization) require especially careful maintainer review. Do not merge with failing required checks or unresolved blocking feedback. Coordinate changes to shared contracts with the frontend repository. Maintainers determine final merge and release decisions.

## Reporting bugs versus vulnerabilities

Ordinary bugs and documentation issues may use public GitHub Issues with synthetic reproduction data. Suspected security weaknesses must be reported privately as explained in [SECURITY.md](SECURITY.md). Do not probe deployed services, access other users' accounts, or publish exploit details without authorization.

## Licensing

Public visibility alone does not grant reuse or redistribution rights. Consult the repository's published license, if any, or contact the maintainers before assuming permissions.
