package httpdelivery

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/google/uuid"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/user/usecase"
)

const (
	maxRegistrationBodyBytes = 16 * 1024
	maxLoginBodyBytes        = 8 * 1024
	maxRefreshBodyBytes      = 8 * 1024
)

// OwnerRegistrationHandler handles public owner registration requests.
type OwnerRegistrationHandler struct {
	RegisterOwnerUseCase *usecase.RegisterOwnerUseCase
}

func NewOwnerRegistrationHandler(
	registerOwner *usecase.RegisterOwnerUseCase,
) *OwnerRegistrationHandler {
	return &OwnerRegistrationHandler{
		RegisterOwnerUseCase: registerOwner,
	}
}

func (h *OwnerRegistrationHandler) RegisterOwner(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h == nil || h.RegisterOwnerUseCase == nil {
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}

	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxRegistrationBodyBytes,
	)

	var request registerOwnerRequest

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

	var extra any

	if err := decoder.Decode(&extra); err != io.EOF {
		http.Error(
			w,
			"invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	result, err := h.RegisterOwnerUseCase.Execute(
		r.Context(),
		request.toUseCaseInput(),
	)
	if err != nil {
		writeRegistrationError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	_ = json.NewEncoder(w).Encode(
		newRegisterOwnerResponse(result),
	)
}

// OwnerLoginHandler handles public owner login requests.
type OwnerLoginHandler struct {
	LoginUseCase *usecase.LoginUseCase
}

func NewOwnerLoginHandler(
	loginUseCase *usecase.LoginUseCase,
) *OwnerLoginHandler {
	return &OwnerLoginHandler{
		LoginUseCase: loginUseCase,
	}
}

func (h *OwnerLoginHandler) Login(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h == nil || h.LoginUseCase == nil {
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}

	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxLoginBodyBytes,
	)

	var request loginRequest

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

	var extra any

	if err := decoder.Decode(&extra); err != io.EOF {
		http.Error(
			w,
			"invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	result, err := h.LoginUseCase.Execute(
		r.Context(),
		request.toUseCaseInput(),
	)
	if err != nil {
		writeLoginError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(
		newLoginResponse(result),
	)
}

// RefreshHandler handles public refresh-token rotation requests.
type RefreshHandler struct {
	RefreshUseCase *usecase.RefreshUseCase
}

func NewRefreshHandler(
	refreshUseCase *usecase.RefreshUseCase,
) *RefreshHandler {
	return &RefreshHandler{
		RefreshUseCase: refreshUseCase,
	}
}

func (h *RefreshHandler) Refresh(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h == nil || h.RefreshUseCase == nil {
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}

	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxRefreshBodyBytes,
	)

	var request refreshRequest

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

	var extra any

	if err := decoder.Decode(&extra); err != io.EOF {
		http.Error(
			w,
			"invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	tenantID, err := uuid.Parse(request.TenantID)
	if err != nil || tenantID == uuid.Nil {
		http.Error(
			w,
			"invalid refresh details",
			http.StatusBadRequest,
		)
		return
	}

	result, err := h.RefreshUseCase.Execute(
		r.Context(),
		request.toUseCaseInput(tenantID),
	)
	if err != nil {
		writeRefreshError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(
		newRefreshResponse(result),
	)
}
