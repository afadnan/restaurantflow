package httpdelivery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/afadnan/restaurantflow/services/api/internal/authorization"
	"github.com/afadnan/restaurantflow/services/api/internal/middleware"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/order/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/order/usecase"
	platformauth "github.com/afadnan/restaurantflow/services/api/internal/platform/auth"
)

const testKDSJWTSecret = "0123456789abcdef0123456789abcdef"

type fakeUpdateKDSStateExecutor struct {
	called bool
	input  usecase.UpdateKDSStateInput
	err    error
}

func (f *fakeUpdateKDSStateExecutor) Execute(
	ctx context.Context,
	input usecase.UpdateKDSStateInput,
) error {
	f.called = true
	f.input = input
	return f.err
}

func newKDSTestTokenService(t *testing.T) *platformauth.TokenService {
	t.Helper()

	service, err := platformauth.NewTokenService(
		testKDSJWTSecret,
		"restaurantflow-api",
		15*time.Minute,
	)
	if err != nil {
		t.Fatalf("create token service: %v", err)
	}

	return service
}

func newKDSAuthenticatedRequest(
	t *testing.T,
	tokenService *platformauth.TokenService,
	role string,
	state domain.KDSState,
	orderID uuid.UUID,
) *http.Request {
	t.Helper()

	userID := uuid.New()
	tenantID := uuid.New()

	token, _, err := tokenService.GenerateAccessToken(
		userID,
		tenantID,
		role,
	)
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}

	body := `{"state":"` + string(state) + `"}`

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/orders/"+orderID.String()+"/kds",
		strings.NewReader(body),
	)

	req.SetPathValue("orderID", orderID.String())
	req.Header.Set("Authorization", "Bearer "+token)

	return req
}

func TestUpdateKDSState_AuthorizedTransitions(t *testing.T) {
	tests := []struct {
		name  string
		state domain.KDSState
	}{
		{
			name:  "kitchen can start preparing",
			state: domain.KDSStatePreparing,
		},
		{
			name:  "kitchen can mark ready",
			state: domain.KDSStateReady,
		},
		{
			name:  "kitchen can complete",
			state: domain.KDSStateCompleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenService := newKDSTestTokenService(t)
			executor := &fakeUpdateKDSStateExecutor{}

			handler := NewOrderHandler(
				nil,
				executor,
				authorization.NewAuthorizer(),
			)

			protectedHandler := middleware.AuthMiddleware(tokenService)(
				middleware.TenantMiddleware(
					http.HandlerFunc(handler.UpdateKDSState),
				),
			)

			orderID := uuid.New()

			req := newKDSAuthenticatedRequest(
				t,
				tokenService,
				"kitchen",
				tt.state,
				orderID,
			)

			recorder := httptest.NewRecorder()
			protectedHandler.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusNoContent {
				t.Fatalf(
					"status = %d, want %d; body = %s",
					recorder.Code,
					http.StatusNoContent,
					recorder.Body.String(),
				)
			}

			if !executor.called {
				t.Fatal("expected KDS use case to be called")
			}

			if executor.input.OrderID != orderID {
				t.Errorf(
					"order ID = %s, want %s",
					executor.input.OrderID,
					orderID,
				)
			}

			if executor.input.State != tt.state {
				t.Errorf(
					"state = %q, want %q",
					executor.input.State,
					tt.state,
				)
			}

			if executor.input.TenantID == uuid.Nil {
				t.Error("expected tenant ID to be passed to use case")
			}
		})
	}
}

func TestUpdateKDSState_UnauthorizedTransitions(t *testing.T) {
	tests := []struct {
		name  string
		role  string
		state domain.KDSState
	}{
		{
			name:  "cashier cannot start preparing",
			role:  "cashier",
			state: domain.KDSStatePreparing,
		},
		{
			name:  "cashier cannot mark ready",
			role:  "cashier",
			state: domain.KDSStateReady,
		},
		{
			name:  "cashier cannot complete",
			role:  "cashier",
			state: domain.KDSStateCompleted,
		},
		{
			name:  "staff cannot start preparing",
			role:  "staff",
			state: domain.KDSStatePreparing,
		},
		{
			name:  "unknown role cannot start preparing",
			role:  "unknown",
			state: domain.KDSStatePreparing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenService := newKDSTestTokenService(t)
			executor := &fakeUpdateKDSStateExecutor{}

			handler := NewOrderHandler(
				nil,
				executor,
				authorization.NewAuthorizer(),
			)

			protectedHandler := middleware.AuthMiddleware(tokenService)(
				middleware.TenantMiddleware(
					http.HandlerFunc(handler.UpdateKDSState),
				),
			)

			req := newKDSAuthenticatedRequest(
				t,
				tokenService,
				tt.role,
				tt.state,
				uuid.New(),
			)

			recorder := httptest.NewRecorder()
			protectedHandler.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusForbidden {
				t.Fatalf(
					"status = %d, want %d; body = %s",
					recorder.Code,
					http.StatusForbidden,
					recorder.Body.String(),
				)
			}

			if executor.called {
				t.Fatal("use case must not be called for unauthorized request")
			}
		})
	}
}

func TestUpdateKDSState_InvalidTargets(t *testing.T) {
	tests := []struct {
		name  string
		state string
	}{
		{
			name:  "pending is not a valid transition target",
			state: "pending",
		},
		{
			name:  "unknown state is rejected",
			state: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenService := newKDSTestTokenService(t)
			executor := &fakeUpdateKDSStateExecutor{}

			handler := NewOrderHandler(
				nil,
				executor,
				authorization.NewAuthorizer(),
			)

			protectedHandler := middleware.AuthMiddleware(tokenService)(
				middleware.TenantMiddleware(
					http.HandlerFunc(handler.UpdateKDSState),
				),
			)

			req := httptest.NewRequest(
				http.MethodPatch,
				"/api/v1/orders/"+uuid.New().String()+"/kds",
				strings.NewReader(`{"state":"`+tt.state+`"}`),
			)

			req.SetPathValue("orderID", uuid.New().String())

			token, _, err := tokenService.GenerateAccessToken(
				uuid.New(),
				uuid.New(),
				"kitchen",
			)
			if err != nil {
				t.Fatalf("generate access token: %v", err)
			}

			req.Header.Set("Authorization", "Bearer "+token)

			recorder := httptest.NewRecorder()
			protectedHandler.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf(
					"status = %d, want %d; body = %s",
					recorder.Code,
					http.StatusBadRequest,
					recorder.Body.String(),
				)
			}

			if executor.called {
				t.Fatal("use case must not be called for invalid state")
			}
		})
	}
}

func TestUpdateKDSState_MissingRole(t *testing.T) {
	executor := &fakeUpdateKDSStateExecutor{}

	handler := NewOrderHandler(
		nil,
		executor,
		authorization.NewAuthorizer(),
	)

	tenantID := uuid.New()
	orderID := uuid.New()

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/orders/"+orderID.String()+"/kds",
		strings.NewReader(`{"state":"preparing"}`),
	)

	req.SetPathValue("orderID", orderID.String())

	// Supply a tenant but deliberately omit authenticated user/role context.
	req = req.WithContext(
		middleware.WithTenant(req.Context(), tenantID),
	)

	recorder := httptest.NewRecorder()
	handler.UpdateKDSState(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"status = %d, want %d",
			recorder.Code,
			http.StatusUnauthorized,
		)
	}

	if executor.called {
		t.Fatal("use case must not be called without an authenticated role")
	}
}

func TestUpdateKDSState_NilAuthorizer(t *testing.T) {
	tokenService := newKDSTestTokenService(t)
	executor := &fakeUpdateKDSStateExecutor{}

	handler := NewOrderHandler(
		nil,
		executor,
		nil,
	)

	protectedHandler := middleware.AuthMiddleware(tokenService)(
		middleware.TenantMiddleware(
			http.HandlerFunc(handler.UpdateKDSState),
		),
	)

	req := newKDSAuthenticatedRequest(
		t,
		tokenService,
		"kitchen",
		domain.KDSStatePreparing,
		uuid.New(),
	)

	recorder := httptest.NewRecorder()
	protectedHandler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf(
			"status = %d, want %d",
			recorder.Code,
			http.StatusForbidden,
		)
	}

	if executor.called {
		t.Fatal("use case must not be called when authorizer is nil")
	}
}
