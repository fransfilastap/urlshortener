package repository

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// WARNING: Do NOT add m.Down() capability to this codebase.
// Production rollbacks should be new forward migrations, not destructive reversions.
// Running m.Down() would DROP all tables in production.

type MigrationsFS struct {
	fs.FS
}

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