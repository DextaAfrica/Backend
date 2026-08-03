//go:build integration

// Run with: go test -tags=integration ./internal/db/... (requires DATABASE_URL
// pointing at a real, disposable Postgres instance — see the "test" job in
// .github/workflows/ci.yml). Excluded from the default `go test ./...` run
// since it needs a live database, not a fake.
package db

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/DextaAfrica/Backend/internal/config"
)

func TestMigrations_UpAndDownAreReversible(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping migration integration test")
	}

	pool, err := Connect(context.Background(), config.Database{
		URL: databaseURL, MaxConns: 5, MinConns: 1, ConnMaxLifetime: time.Hour,
	})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	if err := Migrate(databaseURL, "file://migrations"); err != nil {
		t.Fatalf("apply migrations up: %v", err)
	}

	m, err := migrate.New("file://migrations", databaseURL)
	if err != nil {
		t.Fatalf("init migrator: %v", err)
	}
	defer m.Close()

	if err := m.Down(); err != nil {
		t.Fatalf("apply migrations down: %v", err)
	}

	// Re-applying up after a full down must succeed cleanly, proving the
	// down migration actually removed everything the up migration created.
	if err := Migrate(databaseURL, "file://migrations"); err != nil {
		t.Fatalf("re-apply migrations up: %v", err)
	}
}
