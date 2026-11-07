package httpdelivery

import (
	"time"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/user/usecase"
)

// registerOwnerRequest represents the public owner registration payload.
type registerOwnerRequest struct {
	BusinessName string `json:"business_name"`
	BusinessSlug string `json:"business_slug"`
	Email        string `json:"email"`
	Password     string `json:"password"`
	FirstName    string `json:"first_name"`
}

// registerOwnerResponse contains only safe, public registration details.
type registerOwnerResponse struct {
	Tenant tenantResponse `json:"tenant"`
	Owner  ownerResponse  `json:"owner"`
}

// loginRequest represents the public owner login payload.
type loginRequest struct {
	BusinessSlug string `json:"business_slug"`
	Email        string `json:"email"`
	Password     string `json:"password"`
}

// loginResponse contains the access token and safe account details.
type loginResponse struct {
	AccessToken string            `json:"access_token"`
	TokenType   string            `json:"token_type"`
	ExpiresAt   time.Time         `json:"expires_at"`
	Tenant      tenantResponse    `json:"tenant"`
	User        loginUserResponse `json:"user"`
}

// loginUserResponse contains safe user information.
type loginUserResponse struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	Role      string `json:"role"`
	Status    string `json:"status"`
}

type tenantResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Status string `json:"status"`
}

type ownerResponse struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	Role      string `json:"role"`
	Status    string `json:"status"`
}

// toUseCaseInput maps HTTP registration input to the application use case.
func (r registerOwnerRequest) toUseCaseInput() usecase.RegisterOwnerInput {
	return usecase.RegisterOwnerInput{
		BusinessName: r.BusinessName,
		BusinessSlug: r.BusinessSlug,
		Email:        r.Email,
		Password:     r.Password,
		FirstName:    r.FirstName,
	}
}

// toLoginUseCaseInput maps HTTP login input to the application use case.
func (r loginRequest) toUseCaseInput() usecase.LoginInput {
	return usecase.LoginInput{
		BusinessSlug: r.BusinessSlug,
		Email:        r.Email,
		Password:     r.Password,
	}
}

// newRegisterOwnerResponse maps the registration result into a safe DTO.
func newRegisterOwnerResponse(
	result *usecase.RegisterOwnerResult,
) registerOwnerResponse {
	return registerOwnerResponse{
		Tenant: tenantResponse{
			ID:     result.Tenant.ID.String(),
			Name:   result.Tenant.Name,
			Slug:   result.Tenant.Slug,
			Status: result.Tenant.Status,
		},
		Owner: ownerResponse{
			ID:        result.Owner.ID.String(),
			TenantID:  result.Owner.TenantID.String(),
			Email:     result.Owner.Email,
			FirstName: result.Owner.FirstName,
			Role:      string(result.Owner.Role),
			Status:    string(result.Owner.Status),
		},
	}
}

// newLoginResponse maps the login result into a safe HTTP response.
func newLoginResponse(
	result *usecase.LoginResult,
) loginResponse {
	return loginResponse{
		AccessToken: result.AccessToken,
		TokenType:   result.TokenType,
		ExpiresAt:   result.ExpiresAt,
		Tenant: tenantResponse{
			ID:     result.Tenant.ID.String(),
			Name:   result.Tenant.Name,
			Slug:   result.Tenant.Slug,
			Status: result.Tenant.Status,
		},
		User: loginUserResponse{
			ID:        result.User.ID.String(),
			TenantID:  result.User.TenantID.String(),
			Email:     result.User.Email,
			FirstName: result.User.FirstName,
			Role:      string(result.User.Role),
			Status:    string(result.User.Status),
		},
	}
}
