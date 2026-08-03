package db

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Migrate applies every pending SQL migration in migrationsPath. It is safe
// to call on every boot: golang-migrate no-ops when the schema is already
// current. Controlled by DATABASE_AUTO_MIGRATE so a production deployment
// can instead run migrations as an explicit release step if preferred.
func Migrate(databaseURL, migrationsPath string) error {
	m, err := migrate.New(migrationsPath, databaseURL)
	if err != nil {
		return fmt.Errorf("db: init migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("db: apply migrations: %w", err)
	}
	return nil
}
