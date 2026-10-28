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

func TestIntegrationMenuItemAddonHTTP(t *testing.T) {
	integration := newVariantIntegration(t)
	fixture := integration.createFixture(t)

	addonDeps := usecase.AddonUseCaseDependencies{
		DB:     integration.db,
		Addons: repository.NewPostgresAddonRepository(),
		Events: events.NewPostgresEventRepository(),
	}

	itemAddonDeps := usecase.ItemAddonUseCaseDependencies{
		DB:         integration.db,
		ItemAddons: repository.NewPostgresItemAddonRepository(),
		Events:     events.NewPostgresEventRepository(),
	}

	addonHandler := menuhttp.NewAddonHandler(
		&usecase.CreateAddonUseCase{
			AddonUseCaseDependencies: addonDeps,
		},
		nil, nil, nil, nil,
	)

	itemAddonHandler := menuhttp.NewItemAddonHandler(
		&usecase.AttachItemAddonUseCase{
			ItemAddonUseCaseDependencies: itemAddonDeps,
		},
		&usecase.GetItemAddonUseCase{
			ItemAddonUseCaseDependencies: itemAddonDeps,
		},
		&usecase.ListItemAddonsUseCase{
			ItemAddonUseCaseDependencies: itemAddonDeps,
		},
		&usecase.UpdateItemAddonUseCase{
			ItemAddonUseCaseDependencies: itemAddonDeps,
		},
		&usecase.DetachItemAddonUseCase{
			ItemAddonUseCaseDependencies: itemAddonDeps,
		},
	)

	mux := http.NewServeMux()
	menuhttp.RegisterAddonRoutes(mux, addonHandler)
	menuhttp.RegisterItemAddonRoutes(mux, itemAddonHandler)

	addonBasePath := fmt.Sprintf(
		"/api/v1/restaurants/%s/menu/addons",
		fixture.restaurantID,
	)

	itemAddonBasePath := fmt.Sprintf(
		"/api/v1/restaurants/%s/menu/items/%s/addons",
		fixture.restaurantID,
		fixture.itemID,
	)

	type itemAddonJSON struct {
		TenantID     string `json:"tenant_id"`
		RestaurantID string `json:"restaurant_id"`
		ItemID       string `json:"item_id"`
		AddonID      string `json:"addon_id"`
		IsRequired   bool   `json:"is_required"`
		MaxQuantity  int32  `json:"max_quantity"`
	}

	request := func(
		method, path, body string,
		tenantID uuid.UUID,
	) *httptest.ResponseRecorder {
		t.Helper()

		ctx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		req := httptest.NewRequest(
			method,
			path,
			strings.NewReader(body),
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

	decodeAssociation := func(
		recorder *httptest.ResponseRecorder,
	) itemAddonJSON {
		t.Helper()

		var result itemAddonJSON
		require.NoError(t, json.Unmarshal(
			recorder.Body.Bytes(),
			&result,
		))
		return result
	}

	// Create an actual add-on through the Add-on HTTP endpoint.
	createResponse := request(
		http.MethodPost,
		addonBasePath,
		`{"name":"Item HTTP Extra Cheese","description":"Test topping","price":"25.00"}`,
		fixture.tenantID,
	)
	require.Equal(t, http.StatusCreated, createResponse.Code,
		createResponse.Body.String())

	var createdAddon struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(
		createResponse.Body.Bytes(),
		&createdAddon,
	))

	addonID, err := uuid.Parse(createdAddon.ID)
	require.NoError(t, err)

	associationPath := itemAddonBasePath + "/" + addonID.String()

	t.Run("attach addon", func(t *testing.T) {
		body := fmt.Sprintf(
			`{"addon_id":%q,"is_required":false,"max_quantity":2}`,
			addonID.String(),
		)

		response := request(
			http.MethodPost,
			itemAddonBasePath,
			body,
			fixture.tenantID,
		)

		require.Equal(t, http.StatusCreated, response.Code,
			response.Body.String())

		association := decodeAssociation(response)

		require.Equal(t, fixture.tenantID.String(), association.TenantID)
		require.Equal(t, fixture.restaurantID.String(), association.RestaurantID)
		require.Equal(t, fixture.itemID.String(), association.ItemID)
		require.Equal(t, addonID.String(), association.AddonID)
		require.False(t, association.IsRequired)
		require.EqualValues(t, 2, association.MaxQuantity)

		require.Equal(t, 1, integration.countEvents(
			t,
			fixture.tenantID,
			fixture.itemID,
			string(domain.EventTypeItemAddonAttached),
		))
	})

	t.Run("get association", func(t *testing.T) {
		response := request(
			http.MethodGet,
			associationPath,
			"",
			fixture.tenantID,
		)

		require.Equal(t, http.StatusOK, response.Code,
			response.Body.String())

		association := decodeAssociation(response)
		require.Equal(t, addonID.String(), association.AddonID)
		require.EqualValues(t, 2, association.MaxQuantity)
	})

	t.Run("list associations", func(t *testing.T) {
		response := request(
			http.MethodGet,
			itemAddonBasePath+"?limit=10&offset=0",
			"",
			fixture.tenantID,
		)

		require.Equal(t, http.StatusOK, response.Code,
			response.Body.String())

		var result struct {
			Addons []itemAddonJSON `json:"addons"`
			Limit  int             `json:"limit"`
			Offset int             `json:"offset"`
		}

		require.NoError(t, json.Unmarshal(
			response.Body.Bytes(),
			&result,
		))

		require.Equal(t, 10, result.Limit)
		require.Zero(t, result.Offset)
		require.Len(t, result.Addons, 1)
		require.Equal(t, addonID.String(), result.Addons[0].AddonID)
	})

	t.Run("duplicate attachment rejected", func(t *testing.T) {
		body := fmt.Sprintf(
			`{"addon_id":%q,"is_required":false,"max_quantity":2}`,
			addonID.String(),
		)

		response := request(
			http.MethodPost,
			itemAddonBasePath,
			body,
			fixture.tenantID,
		)

		require.Equal(t, http.StatusConflict, response.Code,
			response.Body.String())

		require.Equal(t, 1, integration.countEvents(
			t,
			fixture.tenantID,
			fixture.itemID,
			string(domain.EventTypeItemAddonAttached),
		))
	})

	t.Run("update association", func(t *testing.T) {
		response := request(
			http.MethodPut,
			associationPath,
			`{"is_required":true,"max_quantity":3}`,
			fixture.tenantID,
		)

		require.Equal(t, http.StatusOK, response.Code,
			response.Body.String())

		association := decodeAssociation(response)
		require.True(t, association.IsRequired)
		require.EqualValues(t, 3, association.MaxQuantity)

		require.Equal(t, 1, integration.countEvents(
			t,
			fixture.tenantID,
			fixture.itemID,
			string(domain.EventTypeItemAddonUpdated),
		))
	})

	t.Run("invalid quantity rejected", func(t *testing.T) {
		response := request(
			http.MethodPut,
			associationPath,
			`{"is_required":true,"max_quantity":-1}`,
			fixture.tenantID,
		)

		require.Equal(t, http.StatusBadRequest, response.Code,
			response.Body.String())

		require.Equal(t, 1, integration.countEvents(
			t,
			fixture.tenantID,
			fixture.itemID,
			string(domain.EventTypeItemAddonUpdated),
		))
	})

	t.Run("unknown JSON field rejected", func(t *testing.T) {
		body := fmt.Sprintf(
			`{"addon_id":%q,"is_required":false,"max_quantity":2,"unexpected":true}`,
			addonID.String(),
		)

		response := request(
			http.MethodPost,
			itemAddonBasePath,
			body,
			fixture.tenantID,
		)

		require.Equal(t, http.StatusBadRequest, response.Code)
	})

	t.Run("invalid addon ID rejected", func(t *testing.T) {
		response := request(
			http.MethodPost,
			itemAddonBasePath,
			`{"addon_id":"not-a-uuid","max_quantity":2}`,
			fixture.tenantID,
		)

		require.Equal(t, http.StatusBadRequest, response.Code)
	})

	t.Run("invalid item ID rejected", func(t *testing.T) {
		path := fmt.Sprintf(
			"/api/v1/restaurants/%s/menu/items/not-a-uuid/addons",
			fixture.restaurantID,
		)

		response := request(
			http.MethodGet,
			path,
			"",
			fixture.tenantID,
		)

		require.Equal(t, http.StatusBadRequest, response.Code)
	})

	t.Run("invalid pagination rejected", func(t *testing.T) {
		response := request(
			http.MethodGet,
			itemAddonBasePath+"?offset=-1",
			"",
			fixture.tenantID,
		)

		require.Equal(t, http.StatusBadRequest, response.Code)
	})

	t.Run("missing tenant rejected", func(t *testing.T) {
		response := request(
			http.MethodGet,
			associationPath,
			"",
			uuid.Nil,
		)

		require.Equal(t, http.StatusUnauthorized, response.Code)
	})

	t.Run("other tenant cannot read association", func(t *testing.T) {
		response := request(
			http.MethodGet,
			associationPath,
			"",
			uuid.New(),
		)

		require.Equal(t, http.StatusNotFound, response.Code,
			response.Body.String())
	})

	t.Run("wrong restaurant cannot read association", func(t *testing.T) {
		path := fmt.Sprintf(
			"/api/v1/restaurants/%s/menu/items/%s/addons/%s",
			uuid.New(),
			fixture.itemID,
			addonID,
		)

		response := request(
			http.MethodGet,
			path,
			"",
			fixture.tenantID,
		)

		require.Equal(t, http.StatusNotFound, response.Code,
			response.Body.String())
	})

	t.Run("detach association", func(t *testing.T) {
		response := request(
			http.MethodDelete,
			associationPath,
			"",
			fixture.tenantID,
		)

		require.Equal(t, http.StatusOK, response.Code,
			response.Body.String())

		require.Equal(t, 1, integration.countEvents(
			t,
			fixture.tenantID,
			fixture.itemID,
			string(domain.EventTypeItemAddonDetached),
		))
	})

	t.Run("detached association cannot be fetched", func(t *testing.T) {
		response := request(
			http.MethodGet,
			associationPath,
			"",
			fixture.tenantID,
		)

		require.Equal(t, http.StatusNotFound, response.Code,
			response.Body.String())
	})
}
