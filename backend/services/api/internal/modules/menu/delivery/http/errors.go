package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/usecase"
)

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidTenantID):
		http.Error(w, err.Error(), http.StatusUnauthorized)

	case errors.Is(err, domain.ErrInvalidRestaurantID),
		errors.Is(err, domain.ErrInvalidCategoryID),
		errors.Is(err, domain.ErrInvalidCategoryName),
		errors.Is(err, domain.ErrInvalidSortOrder),
		errors.Is(err, domain.ErrInvalidCategoryInput),
		errors.Is(err, usecase.ErrInvalidPagination):
		http.Error(w, err.Error(), http.StatusBadRequest)

	case errors.Is(err, domain.ErrCategoryNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)

	case errors.Is(err, usecase.ErrUseCaseNotConfigured):
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)

	default:
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
