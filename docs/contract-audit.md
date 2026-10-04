# Laravel contract audit status

Source of truth: the sibling Laravel repository's routes, controllers, resources and enabled Passport routes. Feature coverage in `feature-parity.md` does not establish exact response compatibility.

## Verified changes

- `/api/login` success message is `Login successful`, with a `token` field.
- `/api/login` invalid credentials returns HTTP 401 and `Invalid credentials`.
- `/api/logout` success is the single-field response `{"message":"Logged out successfully"}`.
- The independent frontend retains the original UI source and public assets; typecheck, production build and all 12 existing browser tests passed after extraction.
- Full Go tests and `go vet` passed after the response changes. The localization test fixture now enables OAuth for its OAuth request-validation cases.

## Remaining compatibility work

- Laravel exposes `/web3/nonce`, `/web3/verify`, `/web3/link/nonce` and `/web3/link`; Go currently registers their `/api/web3/*` counterparts. The existing Next proxy does not supply those legacy paths.
- Compare `/api/user` serialization against actual Laravel model attributes, nulls and timestamp formatting. Go currently serializes its domain user, including its additional username field.
- Verify login validation, unknown fields, form input, guest middleware, rate-limit responses and unauthenticated error contracts against Laravel.
- Inventory web actions, redirects, CSRF/session behavior and Passport errors with method/status/header/body fixtures. Replacing rendered Blade pages with Next pages requires a separate review of browser behavior.
- Validate `/api/me` and `/api/users/{user}` resources, avatar URLs and error cases with shared fixtures rather than relying solely on feature tests.

No production database migration or service cutover has been performed. Exact compatibility is not yet certified.
