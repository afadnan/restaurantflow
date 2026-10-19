package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/afadnan/restaurantflow/services/api/internal/middleware"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/usecase"
)

type CategoryHandler struct {
	CreateCategoryUseCase     *usecase.CreateCategoryUseCase
	GetCategoryUseCase        *usecase.GetCategoryUseCase
	ListCategoriesUseCase     *usecase.ListCategoriesUseCase
	UpdateCategoryUseCase     *usecase.UpdateCategoryUseCase
	DeactivateCategoryUseCase *usecase.DeactivateCategoryUseCase
}

func NewCategoryHandler(
	create *usecase.CreateCategoryUseCase,
	get *usecase.GetCategoryUseCase,
	list *usecase.ListCategoriesUseCase,
	update *usecase.UpdateCategoryUseCase,
	deactivate *usecase.DeactivateCategoryUseCase,
) *CategoryHandler {
	return &CategoryHandler{
		CreateCategoryUseCase:     create,
		GetCategoryUseCase:        get,
		ListCategoriesUseCase:     list,
		UpdateCategoryUseCase:     update,
		DeactivateCategoryUseCase: deactivate,
	}
}

func (h *CategoryHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.TenantFromContext(r.Context())
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	restaurantID, err := parseUUIDPath(r, "restaurantID")
	if err != nil {
		writeDomainError(w, domain.ErrInvalidRestaurantID)
		return
	}

	var request createCategoryRequest
	if err := decodeJSON(r, &request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	category, err := h.CreateCategoryUseCase.Execute(r.Context(), usecase.CreateCategoryInput{
		TenantID:     tenantID,
		RestaurantID: restaurantID,
		Name:         request.Name,
		Description:  request.Description,
		SortOrder:    request.SortOrder,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, newCategoryResponse(category))
}

func (h *CategoryHandler) GetCategory(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.TenantFromContext(r.Context())
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	restaurantID, err := parseUUIDPath(r, "restaurantID")
	if err != nil {
		writeDomainError(w, domain.ErrInvalidRestaurantID)
		return
	}

	categoryID, err := parseUUIDPath(r, "categoryID")
	if err != nil {
		writeDomainError(w, domain.ErrInvalidCategoryID)
		return
	}

	category, err := h.GetCategoryUseCase.Execute(r.Context(), usecase.GetCategoryInput{
		TenantID: tenantID, RestaurantID: restaurantID, CategoryID: categoryID,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newCategoryResponse(category))
}

func (h *CategoryHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.TenantFromContext(r.Context())
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	restaurantID, err := parseUUIDPath(r, "restaurantID")
	if err != nil {
		writeDomainError(w, domain.ErrInvalidRestaurantID)
		return
	}

	limit, err := parseQueryInt(r, "limit", 50, 1)
	if err != nil {
		writeDomainError(w, usecase.ErrInvalidPagination)
		return
	}

	offset, err := parseQueryInt(r, "offset", 0, 0)
	if err != nil {
		writeDomainError(w, usecase.ErrInvalidPagination)
		return
	}

	categories, err := h.ListCategoriesUseCase.Execute(r.Context(), usecase.ListCategoriesInput{
		TenantID: tenantID, RestaurantID: restaurantID, Limit: limit, Offset: offset,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	response := make([]categoryResponse, 0, len(categories))
	for _, category := range categories {
		response = append(response, newCategoryResponse(category))
	}

	writeJSON(w, http.StatusOK, listCategoriesResponse{
		Categories: response,
		Limit:      min(limit, 100),
		Offset:     offset,
	})
}

func (h *CategoryHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.TenantFromContext(r.Context())
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	restaurantID, err := parseUUIDPath(r, "restaurantID")
	if err != nil {
		writeDomainError(w, domain.ErrInvalidRestaurantID)
		return
	}

	categoryID, err := parseUUIDPath(r, "categoryID")
	if err != nil {
		writeDomainError(w, domain.ErrInvalidCategoryID)
		return
	}

	var request updateCategoryRequest
	if err := decodeJSON(r, &request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	category, err := h.UpdateCategoryUseCase.Execute(r.Context(), usecase.UpdateCategoryInput{
		TenantID: tenantID, RestaurantID: restaurantID, CategoryID: categoryID,
		Name: request.Name, Description: request.Description, SortOrder: request.SortOrder,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newCategoryResponse(category))
}

func (h *CategoryHandler) DeactivateCategory(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.TenantFromContext(r.Context())
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	restaurantID, err := parseUUIDPath(r, "restaurantID")
	if err != nil {
		writeDomainError(w, domain.ErrInvalidRestaurantID)
		return
	}

	categoryID, err := parseUUIDPath(r, "categoryID")
	if err != nil {
		writeDomainError(w, domain.ErrInvalidCategoryID)
		return
	}

	category, err := h.DeactivateCategoryUseCase.Execute(r.Context(), usecase.DeactivateCategoryInput{
		TenantID: tenantID, RestaurantID: restaurantID, CategoryID: categoryID,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newCategoryResponse(category))
}

func parseUUIDPath(r *http.Request, key string) (uuid.UUID, error) {
	value := strings.TrimSpace(r.PathValue(key))
	if value == "" {
		return uuid.Nil, errors.New("missing UUID path value")
	}

	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, errors.New("invalid UUID path value")
	}

	return id, nil
}

func parseQueryInt(r *http.Request, key string, defaultValue, minimum int) (int, error) {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < minimum {
		return 0, usecase.ErrInvalidPagination
	}

	return parsed, nil
}

func decodeJSON(r *http.Request, destination any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return err
	}

	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return errors.New("request body must contain a single JSON value")
	}

	return nil
}
