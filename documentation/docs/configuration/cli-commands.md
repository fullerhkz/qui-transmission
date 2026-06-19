---
sidebar_position: 5
title: CLI Commands
---

# CLI Commands

## Generate Configuration File

Create a default configuration file without starting the server:

```bash
# Generate config in OS-specific default location
./qui-Transmission generate-config

# Generate config in custom directory
./qui-Transmission generate-config --config-dir /path/to/config/

# Generate config with custom filename
./qui-Transmission generate-config --config-dir /path/to/myconfig.toml
```

## User Management

Create and manage user accounts from the command line:

```bash
# Create initial user account
./qui-Transmission create-user --username admin --password mypassword

# Create user with prompts (secure password input)
./qui-Transmission create-user --username admin

# Change password for existing user (no old password required)
./qui-Transmission change-password --username admin --new-password mynewpassword

# Change password with secure prompt
./qui-Transmission change-password --username admin

# Pipe passwords for scripting (works with both commands)
echo "mypassword" | ./qui-Transmission create-user --username admin
echo "newpassword" | ./qui-Transmission change-password --username admin
printf "password" | ./qui-Transmission change-password --username admin
./qui-Transmission change-password --username admin < password.txt

# All commands support custom config/data directories
./qui-Transmission create-user --config-dir /path/to/config/ --username admin
```

### Notes

- Only one user account is allowed in the system
- Passwords must be at least 8 characters long
- Interactive prompts use secure input (passwords are masked)
- Supports piped input for automation and scripting
- Commands will create the database if it doesn't exist
- No password confirmation required - perfect for automation

### Reset a Forgotten Password {#reset-password}

If you've forgotten your password, use the `change-password` command to set a new one. No old password is required.

**Linux / macOS:**

```bash
./qui-Transmission change-password --username admin --new-password mynewpassword
```

**Windows (Command Prompt):**

Navigate to the folder containing `qui-Transmission.exe` and run:

```batch
qui-Transmission.exe change-password --username admin --new-password mynewpassword
```

**Docker:**

```bash
docker exec -it <container-name> qui-Transmission change-password --username admin --new-password mynewpassword
```

Replace `admin` with your username and `mynewpassword` with your desired password (minimum 8 characters).

## Update Command

Keep your qui-Transmission installation up-to-date:

```bash
# Update to the latest version
./qui-Transmission update
```

## Command Line Flags

```bash
# Specify config directory (config.toml will be created inside)
./qui-Transmission serve --config-dir /path/to/config/

# Specify data directory for database and other data files
./qui-Transmission serve --data-dir /path/to/data/
```

## Database Migration

Offline SQLite to Postgres migration:

```bash
# 0) Stop qui-Transmission first (no writes during migration)
#    (example) docker compose stop qui-Transmission

# 1) Create the target Postgres database first (required)
#    (example) createdb -h localhost -p 5432 -U user qui-Transmission
#    (or in psql) CREATE DATABASE qui-Transmission;

# 2) Optional: backup the SQLite file
cp /path/to/qui-Transmission.db /path/to/qui-Transmission.db.bak

# 3) Validate source + destination without importing rows
./qui-Transmission db migrate \
  --from-sqlite /path/to/qui-Transmission.db \
  --to-postgres "postgres://user:pass@localhost:5432/qui-Transmission?sslmode=disable" \
  --dry-run

# 4) Apply migration (schema bootstrap + table copy + identity reset)
./qui-Transmission db migrate \
  --from-sqlite /path/to/qui-Transmission.db \
  --to-postgres "postgres://user:pass@localhost:5432/qui-Transmission?sslmode=disable" \
  --apply

# 5) Point qui-Transmission at Postgres and start it again
#    - config.toml: databaseEngine=postgres + databaseDsn=...
#    - or env: QUI__DATABASE_ENGINE=postgres + QUI__DATABASE_DSN=...
```

Notes:

- Run this while qui-Transmission is stopped.
- Create the target Postgres database before running migration.
- `--dry-run` and `--apply` are mutually exclusive.
- The command copies all runtime tables except migration history.
- The migrator bootstraps schema/tables inside the destination DB, but does not create the database itself.
- The output includes per-table row counts for SQLite and Postgres.

### FAQ

**Q: Why is `cross_seed_feed_items` row count lower in Postgres after migration?**

This is expected when the SQLite file contains historical rows whose `indexer_id` no longer exists in `torznab_indexers`.
Postgres enforces the foreign key strictly, so migration keeps only rows that still have valid parent records.

You can verify this in SQLite:

```sql
SELECT COUNT(*) AS orphaned_rows
FROM cross_seed_feed_items f
LEFT JOIN torznab_indexers i ON i.id = f.indexer_id
WHERE i.id IS NULL;
```

If `orphaned_rows` matches the migration delta (`sqlite_count - postgres_count`), migration behavior is working as intended.
