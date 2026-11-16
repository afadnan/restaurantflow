package authorization

import (
	"testing"

	userdomain "github.com/afadnan/restaurantflow/services/api/internal/modules/user/domain"
)

func TestAuthorizerRestaurantLifecyclePermissions(t *testing.T) {
	authorizer := NewAuthorizer()

	tests := []struct {
		name       string
		role       userdomain.Role
		permission Permission
		want       bool
	}{
		{
			name:       "owner can create restaurant",
			role:       userdomain.RoleOwner,
			permission: PermissionRestaurantCreate,
			want:       true,
		},
		{
			name:       "owner can deactivate restaurant",
			role:       userdomain.RoleOwner,
			permission: PermissionRestaurantDeactivate,
			want:       true,
		},
		{
			name:       "admin cannot create restaurant",
			role:       userdomain.RoleAdmin,
			permission: PermissionRestaurantCreate,
			want:       false,
		},
		{
			name:       "admin cannot deactivate restaurant",
			role:       userdomain.RoleAdmin,
			permission: PermissionRestaurantDeactivate,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := authorizer.HasPermission(tt.role, tt.permission)
			if got != tt.want {
				t.Fatalf(
					"HasPermission(%q, %q) = %v, want %v",
					tt.role,
					tt.permission,
					got,
					tt.want,
				)
			}
		})
	}
}

func TestAuthorizerPolicyIsIsolatedFromGlobalPolicyChanges(t *testing.T) {
	originalPermissions := rolePermissions
	defer func() {
		rolePermissions = originalPermissions
	}()

	authorizer := NewAuthorizer()

	if !authorizer.HasPermission(
		userdomain.RoleAdmin,
		PermissionMenuRead,
	) {
		t.Fatal("admin should initially have menu read permission")
	}

	rolePermissions[userdomain.RoleAdmin][PermissionMenuRead] = struct{}{}
	delete(
		rolePermissions[userdomain.RoleAdmin],
		PermissionMenuRead,
	)

	if !authorizer.HasPermission(
		userdomain.RoleAdmin,
		PermissionMenuRead,
	) {
		t.Fatal("existing authorizer must not be affected by global policy changes")
	}
}
