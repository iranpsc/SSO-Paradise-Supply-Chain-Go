# Local Laravel snapshot import, 2026-10-13

## Final installation into the existing sso_go database

After review, the user explicitly selected Laravel identities as the authoritative data. The existing five local users had conflicting IDs, and three member codes overlapped unrelated Laravel accounts. The prepared Go data was first installed into a restored backup rehearsal and validated, then installed into the existing `sso_go` database. The database name and local MYSQL_DATABASE setting remain `sso_go`.

Before installation, the existing database was backed up to `C:/Users/Diablo/Downloads/sso_go_before_laravel_import_20261013_final.sql`. Its Go schema was upgraded from migration 9 to 11. All previous local rows are preserved in `archive_local_<table>` tables. Old local sessions and tokens are archived only, preventing their user IDs from resolving to newly installed Laravel identities. Application data replacement was transactional, with foreign-key checks enabled. No Laravel user/client/member identifier was remapped.

The installed counts and validation results below now also apply to `sso_go`. All original profile fields matched; old local credentials have zero matches in active credential tables; the member-code sequence is above the imported member codes. The 30 files are still pending their storage backup. No API or cleanup worker was started during this operation.

The final SQL export is `C:/Users/Diablo/Downloads/sso_go_after_laravel_import_20261013.sql`. The reproducible installer is `scripts/install-prepared-laravel-import.py`; it refuses unexpected schemas/user counts and requires explicit replacement intent. Do not rerun it on the populated target.

The following sections describe the initial staging and archive preparation before that final installation.

The supplied SQL was restored on local WAMP into `sso_laravel_snapshot_20261013_125229`. The original `sso_go` database (5 users and 1 client) was not overwritten. Application rows were imported atomically into the isolated Go database `sso_go_import_20261013`.

## Results

| Data | Imported for application use |
| --- | ---: |
| Users, original IDs and password hashes | 7,963 |
| Original personal/company profiles | 6,508 |
| Default profiles for users without a source profile | 1,455 |
| OAuth clients, original IDs and compatible secrets | 16 |
| Wallets | 4 |
| Access/refresh token pairs | 1,993 |
| Authorization codes | 11,072 |
| Password reset records | 6,715 |
| Remember-token digests | 88 |

Every original table is also copied into `archive_laravel_<original_table>` in the new Go database. This retains all source values, token records, original profile/media IDs, and file metadata. These archives are not authentication tables and never authorize requests. The archives and SQL export contain sensitive information and must not be published or committed.

Source referential defects were excluded from functional import using a separate view-only adapter `sso_laravel_adapter_20261013`: 46 access tokens, 8,767 refresh tokens (including those attached to the 46 excluded access tokens), 1,679 authorization codes, and 1 password reset record cannot be linked to existing owners/clients/grants. All originals remain in the archive. No user, client or missing grant was invented to make these credentials work.

Thirty media records are deferred because the SQL has no file contents and no matching files were available. `archive_laravel_media` and `archive_laravel_personal_infos` retain the metadata/ownership needed to load files after receiving the storage/app backup. No empty image blobs were inserted. Browser file sessions and OAuth signing keys are also outside the SQL backup; old signed/encrypted credentials require the original keys.

Fourteen profiles contain `is_verified=-1`. The importer preserves this exact integer, and the MySQL adapter interprets it using nonzero truthiness, matching the current Laravel model/views. Reading and editing imported profiles was regression-tested without changing the raw status.

## Validation

User IDs, names, password hashes, normalized emails, member codes, referral values and mobile numbers matched the archive with zero differences. All 6,508 original profiles matched their source fields and raw verification status. All 16 client secrets were checked for compatible verification. A request to `/oauth/authorize` with client 38 and `https://dev-api.metarang.com/api/auth/callback` returned HTTP 302 to the login page, rather than invalid_client. The Laravel import regression tests passed on MySQL.

The verification script uses environment connection settings and prints only counts:

```powershell
$env:MYSQL_HOST='127.0.0.1'
$env:MYSQL_USER='root'
$env:MYSQL_PASSWORD=''
$env:MYSQL_DATABASE='sso_go_import_20261013'
$env:LARAVEL_MYSQL_DSN='root@tcp(127.0.0.1:3306)/sso_laravel_adapter_20261013'
go run scripts/verify-laravel-import.go
```

To point the local Go app at these imported records, set MYSQL_DATABASE to `sso_go_import_20261013`, with the appropriate local MySQL credentials. No app environment file was changed automatically. Do not reapply the full importer to the now-populated target. Media must be added separately when its backup arrives.

The existing worker cleanup policy would delete 7,252 historical unverified users. No worker was launched against this database during import. Review that policy before starting the worker for local evaluation; the original archived rows remain preserved.

The converted SQL export is `C:/Users/Diablo/Downloads/sso_go_import_20261013.sql`. It includes the Go schema, imported application records and original-data archives, but no media blobs or original RSA key.

## Reproducing deferred import into another empty target

Restore the dump into an isolated snapshot, migrate a separate empty Go target, and run `scripts/prepare-laravel-deferred-import.py` with three distinct source/target/adapter database names. The script refuses nonempty application targets and existing archive/view destinations. Use `--defaults-extra-file` for a private MySQL credentials file when needed. It leaves the source unchanged, archives every table, and exposes only referentially valid records to the strict Go importer. Run the normal importer against the adapter first without `--apply`, review counts, then run with `--apply`. Keep source writes stopped throughout preparation/import. This is a deferred-media workflow, not a merge into an existing populated database.


## Final WAMP state: application tables only

On 2026-10-13, at the user's request, all 37 archive tables were removed from `sso_go` after an external SQL backup. Only the 23 application tables remain. Counts in every application table were checked before and after removal and are unchanged: 7,963 users/profiles and 16 OAuth clients remain in the Go schema. Runtime does not depend on archive tables; migrations do not recreate deleted archives. Earlier archive descriptions above record the import process, not the final database state.

Backup: `C:/Users/Diablo/Downloads/sso_go_before_archive_removal_20261013_135626.sql`. Updated application-only export: `C:/Users/Diablo/Downloads/sso_go_after_laravel_import_20261013.sql`. Media metadata and invalid orphan credentials remain recoverable from the backup; files are still pending.

After removing archives, run the verifier with `LARAVEL_VERIFY_RUNTIME_ONLY=1` and the normal target MYSQL settings to check application counts and the actual client-38 authorization handler. This mode deliberately skips source comparisons and original client-secret checks, which were performed before removal. The full verifier requires source archive tables.


## Credential table naming (migration 13)

WAMP sso_go now uses password_reset_tokens and remember_tokens. Migration 13 atomically renames both prior legacy-prefixed tables and preserves their rows, hashes, expiry dates, indexes and foreign keys. Runtime SQL, importer, cleanup and database documentation use the new names. Historical creation migrations keep their original names and migration 13 upgrades them. The SQL export above was refreshed. Before/after counts (6,715 reset records, 88 remember records) and MySQL table checksums matched exactly.
