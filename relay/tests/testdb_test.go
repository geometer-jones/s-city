package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func openIntegrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	dbURL := strings.TrimSpace(os.Getenv("SCITY_INTEGRATION_DB_URL"))
	if dbURL == "" {
		dbURL = "postgres://s_city:s_city@localhost:5432/s_city?sslmode=disable"
	}

	adminCfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		t.Skipf("skip integration tests: cannot parse DB URL: %v", err)
	}
	adminPool, err := pgxpool.NewWithConfig(ctx, adminCfg)
	if err != nil {
		t.Skipf("skip integration tests: cannot open DB pool: %v", err)
	}
	if err := adminPool.Ping(ctx); err != nil {
		adminPool.Close()
		t.Skipf("skip integration tests: DB unavailable at %q: %v", dbURL, err)
	}

	schema := fmt.Sprintf("itest_%d", time.Now().UnixNano())
	if _, err := adminPool.Exec(ctx, fmt.Sprintf(`CREATE SCHEMA "%s"`, schema)); err != nil {
		adminPool.Close()
		t.Fatalf("create schema %s: %v", schema, err)
	}

	testCfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		adminPool.Close()
		t.Fatalf("parse DB URL for test pool: %v", err)
	}
	if testCfg.ConnConfig.RuntimeParams == nil {
		testCfg.ConnConfig.RuntimeParams = make(map[string]string)
	}
	testCfg.ConnConfig.RuntimeParams["search_path"] = schema

	testPool, err := pgxpool.NewWithConfig(ctx, testCfg)
	if err != nil {
		adminPool.Close()
		t.Fatalf("open test DB pool: %v", err)
	}
	if err := testPool.Ping(ctx); err != nil {
		testPool.Close()
		adminPool.Close()
		t.Fatalf("ping test DB pool: %v", err)
	}

	if err := applyMigrationsForTests(ctx, testPool); err != nil {
		testPool.Close()
		_, _ = adminPool.Exec(ctx, fmt.Sprintf(`DROP SCHEMA "%s" CASCADE`, schema))
		adminPool.Close()
		t.Fatalf("apply migrations: %v", err)
	}

	t.Cleanup(func() {
		testPool.Close()
		_, _ = adminPool.Exec(ctx, fmt.Sprintf(`DROP SCHEMA "%s" CASCADE`, schema))
		adminPool.Close()
	})

	return testPool
}

func applyMigrationsForTests(ctx context.Context, pool *pgxpool.Pool) error {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("resolve test file path")
	}
	relayRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), ".."))
	migrationDir := filepath.Join(relayRoot, "src", "storage", "migrations")

	entries, err := os.ReadDir(migrationDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".sql" {
			files = append(files, filepath.Join(migrationDir, entry.Name()))
		}
	}
	sort.Strings(files)

	for _, path := range files {
		sql, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", path, err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("exec migration %s: %w", path, err)
		}
	}
	return nil
}
