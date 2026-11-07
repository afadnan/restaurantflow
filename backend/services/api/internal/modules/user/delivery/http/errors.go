package httpdelivery

import (
	"errors"
	"net/http"

	tenantdomain "github.com/afadnan/restaurantflow/services/api/internal/modules/tenant/domain"
	tenantrepository "github.com/afadnan/restaurantflow/services/api/internal/modules/tenant/repository"
	userdomain "github.com/afadnan/restaurantflow/services/api/internal/modules/user/domain"
	userrepository "github.com/afadnan/restaurantflow/services/api/internal/modules/user/repository"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/user/usecase"
)

func writeRegistrationError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(err, tenantrepository.ErrTenantSlugExists):
		http.Error(
			w,
			"business slug already exists",
			http.StatusConflict,
		)

	case errors.Is(err, userrepository.ErrEmailExists):
		http.Error(
			w,
			"email already exists",
			http.StatusConflict,
		)

	case errors.Is(err, tenantdomain.ErrInvalidTenantName),
		errors.Is(err, tenantdomain.ErrInvalidTenantSlug),
		errors.Is(err, userdomain.ErrInvalidEmail),
		errors.Is(err, userdomain.ErrInvalidName),
		errors.Is(err, usecase.ErrInvalidRegistrationPassword):
		http.Error(
			w,
			"invalid registration details",
			http.StatusBadRequest,
		)

	default:
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
	}
}

// writeLoginError maps login failures to safe public HTTP responses.
func writeLoginError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(err, usecase.ErrInvalidCredentials):
		// Do not disclose whether the business, email, password,
		// or account status caused the authentication failure.
		http.Error(
			w,
			"invalid business slug, email or password",
			http.StatusUnauthorized,
		)

	case errors.Is(err, usecase.ErrInvalidLoginInput):
		http.Error(
			w,
			"invalid login details",
			http.StatusBadRequest,
		)

	default:
		// Never expose internal errors or database details.
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
	}
}
