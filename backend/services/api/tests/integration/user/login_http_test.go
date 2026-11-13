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
	"github.com/afadnan/restaurantflow/services/api/internal/platform/auth"
)

func TestIntegrationOwnerLoginHTTP(t *testing.T) {
	i := newIntegrationDB(t)

	tenantID := i.createTenant(t)
	slug := "user-it-" + tenantID.String()

	user := createLoginTestUser(
		t,
		i,
		tenantID,
		"owner@example.com",
		integrationLoginPassword,
	)

	loginUseCase, tokenService := newLoginTestUseCase(i)

	registrationHandler := userhttp.NewOwnerRegistrationHandler(
		newOwnerRegistrationUseCase(
			i,
			userrepository.NewPostgresUserRepository(),
		),
	)

	loginHandler := userhttp.NewOwnerLoginHandler(loginUseCase)

	mux := http.NewServeMux()
	userhttp.RegisterRoutes(mux, registrationHandler, loginHandler)

	sendLoginRequest := func(body string) *httptest.ResponseRecorder {
		t.Helper()

		ctx, cancel := context.WithTimeout(
			context.Background(),
			15*time.Second,
		)
		defer cancel()

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/auth/login",
			strings.NewReader(body),
		).WithContext(ctx)

		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, req)

		return recorder
	}

	makeLoginBody := func(
		businessSlug string,
		email string,
		password string,
	) string {
		t.Helper()

		body, err := json.Marshal(map[string]string{
			"business_slug": businessSlug,
			"email":         email,
			"password":      password,
		})
		require.NoError(t, err)

		return string(body)
	}

	// ---------------------------------------------------------
	// Test 1: Successful owner login
	// ---------------------------------------------------------

	t.Run("successful login", func(t *testing.T) {
		recorder := sendLoginRequest(
			makeLoginBody(
				slug,
				" OWNER@EXAMPLE.COM ",
				integrationLoginPassword,
			),
		)

		require.Equal(
			t,
			http.StatusOK,
			recorder.Code,
			recorder.Body.String(),
		)

		require.Equal(
			t,
			"application/json",
			recorder.Header().Get("Content-Type"),
		)

		var response struct {
			AccessToken           string    `json:"access_token"`
			TokenType             string    `json:"token_type"`
			ExpiresAt             time.Time `json:"expires_at"`
			RefreshToken          string    `json:"refresh_token"`
			RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`

			Tenant struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				Slug   string `json:"slug"`
				Status string `json:"status"`
			} `json:"tenant"`

			User struct {
				ID        string `json:"id"`
				TenantID  string `json:"tenant_id"`
				Email     string `json:"email"`
				FirstName string `json:"first_name"`
				Role      string `json:"role"`
				Status    string `json:"status"`
			} `json:"user"`
		}

		require.NoError(
			t,
			json.Unmarshal(recorder.Body.Bytes(), &response),
		)

		// Verify access token response.
		require.NotEmpty(t, response.AccessToken)
		require.Equal(t, "Bearer", response.TokenType)
		require.False(t, response.ExpiresAt.IsZero())

		// Verify refresh token response.
		require.NotEmpty(t, response.RefreshToken)
		require.False(t, response.RefreshTokenExpiresAt.IsZero())

		require.WithinDuration(
			t,
			time.Now().Add(30*24*time.Hour),
			response.RefreshTokenExpiresAt,
			2*time.Minute,
		)

		// Verify tenant response.
		require.Equal(t, tenantID.String(), response.Tenant.ID)
		require.Equal(t, slug, response.Tenant.Slug)
		require.Equal(t, "active", response.Tenant.Status)

		// Verify user response.
		require.Equal(t, user.ID.String(), response.User.ID)
		require.Equal(t, tenantID.String(), response.User.TenantID)
		require.Equal(t, "owner@example.com", response.User.Email)
		require.Equal(t, "Integration", response.User.FirstName)
		require.Equal(t, "owner", response.User.Role)
		require.Equal(t, "active", response.User.Status)

		// Verify the returned access token.
		claims, err := tokenService.ValidateAccessToken(
			response.AccessToken,
		)
		require.NoError(t, err)
		require.Equal(t, user.ID.String(), claims.Subject)
		require.Equal(t, tenantID.String(), claims.TenantID)
		require.Equal(t, "owner", claims.Role)

		// Never expose password data in the HTTP response.
		require.NotContains(t, recorder.Body.String(), "password_hash")
		require.NotContains(t, recorder.Body.String(), `"password"`)
		require.NotContains(
			t,
			recorder.Body.String(),
			integrationLoginPassword,
		)

		// Verify LastLoginAt and refresh-token persistence.
		ctx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		tx, err := i.db.BeginTx(ctx, tenantID)
		require.NoError(t, err)

		defer func() {
			_ = tx.Rollback(context.Background())
		}()

		stored, err := userrepository.NewPostgresUserRepository().
			GetByID(ctx, tx, tenantID, user.ID)

		require.NoError(t, err)
		require.NotNil(t, stored.LastLoginAt)

		// Hash the returned plaintext refresh token.
		// The database must store only this hash, never the plaintext.
		refreshTokenHash, err := auth.HashRefreshToken(
			response.RefreshToken,
		)
		require.NoError(t, err)

		storedRefreshToken, err := userrepository.
			NewPostgresRefreshTokenRepository().
			GetByHash(ctx, tx, tenantID, refreshTokenHash)

		require.NoError(t, err)
		require.NotNil(t, storedRefreshToken)

		require.Equal(
			t,
			tenantID,
			storedRefreshToken.TenantID,
		)

		require.Equal(
			t,
			user.ID,
			storedRefreshToken.UserID,
		)

		require.Equal(
			t,
			refreshTokenHash,
			storedRefreshToken.TokenHash,
		)

		require.WithinDuration(
			t,
			response.RefreshTokenExpiresAt,
			storedRefreshToken.ExpiresAt,
			time.Second,
		)

		require.Nil(t, storedRefreshToken.RevokedAt)
		require.Nil(t, storedRefreshToken.ReplacedByTokenID)

		require.NoError(t, tx.Rollback(ctx))
	})

	// ---------------------------------------------------------
	// Test 2: Invalid credentials return the same HTTP response
	// ---------------------------------------------------------

	t.Run("invalid credentials return generic unauthorized response", func(t *testing.T) {
		testCases := []struct {
			name  string
			slug  string
			email string
			pass  string
		}{
			{
				name:  "wrong password",
				slug:  slug,
				email: "owner@example.com",
				pass:  "WrongPassword123!",
			},
			{
				name:  "unknown email",
				slug:  slug,
				email: "missing@example.com",
				pass:  integrationLoginPassword,
			},
			{
				name:  "unknown business slug",
				slug:  "missing-" + uuid.NewString(),
				email: "owner@example.com",
				pass:  integrationLoginPassword,
			},
		}

		var expectedBody string

		for index, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				recorder := sendLoginRequest(
					makeLoginBody(tc.slug, tc.email, tc.pass),
				)

				require.Equal(
					t,
					http.StatusUnauthorized,
					recorder.Code,
					recorder.Body.String(),
				)

				require.Contains(
					t,
					recorder.Body.String(),
					"invalid business slug, email or password",
				)

				// All credential failures must have the same response.
				if index == 0 {
					expectedBody = recorder.Body.String()
				} else {
					require.Equal(
						t,
						expectedBody,
						recorder.Body.String(),
					)
				}
			})
		}
	})

	// ---------------------------------------------------------
	// Test 3: Inactive owner cannot log in
	// ---------------------------------------------------------

	t.Run("inactive owner cannot login", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		tx, err := i.db.BeginTx(ctx, tenantID)
		require.NoError(t, err)

		_, err = tx.Raw().Exec(
			ctx,
			"UPDATE users SET status = 'inactive' WHERE id = $1",
			user.ID,
		)
		require.NoError(t, err)

		require.NoError(t, tx.Commit(ctx))

		recorder := sendLoginRequest(
			makeLoginBody(
				slug,
				"owner@example.com",
				integrationLoginPassword,
			),
		)

		require.Equal(
			t,
			http.StatusUnauthorized,
			recorder.Code,
			recorder.Body.String(),
		)

		require.Contains(
			t,
			recorder.Body.String(),
			"invalid business slug, email or password",
		)
	})

	// ---------------------------------------------------------
	// Test 4: Malformed JSON is rejected
	// ---------------------------------------------------------

	t.Run("malformed JSON rejected", func(t *testing.T) {
		recorder := sendLoginRequest(`{
			"business_slug": "` + slug + `",
			"email": "owner@example.com",
			"password": "SecureLoginPassword123!"
		`)

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
	})

	// ---------------------------------------------------------
	// Test 5: Unknown JSON fields are rejected
	// ---------------------------------------------------------

	t.Run("unknown JSON field rejected", func(t *testing.T) {
		body := makeLoginBody(
			slug,
			"owner@example.com",
			integrationLoginPassword,
		)

		var request map[string]any
		require.NoError(
			t,
			json.Unmarshal([]byte(body), &request),
		)

		request["admin"] = true

		bodyBytes, err := json.Marshal(request)
		require.NoError(t, err)

		recorder := sendLoginRequest(string(bodyBytes))

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
	})
}
