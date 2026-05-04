# Migration Production Safety Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add production safety guards to golang-migrate — config-gated auto-migration, baseline detection, a `migrate` subcommand, and Down() protection.

**Architecture:** Add `AutoMigrate` and `ExpectedSchemaVersion` config fields. Refactor `RunMigrations` to detect baseline scenarios. Add a `migrate` subcommand in `main.go` for explicit migration control. Disable auto-migration in production Docker config.

**Tech Stack:** Go, golang-migrate/v4, pgx/v5, Docker Compose

---

### Task 1: Add config fields for AutoMigrate and ExpectedSchemaVersion

**Files:**
- Modify: `internal/config/config.go:12-29` (Config struct)
- Modify: `internal/config/config.go:37-50` (NewConfig function)

**Step 1: Add fields to Config struct**

In `internal/config/config.go`, add two new fields to the `Config` struct after `SessionMaxAge`:

```go
AutoMigrate            bool
ExpectedSchemaVersion  int
```

**Step 2: Populate fields in NewConfig**

In the `NewConfig` return struct, add:

```go
AutoMigrate:           getEnvAsBool("AUTO_MIGRATE", true),
ExpectedSchemaVersion: getEnvAsInt("EXPECTED_SCHEMA_VERSION", 0),
```

**Step 3: Add getEnvAsBool helper**

Add a new helper function after `getEnvAsDuration`:

```go
func getEnvAsBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		parsed, err := strconv.ParseBool(value)
		if err == nil {
			return parsed
		}
	}
	return defaultValue
}
```

**Step 4: Run tests**

Run: `go vet ./internal/config/`
Expected: No errors

**Step 5: Commit**

```bash
git add internal/config/config.go
git commit -m "feat: add AutoMigrate and ExpectedSchemaVersion config fields"
```

---

### Task 2: Add baseline detection and version check to migrate.go

**Files:**
- Modify: `internal/repository/migrate.go`

**Step 1: Add baseline detection and version check functions**

Add the following after the `RunMigrations` function in `internal/repository/migrate.go`. Also add a warning comment about Down() and add the new helper functions:

Add imports for `"database/sql"` and `"github.com/lib/pq"` (for standalone check queries). But actually, we should use the pgx driver already in the project to check for baseline. We'll use `migrate.NewWithSourceInstance` to get the version, and if that fails with "no such table", we query directly.

Add a new helper that creates a migrate instance (reusable):

```go
// WARNING: Do NOT add m.Down() capability to this codebase.
// Production rollbacks should be new forward migrations, not destructive reversions.
// Running m.Down() would DROP all tables in production.

func newMigrateInstance(dbURL string, migrationsFS fs.FS) (*migrate.Migrate, error) {
	d, err := iofs.New(migrationsFS, "db/migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to create migration source: %w", err)
	}

	migrateURL := dbURL
	if strings.HasPrefix(dbURL, "postgres://") {
		migrateURL = "pgx5://" + strings.TrimPrefix(dbURL, "postgres://")
	} else if strings.HasPrefix(dbURL, "postgresql://") {
		migrateURL = "pgx5://" + strings.TrimPrefix(dbURL, "postgresql://")
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, migrateURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}
	return m, nil
}
```

Refactor `RunMigrations` to use it:

```go
func RunMigrations(dbURL string, migrationsFS fs.FS) error {
	m, err := newMigrateInstance(dbURL, migrationsFS)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
```

Add `CheckBaseline` function:

```go
func CheckBaseline(dbURL string) error {
	m, err := newMigrateInstance(dbURL, MigrationsFS{})
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	_, _, err = m.Version()
	if err == nil {
		return nil
	}

	if !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("failed to check migration version: %w", err)
	}

	return fmt.Errorf("database has no schema_migrations table; if this is an existing database, run './urlshortener migrate force <version>' to baseline")
}
```

Add `GetSchemaVersion` function:

```go
func GetSchemaVersion(dbURL string, migrationsFS fs.FS) (uint, bool, error) {
	m, err := newMigrateInstance(dbURL, migrationsFS)
	if err != nil {
		return 0, false, err
	}
	defer m.Close()

	version, dirty, err := m.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			return 0, false, nil
		}
		return 0, false, err
	}
	return version, dirty, nil
}
```

Add `ForceVersion` function:

```go
func ForceVersion(dbURL string, migrationsFS fs.FS, version int) error {
	m, err := newMigrateInstance(dbURL, migrationsFS)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Force(version); err != nil {
		return fmt.Errorf("failed to force migration version to %d: %w", version, err)
	}

	return nil
}
```

Add `ValidateSchemaVersion` function:

```go
func ValidateSchemaVersion(dbURL string, migrationsFS fs.FS, expectedVersion int) error {
	version, dirty, err := GetSchemaVersion(dbURL, migrationsFS)
	if err != nil {
		return fmt.Errorf("failed to get schema version: %w", err)
	}

	if dirty {
		return fmt.Errorf("database schema is dirty; manual intervention required")
	}

	if int(version) < expectedVersion {
		return fmt.Errorf("schema version %d is behind expected %d; run migrations manually or set AUTO_MIGRATE=true", version, expectedVersion)
	}

	if int(version) > expectedVersion {
		return fmt.Errorf("schema version %d is ahead of expected %d; possible downgrade attempt", version, expectedVersion)
	}

	return nil
}
```

**Step 2: Run vet**

Run: `go vet ./internal/repository/`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/repository/migrate.go
git commit -m "feat: add baseline detection, version check, and force version to migrate"
```

---

### Task 3: Add migrate subcommand to main.go

**Files:**
- Modify: `cmd/urlshortener/main.go`

**Step 1: Add subcommand routing in main.go**

Replace the `main()` function to detect subcommands, and extract server start into a `runServer()` function.

Add `"fmt"` and `"os"` to imports (os is already there).

New `main()`:

```go
func main() {
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		runMigrateCommand()
		return
	}

	runServer()
}
```

Add `runServer()` (current main logic, with conditional migration):

```go
func runServer() {
	cfg := config.NewConfig()

	logger.InitLogger(cfg.LogLevel, cfg.LogFormat)

	if cfg.AutoMigrate {
		if err := repository.RunMigrations(cfg.PostgresURL, urlshortener.MigrationsFS); err != nil {
			log.Fatal().Err(err).Msg("Failed to run database migrations")
		}
	} else {
		log.Info().Msg("Migrations skipped (AUTO_MIGRATE=false)")
	}

	if cfg.ExpectedSchemaVersion > 0 {
		if err := repository.ValidateSchemaVersion(cfg.PostgresURL, urlshortener.MigrationsFS, cfg.ExpectedSchemaVersion); err != nil {
			log.Fatal().Err(err).Msg("Schema version check failed")
		}
		log.Info().Int("version", cfg.ExpectedSchemaVersion).Msg("Schema version validated")
	}

	db, err := repository.NewPostgresRepository(cfg.PostgresURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	// ... rest of current main.go unchanged ...

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server shutdown failed")
	}

	log.Info().Msg("Server gracefully stopped")
}
```

Add `runMigrateCommand()`:

```go
func runMigrateCommand() {
	cfg := config.NewConfig()

	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: urlshortener migrate <up|version|force> [version]")
		os.Exit(1)
	}

	switch os.Args[2] {
	case "up":
		if err := repository.RunMigrations(cfg.PostgresURL, urlshortener.MigrationsFS); err != nil {
			fmt.Fprintf(os.Stderr, "Error running migrations: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Migrations applied successfully")
	case "version":
		version, dirty, err := repository.GetSchemaVersion(cfg.PostgresURL, urlshortener.MigrationsFS)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting schema version: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Schema version: %d (dirty: %v)\n", version, dirty)
	case "force":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "Usage: urlshortener migrate force <version>")
			os.Exit(1)
		}
		version, err := strconv.Atoi(os.Args[3])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid version number: %v\n", err)
			os.Exit(1)
		}
		if err := repository.ForceVersion(cfg.PostgresURL, urlshortener.MigrationsFS, version); err != nil {
			fmt.Fprintf(os.Stderr, "Error forcing version: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Forced schema version to %d\n", version)
	default:
		fmt.Fprintf(os.Stderr, "Unknown migrate command: %s\n", os.Args[2])
		fmt.Fprintln(os.Stderr, "Usage: urlshortener migrate <up|version|force> [version]")
		os.Exit(1)
	}
}
```

Add `"strconv"` to imports.

**Step 2: Run vet**

Run: `go vet ./cmd/urlshortener/`
Expected: No errors

**Step 3: Commit**

```bash
git add cmd/urlshortener/main.go
git commit -m "feat: add migrate subcommand and config-gated auto-migration"
```

---

### Task 4: Update docker-compose.yml for production safety

**Files:**
- Modify: `docker-compose.yml`

**Step 1: Add AUTO_MIGRATE and EXPECTED_SCHEMA_VERSION to production service**

In the `app` service environment section, add these lines after `LOG_FORMAT`:

```yaml
      - AUTO_MIGRATE=${AUTO_MIGRATE:-false}
      - EXPECTED_SCHEMA_VERSION=${EXPECTED_SCHEMA_VERSION:-1}
```

The dev `app-dev` service intentionally does NOT get these — it inherits the defaults (`AUTO_MIGRATE=true`, `EXPECTED_SCHEMA_VERSION=0`).

**Step 2: Commit**

```bash
git add docker-compose.yml
git commit -m "feat: disable auto-migration in production docker-compose"
```

---

### Task 5: Integration verification

**Step 1: Run go vet across entire project**

Run: `go vet ./...`
Expected: No errors

**Step 2: Run short tests**

Run: `go test -short ./...`
Expected: All tests pass (cache integration test may fail without Docker — that's expected)

**Step 3: Build the binary**

Run: `go build -o /tmp/urlshortener ./cmd/urlshortener`
Expected: Builds successfully (may warn about missing web/dist embed but that's fine)

**Step 4: Verify subcommand help**

Run: `/tmp/urlshortener migrate` (should print usage and exit with code 1)
Expected: Usage message printed to stderr

**Step 5: Final commit**

If any minor fixes were needed, commit them. Otherwise, no action needed.