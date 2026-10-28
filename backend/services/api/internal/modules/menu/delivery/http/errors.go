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
		errors.Is(err, domain.ErrInvalidItemID),
		errors.Is(err, domain.ErrInvalidItemName),
		errors.Is(err, domain.ErrInvalidItemStatus),
		errors.Is(err, domain.ErrInvalidItemInput),
		errors.Is(err, domain.ErrInvalidItemPrice),
		errors.Is(err, domain.ErrInvalidAddonID),
		errors.Is(err, domain.ErrInvalidAddonName),
		errors.Is(err, domain.ErrInvalidAddonPrice),
		errors.Is(err, domain.ErrInvalidAddonInput),
		errors.Is(err, domain.ErrInvalidAddonMaxQuantity),
		errors.Is(err, domain.ErrInvalidItemAddonInput),
		errors.Is(err, usecase.ErrInvalidPagination):
		http.Error(w, err.Error(), http.StatusBadRequest)

	case errors.Is(err, domain.ErrCategoryNotFound),
		errors.Is(err, domain.ErrItemNotFound),
		errors.Is(err, domain.ErrAddonNotFound),
		errors.Is(err, domain.ErrItemAddonNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)

	case errors.Is(err, domain.ErrAddonInactive),
		errors.Is(err, domain.ErrItemAddonAlreadyExists):
		http.Error(w, err.Error(), http.StatusConflict)

	case errors.Is(err, usecase.ErrUseCaseNotConfigured),
		errors.Is(err, usecase.ErrItemUseCaseNotConfigured),
		errors.Is(err, usecase.ErrAddonUseCaseNotConfigured),
		errors.Is(err, usecase.ErrItemAddonUseCaseNotConfigured):
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
