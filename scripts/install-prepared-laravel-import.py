"""Install validated Go-format data into the existing Go schema.

Requires explicit replacement intent. Keeps foreign keys enabled, archives
all existing rows, and replaces application data in a single transaction.
Run cmd/migrate and take a tested SQL backup first; stop API/worker writers.
"""

import argparse
import json
import re
import subprocess


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--mysql", required=True)
    parser.add_argument("--defaults-extra-file")
    parser.add_argument("--source", required=True, help="Validated Go-format staging database")
    parser.add_argument("--target", required=True, help="Existing Go database")
    parser.add_argument("--archive-prefix", default="archive_local_")
    parser.add_argument("--expected-existing-users", type=int, required=True)
    parser.add_argument("--replace-application-data", action="store_true", required=True)
    args = parser.parse_args()
    for name in (args.source, args.target, args.archive_prefix):
        if not re.fullmatch(r"[a-zA-Z0-9_]+", name):
            parser.error("Invalid database/archive identifier")
    if args.source == args.target:
        parser.error("Source and target must differ")
    command = [args.mysql]
    if args.defaults_extra_file:
        command.append("--defaults-extra-file=" + args.defaults_extra_file)
    else:
        command += ["--host=127.0.0.1", "--user=root"]
    command += ["--batch", "--skip-column-names", "--default-character-set=utf8mb4"]

    def sql(query):
        result = subprocess.run(command, input=query, text=True, encoding="utf-8", capture_output=True)
        if result.returncode:
            # Keep credential-bearing row values out of tool output.
            raise RuntimeError("Database operation failed; current transaction was rolled back")
        return result.stdout.strip()

    source, target = ("`" + name + "`" for name in (args.source, args.target))
    if int(sql(f"SELECT COUNT(*) FROM {target}.users")) != args.expected_existing_users:
        raise RuntimeError("Target user count changed; refusing replacement")
    if int(sql(f"SELECT COUNT(*) FROM {source}.users")) == 0:
        raise RuntimeError("Prepared source has no users")
    versions = sql(f"SELECT MAX(version) FROM {source}.schema_migrations UNION ALL SELECT MAX(version) FROM {target}.schema_migrations").splitlines()
    if len(versions) != 2 or versions[0] != versions[1]:
        raise RuntimeError("Schema versions differ; migrate the target before proceeding")
    tables = sql(f"SELECT table_name FROM information_schema.tables WHERE table_schema='{args.source}' AND table_type='BASE TABLE' ORDER BY table_name").splitlines()
    application = [t for t in tables if not t.startswith("archive_laravel_") and t != "schema_migrations"]
    source_archives = [t for t in tables if t.startswith("archive_laravel_")]
    existing = sql(f"SELECT table_name FROM information_schema.tables WHERE table_schema='{args.target}' AND table_type='BASE TABLE' ORDER BY table_name").splitlines()
    required = set(application + ["schema_migrations"])
    if set(existing) != required:
        raise RuntimeError("Target has unexpected or missing tables; refusing replacement")
    for table in tables + existing:
        if not re.fullmatch(r"[a-zA-Z0-9_]+", table) or len(table) > 64:
            raise RuntimeError("Invalid table/archive name")
    if any(len(args.archive_prefix + table) > 64 for table in existing):
        raise RuntimeError("Archive prefix is too long")
    columns = {}
    for table in application:
        def table_columns(database):
            return sql(f"SELECT column_name FROM information_schema.columns WHERE table_schema='{database}' AND table_name='{table}' ORDER BY ordinal_position").splitlines()
        columns[table] = table_columns(args.target)
        if set(columns[table]) != set(table_columns(args.source)):
            raise RuntimeError("Application column sets differ")
    edges = sql(f"SELECT table_name,referenced_table_name FROM information_schema.key_column_usage WHERE table_schema='{args.target}' AND referenced_table_name IS NOT NULL").splitlines()
    dependencies = {t: set() for t in application}
    for edge in edges:
        child, parent = edge.split("\t")
        if child in dependencies and parent in dependencies:
            dependencies[child].add(parent)
    order = []
    while len(order) < len(application):
        available = sorted(t for t in application if t not in order and dependencies[t].issubset(order))
        if not available:
            raise RuntimeError("Foreign-key dependency cycle")
        order.extend(available)
    before = {t: int(sql(f"SELECT COUNT(*) FROM {target}.`{t}`")) for t in existing}
    for table in existing:
        sql(f"CREATE TABLE {target}.`{args.archive_prefix}{table}` LIKE {target}.`{table}`")
    for table in source_archives:
        sql(f"CREATE TABLE {target}.`{table}` LIKE {source}.`{table}`")
    statements = ["SET time_zone='+00:00'", "START TRANSACTION"]
    for table in existing:
        statements.append(f"INSERT INTO {target}.`{args.archive_prefix}{table}` SELECT * FROM {target}.`{table}`")
    for table in reversed(order):
        statements.append(f"DELETE FROM {target}.`{table}`")
    for table in order:
        names = ",".join("`" + c + "`" for c in columns[table])
        statements.append(f"INSERT INTO {target}.`{table}` ({names}) SELECT {names} FROM {source}.`{table}`")
    for table in source_archives:
        statements.append(f"INSERT INTO {target}.`{table}` SELECT * FROM {source}.`{table}`")
    statements.append("COMMIT")
    sql(";\n".join(statements) + ";")
    for table, count in before.items():
        if int(sql(f"SELECT COUNT(*) FROM {target}.`{args.archive_prefix}{table}`")) != count:
            raise RuntimeError("Existing-data archive count mismatch")
    after = {}
    for table in application + source_archives:
        count = int(sql(f"SELECT COUNT(*) FROM {target}.`{table}`"))
        if count != int(sql(f"SELECT COUNT(*) FROM {source}.`{table}`")):
            raise RuntimeError("Installed-data count mismatch")
        after[table] = count
    print(json.dumps({"target": args.target, "old_counts_archived": before, "installed_counts": after}, indent=2))


if __name__ == "__main__":
    main()
