package user_test

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

	userhttp "github.com/afadnan/restaurantflow/services/api/internal/modules/user/delivery/http"
	userrepository "github.com/afadnan/restaurantflow/services/api/internal/modules/user/repository"
)

func TestIntegrationOwnerRegistrationHTTP(t *testing.T) {
	integration := newIntegrationDB(t)
	cleanupPool := newOwnerCleanupPool(t)

	registerOwner := newOwnerRegistrationUseCase(
		integration,
		userrepository.NewPostgresUserRepository(),
	)

	handler := userhttp.NewOwnerRegistrationHandler(registerOwner)

	mux := http.NewServeMux()
	userhttp.RegisterRoutes(mux, handler, nil)

	// ---------------------------------------------------------
	// Test 1: Successful owner registration
	// ---------------------------------------------------------

	t.Run("successful registration", func(t *testing.T) {
		slug := "owner-http-" + uuid.NewString()

		body, err := json.Marshal(map[string]string{
			"business_name": "HTTP Integration Cafe",
			"business_slug": slug,
			"email":         "owner@example.com",
			"password":      "SecureRegistrationPassword123!",
			"first_name":    "Integration",
		})
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(
			context.Background(),
			15*time.Second,
		)
		defer cancel()

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/auth/register/owner",
			strings.NewReader(string(body)),
		).WithContext(ctx)

		req.Header.Set("Content-Type", "application/json")

		// Registration must work without X-Tenant-ID.
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, req)

		require.Equal(
			t,
			http.StatusCreated,
			recorder.Code,
			recorder.Body.String(),
		)

		require.Equal(
			t,
			"application/json",
			recorder.Header().Get("Content-Type"),
		)

		var response struct {
			Tenant struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				Slug   string `json:"slug"`
				Status string `json:"status"`
			} `json:"tenant"`

			Owner struct {
				ID        string `json:"id"`
				TenantID  string `json:"tenant_id"`
				Email     string `json:"email"`
				FirstName string `json:"first_name"`
				Role      string `json:"role"`
				Status    string `json:"status"`
			} `json:"owner"`
		}

		require.NoError(
			t,
			json.Unmarshal(recorder.Body.Bytes(), &response),
		)

		tenantID, err := uuid.Parse(response.Tenant.ID)
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, tenantID)

		ownerID, err := uuid.Parse(response.Owner.ID)
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, ownerID)

		// Clean up test-created records.
		cleanupRegisteredOwner(
			t,
			cleanupPool,
			tenantID,
			ownerID,
		)

		require.Equal(t, slug, response.Tenant.Slug)
		require.Equal(t, "HTTP Integration Cafe", response.Tenant.Name)
		require.Equal(t, "active", response.Tenant.Status)

		require.Equal(t, tenantID.String(), response.Owner.TenantID)
		require.Equal(t, "owner@example.com", response.Owner.Email)
		require.Equal(t, "Integration", response.Owner.FirstName)
		require.Equal(t, "owner", response.Owner.Role)
		require.Equal(t, "active", response.Owner.Status)

		// Never expose passwords or password hashes.
		require.NotContains(
			t,
			recorder.Body.String(),
			"password_hash",
		)
		require.NotContains(
			t,
			recorder.Body.String(),
			`"password"`,
		)
		require.NotContains(
			t,
			recorder.Body.String(),
			"SecureRegistrationPassword123!",
		)

		// Verify the tenant was committed.
		require.Equal(
			t,
			1,
			countTenantBySlug(t, integration, slug),
		)

		// Verify the owner was committed under the tenant.
		tx, err := integration.db.BeginTx(ctx, tenantID)
		require.NoError(t, err)
		defer func() {
			_ = tx.Rollback(context.Background())
		}()

		storedOwner, err := userrepository.NewPostgresUserRepository().
			GetByID(ctx, tx, tenantID, ownerID)

		require.NoError(t, err)
		require.Equal(t, tenantID, storedOwner.TenantID)
		require.Equal(t, "owner@example.com", storedOwner.Email)
		require.Equal(t, "owner", string(storedOwner.Role))

		require.NoError(t, tx.Rollback(ctx))
	})

	// ---------------------------------------------------------
	// Test 2: Invalid password rejected
	// ---------------------------------------------------------

	t.Run("invalid password rejected", func(t *testing.T) {
		slug := "owner-http-invalid-" + uuid.NewString()

		body, err := json.Marshal(map[string]string{
			"business_name": "Invalid Password Cafe",
			"business_slug": slug,
			"email":         "owner@example.com",
			"password":      "short",
			"first_name":    "Integration",
		})
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(
			context.Background(),
			15*time.Second,
		)
		defer cancel()

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/auth/register/owner",
			strings.NewReader(string(body)),
		).WithContext(ctx)

		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, req)

		// Invalid password must return 400.
		require.Equal(
			t,
			http.StatusBadRequest,
			recorder.Code,
			recorder.Body.String(),
		)

		require.Contains(
			t,
			recorder.Body.String(),
			"invalid registration details",
		)

		// No tenant should be created.
		require.Zero(
			t,
			countTenantBySlug(t, integration, slug),
		)
	})

	// ---------------------------------------------------------
	// Test 3: Malformed JSON rejected
	// ---------------------------------------------------------

	t.Run("malformed JSON rejected", func(t *testing.T) {
		slug := "owner-http-malformed-" + uuid.NewString()

		// Intentionally malformed JSON: missing closing brace.
		body := `{
			"business_name": "Malformed JSON Cafe",
			"business_slug": "` + slug + `",
			"email": "owner@example.com",
			"password": "SecureRegistrationPassword123!",
			"first_name": "Integration"
		`

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/auth/register/owner",
			strings.NewReader(body),
		)

		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, req)

		require.Equal(
			t,
			http.StatusBadRequest,
			recorder.Code,
			recorder.Body.String(),
		)

		require.Contains(
			t,
			recorder.Body.String(),
			"invalid request body",
		)

		// Malformed JSON must not create a tenant.
		require.Zero(
			t,
			countTenantBySlug(t, integration, slug),
		)
	})

	// ---------------------------------------------------------
	// Test 4: Unknown JSON field rejected
	// ---------------------------------------------------------

	t.Run("unknown JSON field rejected", func(t *testing.T) {
		slug := "owner-http-unknown-" + uuid.NewString()

		body, err := json.Marshal(map[string]string{
			"business_name": "Unknown Field Cafe",
			"business_slug": slug,
			"email":         "owner@example.com",
			"password":      "SecureRegistrationPassword123!",
			"first_name":    "Integration",
			"admin":         "true", // Unexpected field
		})
		require.NoError(t, err)

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/auth/register/owner",
			strings.NewReader(string(body)),
		)

		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, req)

		require.Equal(
			t,
			http.StatusBadRequest,
			recorder.Code,
			recorder.Body.String(),
		)

		require.Contains(
			t,
			recorder.Body.String(),
			"invalid request body",
		)

		// Unknown fields must not create a tenant.
		require.Zero(
			t,
			countTenantBySlug(t, integration, slug),
		)
	})

	// ---------------------------------------------------------
	// Test 5: Duplicate business slug rejected
	// ---------------------------------------------------------

	t.Run("duplicate business slug rejected", func(t *testing.T) {
		slug := "owner-http-duplicate-" + uuid.NewString()

		makeBody := func(email string) []byte {
			body, err := json.Marshal(map[string]string{
				"business_name": "Duplicate Slug Cafe",
				"business_slug": slug,
				"email":         email,
				"password":      "SecureRegistrationPassword123!",
				"first_name":    "Integration",
			})
			require.NoError(t, err)
			return body
		}

		sendRequest := func(body []byte) *httptest.ResponseRecorder {
			req := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/auth/register/owner",
				strings.NewReader(string(body)),
			)
			req.Header.Set("Content-Type", "application/json")

			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, req)
			return recorder
		}

		// First registration should succeed.
		first := sendRequest(makeBody("first@example.com"))

		require.Equal(
			t,
			http.StatusCreated,
			first.Code,
			first.Body.String(),
		)

		var response struct {
			Tenant struct {
				ID string `json:"id"`
			} `json:"tenant"`
			Owner struct {
				ID string `json:"id"`
			} `json:"owner"`
		}

		require.NoError(t, json.Unmarshal(first.Body.Bytes(), &response))

		tenantID, err := uuid.Parse(response.Tenant.ID)
		require.NoError(t, err)

		ownerID, err := uuid.Parse(response.Owner.ID)
		require.NoError(t, err)

		// Ensure the first registration is cleaned up.
		cleanupRegisteredOwner(t, cleanupPool, tenantID, ownerID)

		// Second registration uses the same slug but a different email.
		second := sendRequest(makeBody("second@example.com"))

		require.Equal(
			t,
			http.StatusConflict,
			second.Code,
			second.Body.String(),
		)

		require.Contains(
			t,
			second.Body.String(),
			"business slug already exists",
		)

		// Only the original tenant should exist.
		require.Equal(
			t,
			1,
			countTenantBySlug(t, integration, slug),
		)
	})

	// ---------------------------------------------------------
	// Test 6: Owner insertion failure rolls back tenant
	// ---------------------------------------------------------

	t.Run("owner insertion failure rolls back tenant", func(t *testing.T) {
		slug := "owner-http-rollback-" + uuid.NewString()

		// Inject a repository that simulates owner insertion failure.
		failingUseCase := newOwnerRegistrationUseCase(
			integration,
			&failingOwnerRepository{},
		)

		failingHandler := userhttp.NewOwnerRegistrationHandler(failingUseCase)

		failingMux := http.NewServeMux()
		userhttp.RegisterRoutes(failingMux, failingHandler, nil)

		body, err := json.Marshal(map[string]string{
			"business_name": "Rollback Test Cafe",
			"business_slug": slug,
			"email":         "owner@example.com",
			"password":      "SecureRegistrationPassword123!",
			"first_name":    "Integration",
		})
		require.NoError(t, err)

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/auth/register/owner",
			strings.NewReader(string(body)),
		)
		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		failingMux.ServeHTTP(recorder, req)

		// Unexpected repository failure must return HTTP 500.
		require.Equal(
			t,
			http.StatusInternalServerError,
			recorder.Code,
			recorder.Body.String(),
		)

		// Tenant creation must be rolled back.
		require.Zero(
			t,
			countTenantBySlug(t, integration, slug),
		)
	})
}
