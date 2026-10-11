# Security Policy

Paradise Supply Chain SSO is authentication infrastructure. Security reports are welcome and should be handled confidentially to protect users and deployments. This policy applies to the public Go backend repository; the [separate frontend repository](https://github.com/iranpsc/SSO-Paradise-Supply-Chain-Go-Front) has its own code and may require a coordinated fix.

## Supported versions

Security maintenance is focused on the current `main` branch and any release explicitly designated as supported by the maintainers. No long-term-support schedule, fixed response-time SLA, or blanket support for older commits is implied. Check repository releases and maintainer announcements for current status.

## Report a vulnerability privately

**Do not open a public issue, discussion, pull request, or social-media post containing an undisclosed vulnerability.** First check the repository's [Security advisories](https://github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/security/advisories). If private vulnerability reporting is enabled, use **Report a vulnerability** there. If that option is unavailable, contact the repository owner through an established private channel and request a secure reporting route; do not include exploit details in a public message. This document does not invent an unverified security email address.

Please include, as safely as possible: affected commit/version and component; concise impact and prerequisites; minimal reproduction using synthetic data; expected versus observed behavior; any relevant sanitized logs; and whether the issue also affects the sibling frontend or Laravel interoperability. Do **not** send production secrets, customer records, private keys, live access/refresh tokens, session cookies, or raw database exports. Coordinate secure transfer with maintainers if sensitive evidence is genuinely necessary.

Maintainers should acknowledge and triage privately, assess impact, coordinate a fix and validation, and agree on responsible disclosure. No guaranteed acknowledgment or remediation deadline is promised. Public disclosure should be coordinated after a fix or mitigation is available, taking affected users into account.

## Particularly sensitive areas

Reports involving the following are especially relevant:

- Authentication and account recovery: login, registration, password reset/change, signed email verification, remember-me, and account enumeration.
- OAuth2 authorization-code/PKCE, consent/client policy, redirect URI validation, RS256 signing/verification, token rotation, replay, and revocation.
- Session/cookie security: origin checks, CSRF, expiry, cache consistency, secure attributes, and Laravel session/remember-cookie compatibility.
- Wallet authentication and linking: nonce uniqueness and atomic consumption, signature validation, replay, and account binding.
- Authorization and private assets: profile/document access, cross-account data exposure, and privilege boundaries.
- Data migration: read-only source handling, import integrity, legacy credential/session compatibility, and sensitive data exposure.
- Infrastructure: MySQL/Redis trust boundaries, SMTP queues and workers, throttling, deployment configuration, key handling, and supply-chain vulnerabilities.

The list describes review priorities, not an assertion that a vulnerability exists.

## Safe research rules

Use only systems and accounts you own or have explicit permission to test. Prefer a local isolated deployment with synthetic data. Do not perform denial-of-service, spam, phishing, social engineering, credential stuffing, destructive tests, data exfiltration, or access to other users' information. Stop and report privately if you unexpectedly encounter sensitive data. Avoid automated scanning of production infrastructure without written authorization. Do not disclose an unpatched issue publicly before coordinating with maintainers.

## Security requirements for contributors and operators

- Never commit `.env`, production dumps, private signing/application keys, session material, wallet secrets, SMTP credentials, or personal documents. Treat exposed secrets as compromised: revoke/rotate them and investigate impact; deleting a commit alone is insufficient.
- Use HTTPS and appropriately scoped, stable, securely stored production signing keys. Production/staging configuration validates HTTPS, stable keys, and TLS SMTP; review [`docs/production.md`](docs/production.md).
- Keep MySQL authoritative for session expiration, credential state, and revocation; optional Redis must not bypass security decisions.
- Preserve PKCE, redirect validation, client consent policy, refresh-token rotation/replay detection, nonce atomicity, and origin/cookie protections.
- Keep browser tokens out of `localStorage`; redact credentials and identifiers from logs and telemetry.
- Use separate least-privilege databases and secrets per environment. Never run disposable test or import procedures against production. Back up and rehearse migrations before deployment.
- Keep Go modules, images, and deployment dependencies updated and review security-sensitive dependency changes. Verify third-party security alerts against the actual affected versions and usage.
- Coordinate cross-repository issues with the [frontend](https://github.com/iranpsc/SSO-Paradise-Supply-Chain-Go-Front), and Laravel compatibility changes with the migration documentation.

## Public repository configuration (maintainer checklist)

Maintainers should enable GitHub private vulnerability reporting, configure branch protection and required reviews/checks, enable appropriate dependency/security scanning and secret protection where available, restrict deployment secrets and permissions, and establish an internal incident-response contact and key-rotation procedure. These are recommendations, **not** claims that settings are currently enabled.

For non-security bugs, use regular GitHub Issues with sanitized reproduction steps. For contribution rules, see [CONTRIBUTING.md](CONTRIBUTING.md).
