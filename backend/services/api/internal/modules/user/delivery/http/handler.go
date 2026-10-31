package httpdelivery

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/user/usecase"
)

const maxRegistrationBodyBytes = 16 * 1024

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

	// Limit the size of the public registration request.
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

	// Reject multiple JSON values in the same request body.
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
