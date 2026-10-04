# Source feature coverage

This audit follows the checked-in Laravel application/routes/providers and their enabled Passport routes. It distinguishes implemented behavior from deployment-specific configuration. Disabled package APIs are not counted as application features.

| Laravel source | Go implementation | Validation |
|---|---|---|
| routes/api.php: login/logout/me/user/users | httpapi server.go + application/auth/oauth + public profile | HTTP contract, JWT/revocation, public names/avatar tests |
| Auth Login/Register + remember | auth.go, signed verification, callback store, first-party cookies | Go auth/HTTP tests + account browser flow |
| Forgot/Reset/NewPassword | Separate password policies, old-link import, automatic reset login, breach check | Replay/revocation/password-cache tests + browser reset |
| ConfirmPassword | Session-bound three-hour confirmation | HTTP/session attribute tests |
| VerificationController + custom signed URL middleware | Native signed URL, same-account guard, back_url single use | Actual Laravel URL validator + HTTP verification tests |
| AccountController | Display/edit identity, email reverification, avatars | Account and image HTTP/browser tests |
| PersonalInfoController | Atomic personal/company fields and owner-only document uploads/reads | Profile validation/storage/private-media tests |
| Web3AuthController + MetarangWalletClient | Separate login/link nonces, signature recovery, ownership, Metarang lookup | Known PHP signature vectors, nonce race, wallet browser flow |
| AppendWalletLoginToOAuthCallback | One-use wallet_login flag on successful callback | HTTP callback consumption/replay tests |
| App custom Passport Client | Automatic authorization; exact redirects, legacy confidential clients | OAuth PKCE/redirect/replay tests |
| Passport enabled authorize/token/deny/transient refresh | GET/POST/DELETE authorize, token, browser refresh; extra revoke endpoint | OAuth HTTP/application tests |
| Passport persisted access/refresh/auth codes | Snapshot import + reused RSA key + Defuse V2 decoding | Actual Passport JWT/Defuse interoperability; imported token/code tests |
| Custom verify/reset notifications | Ported Blade layouts/copy and fa/en translations; MIME HTML/text | Escaping/RTL template tests; fake local SMTP test |
| DeleteUnverifiedUsers + bootstrap scheduler | Dry-run/apply cleanup; worker startup + local midnight schedule | Cleanup/cascade and Tehran midnight tests |
| Laravel file sessions/remember cookies | Primitive file import; AES-CBC/MAC/name-bound cookie bridge | Actual Laravel Encrypter + logout/reset/restore-race tests |
| MySQL/Redis runtime | Versioned safe migration, atomic columns, explicit indexes/FKs, authoritative auth checks | Legacy/fresh/resume/import preservation and cache tests |
| API/Web3/verification and failed-login throttles | Shared user/IP budgets; five failed logins per identifier/IP minute | Concurrent quota, failure-window/isolation/reset tests; account browser flow without quota bypass |
| Production operations | Strict config, durable mail queue, shared rate state, graceful shutdown, worker, deployment files | Config/lease/concurrent-rate tests; Go/Next native builds |

The inspected AppServiceProvider enables password grants; the installed Passport package enables device routes by default. Password grant and refresh are implemented, and imported client flags retain their original grant restrictions. The legacy client model advertises personal_access, password/refresh_token or authorization_code/refresh_token; it advertises neither device_code nor client_credentials. No device view or source device-code migration is configured. Device routes preserve the resulting errors/redirects; they do not provision a new successful device flow. registerJsonApiRoutes remains disabled. Ownerless machine-token imports require explicit review and are rejected. User-facing client/token management endpoints are therefore not missing application features. New clients can be provisioned with cmd/manage client:create.

Intentional Go extensions: username login, profile scope, REST/JSON frontend support routes, BLOB media storage, retaining existing documents for text-only edits, stable member codes and shared persistent throttling. Normal runtime browser sessions refresh their idle deadline on activity, matching Laravel; fixed credentials and migrated older Go sessions retain their expiry. Current password/image validation and compact public resource semantics remain documented in migration.md.

Limits of evidence: browser Web3 used a mocked injected wallet; real WalletConnect/Metarang, SMTP deliverability, public TLS and actual connected clients need deployment validation. Source imports used synthetic fixtures, never live data. Non-default source session/encryption/client-ID configurations listed in production.md require adaptation if actually deployed. Container files were authored but not built here because Docker is unavailable. Passing tests are not a guarantee of no bugs in an untested live environment.
