# Migration Production Safety Design

## Problem

The app runs `golang-migrate` `m.Up()` on every startup with no production safety guards. Three risk scenarios:

1. **Accidental destructive migration** — `m.Down()` could DROP all tables in production
2. **Unreviewed migrations auto-applied** — new migrations run automatically on app startup in production
3. **Existing DB without schema_migrations** — baseline scenario where tables exist but lack golang-migrate tracking, causing conflicts or duplicate creation

## Design

### Approach: Config-gated auto-migrate

Preserve dev convenience (auto-migrate on startup), add production safety via configuration.

### 1. Configuration & Control Flow

**New config values in `config.go`:**

- `AUTO_MIGRATE` (bool, default `true`) — controls whether `RunMigrations` is called during app startup
- `EXPECTED_SCHEMA_VERSION` (int, default `0`, meaning no check) — if set, app validates the DB schema version matches before starting

**Startup flow in `main.go`:**

1. If `AUTO_MIGRATE=true` → run `RunMigrations()` as before
2. If `AUTO_MIGRATE=false` → skip migrations, log `"Migrations skipped (AUTO_MIGRATE=false)"`
3. If `EXPECTED_SCHEMA_VERSION > 0` → after migrations (or skip), validate current DB version matches
   - Version too low → fatal error: `"Schema version %d is behind expected %d. Run migrations manually."`
   - Version too high → fatal error: `"Schema version %d is ahead of expected %d. Possible downgrade attempt."`
   - Version matches → continue

**Docker compose changes:**

- Production `app` service: `AUTO_MIGRATE=false`, `EXPECTED_SCHEMA_VERSION=1`
- Dev `app-dev` service: no changes (inherits `AUTO_MIGRATE=true` default)

### 2. Baseline Detection

`RunMigrations` checks for a baseline scenario — tables exist but `schema_migrations` doesn't:

```
1. Create migrate instance
2. Call m.Version()
3. If error is "no such table schema_migrations":
   a. Query: do core tables (urls, clicks) exist?
   b. If YES → return error with clear message to run baseline command
   c. If NO → fresh database, safe to run m.Up()
4. Otherwise, proceed with m.Up() as normal
```

### 3. Separate `migrate` Subcommand

New CLI subcommands via `os.Args` detection in `main()`:

```
./urlshortener                  → start server (current behavior)
./urlshortener migrate up       → run migrations and exit
./urlshortener migrate force <version> → baseline existing DB at given version
./urlshortener migrate version          → print current schema version
```

### 4. Down() Protection

- No `m.Down()` call anywhere in the codebase (already the case — keep it this way)
- Add explicit comment block in `migrate.go` warning against adding Down() capability
- The `migrate` subcommand does NOT support `down` — only `up`, `force`, and `version`
- Production rollbacks should be new forward migrations, not destructive reversions

### 5. Docker Compose & Production Deploys

**Production deploy workflow:**

1. Deploy new app container with updated `EXPECTED_SCHEMA_VERSION=N`
2. Run `docker compose exec app ./urlshortener migrate up`
3. Restart app (or let healthcheck/restart policy handle it)

**Docker compose production service config:**

```yaml
app:
  environment:
    - AUTO_MIGRATE=false
    - EXPECTED_SCHEMA_VERSION=1
```

## Files to Change

- `internal/config/config.go` — add `AutoMigrate` and `ExpectedSchemaVersion` fields
- `internal/repository/migrate.go` — add baseline detection, version check helper, subcommand logic, Down() warning comment
- `cmd/urlshortener/main.go` — add subcommand routing, conditionally call RunMigrations, add ExpectedSchemaVersion validation
- `docker-compose.yml` — add `AUTO_MIGRATE=false` and `EXPECTED_SCHEMA_VERSION=1` to production `app` service