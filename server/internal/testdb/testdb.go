// Package testdb opens a migrated, throwaway SQLite database for tests.
package testdb

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/canavan-a/super-atlas/server/internal/db"
)

func Open(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()
	conn, err := db.Connect(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := db.Migrate(ctx, conn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return conn
}
