package authorization

import userdomain "github.com/afadnan/restaurantflow/services/api/internal/modules/user/domain"

// Authorizer evaluates whether a role has a specific permission.
type Authorizer struct {
	permissions map[userdomain.Role]map[Permission]struct{}
}

// NewAuthorizer creates an authorizer using an isolated copy of the
// application's RBAC policy.
func NewAuthorizer() *Authorizer {
	permissions := make(
		map[userdomain.Role]map[Permission]struct{},
		len(rolePermissions),
	)

	for role, rolePermissionSet := range rolePermissions {
		copiedSet := make(
			map[Permission]struct{},
			len(rolePermissionSet),
		)

		for permission := range rolePermissionSet {
			copiedSet[permission] = struct{}{}
		}

		permissions[role] = copiedSet
	}

	return &Authorizer{
		permissions: permissions,
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
