package httpdelivery

import (
	"net/http"

	"github.com/afadnan/restaurantflow/services/api/internal/middleware"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/usecase"
)

type AddonHandler struct {
	CreateAddonUseCase     *usecase.CreateAddonUseCase
	GetAddonUseCase        *usecase.GetAddonUseCase
	ListAddonsUseCase      *usecase.ListAddonsUseCase
	UpdateAddonUseCase     *usecase.UpdateAddonUseCase
	DeactivateAddonUseCase *usecase.DeactivateAddonUseCase
}

func NewAddonHandler(
	create *usecase.CreateAddonUseCase,
	get *usecase.GetAddonUseCase,
	list *usecase.ListAddonsUseCase,
	update *usecase.UpdateAddonUseCase,
	deactivate *usecase.DeactivateAddonUseCase,
) *AddonHandler {
	return &AddonHandler{
		CreateAddonUseCase:     create,
		GetAddonUseCase:        get,
		ListAddonsUseCase:      list,
		UpdateAddonUseCase:     update,
		DeactivateAddonUseCase: deactivate,
	}
}

func (h *AddonHandler) CreateAddon(w http.ResponseWriter, r *http.Request) {
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

	var request createAddonRequest
	if err := decodeJSON(r, &request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if h == nil || h.CreateAddonUseCase == nil {
		writeDomainError(w, usecase.ErrAddonUseCaseNotConfigured)
		return
	}

	addon, err := h.CreateAddonUseCase.Execute(
		r.Context(),
		usecase.CreateAddonInput{
			TenantID:     tenantID,
			RestaurantID: restaurantID,
			Name:         request.Name,
			Description:  request.Description,
			Price:        request.Price,
		},
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, newAddonResponse(addon))
}

func (h *AddonHandler) GetAddon(w http.ResponseWriter, r *http.Request) {
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

	addonID, err := parseUUIDPath(r, "addonID")
	if err != nil {
		writeDomainError(w, domain.ErrInvalidAddonID)
		return
	}

	if h == nil || h.GetAddonUseCase == nil {
		writeDomainError(w, usecase.ErrAddonUseCaseNotConfigured)
		return
	}

	addon, err := h.GetAddonUseCase.Execute(
		r.Context(),
		usecase.GetAddonInput{
			TenantID:     tenantID,
			RestaurantID: restaurantID,
			AddonID:      addonID,
		},
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newAddonResponse(addon))
}

func (h *AddonHandler) ListAddons(w http.ResponseWriter, r *http.Request) {
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

	if h == nil || h.ListAddonsUseCase == nil {
		writeDomainError(w, usecase.ErrAddonUseCaseNotConfigured)
		return
	}

	addons, err := h.ListAddonsUseCase.Execute(
		r.Context(),
		usecase.ListAddonsInput{
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

	response := make([]addonResponse, 0, len(addons))
	for _, addon := range addons {
		response = append(response, newAddonResponse(addon))
	}

	writeJSON(w, http.StatusOK, listAddonsResponse{
		Addons: response,
		Limit:  min(limit, 100),
		Offset: offset,
	})
}

func (h *AddonHandler) UpdateAddon(w http.ResponseWriter, r *http.Request) {
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

	addonID, err := parseUUIDPath(r, "addonID")
	if err != nil {
		writeDomainError(w, domain.ErrInvalidAddonID)
		return
	}

	var request updateAddonRequest
	if err := decodeJSON(r, &request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if h == nil || h.UpdateAddonUseCase == nil {
		writeDomainError(w, usecase.ErrAddonUseCaseNotConfigured)
		return
	}

	addon, err := h.UpdateAddonUseCase.Execute(
		r.Context(),
		usecase.UpdateAddonInput{
			TenantID:     tenantID,
			RestaurantID: restaurantID,
			AddonID:      addonID,
			Name:         request.Name,
			Description:  request.Description,
			Price:        request.Price,
		},
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newAddonResponse(addon))
}

func (h *AddonHandler) DeactivateAddon(w http.ResponseWriter, r *http.Request) {
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

	addonID, err := parseUUIDPath(r, "addonID")
	if err != nil {
		writeDomainError(w, domain.ErrInvalidAddonID)
		return
	}

	if h == nil || h.DeactivateAddonUseCase == nil {
		writeDomainError(w, usecase.ErrAddonUseCaseNotConfigured)
		return
	}

	addon, err := h.DeactivateAddonUseCase.Execute(
		r.Context(),
		usecase.DeactivateAddonInput{
			TenantID:     tenantID,
			RestaurantID: restaurantID,
			AddonID:      addonID,
		},
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newAddonResponse(addon))
}
