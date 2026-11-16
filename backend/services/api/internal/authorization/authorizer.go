package authorization

import userdomain "github.com/afadnan/restaurantflow/services/api/internal/modules/user/domain"

// Authorizer evaluates whether a role has a specific permission.
type Authorizer struct {
	permissions map[userdomain.Role]map[Permission]struct{}
}

// NewAuthorizer creates an authorizer using the application's RBAC policy.
func NewAuthorizer() *Authorizer {
	return &Authorizer{
		permissions: rolePermissions,
	}
}

// HasPermission returns false for unknown roles, unknown permissions,
// empty permissions, or an uninitialized authorizer.
func (a *Authorizer) HasPermission(
	role userdomain.Role,
	permission Permission,
) bool {
	if a == nil || !role.IsValid() || permission == "" {
		return false
	}

	rolePermissions, ok := a.permissions[role]
	if !ok {
		return false
	}

	_, allowed := rolePermissions[permission]
	return allowed
}
