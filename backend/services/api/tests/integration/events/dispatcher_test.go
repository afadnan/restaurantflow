package events_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"

	"github.com/afadnan/restaurantflow/services/api/internal/config"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/database"
	db "github.com/afadnan/restaurantflow/services/api/internal/platform/postgres/db"
)

func newWorkerIntegrationDB(t *testing.T) *db.Queries {
	t.Helper()

	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("set RUN_INTEGRATION_TESTS=1 to run PostgreSQL integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := config.Load()
	require.NoError(t, err)

	if user := os.Getenv("OUTBOX_TEST_DATABASE_USER"); user != "" {
		cfg.Database.User = user
	}

	if password := os.Getenv("OUTBOX_TEST_DATABASE_PASSWORD"); password != "" {
		cfg.Database.Password = password
	}

	pool, err := database.NewPostgresPool(ctx, cfg.Database)
	require.NoError(t, err)

	t.Cleanup(func() {
		pool.Close()
	})

	require.NoError(t, pool.Ping(ctx))

	return db.New(pool)
}

func insertOutboxEvent(
	t *testing.T,
	q *db.Queries,
	eventID uuid.UUID,
	tenantID uuid.UUID,
) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		ID:          pgtype.UUID{Bytes: eventID, Valid: true},
		TenantID:    pgtype.UUID{Bytes: tenantID, Valid: true},
		AggregateID: pgtype.UUID{Bytes: uuid.New(), Valid: true},
		EventType:   "test.event",
		Payload:     []byte(`{}`),
		OccurredAt:  time.Now().UTC(),
	})
	require.NoError(t, err)
}

func TestConcurrentOutboxClaimsDoNotClaimSameEvent(t *testing.T) {
	q := newWorkerIntegrationDB(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	eventID := uuid.New()
	tenantID := uuid.New()

	insertOutboxEvent(t, q, eventID, tenantID)

	type result struct {
		rows []db.ClaimOutboxEventsRow
		err  error
	}

	results := make(chan result, 2)
	start := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(2)

	claim := func() {
		defer wg.Done()

		<-start

		token := uuid.New()

		rows, err := q.ClaimOutboxEvents(
			ctx,
			db.ClaimOutboxEventsParams{
				ClaimToken: pgtype.UUID{
					Bytes: token,
					Valid: true,
				},
				LeaseSeconds: 30,
				BatchSize:    1,
			},
		)

		results <- result{
			rows: rows,
			err:  err,
		}
	}

	go claim()
	go claim()

	close(start)

	wg.Wait()
	close(results)

	var claimed int

	for result := range results {
		require.NoError(t, result.err)

		for _, row := range result.rows {
			if row.ID.Bytes == eventID {
				claimed++
			}
		}
	}

	require.Equal(
		t,
		1,
		claimed,
		"the same outbox event must not be claimed by two concurrent workers",
	)
}

func TestOutboxClaimSkipsPublishedEvent(t *testing.T) {
	q := newWorkerIntegrationDB(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	eventID := uuid.New()
	tenantID := uuid.New()

	insertOutboxEvent(t, q, eventID, tenantID)

	_, err := q.MarkOutboxEventPublished(
		ctx,
		db.MarkOutboxEventPublishedParams{
			ID:         eventIDPGUUID(eventID),
			ClaimToken: eventIDPGUUID(uuid.New()),
		},
	)
	require.NoError(t, err)

	rows, err := q.ClaimOutboxEvents(
		ctx,
		db.ClaimOutboxEventsParams{
			ClaimToken:   eventIDPGUUID(uuid.New()),
			LeaseSeconds: 30,
			BatchSize:    1,
		},
	)
	require.NoError(t, err)

	require.Empty(t, rows)
}

func TestOutboxClaimReclaimsExpiredLease(t *testing.T) {
	q := newWorkerIntegrationDB(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	eventID := uuid.New()
	tenantID := uuid.New()
	firstToken := uuid.New()

	insertOutboxEvent(t, q, eventID, tenantID)

	_, err := q.ClaimOutboxEvents(
		ctx,
		db.ClaimOutboxEventsParams{
			ClaimToken:   eventIDPGUUID(firstToken),
			LeaseSeconds: 1,
			BatchSize:    1,
		},
	)
	require.NoError(t, err)

	_, err = q.MarkOutboxEventFailed(
		ctx,
		db.MarkOutboxEventFailedParams{
			ID:         eventIDPGUUID(eventID),
			ClaimToken: eventIDPGUUID(firstToken),
			LastError: pgtype.Text{
				String: "test",
				Valid:  true,
			},
		},
	)
	require.NoError(t, err)

	rows, err := q.ClaimOutboxEvents(
		ctx,
		db.ClaimOutboxEventsParams{
			ClaimToken:   eventIDPGUUID(uuid.New()),
			LeaseSeconds: 30,
			BatchSize:    1,
		},
	)
	require.NoError(t, err)

	require.Len(t, rows, 1)
	require.Equal(t, eventID, rows[0].ID.Bytes)
}

func eventIDPGUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: id,
		Valid: true,
	}
}
