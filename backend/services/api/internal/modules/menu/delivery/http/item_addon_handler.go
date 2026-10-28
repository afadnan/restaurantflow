package httpdelivery

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/afadnan/restaurantflow/services/api/internal/middleware"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/usecase"
)

type ItemAddonHandler struct {
	AttachItemAddonUseCase *usecase.AttachItemAddonUseCase
	GetItemAddonUseCase    *usecase.GetItemAddonUseCase
	ListItemAddonsUseCase  *usecase.ListItemAddonsUseCase
	UpdateItemAddonUseCase *usecase.UpdateItemAddonUseCase
	DetachItemAddonUseCase *usecase.DetachItemAddonUseCase
}

func NewItemAddonHandler(
	attach *usecase.AttachItemAddonUseCase,
	get *usecase.GetItemAddonUseCase,
	list *usecase.ListItemAddonsUseCase,
	update *usecase.UpdateItemAddonUseCase,
	detach *usecase.DetachItemAddonUseCase,
) *ItemAddonHandler {
	return &ItemAddonHandler{
		AttachItemAddonUseCase: attach,
		GetItemAddonUseCase:    get,
		ListItemAddonsUseCase:  list,
		UpdateItemAddonUseCase: update,
		DetachItemAddonUseCase: detach,
	}
}

// itemAddonPathIDs extracts the restaurant and item identifiers.
// Both identifiers are validated before invoking any use case.
func itemAddonPathIDs(r *http.Request) (uuid.UUID, uuid.UUID, error) {
	restaurantID, err := parseUUIDPath(r, "restaurantID")
	if err != nil {
		return uuid.Nil, uuid.Nil, domain.ErrInvalidRestaurantID
	}

	itemID, err := parseUUIDPath(r, "itemID")
	if err != nil {
		return uuid.Nil, uuid.Nil, domain.ErrInvalidItemID
	}

	return restaurantID, itemID, nil
}

func (h *ItemAddonHandler) AttachItemAddon(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.TenantFromContext(r.Context())
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	restaurantID, itemID, err := itemAddonPathIDs(r)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	var request attachItemAddonRequest
	if err := decodeJSON(r, &request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	addonID, err := uuid.Parse(request.AddonID)
	if err != nil || addonID == uuid.Nil {
		writeDomainError(w, domain.ErrInvalidAddonID)
		return
	}

	if h == nil || h.AttachItemAddonUseCase == nil {
		writeDomainError(w, usecase.ErrItemAddonUseCaseNotConfigured)
		return
	}

	association, err := h.AttachItemAddonUseCase.Execute(
		r.Context(),
		usecase.AttachItemAddonInput{
			TenantID:     tenantID,
			RestaurantID: restaurantID,
			ItemID:       itemID,
			AddonID:      addonID,
			IsRequired:   request.IsRequired,
			MaxQuantity:  request.MaxQuantity,
		},
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, newItemAddonResponse(association))
}

func (h *ItemAddonHandler) GetItemAddon(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.TenantFromContext(r.Context())
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	restaurantID, itemID, err := itemAddonPathIDs(r)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	addonID, err := parseUUIDPath(r, "addonID")
	if err != nil {
		writeDomainError(w, domain.ErrInvalidAddonID)
		return
	}

	if h == nil || h.GetItemAddonUseCase == nil {
		writeDomainError(w, usecase.ErrItemAddonUseCaseNotConfigured)
		return
	}

	association, err := h.GetItemAddonUseCase.Execute(
		r.Context(),
		usecase.GetItemAddonInput{
			TenantID:     tenantID,
			RestaurantID: restaurantID,
			ItemID:       itemID,
			AddonID:      addonID,
		},
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newItemAddonResponse(association))
}

func (h *ItemAddonHandler) ListItemAddons(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.TenantFromContext(r.Context())
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	restaurantID, itemID, err := itemAddonPathIDs(r)
	if err != nil {
		writeDomainError(w, err)
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

	if h == nil || h.ListItemAddonsUseCase == nil {
		writeDomainError(w, usecase.ErrItemAddonUseCaseNotConfigured)
		return
	}

	associations, err := h.ListItemAddonsUseCase.Execute(
		r.Context(),
		usecase.ListItemAddonsInput{
			TenantID:     tenantID,
			RestaurantID: restaurantID,
			ItemID:       itemID,
			Limit:        limit,
			Offset:       offset,
		},
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	response := make([]itemAddonResponse, 0, len(associations))
	for _, association := range associations {
		response = append(response, newItemAddonResponse(association))
	}

	writeJSON(w, http.StatusOK, listItemAddonsResponse{
		Addons: response,
		Limit:  min(limit, 100),
		Offset: offset,
	})
}

func (h *ItemAddonHandler) UpdateItemAddon(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.TenantFromContext(r.Context())
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	restaurantID, itemID, err := itemAddonPathIDs(r)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	addonID, err := parseUUIDPath(r, "addonID")
	if err != nil {
		writeDomainError(w, domain.ErrInvalidAddonID)
		return
	}

	var request updateItemAddonRequest
	if err := decodeJSON(r, &request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if h == nil || h.UpdateItemAddonUseCase == nil {
		writeDomainError(w, usecase.ErrItemAddonUseCaseNotConfigured)
		return
	}

	association, err := h.UpdateItemAddonUseCase.Execute(
		r.Context(),
		usecase.UpdateItemAddonInput{
			TenantID:     tenantID,
			RestaurantID: restaurantID,
			ItemID:       itemID,
			AddonID:      addonID,
			IsRequired:   request.IsRequired,
			MaxQuantity:  request.MaxQuantity,
		},
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newItemAddonResponse(association))
}

func (h *ItemAddonHandler) DetachItemAddon(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.TenantFromContext(r.Context())
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	restaurantID, itemID, err := itemAddonPathIDs(r)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	addonID, err := parseUUIDPath(r, "addonID")
	if err != nil {
		writeDomainError(w, domain.ErrInvalidAddonID)
		return
	}

	if h == nil || h.DetachItemAddonUseCase == nil {
		writeDomainError(w, usecase.ErrItemAddonUseCaseNotConfigured)
		return
	}

	association, err := h.DetachItemAddonUseCase.Execute(
		r.Context(),
		usecase.DetachItemAddonInput{
			TenantID:     tenantID,
			RestaurantID: restaurantID,
			ItemID:       itemID,
			AddonID:      addonID,
		},
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newItemAddonResponse(association))
}
