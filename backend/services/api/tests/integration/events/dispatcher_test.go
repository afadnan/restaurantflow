package events_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/afadnan/restaurantflow/services/api/internal/config"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/database"
	postgresdb "github.com/afadnan/restaurantflow/services/api/internal/platform/postgres/db"
)

type dispatcherIntegrationDB struct {
	appPool    *pgxpool.Pool
	workerPool *pgxpool.Pool
}

func newDispatcherIntegrationDB(t *testing.T) *dispatcherIntegrationDB {
	t.Helper()

	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip(
			"set RUN_INTEGRATION_TESTS=1 to run PostgreSQL integration tests",
		)
	}

	workerUser := os.Getenv("OUTBOX_TEST_DATABASE_USER")
	workerPassword := os.Getenv("OUTBOX_TEST_DATABASE_PASSWORD")

	if workerUser == "" || workerPassword == "" {
		t.Fatal(
			"OUTBOX_TEST_DATABASE_USER and OUTBOX_TEST_DATABASE_PASSWORD are required",
		)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	cfg, err := config.Load()
	require.NoError(t, err)

	appPool, err := database.NewPostgresPool(ctx, cfg.Database)
	require.NoError(t, err)

	workerCfg := cfg.Database
	workerCfg.User = workerUser
	workerCfg.Password = workerPassword

	workerPool, err := database.NewPostgresPool(ctx, workerCfg)
	require.NoError(t, err)

	t.Cleanup(func() {
		workerPool.Close()
		appPool.Close()
	})

	require.NoError(t, appPool.Ping(ctx))
	require.NoError(t, workerPool.Ping(ctx))

	return &dispatcherIntegrationDB{
		appPool:    appPool,
		workerPool: workerPool,
	}
}

func resetOutboxEvents(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Fatal("refusing to reset outbox_events outside integration tests")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	_, err := pool.Exec(
		ctx,
		`TRUNCATE TABLE outbox_events`,
	)

	require.NoError(t, err)
}

func createOutboxTenant(
	t *testing.T,
	pool *pgxpool.Pool,
) uuid.UUID {
	t.Helper()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	tenantID := uuid.New()

	_, err := pool.Exec(
		ctx,
		`
		INSERT INTO tenants (
			id,
			name,
			slug
		)
		VALUES ($1, $2, $3)
		`,
		tenantID,
		"Outbox Integration Tenant",
		"outbox-integration-"+tenantID.String(),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		cleanupOutboxTenant(t, pool, tenantID)
	})

	return tenantID
}

func cleanupOutboxTenant(
	t *testing.T,
	pool *pgxpool.Pool,
	tenantID uuid.UUID,
) {
	t.Helper()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	_, err := pool.Exec(
		ctx,
		`DELETE FROM tenants WHERE id = $1`,
		tenantID,
	)

	require.NoError(t, err)
}

func createTestOutboxEvent(
	t *testing.T,
	pool *pgxpool.Pool,
	tenantID uuid.UUID,
) uuid.UUID {
	t.Helper()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	_, err = tx.Exec(
		ctx,
		`SELECT set_config('app.tenant_id', $1, true)`,
		tenantID.String(),
	)
	require.NoError(t, err)

	eventID := uuid.New()
	aggregateID := uuid.New()

	queries := postgresdb.New(tx)

	err = queries.CreateOutboxEvent(
		ctx,
		postgresdb.CreateOutboxEventParams{
			ID: pgtype.UUID{
				Bytes: eventID,
				Valid: true,
			},
			TenantID: pgtype.UUID{
				Bytes: tenantID,
				Valid: true,
			},
			AggregateID: pgtype.UUID{
				Bytes: aggregateID,
				Valid: true,
			},
			EventType: "order.created",
			Payload: []byte(
				`{"event_id":"` + eventID.String() + `"}`,
			),
			OccurredAt: pgtype.Timestamptz{
				Time:  time.Now().UTC(),
				Valid: true,
			},
		},
	)

	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))

	return eventID
}

func claimEvents(
	t *testing.T,
	pool *pgxpool.Pool,
	claimToken uuid.UUID,
	leaseSeconds int32,
	batchSize int32,
) []postgresdb.ClaimOutboxEventsRow {
	t.Helper()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	queries := postgresdb.New(pool)

	rows, err := queries.ClaimOutboxEvents(
		ctx,
		postgresdb.ClaimOutboxEventsParams{
			ClaimToken: pgtype.UUID{
				Bytes: claimToken,
				Valid: true,
			},
			LeaseSeconds: leaseSeconds,
			BatchSize:    batchSize,
		},
	)

	require.NoError(t, err)

	return rows
}

func readOutboxState(
	t *testing.T,
	pool *pgxpool.Pool,
	eventID uuid.UUID,
) (
	publishedAt *time.Time,
	claimedAt *time.Time,
	claimToken *uuid.UUID,
	attempts int32,
) {
	t.Helper()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	var published pgtype.Timestamptz
	var claimed pgtype.Timestamptz
	var token pgtype.UUID

	err := pool.QueryRow(
		ctx,
		`
		SELECT
			published_at,
			claimed_at,
			claim_token,
			attempts
		FROM outbox_events
		WHERE id = $1
		`,
		eventID,
	).Scan(
		&published,
		&claimed,
		&token,
		&attempts,
	)

	require.NoError(t, err)

	if published.Valid {
		value := published.Time
		publishedAt = &value
	}

	if claimed.Valid {
		value := claimed.Time
		claimedAt = &value
	}

	if token.Valid {
		value := uuid.UUID(token.Bytes)
		claimToken = &value
	}

	return
}

func TestOutboxClaim_ConcurrentWorkersClaimEventOnce(t *testing.T) {
	db := newDispatcherIntegrationDB(t)
	resetOutboxEvents(t, db.workerPool)

	tenantID := createOutboxTenant(
		t,
		db.appPool,
	)

	eventID := createTestOutboxEvent(
		t,
		db.appPool,
		tenantID,
	)

	const (
		leaseSeconds int32 = 60
		batchSize    int32 = 1
	)

	type result struct {
		rows []postgresdb.ClaimOutboxEventsRow
		err  error
	}

	results := make(chan result, 2)
	start := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(2)

	claim := func() {
		defer wg.Done()

		<-start

		ctx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()

		token := uuid.New()

		rows, err := postgresdb.New(
			db.workerPool,
		).ClaimOutboxEvents(
			ctx,
			postgresdb.ClaimOutboxEventsParams{
				ClaimToken: pgtype.UUID{
					Bytes: token,
					Valid: true,
				},
				LeaseSeconds: leaseSeconds,
				BatchSize:    batchSize,
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

	totalClaims := 0

	for result := range results {
		require.NoError(t, result.err)

		for _, row := range result.rows {
			require.Equal(
				t,
				eventID,
				uuid.UUID(row.ID.Bytes),
			)

			totalClaims++
		}
	}

	require.Equal(
		t,
		1,
		totalClaims,
	)

	publishedAt, claimedAt, claimToken, attempts := readOutboxState(
		t,
		db.workerPool,
		eventID,
	)

	require.Nil(t, publishedAt)
	require.NotNil(t, claimedAt)
	require.NotNil(t, claimToken)
	require.Equal(t, int32(0), attempts)
}

func TestOutboxClaim_DistributesEventsAcrossWorkers(t *testing.T) {
	db := newDispatcherIntegrationDB(t)
	resetOutboxEvents(t, db.workerPool)

	tenantID := createOutboxTenant(
		t,
		db.appPool,
	)

	eventA := createTestOutboxEvent(
		t,
		db.appPool,
		tenantID,
	)

	eventB := createTestOutboxEvent(
		t,
		db.appPool,
		tenantID,
	)

	tokenA := uuid.New()
	tokenB := uuid.New()

	type result struct {
		rows []postgresdb.ClaimOutboxEventsRow
		err  error
	}

	results := make(chan result, 2)
	start := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(2)

	claim := func(token uuid.UUID) {
		defer wg.Done()

		<-start

		ctx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()

		rows, err := postgresdb.New(
			db.workerPool,
		).ClaimOutboxEvents(
			ctx,
			postgresdb.ClaimOutboxEventsParams{
				ClaimToken: pgtype.UUID{
					Bytes: token,
					Valid: true,
				},
				LeaseSeconds: 60,
				BatchSize:    1,
			},
		)

		results <- result{
			rows: rows,
			err:  err,
		}
	}

	go claim(tokenA)
	go claim(tokenB)

	close(start)

	wg.Wait()
	close(results)

	var claimedIDs []uuid.UUID

	for result := range results {
		require.NoError(t, result.err)
		require.Len(t, result.rows, 1)

		claimedIDs = append(
			claimedIDs,
			uuid.UUID(result.rows[0].ID.Bytes),
		)
	}

	require.Len(t, claimedIDs, 2)

	require.ElementsMatch(
		t,
		[]uuid.UUID{
			eventA,
			eventB,
		},
		claimedIDs,
	)
}

func TestOutboxClaim_ActiveLeasePreventsReclaim(t *testing.T) {
	db := newDispatcherIntegrationDB(t)
	resetOutboxEvents(t, db.workerPool)

	tenantID := createOutboxTenant(
		t,
		db.appPool,
	)

	eventID := createTestOutboxEvent(
		t,
		db.appPool,
		tenantID,
	)

	firstToken := uuid.New()
	secondToken := uuid.New()

	firstRows := claimEvents(
		t,
		db.workerPool,
		firstToken,
		60,
		1,
	)

	require.Len(t, firstRows, 1)

	require.Equal(
		t,
		eventID,
		uuid.UUID(firstRows[0].ID.Bytes),
	)

	secondRows := claimEvents(
		t,
		db.workerPool,
		secondToken,
		60,
		1,
	)

	require.Empty(t, secondRows)

	_, claimedAt, claimToken, _ := readOutboxState(
		t,
		db.workerPool,
		eventID,
	)

	require.NotNil(t, claimedAt)
	require.NotNil(t, claimToken)

	require.Equal(
		t,
		firstToken,
		*claimToken,
	)
}

func TestOutboxClaim_ExpiredLeaseCanBeReclaimed(t *testing.T) {
	db := newDispatcherIntegrationDB(t)
	resetOutboxEvents(t, db.workerPool)

	tenantID := createOutboxTenant(
		t,
		db.appPool,
	)

	eventID := createTestOutboxEvent(
		t,
		db.appPool,
		tenantID,
	)

	firstToken := uuid.New()
	secondToken := uuid.New()

	rows := claimEvents(
		t,
		db.workerPool,
		firstToken,
		60,
		1,
	)

	require.Len(t, rows, 1)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	_, err := db.workerPool.Exec(
		ctx,
		`
		UPDATE outbox_events
		SET claimed_at = NOW() - INTERVAL '5 minutes'
		WHERE id = $1
		`,
		eventID,
	)

	require.NoError(t, err)

	reclaimed := claimEvents(
		t,
		db.workerPool,
		secondToken,
		60,
		1,
	)

	require.Len(t, reclaimed, 1)

	require.Equal(
		t,
		eventID,
		uuid.UUID(reclaimed[0].ID.Bytes),
	)

	_, claimedAt, claimToken, _ := readOutboxState(
		t,
		db.workerPool,
		eventID,
	)

	require.NotNil(t, claimedAt)
	require.NotNil(t, claimToken)

	require.Equal(
		t,
		secondToken,
		*claimToken,
	)
}

func TestOutboxClaim_PublishedEventIsSkipped(t *testing.T) {
	db := newDispatcherIntegrationDB(t)
	resetOutboxEvents(t, db.workerPool)

	tenantID := createOutboxTenant(
		t,
		db.appPool,
	)

	eventID := createTestOutboxEvent(
		t,
		db.appPool,
		tenantID,
	)

	claimToken := uuid.New()

	rows := claimEvents(
		t,
		db.workerPool,
		claimToken,
		60,
		1,
	)

	require.Len(t, rows, 1)

	require.Equal(
		t,
		eventID,
		uuid.UUID(rows[0].ID.Bytes),
	)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	err := postgresdb.New(
		db.workerPool,
	).MarkOutboxEventPublished(
		ctx,
		postgresdb.MarkOutboxEventPublishedParams{
			ID: pgtype.UUID{
				Bytes: eventID,
				Valid: true,
			},
			ClaimToken: pgtype.UUID{
				Bytes: claimToken,
				Valid: true,
			},
		},
	)

	require.NoError(t, err)

	nextToken := uuid.New()

	nextRows := claimEvents(
		t,
		db.workerPool,
		nextToken,
		60,
		1,
	)

	require.Empty(t, nextRows)

	publishedAt, claimedAt, storedToken, attempts := readOutboxState(
		t,
		db.workerPool,
		eventID,
	)

	require.NotNil(t, publishedAt)
	require.Nil(t, claimedAt)
	require.Nil(t, storedToken)
	require.Equal(t, int32(0), attempts)
}

func TestOutboxClaim_WrongClaimTokenCannotCompleteEvent(t *testing.T) {
	db := newDispatcherIntegrationDB(t)
	resetOutboxEvents(t, db.workerPool)

	tenantID := createOutboxTenant(
		t,
		db.appPool,
	)

	eventID := createTestOutboxEvent(
		t,
		db.appPool,
		tenantID,
	)

	ownerToken := uuid.New()
	wrongToken := uuid.New()

	rows := claimEvents(
		t,
		db.workerPool,
		ownerToken,
		60,
		1,
	)

	require.Len(t, rows, 1)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	err := postgresdb.New(
		db.workerPool,
	).MarkOutboxEventPublished(
		ctx,
		postgresdb.MarkOutboxEventPublishedParams{
			ID: pgtype.UUID{
				Bytes: eventID,
				Valid: true,
			},
			ClaimToken: pgtype.UUID{
				Bytes: wrongToken,
				Valid: true,
			},
		},
	)

	require.NoError(t, err)

	publishedAt, claimedAt, storedToken, attempts := readOutboxState(
		t,
		db.workerPool,
		eventID,
	)

	require.Nil(t, publishedAt)
	require.NotNil(t, claimedAt)
	require.NotNil(t, storedToken)

	require.Equal(
		t,
		ownerToken,
		*storedToken,
	)

	require.Equal(
		t,
		int32(0),
		attempts,
	)
}

func TestOutboxClaim_WrongClaimTokenCannotMarkFailed(t *testing.T) {
	db := newDispatcherIntegrationDB(t)
	resetOutboxEvents(t, db.workerPool)

	tenantID := createOutboxTenant(
		t,
		db.appPool,
	)

	eventID := createTestOutboxEvent(
		t,
		db.appPool,
		tenantID,
	)

	ownerToken := uuid.New()
	wrongToken := uuid.New()

	rows := claimEvents(
		t,
		db.workerPool,
		ownerToken,
		60,
		1,
	)

	require.Len(t, rows, 1)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	err := postgresdb.New(
		db.workerPool,
	).MarkOutboxEventFailed(
		ctx,
		postgresdb.MarkOutboxEventFailedParams{
			ID: pgtype.UUID{
				Bytes: eventID,
				Valid: true,
			},
			ClaimToken: pgtype.UUID{
				Bytes: wrongToken,
				Valid: true,
			},
			LastError: pgtype.Text{
				String: "wrong worker",
				Valid:  true,
			},
		},
	)

	require.NoError(t, err)

	publishedAt, claimedAt, storedToken, attempts := readOutboxState(
		t,
		db.workerPool,
		eventID,
	)

	require.Nil(t, publishedAt)
	require.NotNil(t, claimedAt)
	require.NotNil(t, storedToken)

	require.Equal(
		t,
		ownerToken,
		*storedToken,
	)

	require.Equal(
		t,
		int32(0),
		attempts,
	)
}
