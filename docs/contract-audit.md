# Laravel contract compatibility

Source of truth: the sibling Laravel repository's routes, controllers, resources and enabled Passport routes. Feature coverage in `feature-parity.md` does not establish exact response compatibility.

## Verified changes

- `/api/login` success message is `Login successful`, with a `token` field.
- `/api/login` invalid credentials returns HTTP 401 and `Invalid credentials`.
- `/api/logout` success is the single-field response `{"message":"Logged out successfully"}`.
- The independent frontend retains the existing visual design and public assets; typecheck, production build and all 14 browser tests passed, including the two new legacy-contract tests.
- Full Go tests and `go vet` passed after the response changes. The localization test fixture now enables OAuth for its OAuth request-validation cases.

## Implementation of previously outstanding items

- Legacy Web3 paths now expose the original methods, English signing messages and success/error messages, CSRF, guards and throttles. The independent frontend proxies those paths.
- `/api/user` now serializes original Laravel model keys, hidden fields excluded, nullable values and UTC timestamps with six fractional digits.
- Login accepts JSON/forms and ignores unknown keys; source validation summaries, guest redirects, credential/unauthenticated messages and throttle responses are preserved.
- Original account, personal-info, registration, logout, password and verification actions have their source methods, method override, guards and redirects. Signed one-use flash cookies carry form errors to the existing frontend notice component. Imported Laravel sessions retain their CSRF tokens.
- Passport returns League error/status/hint shapes and four-field token responses. Password grants and refresh rotation are implemented. Imported client flags retain grant restrictions, legacy scopes default to empty, PKCE supports source methods, and source token TTL defaults are retained. OAuth guest authorization redirects to `/login`; signed flash state carries the intended authorization URL for resumption after login/verification.
- `/api/me` and `/api/users/{user}` resources, verified full names, model-not-found errors and preserved imported avatar URLs are covered. `/storage/{media}/{filename}` serves imported public avatars while personal documents remain private.
- `/account/edit` and `/personal-info/edit` retain their URLs in Next. The legacy `/personal-info` form requires fresh scans on submission; the `/api/personal-info` extension keeps its document-preservation policy. Existing frontend visual design is retained.

## Evidence

`scripts/laravel-contracts.php` boots the actual source with synthetic models and array cache/session without querying production data. Its output is committed at `internal/transport/httpapi/testdata/laravel-contracts.json`. Tests compare Go responses against those fixtures and rerun the probe when PHP and the sibling vendor directory are available. Fixtures cover login validators, model/resource serialization, nonce messages, League errors, enabled grants and the route inventory. HTTP tests additionally cover forms, CSRF, guards, password/refresh replay, imported sessions and avatar paths. Validation includes full Go tests/vet, frontend typecheck/build and 14 browser tests.

## Device route configuration

Installed Passport enables device routes by default, but this application's legacy client model advertises only authorization_code/refresh_token, password/refresh_token or personal_access. No device view or source device-code migration is configured. Go preserves the resulting configuration: legacy clients get `unauthorized_client`, invalid user codes redirect back, and requests requiring the absent view fail. This does not configure a new successful device flow.

Fixtures cover the named cases; route coverage does not prove every possible response body. Browser pages use Next instead of reproducing Blade HTML. Real Metarang/WalletConnect, SMTP delivery, public TLS and connected third-party clients still require deployment acceptance testing. No production database import or service cutover has been performed.
