# Database dictionary

Canonical SQL: internal/infrastructure/mysql/schema.sql, features.sql and oauth.sql. Apply through cmd/migrate. All tables use InnoDB. Human text uses utf8mb4; hashes/protocol identifiers explicitly use ASCII with binary collation. IDs/FKs are signed BIGINT to preserve Laravel IDs within Go int64. Times are UTC DATETIME(6), not formatted VARCHAR. Nullable values represent absent data; Iranian phone/national/company identifiers are strings so leading zeros survive.

Lengths follow inspected validation and protocol formats. Names/addresses retain Laravel's 255-character allowance. Company registration/national/tax identifiers allow 32 characters because the source does not establish a fixed-width domain contract; narrowing to an invented numeric width would lose valid legacy data. Migration/import rejects values exceeding these limits. SQL VARCHAR counts characters; password validation additionally limits UTF-8 bytes to bcrypt's 72.

## Accounts and personal information

| Table.column | Type | Purpose / constraints |
|---|---|---|
| users.id | BIGINT AUTO_INCREMENT | Primary key |
| users.name | VARCHAR(255) | Display name, required |
| users.username | VARCHAR(30) NULL | Unique, normalized ASCII username; absent on imported accounts |
| users.email | VARCHAR(255) NULL | Unique normalized email; wallet-only accounts may omit |
| users.password_hash | VARCHAR(255) NULL | Bcrypt; NULL for wallet-only accounts; reserve space for hash evolution |
| users.code | VARCHAR(32) NULL | Unique stable member code, e.g. hm2000001 |
| users.referral | VARCHAR(32) | Referral code, default empty |
| users.mobile | VARCHAR(11) ASCII NULL | Imported account mobile |
| users.email_verified_at | DATETIME(6) NULL | NULL until verified |
| users.created_at / updated_at | DATETIME(6) | Required; updated_at automatically tracks updates |
| personal_infos.user_id | BIGINT | PK + FK users; one profile per account |
| personal_infos.is_verified | TINYINT | Verification flag, default 0, server-controlled |
| personal_infos.is_company | BOOLEAN NULL | Personal/company choice |
| personal_infos.first_name / last_name | VARCHAR(255) | Default empty |
| personal_infos.mobile / telephone | VARCHAR(11) ASCII | Iranian numbers, preserves leading zero |
| personal_infos.national_code | VARCHAR(10) ASCII | Ten-digit national identifier |
| personal_infos.address / company_address | VARCHAR(255) | Validated address |
| personal_infos.company_name / company_executive_name | VARCHAR(255) | Organization/contact names |
| personal_infos.company_registration_number / company_national_number / company_tax_number | VARCHAR(32) | Identifier strings |
| personal_infos.verification_messages | TEXT | Administrative verification feedback; default empty |
| code_sequence.id / value | INT / BIGINT | Singleton PK id=1; transactional counter, initial 2000000 |

Each profile field is a separate column. There is no mutable profile JSON payload. profile_details(user_id BIGINT PK/FK, payload TEXT) is a read-only legacy archive retained to avoid losing historical data.

## Sessions, actions, media and wallets

| Table.column | Type | Purpose / constraints |
|---|---|---|
| sessions.token_hash | CHAR(64) ASCII binary | PK; SHA-256 hex, no raw bearer stored |
| sessions.user_id / expires_at | BIGINT / DATETIME(6) | Owner FK; current idle deadline or fixed credential expiry |
| sessions.idle_seconds | INT UNSIGNED | Version 7; 0 keeps fixed expiry, positive values refresh the live session deadline on activity; no additional index needed |
| actions.user_id / kind | BIGINT / VARCHAR(6) ASCII binary | Composite PK; kind CHECK verify/reset |
| actions.token_hash / email / expires_at | CHAR(64) ASCII binary / VARCHAR(255) / DATETIME(6) | One-use verification/reset action |
| media.user_id / kind | BIGINT / VARCHAR(20) ASCII binary | Composite PK; avatar or document collection |
| media.content_type / data | VARCHAR(32) ASCII binary / MEDIUMBLOB | Validated jpg/png/webp; application max 1 MiB (BLOB's 64 KiB is insufficient) |
| wallets.user_id / address | BIGINT / CHAR(42) ASCII binary | PK owner; UNIQUE normalized EVM 0x address |
| challenges.key / message / expires_at | VARCHAR(100) ASCII binary / TEXT / DATETIME(6) | PK namespaced nonce key, exact message and expiry |
| session_attributes.token_hash / name | CHAR(64) / VARCHAR(32), ASCII binary | Composite PK; FK to session; password confirmation/wallet callback state |
| session_attributes.value / expires_at | TEXT / DATETIME(6) | Attribute payload with absolute expiry |
| registration_callbacks.user_id / url / expires_at | BIGINT / VARCHAR(2048) / DATETIME(6) | PK/FK owner, one-use callback |

## OAuth

| Table.column | Type | Purpose / constraints |
|---|---|---|
| oauth_clients.id / name | BIGINT AUTO_INCREMENT / VARCHAR(255) | PK and client display name |
| oauth_clients.purpose | VARCHAR(16) ASCII binary NULL UNIQUE | Optional personal_access designation |
| oauth_clients.secret_hash / revoked / created_at | VARCHAR(255) NULL / BOOLEAN / DATETIME(6) | NULL secret for public client; bcrypt otherwise |
| oauth_client_redirects.client_id / uri_hash | BIGINT / CHAR(64) ASCII binary | Composite PK; SHA-256 URI hash indexes long URLs |
| oauth_client_redirects.uri | VARCHAR(2048) utf8mb4_bin | Exact redirect, one row per URI; comparison also checks original URI |
| oauth_grants.id / user_id / client_id | BIGINT | PK id and owner/client FKs |
| oauth_grants.revoked_at | DATETIME(6) NULL | Grant-wide revocation |
| oauth_grant_scopes.grant_id / scope | BIGINT / VARCHAR(32) ASCII binary | Composite PK; one scope per row |
| oauth_codes.token_hash / grant_id | CHAR(64) ASCII binary / BIGINT | PK one-use code digest and grant FK |
| oauth_codes.redirect_uri / challenge | VARCHAR(2048) utf8mb4_bin / CHAR(43) ASCII binary | Bound redirect and S256 base64url challenge; empty challenge permitted for legacy confidential clients |
| oauth_codes.expires_at / consumed_at | DATETIME(6) / DATETIME(6) NULL | Expiry and consumption |
| oauth_tokens.access_hash / refresh_hash | CHAR(64) ASCII binary | Access PK, refresh UNIQUE; SHA-256 token-ID digests |
| oauth_tokens.grant_id | BIGINT | Grant FK; generations remain linked for replay revocation |
| oauth_tokens.access_expires_at / refresh_expires_at / consumed_at | DATETIME(6), last nullable | Absolute TTL and refresh consumption |

OAuth redirects/scopes use child rows instead of comma lists/JSON. JWT jti is 80 hex characters, matching Passport; storage retains its SHA-256 digest (64). Token generations and grant-level revocation support refresh replay detection.

## Indexes and relationships

PKs and UNIQUE constraints above provide identity lookups. Additional indexes correspond to actual filtering/cleanup paths:

| Table | Index columns | Query purpose |
|---|---|---|
| users | (email_verified_at, created_at) | Unverified accounts older than cutoff |
| sessions | (user_id), (expires_at) | Owner revocation; expired cleanup |
| actions | (expires_at), (email, kind, token_hash) | Expired cleanup; reset-token lookup |
| challenges, session_attributes, registration_callbacks | (expires_at) on each | Expired cleanup |
| oauth_grants | (user_id), (client_id) | User/client revocation and FK lookup |
| oauth_codes | (grant_id), (expires_at) | Grant lookup; expired cleanup |
| oauth_tokens | (grant_id), (refresh_expires_at) | Grant revocation; expired cleanup |

All owner/client/grant FKs use ON DELETE CASCADE; session_attributes cascades from sessions. Child composite PKs already index their leading FK. No blanket indexes are added to profile columns that are not searched. Foreign keys prevent orphan records, and transactional writes preserve account/profile creation, code allocation, password updates/revocation and one-use token/nonce consumption.

schema_migrations records applied versions and timestamps; immutable historical SQL is under mysql/migrations. Schema migration DDL is resumable, not transactional: see migration.md before upgrading existing data.

## Operational migrations (versions 4–6)

| Table.column | Type | Constraint / purpose |
|---|---|---|
| mail_outbox.id | BIGINT AUTO_INCREMENT | PK, durable delivery identifier |
| mail_outbox.recipient / subject | VARCHAR(255) / VARCHAR(255) | Address and subject |
| mail_outbox.text_body / html_body | MEDIUMTEXT | Multipart alternative bodies; application combined max 1 MiB |
| mail_outbox.state | VARCHAR(10) ASCII binary | CHECK pending/processing/sent/dead |
| mail_outbox.attempts | SMALLINT UNSIGNED | Delivery/lease generation; at most ten attempts |
| mail_outbox.available_at / created_at / expires_at | DATETIME(6) | Due/lease deadline, creation and one-hour notice expiry |
| mail_outbox.sent_at | DATETIME(6) NULL | Successful acknowledgment |
| rate_limits.peer_hash | CHAR(64) ASCII binary | PK; digest of resolved user/IP and budget namespace |
| rate_limits.window_start / blocked_until | DATETIME(6), last nullable | Shared request window and ban deadline |
| rate_limits.request_count / strikes | SMALLINT UNSIGNED / TINYINT UNSIGNED | Atomic budget and persistent capped penalty history |
| legacy_password_resets.user_id | BIGINT | PK/FK owner, cascade |
| legacy_password_resets.token_hash / expires_at | VARCHAR(255) / DATETIME(6) | Preserved Laravel bcrypt reset hash and absolute expiry |
| legacy_remember_tokens.user_id | BIGINT | PK/FK owner, cascade |
| legacy_remember_tokens.token_hash / expires_at | CHAR(64) ASCII binary / DATETIME(6) | Digest of source remember credential and expiry |
| oauth_codes.legacy | BOOLEAN | Defaults false; imported codes bind authenticated encrypted metadata once |

Additional indexes: mail_outbox(state,available_at,id) for worker claims; mail_outbox(expires_at,state) for deadline enforcement; expiry indexes on legacy_password_resets and legacy_remember_tokens. Imported file-session IDs are digested into the existing sessions table; confirmation/callback state uses session_attributes. Primitive parser limits and cookie-name MAC checks are documented in migration.md.
