package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/afadnan/restaurantflow/services/api/internal/authorization"
	userdomain "github.com/afadnan/restaurantflow/services/api/internal/modules/user/domain"
)

func TestRequirePermission(t *testing.T) {
	tests := []struct {
		name           string
		role           userdomain.Role
		permission     authorization.Permission
		wantStatusCode int
	}{
		{
			name:           "manager can read menu",
			role:           userdomain.RoleManager,
			permission:     authorization.PermissionMenuRead,
			wantStatusCode: http.StatusNoContent,
		},
		{
			name:           "staff cannot create menu",
			role:           userdomain.RoleStaff,
			permission:     authorization.PermissionMenuCreate,
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "cashier can collect payments",
			role:           userdomain.RoleCashier,
			permission:     authorization.PermissionPaymentsCollect,
			wantStatusCode: http.StatusNoContent,
		},
		{
			name:           "cashier cannot refund payments",
			role:           userdomain.RoleCashier,
			permission:     authorization.PermissionPaymentsRefund,
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "unknown role is denied",
			role:           userdomain.Role("unknown"),
			permission:     authorization.PermissionMenuRead,
			wantStatusCode: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authorizer := authorization.NewAuthorizer()

			handlerCalled := false
			next := http.HandlerFunc(func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				handlerCalled = true
				w.WriteHeader(http.StatusNoContent)
			})

			handler := RequirePermission(
				authorizer,
				tt.permission,
			)(next)

			request := httptest.NewRequest(
				http.MethodGet,
				"/test",
				nil,
			)

			ctx := context.WithValue(
				request.Context(),
				authenticatedUserContextKey{},
				authenticatedUser{
					role: string(tt.role),
				},
			)

			request = request.WithContext(ctx)

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != tt.wantStatusCode {
				t.Fatalf(
					"status = %d, want %d; body = %s",
					response.Code,
					tt.wantStatusCode,
					response.Body.String(),
				)
			}

			wantHandlerCalled := tt.wantStatusCode == http.StatusNoContent
			if handlerCalled != wantHandlerCalled {
				t.Fatalf(
					"handler called = %v, want %v",
					handlerCalled,
					wantHandlerCalled,
				)
			}
		})
	}
}

func TestRequirePermissionWithoutAuthentication(t *testing.T) {
	handler := RequirePermission(
		authorization.NewAuthorizer(),
		authorization.PermissionMenuRead,
	)(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		t.Fatal("handler must not be called without authentication")
	}))

	request := httptest.NewRequest(
		http.MethodGet,
		"/test",
		nil,
	)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf(
			"status = %d, want %d",
			response.Code,
			http.StatusUnauthorized,
		)
	}
}

func TestRequirePermissionWithNilAuthorizer(t *testing.T) {
	handler := RequirePermission(
		nil,
		authorization.PermissionMenuRead,
	)(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		t.Fatal("handler must not be called with nil authorizer")
	}))

	request := httptest.NewRequest(
		http.MethodGet,
		"/test",
		nil,
	)

	ctx := context.WithValue(
		request.Context(),
		authenticatedUserContextKey{},
		authenticatedUser{
			role: string(userdomain.RoleOwner),
		},
	)

	request = request.WithContext(ctx)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf(
			"status = %d, want %d",
			response.Code,
			http.StatusForbidden,
		)
	}
}
