package user_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/afadnan/restaurantflow/services/api/internal/config"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/database"
)

func newOwnerCleanupPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}

	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" ||
		cfg.Database.Name != "restaurantflow_test" {
		t.Fatal("owner cleanup requires restaurantflow_test")
	}

	user := os.Getenv("USER_TEST_CLEANUP_DATABASE_USER")
	password := os.Getenv("USER_TEST_CLEANUP_DATABASE_PASSWORD")

	if user != "restaurantflow_test_cleanup" || password == "" {
		t.Fatal("dedicated owner cleanup credentials required")
	}

	cfg.Database.User = user
	cfg.Database.Password = password

	ctx, cancel := context.WithTimeout(
		context.Background(), 10*time.Second,
	)
	defer cancel()

	pool, err := database.NewPostgresPool(ctx, cfg.Database)
	if err != nil {
		t.Fatal(err)
	}

	var currentUser, currentDatabase string
	err = pool.QueryRow(
		ctx,
		"SELECT current_user, current_database()",
	).Scan(&currentUser, &currentDatabase)

	if err != nil ||
		currentUser != "restaurantflow_test_cleanup" ||
		currentDatabase != "restaurantflow_test" {
		pool.Close()
		t.Fatalf("unexpected cleanup connection: %v", err)
	}

	t.Cleanup(pool.Close)
	return pool
}

func cleanupRegisteredOwner(
	t *testing.T,
	pool *pgxpool.Pool,
	tenantID uuid.UUID,
	ownerID uuid.UUID,
) {
	t.Helper()

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(
			context.Background(), 10*time.Second,
		)
		defer cancel()

		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Errorf("begin cleanup: %v", err)
			return
		}
		defer func() { _ = tx.Rollback(context.Background()) }()

		_, err = tx.Exec(
			ctx,
			"SELECT set_config('app.tenant_id', $1, true)",
			tenantID.String(),
		)
		if err != nil {
			t.Errorf("set cleanup tenant: %v", err)
			return
		}

		result, err := tx.Exec(ctx, `
			DELETE FROM public.users
			WHERE id = $1 AND tenant_id = $2
		`, ownerID, tenantID)
		if err != nil {
			t.Errorf("delete test owner: %v", err)
			return
		}
		if result.RowsAffected() != 1 {
			t.Errorf("expected one owner deleted, got %d", result.RowsAffected())
			return
		}

		result, err = tx.Exec(ctx, `
			DELETE FROM public.tenants
			WHERE id = $1
		`, tenantID)
		if err != nil {
			t.Errorf("delete test tenant: %v", err)
			return
		}
		if result.RowsAffected() != 1 {
			t.Errorf("expected one tenant deleted, got %d", result.RowsAffected())
			return
		}

		if err := tx.Commit(ctx); err != nil {
			t.Errorf("commit cleanup: %v", err)
			return
		}

		// Verify the tenant no longer exists.
		var count int
		err = pool.QueryRow(
			ctx,
			"SELECT COUNT(*) FROM public.tenants WHERE id = $1",
			tenantID,
		).Scan(&count)

		if err != nil || count != 0 {
			t.Errorf(
				"%s",
				fmt.Sprintf("cleanup verification failed: count=%d err=%v", count, err),
			)
		}
	})
}
