package httpdelivery

import (
	"net/http"

	"github.com/afadnan/restaurantflow/services/api/internal/middleware"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/usecase"
	"github.com/google/uuid"
)

type ItemHandler struct {
	CreateItemUseCase     *usecase.CreateItemUseCase
	GetItemUseCase        *usecase.GetItemUseCase
	ListItemsUseCase      *usecase.ListItemsUseCase
	UpdateItemUseCase     *usecase.UpdateItemUseCase
	DeactivateItemUseCase *usecase.DeactivateItemUseCase
}

func NewItemHandler(
	create *usecase.CreateItemUseCase,
	get *usecase.GetItemUseCase,
	list *usecase.ListItemsUseCase,
	update *usecase.UpdateItemUseCase,
	deactivate *usecase.DeactivateItemUseCase,
) *ItemHandler {
	return &ItemHandler{
		CreateItemUseCase:     create,
		GetItemUseCase:        get,
		ListItemsUseCase:      list,
		UpdateItemUseCase:     update,
		DeactivateItemUseCase: deactivate,
	}
}

func (h *ItemHandler) CreateItem(w http.ResponseWriter, r *http.Request) {
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

	var request createItemRequest
	if err := decodeJSON(r, &request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	categoryID, err := uuid.Parse(request.CategoryID)
	if err != nil || categoryID == uuid.Nil {
		writeDomainError(w, domain.ErrInvalidCategoryID)
		return
	}

	item, err := h.CreateItemUseCase.Execute(
		r.Context(),
		usecase.CreateItemInput{
			TenantID:     tenantID,
			RestaurantID: restaurantID,
			CategoryID:   categoryID,
			Name:         request.Name,
			Description:  request.Description,
			SKU:          request.SKU,
			Price:        request.Price,
			Status:       request.Status,
			ImageURL:     request.ImageURL,
			SortOrder:    request.SortOrder,
			IsVegetarian: request.IsVegetarian,
			IsVegan:      request.IsVegan,
			IsAvailable:  request.IsAvailable,
		},
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, newItemResponse(item))
}

func (h *ItemHandler) GetItem(w http.ResponseWriter, r *http.Request) {
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

	itemID, err := parseUUIDPath(r, "itemID")
	if err != nil {
		writeDomainError(w, domain.ErrInvalidItemID)
		return
	}

	item, err := h.GetItemUseCase.Execute(
		r.Context(),
		usecase.GetItemInput{
			TenantID:     tenantID,
			RestaurantID: restaurantID,
			ItemID:       itemID,
		},
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newItemResponse(item))
}

func (h *ItemHandler) ListItems(w http.ResponseWriter, r *http.Request) {
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

	items, err := h.ListItemsUseCase.Execute(
		r.Context(),
		usecase.ListItemsInput{
			TenantID:     tenantID,
			RestaurantID: restaurantID,
			Limit:        limit,
			Offset:       offset,
		},
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	response := make([]itemResponse, 0, len(items))
	for _, item := range items {
		response = append(response, newItemResponse(item))
	}

	writeJSON(w, http.StatusOK, listItemsResponse{
		Items:  response,
		Limit:  min(limit, 100),
		Offset: offset,
	})
}

func (h *ItemHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
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

	itemID, err := parseUUIDPath(r, "itemID")
	if err != nil {
		writeDomainError(w, domain.ErrInvalidItemID)
		return
	}

	var request updateItemRequest
	if err := decodeJSON(r, &request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	categoryID, err := uuid.Parse(request.CategoryID)
	if err != nil || categoryID == uuid.Nil {
		writeDomainError(w, domain.ErrInvalidCategoryID)
		return
	}

	item, err := h.UpdateItemUseCase.Execute(
		r.Context(),
		usecase.UpdateItemInput{
			TenantID:     tenantID,
			RestaurantID: restaurantID,
			ItemID:       itemID,
			CategoryID:   categoryID,
			Name:         request.Name,
			Description:  request.Description,
			SKU:          request.SKU,
			Price:        request.Price,
			Status:       request.Status,
			ImageURL:     request.ImageURL,
			SortOrder:    request.SortOrder,
			IsVegetarian: request.IsVegetarian,
			IsVegan:      request.IsVegan,
			IsAvailable:  request.IsAvailable,
		},
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newItemResponse(item))
}

func (h *ItemHandler) DeactivateItem(w http.ResponseWriter, r *http.Request) {
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

	itemID, err := parseUUIDPath(r, "itemID")
	if err != nil {
		writeDomainError(w, domain.ErrInvalidItemID)
		return
	}

	item, err := h.DeactivateItemUseCase.Execute(
		r.Context(),
		usecase.DeactivateItemInput{
			TenantID:     tenantID,
			RestaurantID: restaurantID,
			ItemID:       itemID,
		},
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newItemResponse(item))
}
