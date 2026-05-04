# sqlc + golang-migrate + Testing Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace hand-written SQL with sqlc-generated type-safe queries, add golang-migrate for DB migrations, add Makefile targets, and improve test coverage.

**Architecture:** Full sqlc replacement — `PostgresRepository` delegates to sqlc's `Queries` struct while keeping the `URLRepository` interface unchanged. Migrations run via golang-migrate at startup instead of `InitSchema()`. sqlc-generated code lives in `internal/db/sqlc/`.

**Tech Stack:** sqlc, golang-migrate/migrate/v4, pgx/v5, testcontainers-go

---

### Task 1: Add dependencies

**Files:**
- Modify: `go.mod`

**Step 1: Add golang-migrate dependency**

Run:
```bash
cd /Users/finnarc/Repo/urlshortener
go get github.com/golang-migrate/migrate/v4
go get github.com/golang-migrate/migrate/v4/database/pgx/v5
go get github.com/golang-migrate/migrate/v4/source/file
```

**Step 2: Add sqlc as a tool dependency**

Run:
```bash
go get github.com/sqlc-dev/sqlc
```

Note: sqlc is a dev tool used via `go generate` / `make sqlc-generate`, not imported at runtime (the generated code uses pgx/v5 which is already a dependency).

**Step 3: Tidy dependencies**

Run:
```bash
go mod tidy
```

**Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add golang-migrate and sqlc dependencies"
```

---

### Task 2: Create migration files

**Files:**
- Create: `db/migrations/000001_init_schema.up.sql`
- Create: `db/migrations/000001_init_schema.down.sql`

**Step 1: Create directory structure**

Run:
```bash
mkdir -p db/migrations
```

**Step 2: Create the up migration**

Create `db/migrations/000001_init_schema.up.sql`:

```sql
CREATE TABLE IF NOT EXISTS urls (
    id SERIAL PRIMARY KEY,
    original TEXT NOT NULL,
    short TEXT NOT NULL UNIQUE,
    title TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP,
    clicks BIGINT NOT NULL DEFAULT 0,
    creator_reference TEXT,
    deleted_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_urls_short ON urls(short);
CREATE INDEX IF NOT EXISTS idx_urls_original ON urls(original);

CREATE TABLE IF NOT EXISTS clicks (
    id SERIAL PRIMARY KEY,
    url_id BIGINT NOT NULL REFERENCES urls(id) ON DELETE CASCADE,
    url_short TEXT NOT NULL,
    ip TEXT NOT NULL,
    location TEXT,
    browser TEXT,
    device TEXT,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_clicks_url_id ON clicks(url_id);
CREATE INDEX IF NOT EXISTS idx_clicks_url_short ON clicks(url_short);

CREATE TABLE IF NOT EXISTS url_history (
    id SERIAL PRIMARY KEY,
    url_id BIGINT NOT NULL REFERENCES urls(id) ON DELETE CASCADE,
    url_short TEXT NOT NULL,
    action TEXT NOT NULL,
    old_value JSONB,
    new_value JSONB,
    modified_at TIMESTAMP NOT NULL DEFAULT NOW(),
    modified_by TEXT
);

CREATE INDEX IF NOT EXISTS idx_url_history_url_id ON url_history(url_id);
CREATE INDEX IF NOT EXISTS idx_url_history_url_short ON url_history(url_short);
```

**Step 3: Create the down migration**

Create `db/migrations/000001_init_schema.down.sql`:

```sql
DROP TABLE IF EXISTS url_history;
DROP TABLE IF EXISTS clicks;
DROP TABLE IF EXISTS urls;
```

**Step 4: Commit**

```bash
git add db/migrations/
git commit -m "feat: add initial database migration matching existing schema"
```

---

### Task 3: Create sqlc query files

**Files:**
- Create: `db/queries/urls.sql`
- Create: `db/queries/clicks.sql`
- Create: `db/queries/url_history.sql`

**Step 1: Create queries directory**

Run:
```bash
mkdir -p db/queries
```

**Step 2: Create urls.sql**

Create `db/queries/urls.sql`:

```sql
-- name: CreateURL :one
INSERT INTO urls (original, short, title, created_at, expires_at, clicks, creator_reference, deleted_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, original, short, title, created_at, expires_at, clicks, creator_reference, deleted_at;

-- name: GetURLByShort :one
SELECT id, original, short, title, created_at, expires_at, clicks, creator_reference, deleted_at
FROM urls
WHERE short = $1 AND deleted_at IS NULL;

-- name: GetURLByOriginal :one
SELECT id, original, short, title, created_at, expires_at, clicks, creator_reference, deleted_at
FROM urls
WHERE original = $1 AND deleted_at IS NULL;

-- name: GetURLsByCreator :many
SELECT id, original, short, title, created_at, expires_at, clicks, creator_reference, deleted_at
FROM urls
WHERE creator_reference = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: IncrementClicks :one
UPDATE urls SET clicks = clicks + 1
WHERE short = $1 AND deleted_at IS NULL
RETURNING clicks;

-- name: SoftDeleteURL :exec
UPDATE urls SET deleted_at = NOW()
WHERE short = $1 AND deleted_at IS NULL;

-- name: SoftDeleteURLWithCreator :exec
UPDATE urls SET deleted_at = NOW()
WHERE short = $1 AND creator_reference = $2 AND deleted_at IS NULL;

-- name: HardDeleteURL :exec
DELETE FROM urls WHERE short = $1;

-- name: UpdateURL :one
UPDATE urls SET original = $1, title = $2, expires_at = $3
WHERE short = $4 AND deleted_at IS NULL
RETURNING id, original, short, title, created_at, expires_at, clicks, creator_reference, deleted_at;

-- name: UpdateURLWithCreator :one
UPDATE urls SET original = $1, title = $2, expires_at = $3
WHERE short = $4 AND creator_reference = $5 AND deleted_at IS NULL
RETURNING id, original, short, title, created_at, expires_at, clicks, creator_reference, deleted_at;
```

**Step 3: Create clicks.sql**

Create `db/queries/clicks.sql`:

```sql
-- name: StoreClick :one
INSERT INTO clicks (url_id, url_short, ip, location, browser, device, timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, url_id, url_short, ip, location, browser, device, timestamp;

-- name: GetClicksByShort :many
SELECT id, url_id, url_short, ip, location, browser, device, timestamp
FROM clicks
WHERE url_short = $1
ORDER BY timestamp DESC;

-- name: HasRecentClick :one
SELECT EXISTS(
    SELECT 1 FROM clicks
    WHERE url_short = $1
    AND ip = $2
    AND browser = $3
    AND device = $4
    AND timestamp > NOW() - INTERVAL '1 hour'
);

-- name: GetTotalClicks :one
SELECT COUNT(*) FROM clicks WHERE url_short = $1;

-- name: GetClicksByBrowser :many
SELECT browser, COUNT(*) AS count FROM clicks
WHERE url_short = $1
GROUP BY browser;

-- name: GetClicksByDevice :many
SELECT device, COUNT(*) AS count FROM clicks
WHERE url_short = $1
GROUP BY device;

-- name: GetClicksByLocation :many
SELECT location, COUNT(*) AS count FROM clicks
WHERE url_short = $1
GROUP BY location;
```

**Step 4: Create url_history.sql**

Create `db/queries/url_history.sql`:

```sql
-- name: LogURLHistory :one
INSERT INTO url_history (url_id, url_short, action, old_value, new_value, modified_at, modified_by)
VALUES ($1, $2, $3, $4, $5, NOW(), $6)
RETURNING id, url_id, url_short, action, old_value, new_value, modified_at, modified_by;
```

**Step 5: Commit**

```bash
git add db/queries/
git commit -m "feat: add sqlc query definitions for urls, clicks, and url_history"
```

---

### Task 4: Create sqlc.yaml and generate code

**Files:**
- Create: `sqlc.yaml`
- Create: `internal/db/sqlc/` (generated)

**Step 1: Create sqlc.yaml**

Create `sqlc.yaml`:

```yaml
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
            go_type:
              import: "encoding/json"
              type: "RawMessage"
          - db_type: "timestamptz"
            go_type:
              import: "time"
              type: "Time"
          - db_type: "timestamp"
            go_type:
              import: "time"
              type: "Time"
```

**Step 2: Create output directory**

Run:
```bash
mkdir -p internal/db/sqlc
```

**Step 3: Generate sqlc code**

Run:
```bash
sqlc generate
```

If sqlc is not installed, install it first:
```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

**Step 4: Verify generated code**

Run:
```bash
ls internal/db/sqlc/
```

Expected output: `db.go`, `models.go`, `urls.sql.go`, `clicks.sql.go`, `url_history.sql.go`

**Step 5: Verify compilation**

Run:
```bash
go build ./...
```

**Step 6: Commit**

```bash
git add sqlc.yaml internal/db/sqlc/
git commit -m "feat: add sqlc config and generate type-safe query code"
```

---

### Task 5: Update Makefile

**Files:**
- Modify: `Makefile`

**Step 1: Add sqlc and migration targets to Makefile**

Append after the existing targets (before the `help` target):

```makefile
# Generate sqlc code
.PHONY: sqlc-generate
sqlc-generate:
	@echo "Generating sqlc code..."
	sqlc generate

# Validate sqlc queries
.PHONY: sqlc-validate
sqlc-validate:
	@echo "Validating sqlc queries..."
	sqlc vet

# Database migrations
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

Also update the help target to include new commands. Add these lines to the help target:

```makefile
	@echo "  sqlc-generate   Generate sqlc code"
	@echo "  sqlc-validate   Validate sqlc queries"
	@echo "  migrate-up     Run database migrations"
	@echo "  migrate-down    Rollback last migration"
	@echo "  migrate-create  Create new migration (requires NAME=)"
	@echo "  migrate-force   Force migration version (requires VERS=)"
```

**Step 2: Commit**

```bash
git add Makefile
git commit -m "feat: add sqlc-generate and migration Makefile targets"
```

---

### Task 6: Create migration helper

**Files:**
- Create: `store/migrate.go`

**Step 1: Create the migration helper**

Create `store/migrate.go`:

```go
package store

import (
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// RunMigrations runs all pending database migrations using the embedded migration files.
func RunMigrations(dbURL string, fs embed.FS) error {
	d, err := iofs.New(fs, "db/migrations")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, dbURL)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
```

**Step 2: Commit**

```bash
git add store/migrate.go
git commit -m "feat: add migration runner helper using golang-migrate"
```

---

### Task 7: Rewrite PostgresRepository to use sqlc

**Files:**
- Modify: `store/postgres_repository.go`

This is the largest task. The `PostgresRepository` struct changes to hold a `*sqlc.Queries` instead of using raw `pool.QueryRow`/`pool.Exec` calls. The `URLRepository` interface remains unchanged.

**Step 1: Rewrite PostgresRepository**

Replace the entire content of `store/postgres_repository.go` with a version that:
1. Imports `sqlc "github.com/fransfilastap/urlshortener/internal/db/sqlc"`
2. Struct holds `queries *sqlc.Queries` (plus pool for migration access)
3. `NewPostgresRepository` creates pgxpool, then creates `sqlc.New(pool)` — keep the retry logic for pool connection
4. Remove `InitSchema` method entirely
5. Each method translates `models.*` ↔ `sqlc.*` and delegates to `queries.*`
6. Keep `Close()` method
7. For `HasRecentClick`, use the sqlc-generated `HasRecentClick` query
8. For `GetClickAnalytics`, call multiple sqlc queries (GetTotalClicks, GetClicksByBrowser, GetClicksByDevice, GetClicksByLocation) and assemble the result

Key translation patterns:
- `models.URL` → `sqlc.CreateURLParams` for inserts
- `sqlc.Url` → `models.URL` for reads
- `models.Click` → `sqlc.StoreClickParams` for inserts
- `sqlc.Click` → `models.Click` for reads
- For `jsonb` fields (old_value, new_value in url_history), convert between `json.RawMessage` and `interface{}` via `json.Marshal`/`json.Unmarshal`

Note: The `fmt.Printf` debug statements in the current `NewPostgresRepository` and `StoreClick` should be removed (they were debug-only and the project uses zerolog).

**Step 2: Verify compilation**

Run:
```bash
go build ./...
```

**Step 3: Commit**

```bash
git add store/postgres_repository.go
git commit -m "refactor: rewrite PostgresRepository to use sqlc-generated queries"
```

---

### Task 8: Update main.go to use migrations

**Files:**
- Modify: `main.go`

**Step 1: Add embed and migration call**

Modify `main.go`:
1. Add `embed` import
2. Add `//go:embed db/migrations` directive to embed migration files
3. Replace `db.InitSchema(context.Background())` with `store.RunMigrations(cfg.PostgresURL, migrationsFS)`
4. Remove the retry logic from `NewPostgresRepository` — instead, keep the pool connection retry but after successful connection, run migrations

Important: The embed directive needs to reference files relative to the module root. You may need a separate file for the embed, or structure it so the `db/migrations` directory is accessible. Consider creating `migrations.go` in the root:

Create `migrations.go`:
```go
package main

import "embed"

//go:embed db/migrations
var migrationsFS embed.FS
```

Then in `main.go`, after creating the repository, call:
```go
if err := store.RunMigrations(cfg.PostgresURL, migrationsFS); err != nil {
    log.Fatal().Err(err).Msg("Failed to run database migrations")
}
```

And remove the `db.InitSchema(context.Background())` call.

**Step 2: Simplify NewPostgresRepository — remove retry log noise**

Since `NewPostgresRepository` no longer needs `InitSchema`, simplify it to just create the pool and return. Keep the retry logic for connection but remove the `fmt.Printf` debug prints (use zerolog instead).

**Step 3: Verify compilation**

Run:
```bash
go build ./...
```

**Step 4: Commit**

```bash
git add main.go migrations.go store/migrate.go
git commit -m "refactor: use golang-migrate instead of InitSchema for DB setup"
```

---

### Task 9: Write sqlc integration tests

**Files:**
- Create: `store/sqlc_integration_test.go`

**Step 1: Create the test file**

Create `store/sqlc_integration_test.go` with integration tests that:
1. Use testcontainers Postgres (reuse `SetupPostgresContainer`)
2. Run migrations (not `InitSchema`) to validate migrations work
3. Test each sqlc query method directly via `sqlc.New(pool)`
4. Cover: CreateURL, GetURLByShort, GetURLByOriginal, GetURLsByCreator, IncrementClicks, SoftDeleteURL, UpdateURL, UpdateURLWithCreator, StoreClick, GetClicksByShort, HasRecentClick, GetClickAnalytics queries, LogURLHistory

Test structure:
```go
//go:build !short

package store

import (
    "context"
    "embed"
    "testing"
    "time"

    sqlc "github.com/fransfilastap/urlshortener/internal/db/sqlc"
    "github.com/fransfilastap/urlshortener/models"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    // embed for migrations
)

//go:embed db/migrations
var testMigrationsFS embed.FS

func TestSqlcQueries_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    ctx := context.Background()
    pgContainer, err := SetupPostgresContainer(ctx)
    require.NoError(t, err)
    defer pgContainer.Teardown(ctx)
    
    // Run migrations
    err = RunMigrations(pgContainer.URI, testMigrationsFS)
    require.NoError(t, err)
    
    // Connect to DB
    repo, err := NewPostgresRepository(pgContainer.URI)
    require.NoError(t, err)
    defer repo.Close()
    
    // Get underlying pool for direct sqlc access
    queries := sqlc.New(repo.Pool())
    
    // ... tests for each query
}
```

Note: Need to expose `Pool()` method on `PostgresRepository` for test access. Add a `Pool() *pgxpool.Pool` method.

**Step 2: Run tests**

Run:
```bash
go test -v ./store/ -run TestSqlcQueries_Integration
```

**Step 3: Commit**

```bash
git add store/sqlc_integration_test.go store/postgres_repository.go
git commit -m "test: add sqlc query integration tests"
```

---

### Task 10: Update PostgresRepository integration tests

**Files:**
- Modify: `store/postgres_repository_test.go`

**Step 1: Update test setup to use migrations**

Replace `repo.InitSchema(ctx)` with `store.RunMigrations(pgContainer.URI, migrationsFS)`. Also remove the `DELETE FROM urls` cleanup hack — use transactions or proper test isolation instead.

Key changes:
1. Add `embed` import and `//go:embed db/migrations` directive
2. Replace `repo.InitSchema(ctx)` with `store.RunMigrations(pgContainer.URI, testMigrationsFS)`
3. Remove `_, err = repo.pool.Exec(ctx, "DELETE FROM urls")` cleanup — instead, use `t.Cleanup` with DELETE or transaction rollback
4. Keep the same test cases (Create, GetByShort, GetByOriginal, etc.)

**Step 2: Run tests**

Run:
```bash
go test -v ./store/ -run TestPostgresRepository_Integration
```

**Step 3: Commit**

```bash
git add store/postgres_repository_test.go
git commit -m "test: update PostgresRepository integration tests to use migrations"
```

---

### Task 11: Final verification and cleanup

**Files:**
- Modify: `store/postgres_repository.go` (remove Pool() if not needed outside tests)

**Step 1: Run all tests**

Run:
```bash
go test -v ./...
```

**Step 2: Run linter**

Run:
```bash
make lint
```

**Step 3: Verify sqlc generate is idempotent**

Run:
```bash
make sqlc-generate
git diff internal/db/sqlc/
```

Expected: No changes (already generated code matches)

**Step 4: Verify build**

Run:
```bash
make build
```

**Step 5: Final commit**

```bash
git add -A
git commit -m "chore: final cleanup and verification for sqlc/migration integration"
```

---

## Summary of all tasks

1. Add dependencies (golang-migrate, sqlc tool)
2. Create migration files (000001_init_schema.up/down.sql)
3. Create sqlc query files (urls.sql, clicks.sql, url_history.sql)
4. Create sqlc.yaml and generate code
5. Update Makefile with sqlc-generate, migrate-up/down/create/force targets
6. Create migration helper (store/migrate.go)
7. Rewrite PostgresRepository to use sqlc
8. Update main.go to use migrations instead of InitSchema
9. Write sqlc integration tests
10. Update PostgresRepository integration tests
11. Final verification and cleanup