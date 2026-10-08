package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/afadnan/restaurantflow/services/api/internal/middleware"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/restaurant/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/restaurant/usecase"
)

type RestaurantHandler struct {
	CreateRestaurantUseCase     *usecase.CreateRestaurantUseCase
	GetRestaurantUseCase        *usecase.GetRestaurantUseCase
	ListRestaurantsUseCase      *usecase.ListRestaurantsUseCase
	UpdateRestaurantUseCase     *usecase.UpdateRestaurantUseCase
	DeactivateRestaurantUseCase *usecase.DeactivateRestaurantUseCase
}

func NewRestaurantHandler(
	createRestaurant *usecase.CreateRestaurantUseCase,
	getRestaurant *usecase.GetRestaurantUseCase,
	listRestaurants *usecase.ListRestaurantsUseCase,
	updateRestaurant *usecase.UpdateRestaurantUseCase,
	deactivateRestaurant *usecase.DeactivateRestaurantUseCase,
) *RestaurantHandler {
	return &RestaurantHandler{
		CreateRestaurantUseCase:     createRestaurant,
		GetRestaurantUseCase:        getRestaurant,
		ListRestaurantsUseCase:      listRestaurants,
		UpdateRestaurantUseCase:     updateRestaurant,
		DeactivateRestaurantUseCase: deactivateRestaurant,
	}
}

func (h *RestaurantHandler) CreateRestaurant(
	w http.ResponseWriter,
	r *http.Request,
) {
	tenantID, err := middleware.TenantFromContext(r.Context())
	if err != nil {
		http.Error(
			w,
			http.StatusText(http.StatusUnauthorized),
			http.StatusUnauthorized,
		)
		return
	}

	var request createRestaurantRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		http.Error(
			w,
			"invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	restaurant, err := h.CreateRestaurantUseCase.Execute(
		r.Context(),
		usecase.CreateRestaurantInput{
			TenantID:     tenantID,
			Name:         request.Name,
			Slug:         request.Slug,
			Description:  request.Description,
			Phone:        request.Phone,
			Email:        request.Email,
			AddressLine1: request.AddressLine1,
			AddressLine2: request.AddressLine2,
			City:         request.City,
			State:        request.State,
			PostalCode:   request.PostalCode,
			Country:      request.Country,
			Latitude:     request.Latitude,
			Longitude:    request.Longitude,
		},
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(
		w,
		http.StatusCreated,
		newRestaurantResponse(restaurant),
	)
}

func (h *RestaurantHandler) GetRestaurant(
	w http.ResponseWriter,
	r *http.Request,
) {
	tenantID, err := middleware.TenantFromContext(r.Context())
	if err != nil {
		http.Error(
			w,
			http.StatusText(http.StatusUnauthorized),
			http.StatusUnauthorized,
		)
		return
	}

	restaurantID, err := parseRestaurantID(r)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	restaurant, err := h.GetRestaurantUseCase.Execute(
		r.Context(),
		usecase.GetRestaurantInput{
			TenantID:     tenantID,
			RestaurantID: restaurantID,
		},
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		newRestaurantResponse(restaurant),
	)
}

func (h *RestaurantHandler) ListRestaurants(
	w http.ResponseWriter,
	r *http.Request,
) {
	tenantID, err := middleware.TenantFromContext(r.Context())
	if err != nil {
		http.Error(
			w,
			http.StatusText(http.StatusUnauthorized),
			http.StatusUnauthorized,
		)
		return
	}

	limit, err := parsePositiveQueryInt(
		r,
		"limit",
		usecase.DefaultRestaurantListLimit,
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	offset, err := parseNonNegativeQueryInt(
		r,
		"offset",
		0,
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	result, err := h.ListRestaurantsUseCase.Execute(
		r.Context(),
		usecase.ListRestaurantsInput{
			TenantID: tenantID,
			Limit:    limit,
			Offset:   offset,
		},
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	restaurants := make(
		[]restaurantResponse,
		0,
		len(result.Restaurants),
	)

	for _, restaurant := range result.Restaurants {
		restaurants = append(
			restaurants,
			newRestaurantResponse(restaurant),
		)
	}

	writeJSON(
		w,
		http.StatusOK,
		listRestaurantsResponse{
			Restaurants: restaurants,
			Limit:       result.Limit,
			Offset:      result.Offset,
		},
	)
}

func (h *RestaurantHandler) UpdateRestaurant(
	w http.ResponseWriter,
	r *http.Request,
) {
	tenantID, err := middleware.TenantFromContext(r.Context())
	if err != nil {
		http.Error(
			w,
			http.StatusText(http.StatusUnauthorized),
			http.StatusUnauthorized,
		)
		return
	}

	restaurantID, err := parseRestaurantID(r)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	var request updateRestaurantRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		http.Error(
			w,
			"invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	restaurant, err := h.UpdateRestaurantUseCase.Execute(
		r.Context(),
		usecase.UpdateRestaurantInput{
			TenantID:     tenantID,
			RestaurantID: restaurantID,
			Name:         request.Name,
			Slug:         request.Slug,
			Description:  request.Description,
			Phone:        request.Phone,
			Email:        request.Email,
			AddressLine1: request.AddressLine1,
			AddressLine2: request.AddressLine2,
			City:         request.City,
			State:        request.State,
			PostalCode:   request.PostalCode,
			Country:      request.Country,
			Latitude:     request.Latitude,
			Longitude:    request.Longitude,
		},
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		newRestaurantResponse(restaurant),
	)
}

func (h *RestaurantHandler) DeactivateRestaurant(
	w http.ResponseWriter,
	r *http.Request,
) {
	tenantID, err := middleware.TenantFromContext(r.Context())
	if err != nil {
		http.Error(
			w,
			http.StatusText(http.StatusUnauthorized),
			http.StatusUnauthorized,
		)
		return
	}

	restaurantID, err := parseRestaurantID(r)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	restaurant, err := h.DeactivateRestaurantUseCase.Execute(
		r.Context(),
		usecase.DeactivateRestaurantInput{
			TenantID:     tenantID,
			RestaurantID: restaurantID,
		},
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		newRestaurantResponse(restaurant),
	)
}

func parseRestaurantID(r *http.Request) (uuid.UUID, error) {
	value := strings.TrimSpace(
		r.PathValue("restaurantID"),
	)

	if value == "" {
		return uuid.Nil, domain.ErrInvalidRestaurantID
	}

	restaurantID, err := uuid.Parse(value)
	if err != nil || restaurantID == uuid.Nil {
		return uuid.Nil, domain.ErrInvalidRestaurantID
	}

	return restaurantID, nil
}

func parsePositiveQueryInt(
	r *http.Request,
	key string,
	defaultValue int,
) (int, error) {
	value := strings.TrimSpace(
		r.URL.Query().Get(key),
	)

	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 0, usecase.ErrInvalidPagination
	}

	return parsed, nil
}

func parseNonNegativeQueryInt(
	r *http.Request,
	key string,
	defaultValue int,
) (int, error) {
	value := strings.TrimSpace(
		r.URL.Query().Get(key),
	)

	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 0, usecase.ErrInvalidPagination
	}

	return parsed, nil
}

func writeDomainError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(err, domain.ErrInvalidTenantID):
		http.Error(
			w,
			err.Error(),
			http.StatusUnauthorized,
		)

	case errors.Is(err, domain.ErrInvalidRestaurantID),
		errors.Is(err, domain.ErrInvalidRestaurantName),
		errors.Is(err, domain.ErrInvalidRestaurantSlug),
		errors.Is(err, domain.ErrInvalidRestaurantStatus),
		errors.Is(err, domain.ErrInvalidLatitude),
		errors.Is(err, domain.ErrInvalidLongitude),
		errors.Is(err, usecase.ErrInvalidPagination):
		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)

	case errors.Is(err, domain.ErrRestaurantNotFound):
		http.Error(
			w,
			err.Error(),
			http.StatusNotFound,
		)

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

func writeJSON(
	w http.ResponseWriter,
	status int,
	value any,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(value)
}
