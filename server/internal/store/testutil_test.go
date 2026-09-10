package store

import (
	"net/url"
	"strings"
	"testing"
)

// dbNameFromDSN pulls the database name out of a "postgres://" DSN (the URL
// path, sans leading slash). Empty string if the DSN can't be parsed.
func dbNameFromDSN(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(u.Path, "/")
}

// requireTestDB fails the test unless the DSN points at a database whose name
// ends in "_test". The store suite runs destructive setup (TRUNCATE in
// newStore, DROP SCHEMA in openPreMigration002Store) straight against
// TEST_DATABASE_URL, so this is the only thing stopping a stray prod DSN in
// the environment from wiping prod.
func requireTestDB(t *testing.T, dsn string) {
	t.Helper()
	if name := dbNameFromDSN(dsn); !strings.HasSuffix(name, "_test") {
		t.Fatalf("refusing to run destructive test setup on non-test database %q (from %q)", name, dsn)
	}
}
