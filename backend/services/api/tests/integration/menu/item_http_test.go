package menu_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	menuhttp "github.com/afadnan/restaurantflow/services/api/internal/modules/menu/delivery/http"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/repository"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/usecase"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/events"
)

func TestIntegrationMenuItemHTTP(t *testing.T) {
	integration := newVariantIntegration(t)
	fixture := integration.createFixture(t)

	// Retrieve the category belonging to the existing fixture item.
	// variantFixture does not expose categoryID directly.
	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	tx, err := integration.db.BeginTx(ctx, fixture.tenantID)
	require.NoError(t, err)

	var categoryID uuid.UUID

	err = tx.Raw().QueryRow(ctx, `
		SELECT category_id
		FROM menu_items
		WHERE id = $1
		  AND tenant_id = $2
		  AND restaurant_id = $3
	`, fixture.itemID, fixture.tenantID, fixture.restaurantID).Scan(&categoryID)

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, categoryID)
	require.NoError(t, tx.Rollback(ctx))

	// Wire the real repositories and transactional use cases.
	deps := usecase.ItemUseCaseDependencies{
		DB:     integration.db,
		Items:  repository.NewPostgresItemRepository(),
		Events: events.NewPostgresEventRepository(),
	}

	handler := menuhttp.NewItemHandler(
		&usecase.CreateItemUseCase{ItemUseCaseDependencies: deps},
		&usecase.GetItemUseCase{ItemUseCaseDependencies: deps},
		&usecase.ListItemsUseCase{ItemUseCaseDependencies: deps},
		&usecase.UpdateItemUseCase{ItemUseCaseDependencies: deps},
		&usecase.DeactivateItemUseCase{ItemUseCaseDependencies: deps},
	)

	mux := http.NewServeMux()
	menuhttp.RegisterItemRoutes(mux, handler)

	basePath := fmt.Sprintf(
		"/api/v1/restaurants/%s/menu/items",
		fixture.restaurantID,
	)

	type itemJSON struct {
		ID           string `json:"id"`
		TenantID     string `json:"tenant_id"`
		RestaurantID string `json:"restaurant_id"`
		CategoryID   string `json:"category_id"`
		Name         string `json:"name"`
		Description  string `json:"description"`
		SKU          string `json:"sku"`
		Price        string `json:"price"`
		Status       string `json:"status"`
		IsAvailable  bool   `json:"is_available"`
		CreatedAt    string `json:"created_at"`
		UpdatedAt    string `json:"updated_at"`
	}

	// Execute requests through the real HTTP router.
	request := func(
		method, path, body string,
		tenantID uuid.UUID,
	) *httptest.ResponseRecorder {
		t.Helper()

		requestCtx, requestCancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer requestCancel()

		req := httptest.NewRequest(
			method,
			path,
			strings.NewReader(body),
		).WithContext(requestCtx)

		if tenantID != uuid.Nil {
			req.Header.Set("X-Tenant-ID", tenantID.String())
		}

		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}

		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, req)

		return recorder
	}

	decodeItem := func(
		recorder *httptest.ResponseRecorder,
	) itemJSON {
		t.Helper()

		var item itemJSON

		require.NoError(
			t,
			json.Unmarshal(recorder.Body.Bytes(), &item),
		)

		return item
	}

	createBody := func(name, price string) string {
		return fmt.Sprintf(
			`{
			"category_id": %q,
			"name": %q,
			"description": "HTTP integration test item",
			"sku": %q,
			"price": %q,
			"status": "active",
			"image_url": "",
			"sort_order": 1,
			"is_vegetarian": false,
			"is_vegan": false,
			"is_available": true
		}`,
			categoryID.String(),
			name,
			"HTTP-"+uuid.NewString(),
			price,
		)
	}

	var itemID uuid.UUID

	t.Run("create item", func(t *testing.T) {
		response := request(
			http.MethodPost,
			basePath,
			createBody("HTTP Chicken Biryani", "249.00"),
			fixture.tenantID,
		)

		require.Equal(
			t,
			http.StatusCreated,
			response.Code,
			response.Body.String(),
		)

		item := decodeItem(response)

		var parseErr error
		itemID, parseErr = uuid.Parse(item.ID)

		require.NoError(t, parseErr)
		require.NotEqual(t, uuid.Nil, itemID)

		require.Equal(t, fixture.tenantID.String(), item.TenantID)
		require.Equal(t, fixture.restaurantID.String(), item.RestaurantID)
		require.Equal(t, categoryID.String(), item.CategoryID)
		require.Equal(t, "HTTP Chicken Biryani", item.Name)
		require.Equal(t, "249.00", item.Price)
		require.Equal(t, "active", item.Status)
		require.True(t, item.IsAvailable)

		_, parseErr = time.Parse(time.RFC3339, item.CreatedAt)
		require.NoError(t, parseErr)

		require.Equal(t, 1, integration.countEvents(
			t,
			fixture.tenantID,
			itemID,
			string(domain.EventTypeItemCreated),
		))
	})

	itemPath := func() string {
		return basePath + "/" + itemID.String()
	}

	t.Run("get item", func(t *testing.T) {
		response := request(
			http.MethodGet,
			itemPath(),
			"",
			fixture.tenantID,
		)

		require.Equal(
			t,
			http.StatusOK,
			response.Code,
			response.Body.String(),
		)

		item := decodeItem(response)

		require.Equal(t, itemID.String(), item.ID)
		require.Equal(t, "HTTP Chicken Biryani", item.Name)
		require.Equal(t, "249.00", item.Price)
	})

	t.Run("list items", func(t *testing.T) {
		response := request(
			http.MethodGet,
			basePath+"?limit=100&offset=0",
			"",
			fixture.tenantID,
		)

		require.Equal(
			t,
			http.StatusOK,
			response.Code,
			response.Body.String(),
		)

		var result struct {
			Items  []itemJSON `json:"items"`
			Limit  int        `json:"limit"`
			Offset int        `json:"offset"`
		}

		require.NoError(
			t,
			json.Unmarshal(response.Body.Bytes(), &result),
		)

		require.Equal(t, 100, result.Limit)
		require.Zero(t, result.Offset)

		found := false

		for _, item := range result.Items {
			if item.ID == itemID.String() {
				found = true
				break
			}
		}

		require.True(
			t,
			found,
			"created item missing from list",
		)
	})

	t.Run("update item", func(t *testing.T) {
		response := request(
			http.MethodPut,
			itemPath(),
			createBody("HTTP Premium Chicken Biryani", "279.00"),
			fixture.tenantID,
		)

		require.Equal(
			t,
			http.StatusOK,
			response.Code,
			response.Body.String(),
		)

		item := decodeItem(response)

		require.Equal(t, itemID.String(), item.ID)
		require.Equal(t, "HTTP Premium Chicken Biryani", item.Name)
		require.Equal(t, "279.00", item.Price)

		require.Equal(t, 1, integration.countEvents(
			t,
			fixture.tenantID,
			itemID,
			string(domain.EventTypeItemUpdated),
		))
	})

	t.Run("invalid JSON rejected", func(t *testing.T) {
		response := request(
			http.MethodPost,
			basePath,
			`{"name":`,
			fixture.tenantID,
		)

		require.Equal(t, http.StatusBadRequest, response.Code)
	})

	t.Run("unknown JSON field rejected", func(t *testing.T) {
		response := request(
			http.MethodPost,
			basePath,
			`{"unexpected_field":true}`,
			fixture.tenantID,
		)

		require.Equal(t, http.StatusBadRequest, response.Code)
	})

	t.Run("invalid price rejected", func(t *testing.T) {
		response := request(
			http.MethodPost,
			basePath,
			createBody("Invalid Price Item", "-10.00"),
			fixture.tenantID,
		)

		require.Equal(
			t,
			http.StatusBadRequest,
			response.Code,
			response.Body.String(),
		)
	})

	t.Run("invalid item UUID rejected", func(t *testing.T) {
		response := request(
			http.MethodGet,
			basePath+"/not-a-uuid",
			"",
			fixture.tenantID,
		)

		require.Equal(t, http.StatusBadRequest, response.Code)
	})

	t.Run("nonexistent item returns 404", func(t *testing.T) {
		response := request(
			http.MethodGet,
			basePath+"/"+uuid.NewString(),
			"",
			fixture.tenantID,
		)

		require.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("invalid pagination rejected", func(t *testing.T) {
		response := request(
			http.MethodGet,
			basePath+"?limit=-1",
			"",
			fixture.tenantID,
		)

		require.Equal(t, http.StatusBadRequest, response.Code)
	})

	t.Run("missing tenant rejected", func(t *testing.T) {
		response := request(
			http.MethodGet,
			itemPath(),
			"",
			uuid.Nil,
		)

		require.Equal(t, http.StatusUnauthorized, response.Code)
	})

	t.Run("different tenant cannot read item", func(t *testing.T) {
		response := request(
			http.MethodGet,
			itemPath(),
			"",
			uuid.New(),
		)

		require.Equal(
			t,
			http.StatusNotFound,
			response.Code,
			response.Body.String(),
		)
	})

	t.Run("deactivate item", func(t *testing.T) {
		response := request(
			http.MethodPatch,
			itemPath()+"/deactivate",
			"",
			fixture.tenantID,
		)

		require.Equal(
			t,
			http.StatusOK,
			response.Code,
			response.Body.String(),
		)

		item := decodeItem(response)

		require.Equal(t, itemID.String(), item.ID)
		require.Equal(t, "inactive", item.Status)
		require.False(t, item.IsAvailable)

		require.Equal(t, 1, integration.countEvents(
			t,
			fixture.tenantID,
			itemID,
			string(domain.EventTypeItemDeactivated),
		))
	})

	t.Run("deactivated item remains readable", func(t *testing.T) {
		response := request(
			http.MethodGet,
			itemPath(),
			"",
			fixture.tenantID,
		)

		require.Equal(
			t,
			http.StatusOK,
			response.Code,
			response.Body.String(),
		)

		item := decodeItem(response)

		require.Equal(t, "inactive", item.Status)
		require.False(t, item.IsAvailable)
	})
}
