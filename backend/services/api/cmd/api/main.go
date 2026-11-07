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
	menuhttp "github.com/afadnan/restaurantflow/services/api/internal/modules/menu/delivery/http"
	menurepository "github.com/afadnan/restaurantflow/services/api/internal/modules/menu/repository"
	menuusecase "github.com/afadnan/restaurantflow/services/api/internal/modules/menu/usecase"
	httpdelivery "github.com/afadnan/restaurantflow/services/api/internal/modules/order/delivery/http"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/order/repository"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/order/usecase"
	httprestaurant "github.com/afadnan/restaurantflow/services/api/internal/modules/restaurant/delivery/http"
	restaurantrepository "github.com/afadnan/restaurantflow/services/api/internal/modules/restaurant/repository"
	restaurantusecase "github.com/afadnan/restaurantflow/services/api/internal/modules/restaurant/usecase"
	tenantrepository "github.com/afadnan/restaurantflow/services/api/internal/modules/tenant/repository"
	userhttp "github.com/afadnan/restaurantflow/services/api/internal/modules/user/delivery/http"
	userrepository "github.com/afadnan/restaurantflow/services/api/internal/modules/user/repository"
	userusecase "github.com/afadnan/restaurantflow/services/api/internal/modules/user/usecase"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/auth"
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
	// Shared event repository
	// -------------------------------------------------------------------------

	// Order and Restaurant share the same event repository so their domain
	// events use the existing transactional outbox infrastructure.
	eventRepository := events.NewPostgresEventRepository()

	// -------------------------------------------------------------------------
	// Order repositories
	// -------------------------------------------------------------------------

	orderRepository := repository.NewPostgresOrderRepository()
	kdsRepository := repository.NewPostgresKDSRepository()
	inventoryRepository := repository.NewPostgresInventoryRepository()

	// -------------------------------------------------------------------------
	// Restaurant repositories
	// -------------------------------------------------------------------------

	restaurantRepository := restaurantrepository.NewPostgresRestaurantRepository()

	// -------------------------------------------------------------------------
	// Menu repositories
	// -------------------------------------------------------------------------

	menuCategoryRepository := menurepository.NewPostgresCategoryRepository()
	menuItemRepository := menurepository.NewPostgresItemRepository()
	menuAddonRepository := menurepository.NewPostgresAddonRepository()
	menuItemAddonRepository := menurepository.NewPostgresItemAddonRepository()
	// -------------------------------------------------------------------------
	// Menu Add-on use cases
	// -------------------------------------------------------------------------

	addonDependencies := menuusecase.AddonUseCaseDependencies{
		DB:     db,
		Addons: menuAddonRepository,
		Events: eventRepository,
		Clock:  menuusecase.SystemClock{},
	}

	createAddonUseCase := &menuusecase.CreateAddonUseCase{
		AddonUseCaseDependencies: addonDependencies,
	}

	getAddonUseCase := &menuusecase.GetAddonUseCase{
		AddonUseCaseDependencies: addonDependencies,
	}

	listAddonsUseCase := &menuusecase.ListAddonsUseCase{
		AddonUseCaseDependencies: addonDependencies,
	}

	updateAddonUseCase := &menuusecase.UpdateAddonUseCase{
		AddonUseCaseDependencies: addonDependencies,
	}

	deactivateAddonUseCase := &menuusecase.DeactivateAddonUseCase{
		AddonUseCaseDependencies: addonDependencies,
	}

	// -------------------------------------------------------------------------
	// Menu Item–Add-on use cases
	// -------------------------------------------------------------------------

	itemAddonDependencies := menuusecase.ItemAddonUseCaseDependencies{
		DB:         db,
		ItemAddons: menuItemAddonRepository,
		Events:     eventRepository,
		Clock:      menuusecase.SystemClock{},
	}

	attachItemAddonUseCase := &menuusecase.AttachItemAddonUseCase{
		ItemAddonUseCaseDependencies: itemAddonDependencies,
	}

	getItemAddonUseCase := &menuusecase.GetItemAddonUseCase{
		ItemAddonUseCaseDependencies: itemAddonDependencies,
	}

	listItemAddonsUseCase := &menuusecase.ListItemAddonsUseCase{
		ItemAddonUseCaseDependencies: itemAddonDependencies,
	}

	updateItemAddonUseCase := &menuusecase.UpdateItemAddonUseCase{
		ItemAddonUseCaseDependencies: itemAddonDependencies,
	}

	detachItemAddonUseCase := &menuusecase.DetachItemAddonUseCase{
		ItemAddonUseCaseDependencies: itemAddonDependencies,
	}

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
	// Restaurant use cases
	// -------------------------------------------------------------------------

	createRestaurantUseCase := &restaurantusecase.CreateRestaurantUseCase{
		DB:          db,
		Restaurants: restaurantRepository,
		Events:      eventRepository,
		Clock:       restaurantusecase.SystemClock{},
	}

	getRestaurantUseCase := &restaurantusecase.GetRestaurantUseCase{
		DB:          db,
		Restaurants: restaurantRepository,
	}

	listRestaurantsUseCase := &restaurantusecase.ListRestaurantsUseCase{
		DB:          db,
		Restaurants: restaurantRepository,
	}

	updateRestaurantUseCase := &restaurantusecase.UpdateRestaurantUseCase{
		DB:          db,
		Restaurants: restaurantRepository,
		Events:      eventRepository,
		Clock:       restaurantusecase.SystemClock{},
	}

	deactivateRestaurantUseCase := &restaurantusecase.DeactivateRestaurantUseCase{
		DB:          db,
		Restaurants: restaurantRepository,
		Events:      eventRepository,
		Clock:       restaurantusecase.SystemClock{},
	}

	// -------------------------------------------------------------------------
	// Menu category use cases
	// -------------------------------------------------------------------------

	createCategoryUseCase := &menuusecase.CreateCategoryUseCase{
		CategoryUseCaseDependencies: menuusecase.CategoryUseCaseDependencies{
			DB:         db,
			Categories: menuCategoryRepository,
			Events:     eventRepository,
			Clock:      menuusecase.SystemClock{},
		},
	}

	getCategoryUseCase := &menuusecase.GetCategoryUseCase{
		CategoryUseCaseDependencies: menuusecase.CategoryUseCaseDependencies{
			DB:         db,
			Categories: menuCategoryRepository,
		},
	}

	listCategoriesUseCase := &menuusecase.ListCategoriesUseCase{
		CategoryUseCaseDependencies: menuusecase.CategoryUseCaseDependencies{
			DB:         db,
			Categories: menuCategoryRepository,
		},
	}

	updateCategoryUseCase := &menuusecase.UpdateCategoryUseCase{
		CategoryUseCaseDependencies: menuusecase.CategoryUseCaseDependencies{
			DB:         db,
			Categories: menuCategoryRepository,
			Events:     eventRepository,
			Clock:      menuusecase.SystemClock{},
		},
	}

	deactivateCategoryUseCase := &menuusecase.DeactivateCategoryUseCase{
		CategoryUseCaseDependencies: menuusecase.CategoryUseCaseDependencies{
			DB:         db,
			Categories: menuCategoryRepository,
			Events:     eventRepository,
		},
	}

	// -------------------------------------------------------------------------
	// Menu Item use cases
	// -------------------------------------------------------------------------

	itemDependencies := menuusecase.ItemUseCaseDependencies{
		DB:     db,
		Items:  menuItemRepository,
		Events: eventRepository,
		Clock:  menuusecase.SystemClock{},
	}

	createItemUseCase := &menuusecase.CreateItemUseCase{
		ItemUseCaseDependencies: itemDependencies,
	}

	getItemUseCase := &menuusecase.GetItemUseCase{
		ItemUseCaseDependencies: itemDependencies,
	}

	listItemsUseCase := &menuusecase.ListItemsUseCase{
		ItemUseCaseDependencies: itemDependencies,
	}

	updateItemUseCase := &menuusecase.UpdateItemUseCase{
		ItemUseCaseDependencies: itemDependencies,
	}

	deactivateItemUseCase := &menuusecase.DeactivateItemUseCase{
		ItemUseCaseDependencies: itemDependencies,
	}

	// -------------------------------------------------------------------------
	// HTTP handlers
	// -------------------------------------------------------------------------

	orderHandler := httpdelivery.NewOrderHandler(
		createOrderUseCase,
		updateKDSStateUseCase,
	)

	restaurantHandler := httprestaurant.NewRestaurantHandler(
		createRestaurantUseCase,
		getRestaurantUseCase,
		listRestaurantsUseCase,
		updateRestaurantUseCase,
		deactivateRestaurantUseCase,
	)

	menuCategoryHandler := menuhttp.NewCategoryHandler(
		createCategoryUseCase,
		getCategoryUseCase,
		listCategoriesUseCase,
		updateCategoryUseCase,
		deactivateCategoryUseCase,
	)

	menuItemHandler := menuhttp.NewItemHandler(
		createItemUseCase,
		getItemUseCase,
		listItemsUseCase,
		updateItemUseCase,
		deactivateItemUseCase,
	)

	// -------------------------------------------------------------------------
	// Authentication: repositories, JWT service and use cases
	// -------------------------------------------------------------------------

	tenantRepository := tenantrepository.NewPostgresTenantRepository()
	userRepository := userrepository.NewPostgresUserRepository()

	tokenService, err := auth.NewTokenService(
		cfg.JWT.Secret,
		cfg.JWT.Issuer,
		cfg.JWT.AccessTokenTTL,
	)
	if err != nil {
		return fmt.Errorf("initialize JWT token service: %w", err)
	}

	// Owner registration
	registerOwnerUseCase := &userusecase.RegisterOwnerUseCase{
		DB:      db,
		Tenants: tenantRepository,
		Users:   userRepository,
	}

	ownerRegistrationHandler := userhttp.NewOwnerRegistrationHandler(
		registerOwnerUseCase,
	)

	// Owner login
	loginUseCase := &userusecase.LoginUseCase{
		DB:      db,
		Tenants: tenantRepository,
		Users:   userRepository,
		Tokens:  tokenService,
	}

	ownerLoginHandler := userhttp.NewOwnerLoginHandler(
		loginUseCase,
	)

	// -------------------------------------------------------------------------
	// Menu Add-on HTTP handler
	// -------------------------------------------------------------------------

	menuAddonHandler := menuhttp.NewAddonHandler(
		createAddonUseCase,
		getAddonUseCase,
		listAddonsUseCase,
		updateAddonUseCase,
		deactivateAddonUseCase,
	)

	menuItemAddonHandler := menuhttp.NewItemAddonHandler(
		attachItemAddonUseCase,
		getItemAddonUseCase,
		listItemAddonsUseCase,
		updateItemAddonUseCase,
		detachItemAddonUseCase,
	)

	// -------------------------------------------------------------------------
	// HTTP routes
	// -------------------------------------------------------------------------

	mux := http.NewServeMux()

	// -------------------------------------------------------------------------
	// Public Owner Registration routes
	// -------------------------------------------------------------------------

	// No TenantMiddleware: the tenant is created during registration.
	userhttp.RegisterRoutes(
		mux,
		ownerRegistrationHandler,
		ownerLoginHandler,
	)

	// Order routes
	httpdelivery.RegisterRoutes(
		mux,
		orderHandler,
		nil,
	)

	// Restaurant routes
	httprestaurant.RegisterRoutes(
		mux,
		restaurantHandler,
	)

	// Menu Category routes
	menuhttp.RegisterCategoryRoutes(
		mux,
		menuCategoryHandler,
	)

	// Menu Item routes
	menuhttp.RegisterItemRoutes(
		mux,
		menuItemHandler,
	)

	// Menu Add-on routes
	menuhttp.RegisterAddonRoutes(
		mux,
		menuAddonHandler,
	)

	// Menu Item–Add-on routes
	menuhttp.RegisterItemAddonRoutes(
		mux,
		menuItemAddonHandler,
	)

	// -------------------------------------------------------------------------
	// Health checks
	// -------------------------------------------------------------------------

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
			w.Header().Set("Content-Type", "application/json")
			http.Error(
				w,
				`{"status":"not_ready"}`,
				http.StatusServiceUnavailable,
			)
			return
		}

		if err := redisClient.Ping(r.Context()).Err(); err != nil {
			w.Header().Set("Content-Type", "application/json")
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

		// The cancellation signal stops the dispatcher before its dependencies
		// are closed.
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
