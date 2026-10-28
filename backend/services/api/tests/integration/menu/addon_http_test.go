package menu_test

import (
	"context"
	"encoding/json"
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

func TestIntegrationMenuAddonHTTP(t *testing.T) {
	integration := newVariantIntegration(t)
	fixture := integration.createFixture(t)

	deps := usecase.AddonUseCaseDependencies{
		DB:     integration.db,
		Addons: repository.NewPostgresAddonRepository(),
		Events: events.NewPostgresEventRepository(),
	}

	handler := menuhttp.NewAddonHandler(
		&usecase.CreateAddonUseCase{
			AddonUseCaseDependencies: deps,
		},
		&usecase.GetAddonUseCase{
			AddonUseCaseDependencies: deps,
		},
		&usecase.ListAddonsUseCase{
			AddonUseCaseDependencies: deps,
		},
		&usecase.UpdateAddonUseCase{
			AddonUseCaseDependencies: deps,
		},
		&usecase.DeactivateAddonUseCase{
			AddonUseCaseDependencies: deps,
		},
	)

	mux := http.NewServeMux()
	menuhttp.RegisterAddonRoutes(mux, handler)

	basePath := "/api/v1/restaurants/" +
		fixture.restaurantID.String() + "/menu/addons"

	type addonJSON struct {
		ID           string `json:"id"`
		TenantID     string `json:"tenant_id"`
		RestaurantID string `json:"restaurant_id"`
		Name         string `json:"name"`
		Price        string `json:"price"`
		IsActive     bool   `json:"is_active"`
		CreatedAt    string `json:"created_at"`
		UpdatedAt    string `json:"updated_at"`
	}

	request := func(
		method, path, body string,
		tenantID uuid.UUID,
	) *httptest.ResponseRecorder {
		t.Helper()

		ctx, cancel := context.WithTimeout(
			context.Background(), 10*time.Second,
		)
		defer cancel()

		req := httptest.NewRequest(
			method, path, strings.NewReader(body),
		).WithContext(ctx)

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

	decodeAddon := func(
		recorder *httptest.ResponseRecorder,
	) addonJSON {
		t.Helper()

		var result addonJSON
		require.NoError(t, json.Unmarshal(
			recorder.Body.Bytes(), &result,
		))
		return result
	}

	var addonID uuid.UUID

	t.Run("create addon", func(t *testing.T) {
		response := request(
			http.MethodPost,
			basePath,
			`{"name":"HTTP Extra Cheese","description":"Optional topping","price":"25.00"}`,
			fixture.tenantID,
		)

		require.Equal(t, http.StatusCreated, response.Code,
			response.Body.String())

		addon := decodeAddon(response)
		var err error
		addonID, err = uuid.Parse(addon.ID)
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, addonID)

		require.Equal(t, fixture.tenantID.String(), addon.TenantID)
		require.Equal(t, fixture.restaurantID.String(), addon.RestaurantID)
		require.Equal(t, "HTTP Extra Cheese", addon.Name)
		require.Equal(t, "25.00", addon.Price)
		require.True(t, addon.IsActive)

		_, err = time.Parse(time.RFC3339, addon.CreatedAt)
		require.NoError(t, err)

		require.Equal(t, 1, integration.countEvents(
			t, fixture.tenantID, addonID,
			string(domain.EventTypeAddonCreated),
		))
	})

	t.Run("get addon", func(t *testing.T) {
		response := request(
			http.MethodGet,
			basePath+"/"+addonID.String(),
			"",
			fixture.tenantID,
		)

		require.Equal(t, http.StatusOK, response.Code,
			response.Body.String())

		addon := decodeAddon(response)
		require.Equal(t, addonID.String(), addon.ID)
		require.Equal(t, "HTTP Extra Cheese", addon.Name)
	})

	t.Run("list addons", func(t *testing.T) {
		response := request(
			http.MethodGet,
			basePath+"?limit=10&offset=0",
			"",
			fixture.tenantID,
		)

		require.Equal(t, http.StatusOK, response.Code,
			response.Body.String())

		var result struct {
			Addons []addonJSON `json:"addons"`
			Limit  int         `json:"limit"`
			Offset int         `json:"offset"`
		}

		require.NoError(t, json.Unmarshal(
			response.Body.Bytes(), &result,
		))

		require.Equal(t, 10, result.Limit)
		require.Zero(t, result.Offset)
		require.Len(t, result.Addons, 1)
		require.Equal(t, addonID.String(), result.Addons[0].ID)
	})

	t.Run("update addon", func(t *testing.T) {
		response := request(
			http.MethodPut,
			basePath+"/"+addonID.String(),
			`{"name":"HTTP Premium Cheese","description":"Updated topping","price":"35.00"}`,
			fixture.tenantID,
		)

		require.Equal(t, http.StatusOK, response.Code,
			response.Body.String())

		addon := decodeAddon(response)
		require.Equal(t, "HTTP Premium Cheese", addon.Name)
		require.Equal(t, "35.00", addon.Price)

		require.Equal(t, 1, integration.countEvents(
			t, fixture.tenantID, addonID,
			string(domain.EventTypeAddonUpdated),
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
			`{"name":"Invalid","price":"10.00","unknown":true}`,
			fixture.tenantID,
		)
		require.Equal(t, http.StatusBadRequest, response.Code)
	})

	t.Run("invalid price rejected", func(t *testing.T) {
		response := request(
			http.MethodPost,
			basePath,
			`{"name":"Invalid Price","price":"not-money"}`,
			fixture.tenantID,
		)
		require.Equal(t, http.StatusBadRequest, response.Code)
	})

	t.Run("invalid addon ID rejected", func(t *testing.T) {
		response := request(
			http.MethodGet,
			basePath+"/not-a-uuid",
			"",
			fixture.tenantID,
		)
		require.Equal(t, http.StatusBadRequest, response.Code)
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
			basePath+"/"+addonID.String(),
			"",
			uuid.Nil,
		)
		require.Equal(t, http.StatusUnauthorized, response.Code)
	})

	t.Run("other tenant cannot read addon", func(t *testing.T) {
		response := request(
			http.MethodGet,
			basePath+"/"+addonID.String(),
			"",
			uuid.New(),
		)
		require.Equal(t, http.StatusNotFound, response.Code,
			response.Body.String())
	})

	t.Run("wrong restaurant cannot read addon", func(t *testing.T) {
		path := "/api/v1/restaurants/" +
			uuid.New().String() +
			"/menu/addons/" + addonID.String()

		response := request(
			http.MethodGet,
			path,
			"",
			fixture.tenantID,
		)
		require.Equal(t, http.StatusNotFound, response.Code,
			response.Body.String())
	})

	t.Run("deactivate addon", func(t *testing.T) {
		response := request(
			http.MethodPatch,
			basePath+"/"+addonID.String()+"/deactivate",
			"",
			fixture.tenantID,
		)

		require.Equal(t, http.StatusOK, response.Code,
			response.Body.String())

		addon := decodeAddon(response)
		require.False(t, addon.IsActive)

		require.Equal(t, 1, integration.countEvents(
			t, fixture.tenantID, addonID,
			string(domain.EventTypeAddonDeactivated),
		))
	})

	t.Run("inactive addon cannot be updated", func(t *testing.T) {
		response := request(
			http.MethodPut,
			basePath+"/"+addonID.String(),
			`{"name":"Should Not Persist","price":"99.00"}`,
			fixture.tenantID,
		)

		require.NotEqual(t, http.StatusOK, response.Code)

		verify := request(
			http.MethodGet,
			basePath+"/"+addonID.String(),
			"",
			fixture.tenantID,
		)
		require.Equal(t, http.StatusOK, verify.Code,
			verify.Body.String())

		addon := decodeAddon(verify)
		require.Equal(t, "HTTP Premium Cheese", addon.Name)
		require.Equal(t, "35.00", addon.Price)
		require.False(t, addon.IsActive)
	})
}
