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
		// Never expose database errors or internal details.
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
	}
}
