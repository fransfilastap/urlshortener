# sqlc + golang-migrate + Testing Integration Design

**Date**: 2026-05-04
**Status**: Approved

## Summary

Add sqlc for type-safe SQL query generation, golang-migrate for database migrations, update the Makefile with new targets, and improve test coverage across the project.

## Approach

**Full sqlc replacement (Approach 1)**: Replace all hand-written SQL in `postgres_repository.go` with sqlc-generated code. The `URLRepository` interface stays unchanged — only `PostgresRepository` internals change to delegate to sqlc's `Queries` struct.

## Project Structure

```
urlshortener/
├── db/
│   ├── migrations/
│   │   ├── 000001_init_schema.up.sql
│   │   └── 000001_init_schema.down.sql
│   └── queries/
│       ├── urls.sql
│       ├── clicks.sql
│       └── url_history.sql
├── sqlc.yaml
├── internal/
│   └── db/
│       └── sqlc/
│           ├── db.go
│           ├── models.go
│           └── *.sql.go
├── store/
│   ├── url_repository.go          # Interface (unchanged)
│   ├── postgres_repository.go     # Rewritten to use sqlc
│   ├── cache_repository.go        # Unchanged
│   ├── url_service.go             # Unchanged
│   ├── postgres_repository_test.go # Updated
│   └── sqlc_test.go               # New: sqlc query integration tests
└── Makefile                        # Updated with new targets
```

## Migrations

Using `golang-migrate/migrate` with numbered SQL files.

### 000001_init_schema.up.sql

Matches the current `InitSchema()` exactly:
- `urls` table: id (SERIAL PK), original (TEXT NOT NULL), short (TEXT NOT NULL UNIQUE), title (TEXT), created_at (TIMESTAMP NOT NULL DEFAULT NOW()), expires_at (TIMESTAMP), clicks (BIGINT NOT NULL DEFAULT 0), creator_reference (TEXT), deleted_at (TIMESTAMP)
- `clicks` table: id (SERIAL PK), url_id (BIGINT NOT NULL REFERENCES urls(id) ON DELETE CASCADE), url_short (TEXT NOT NULL), ip (TEXT NOT NULL), location (TEXT), browser (TEXT), device (TEXT), timestamp (TIMESTAMP NOT NULL DEFAULT NOW())
- `url_history` table: id (SERIAL PK), url_id (BIGINT NOT NULL REFERENCES urls(id) ON DELETE CASCADE), url_short (TEXT NOT NULL), action (TEXT NOT NULL), old_value (JSONB), new_value (JSONB), modified_at (TIMESTAMP NOT NULL DEFAULT NOW()), modified_by (TEXT)
- Indexes: idx_urls_short, idx_urls_original, idx_clicks_url_id, idx_clicks_url_short, idx_url_history_url_id, idx_url_history_url_short

### 000001_init_schema.down.sql

Drops all 3 tables in reverse dependency order.

### Startup change

`InitSchema()` method on `PostgresRepository` is removed. `main.go` calls the migration runner at startup instead.

## sqlc Configuration

```yaml
# sqlc.yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "db/queries/"
    schema: "db/migrations/"
    gen:
      go:
        package: "sqlc"
        out: "internal/db/sqlc"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_interface: true
        emit_exact_table_names: false
        emit_empty_slices: true
        overrides:
          - db_type: "jsonb"
            go_type: "encoding/json.RawMessage"
```

## sqlc Queries

### urls.sql
- `CreateURL` — INSERT into urls, RETURNING all fields
- `GetURLByShort` — SELECT from urls WHERE short = $1 AND deleted_at IS NULL
- `GetURLByOriginal` — SELECT from urls WHERE original = $1 AND deleted_at IS NULL
- `GetURLsByCreator` — SELECT from urls WHERE creator_reference = $1 AND deleted_at IS NULL ORDER BY created_at DESC
- `IncrementClicks` — UPDATE urls SET clicks = clicks + 1 WHERE short = $1 AND deleted_at IS NULL
- `SoftDeleteURL` — UPDATE urls SET deleted_at = NOW() WHERE short = $1 AND deleted_at IS NULL
- `SoftDeleteURLWithCreator` — UPDATE urls SET deleted_at = NOW() WHERE short = $1 AND creator_reference = $2 AND deleted_at IS NULL
- `HardDeleteURL` — DELETE FROM urls WHERE short = $1
- `UpdateURL` — UPDATE urls SET original, title, expires_at WHERE short = $1 AND deleted_at IS NULL
- `UpdateURLWithCreator` — UPDATE urls SET original, title, expires_at WHERE short = $1 AND creator_reference = $2 AND deleted_at IS NULL

### clicks.sql
- `StoreClick` — INSERT into clicks
- `GetClicksByShort` — SELECT from clicks WHERE url_short = $1 ORDER BY timestamp DESC
- `HasRecentClick` — SELECT EXISTS subquery with 1-hour interval
- `GetTotalClicks` — SELECT COUNT(*) FROM clicks WHERE url_short = $1
- `GetClicksByBrowser` — SELECT browser, COUNT(*) FROM clicks WHERE url_short = $1 GROUP BY browser
- `GetClicksByDevice` — SELECT device, COUNT(*) FROM clicks WHERE url_short = $1 GROUP BY device
- `GetClicksByLocation` — SELECT location, COUNT(*) FROM clicks WHERE url_short = $1 GROUP BY location

### url_history.sql
- `LogURLHistory` — INSERT into url_history

## PostgresRepository Refactor

`PostgresRepository` holds a `*sqlc.Queries` instance. Each method:
1. Translates `models.*` → sqlc parameters
2. Calls the corresponding sqlc method
3. Translates sqlc result → `models.*`

This keeps the `URLRepository` interface and `URLService` completely unchanged.

## Makefile Targets

```makefile
# sqlc
.PHONY: sqlc-generate
sqlc-generate:
	@echo "Generating sqlc code..."
	sqlc generate

.PHONY: sqlc-validate
sqlc-validate:
	@echo "Validating sqlc queries..."
	sqlc vet

# migrations
MIGRATE_DB_URL ?= postgres://postgres:postgres@localhost:5432/urlshortener?sslmode=disable

.PHONY: migrate-up
migrate-up:
	@echo "Running migrations..."
	migrate -path db/migrations -database "$(MIGRATE_DB_URL)" up

.PHONY: migrate-down
migrate-down:
	@echo "Rolling back last migration..."
	migrate -path db/migrations -database "$(MIGRATE_DB_URL)" down 1

.PHONY: migrate-create
migrate-create:
	@echo "Creating migration $(NAME)..."
	migrate create -ext sql -dir db/migrations -seq $(NAME)

.PHONY: migrate-force
migrate-force:
	@echo "Forcing migration version $(VERS)..."
	migrate -path db/migrations -database "$(MIGRATE_DB_URL)" force $(VERS)
```

## Testing Plan

### sqlc Query Integration Tests (new: `store/sqlc_test.go`)
- Uses testcontainers Postgres (reuse existing `SetupPostgresContainer`)
- Runs migration files (not `InitSchema`) to validate migrations
- Tests each sqlc query method directly
- Covers: Create, GetByShort, GetByOriginal, GetByCreator, IncrementClicks, Delete, Update, Click operations, History logging

### PostgresRepository Tests (update: `store/postgres_repository_test.go`)
- Switch from `InitSchema()` to migration runner for setup
- Keep the same test structure (the interface doesn't change)

### Existing Tests (unchanged)
- `store/url_service_test.go` — mock-based, unaffected
- `store/cache_repository_test.go` — testcontainers Redis, unaffected
- `handlers/url_handler_test.go` — mock-based, unaffected

### New Dependencies
- `github.com/golang-migrate/migrate/v4` — migration engine
- `github.com/sqlc-dev/sqlc` — code generation (dev dependency, via `go generate`)

## Dependencies to Add

```
# go.mod
github.com/golang-migrate/migrate/v4
github.com/sqlc-dev/sqlc  # tool dependency via go.mod tool directive
```