package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/afadnan/restaurantflow/services/api/internal/config"
	httpdelivery "github.com/afadnan/restaurantflow/services/api/internal/modules/order/delivery/http"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/order/repository"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/order/usecase"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/database"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/events"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
	postgresdb "github.com/afadnan/restaurantflow/services/api/internal/platform/postgres/db"
	redisplatform "github.com/afadnan/restaurantflow/services/api/internal/platform/redis"
)

func main() {
	if err := run(); err != nil {
		log.Printf("api server stopped: %v", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	logger := slog.New(
		slog.NewTextHandler(os.Stdout, nil),
	)

	// -------------------------------------------------------------------------
	// PostgreSQL
	// -------------------------------------------------------------------------

	pool, err := database.NewPostgresPool(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("initialize postgres: %w", err)
	}
	defer pool.Close()

	db := postgres.NewDB(pool)

	log.Printf(
		"postgres connected: host=%s port=%d database=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
	)

	// -------------------------------------------------------------------------
	// Redis
	// -------------------------------------------------------------------------

	redisClient := redisplatform.NewClient(
		fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		"",
		0,
	)

	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Error("redis close failed", "error", err)
		}
	}()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("initialize redis: %w", err)
	}

	log.Printf(
		"redis connected: host=%s port=%d",
		cfg.Redis.Host,
		cfg.Redis.Port,
	)

	redisPublisher := redisplatform.NewPublisher(
		redisClient,
		logger,
	)

	// -------------------------------------------------------------------------
	// Outbox dispatcher
	// -------------------------------------------------------------------------

	outboxStore := postgresdb.New(pool)

	dispatcher := events.NewDispatcher(
		outboxStore,
		redisPublisher,
		logger,
		events.DispatcherConfig{
			BatchSize:    50,
			Lease:        30 * time.Second,
			PollInterval: time.Second,
		},
	)

	var dispatcherWG sync.WaitGroup

	dispatcherWG.Add(1)

	go func() {
		defer dispatcherWG.Done()

		if err := dispatcher.Run(ctx); err != nil &&
			!errors.Is(err, context.Canceled) {
			logger.Error(
				"outbox dispatcher stopped",
				"error",
				err,
			)
		}
	}()

	// -------------------------------------------------------------------------
	// Order repositories
	// -------------------------------------------------------------------------

	orderRepository := repository.NewPostgresOrderRepository()
	kdsRepository := repository.NewPostgresKDSRepository()
	inventoryRepository := repository.NewPostgresInventoryRepository()
	eventRepository := events.NewPostgresEventRepository()

	// -------------------------------------------------------------------------
	// Order use cases
	// -------------------------------------------------------------------------

	createOrderUseCase := &usecase.CreateOrderUseCase{
		DB:        db,
		Orders:    orderRepository,
		KDS:       kdsRepository,
		Inventory: inventoryRepository,
		Events:    eventRepository,
		Clock:     time.Now,
	}

	updateKDSStateUseCase := usecase.NewUpdateKDSStateUseCase(
		db,
		kdsRepository,
		orderRepository,
		eventRepository,
	)

	// -------------------------------------------------------------------------
	// HTTP handlers
	// -------------------------------------------------------------------------

	orderHandler := httpdelivery.NewOrderHandler(
		createOrderUseCase,
		updateKDSStateUseCase,
	)

	// -------------------------------------------------------------------------
	// HTTP routes
	// -------------------------------------------------------------------------

	mux := http.NewServeMux()

	httpdelivery.RegisterRoutes(
		mux,
		orderHandler,
		nil,
	)

	mux.HandleFunc("/healthz", func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	mux.HandleFunc("/readyz", func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		if err := pool.Ping(r.Context()); err != nil {
			http.Error(
				w,
				`{"status":"not_ready"}`,
				http.StatusServiceUnavailable,
			)
			return
		}

		if err := redisClient.Ping(r.Context()).Err(); err != nil {
			http.Error(
				w,
				`{"status":"not_ready"}`,
				http.StatusServiceUnavailable,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})

	// -------------------------------------------------------------------------
	// HTTP server
	// -------------------------------------------------------------------------

	server := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.HTTPPort),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)

	go func() {
		log.Printf(
			"RestaurantFlow API listening on http://localhost:%d",
			cfg.HTTPPort,
		)

		if err := server.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// -------------------------------------------------------------------------
	// Wait for shutdown signal or HTTP server failure
	// -------------------------------------------------------------------------

	select {
	case err := <-serverErr:
		stop()

		// Give the dispatcher a chance to stop before returning and
		// closing PostgreSQL/Redis.
		dispatcherWG.Wait()

		return fmt.Errorf("http server: %w", err)

	case <-ctx.Done():
		log.Println("shutdown signal received")
	}

	// -------------------------------------------------------------------------
	// Graceful HTTP shutdown
	// -------------------------------------------------------------------------

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	// -------------------------------------------------------------------------
	// Graceful dispatcher shutdown
	// -------------------------------------------------------------------------

	dispatcherWG.Wait()

	log.Println("api server stopped")

	return nil
}
