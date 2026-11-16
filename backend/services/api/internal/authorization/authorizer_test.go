package authorization

import (
	"testing"

	userdomain "github.com/afadnan/restaurantflow/services/api/internal/modules/user/domain"
)

func TestAuthorizerRolePermissions(t *testing.T) {
	authorizer := NewAuthorizer()

	tests := []struct {
		name       string
		role       userdomain.Role
		permission Permission
		want       bool
	}{
		{"owner can transfer ownership", userdomain.RoleOwner, PermissionRestaurantTransferOwnership, true},
		{"admin cannot transfer ownership", userdomain.RoleAdmin, PermissionRestaurantTransferOwnership, false},
		{"manager can update operations", userdomain.RoleManager, PermissionRestaurantUpdateOperations, true},
		{"manager cannot update security", userdomain.RoleManager, PermissionRestaurantUpdateSecurity, false},
		{"staff cannot update menu", userdomain.RoleStaff, PermissionMenuUpdate, false},
		{"staff can read menu", userdomain.RoleStaff, PermissionMenuRead, true},
		{"kitchen can start kitchen orders", userdomain.RoleKitchen, PermissionKitchenOrdersStart, true},
		{"kitchen cannot collect payments", userdomain.RoleKitchen, PermissionPaymentsCollect, false},
		{"cashier can collect payments", userdomain.RoleCashier, PermissionPaymentsCollect, true},
		{"cashier cannot refund payments", userdomain.RoleCashier, PermissionPaymentsRefund, false},
		{"manager can refund payments", userdomain.RoleManager, PermissionPaymentsRefund, true},
		{"manager cannot create staff", userdomain.RoleManager, PermissionStaffCreate, false},
		{"admin can create staff", userdomain.RoleAdmin, PermissionStaffCreate, true},
		{"staff can assign tables", userdomain.RoleStaff, PermissionTablesAssign, true},
		{"staff cannot delete tables", userdomain.RoleStaff, PermissionTablesDelete, false},
		{"unknown role is denied", userdomain.Role("unknown"), PermissionMenuRead, false},
		{"unknown permission is denied", userdomain.RoleOwner, Permission("unknown.permission"), false},
		{"empty permission is denied", userdomain.RoleOwner, "", false},
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

func TestAuthorizerNilReceiverDeniesPermission(t *testing.T) {
	var authorizer *Authorizer

	if authorizer.HasPermission(userdomain.RoleOwner, PermissionMenuRead) {
		t.Fatal("nil authorizer must deny permissions")
	}
}
