"""Prepare an isolated, read-only view of a frozen Laravel snapshot.

Retain every original table in archive_laravel_* tables in the empty Go target.
Only orphan OAuth records and media awaiting their files are excluded from the
functional import. No source row is updated or deleted. This script does not
grant authentication rights to archived credentials.
"""

import argparse
import json
import re
import subprocess


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--mysql", required=True)
    parser.add_argument("--defaults-extra-file", help="Private MySQL client credentials file")
    parser.add_argument("--source", required=True)
    parser.add_argument("--target", required=True)
    parser.add_argument("--adapter", required=True)
    args = parser.parse_args()
    for name in (args.source, args.target, args.adapter):
        if not re.fullmatch(r"[a-zA-Z0-9_]+", name):
            parser.error("Database names must contain only ASCII letters, digits and underscores")
    if len({args.source, args.target, args.adapter}) != 3:
        parser.error("Source, target and adapter must be separate databases")
    command = [args.mysql]
    if args.defaults_extra_file:
        command.append("--defaults-extra-file=" + args.defaults_extra_file)
    else:
        command += ["--host=127.0.0.1", "--user=root"]
    command += ["--batch", "--skip-column-names", "--default-character-set=utf8mb4"]

    def sql(query):
        result = subprocess.run(command, input=query, text=True, encoding="utf-8", capture_output=True)
        if result.returncode:
            # Do not print SQL or credential-bearing records from client errors.
            raise RuntimeError("MySQL operation failed; no credentials or row data printed")
        return result.stdout.strip()

    source, target, adapter = ("`" + name + "`" for name in (args.source, args.target, args.adapter))
    occupied = int(sql(f"SELECT (SELECT COUNT(*) FROM {target}.users)+(SELECT COUNT(*) FROM {target}.oauth_clients)"))
    if occupied:
        raise RuntimeError("Target contains application data; refusing to overwrite or merge")
    tables = sql(f"SELECT table_name FROM information_schema.tables WHERE table_schema='{args.source}' AND table_type='BASE TABLE' ORDER BY table_name").splitlines()
    if not tables:
        raise RuntimeError("Source snapshot is empty")
    for table in tables:
        if not re.fullmatch(r"[a-zA-Z0-9_]+", table):
            raise RuntimeError("Unsupported table name")
    sql(f"CREATE DATABASE {adapter} CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci")
    archive_counts = {}
    for table in tables:
        archive = f"{target}.`archive_laravel_{table}`"
        sql(f"CREATE TABLE {archive} LIKE {source}.`{table}`")
        archive_counts[table] = int(sql(f"SELECT COUNT(*) FROM {source}.`{table}`"))
    # DDL occurs before this transaction; original rows are copied atomically.
    statements = ["START TRANSACTION"]
    for table in tables:
        statements.append(f"INSERT INTO {target}.`archive_laravel_{table}` SELECT * FROM {source}.`{table}`")
    statements.append("COMMIT")
    sql(";\n".join(statements) + ";")

    for table in tables:
        if table in ("oauth_access_tokens", "oauth_auth_codes", "oauth_refresh_tokens"):
            continue
        if table in ("password_reset_tokens", "password_resets"):
            sql(f"CREATE VIEW {adapter}.`{table}` AS SELECT r.* FROM {source}.`{table}` r JOIN {source}.users u ON LOWER(TRIM(r.email))=LOWER(u.email)")
        else:
            condition = " WHERE 1=0" if table == "media" else ""
            sql(f"CREATE VIEW {adapter}.`{table}` AS SELECT * FROM {source}.`{table}`{condition}")
    for table in ("oauth_access_tokens", "oauth_auth_codes"):
        if table in tables:
            sql(f"CREATE VIEW {adapter}.`{table}` AS SELECT t.* FROM {source}.`{table}` t JOIN {source}.users u ON u.id=t.user_id JOIN {source}.oauth_clients c ON c.id=t.client_id")
    if "oauth_refresh_tokens" in tables:
        sql(f"CREATE VIEW {adapter}.oauth_refresh_tokens AS SELECT r.* FROM {source}.oauth_refresh_tokens r JOIN {adapter}.oauth_access_tokens a ON a.id=r.access_token_id")
    functional_counts = {}
    for table in tables:
        functional_counts[table] = int(sql(f"SELECT COUNT(*) FROM {adapter}.`{table}`"))
        archived = int(sql(f"SELECT COUNT(*) FROM {target}.`archive_laravel_{table}`"))
        if archived != archive_counts[table]:
            raise RuntimeError("Archive count verification failed")
    print(json.dumps({"source": args.source, "target": args.target, "adapter": args.adapter,
                      "archived_rows": archive_counts, "functional_rows": functional_counts,
                      "deferred_media": archive_counts.get("media", 0)}, indent=2))


if __name__ == "__main__":
    main()
