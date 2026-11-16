package authorization

import userdomain "github.com/afadnan/restaurantflow/services/api/internal/modules/user/domain"

func permissionSet(permissions ...Permission) map[Permission]struct{} {
	result := make(map[Permission]struct{}, len(permissions))

	for _, permission := range permissions {
		result[permission] = struct{}{}
	}

	return result
}

func allPermissions() map[Permission]struct{} {
	return permissionSet(
		PermissionRestaurantRead,
		PermissionRestaurantCreate,
		PermissionRestaurantUpdateProfile,
		PermissionRestaurantUpdateHours,
		PermissionRestaurantUpdateOperations,
		PermissionRestaurantUpdateSecurity,
		PermissionRestaurantDeactivate,
		PermissionRestaurantTransferOwnership,
		PermissionRestaurantDelete,

		PermissionStaffRead,
		PermissionStaffCreate,
		PermissionStaffUpdate,
		PermissionStaffDeactivate,
		PermissionStaffAssignRole,
		PermissionStaffDelete,

		PermissionMenuRead,
		PermissionMenuCreate,
		PermissionMenuUpdate,
		PermissionMenuDelete,
		PermissionMenuManageAddons,

		PermissionOrdersRead,
		PermissionOrdersCreate,
		PermissionOrdersUpdate,
		PermissionOrdersCancel,
		PermissionOrdersComplete,

		PermissionKitchenOrdersRead,
		PermissionKitchenOrdersStart,
		PermissionKitchenOrdersMarkReady,
		PermissionKitchenOrdersComplete,
		PermissionKitchenOrdersReopen,

		PermissionPaymentsRead,
		PermissionPaymentsCollect,
		PermissionPaymentsRefund,
		PermissionPaymentsVoid,
		PermissionPaymentsViewReports,

		PermissionTablesRead,
		PermissionTablesCreate,
		PermissionTablesUpdate,
		PermissionTablesDelete,
		PermissionTablesAssign,
		PermissionTablesRelease,
	)
}

// rolePermissions is the central, code-defined RBAC policy.
//
// Resource ownership, employee hierarchy, order state, and financial
// limits must still be enforced by the relevant use cases.
var rolePermissions = map[userdomain.Role]map[Permission]struct{}{
	userdomain.RoleOwner: allPermissions(),

	userdomain.RoleAdmin: permissionSet(
		PermissionRestaurantRead,
		PermissionRestaurantUpdateProfile,
		PermissionRestaurantUpdateHours,
		PermissionRestaurantUpdateOperations,

		PermissionStaffRead,
		PermissionStaffCreate,
		PermissionStaffUpdate,
		PermissionStaffDeactivate,
		PermissionStaffAssignRole,
		PermissionStaffDelete,

		PermissionMenuRead,
		PermissionMenuCreate,
		PermissionMenuUpdate,
		PermissionMenuDelete,
		PermissionMenuManageAddons,

		PermissionOrdersRead,
		PermissionOrdersCreate,
		PermissionOrdersUpdate,
		PermissionOrdersCancel,
		PermissionOrdersComplete,

		PermissionKitchenOrdersRead,
		PermissionKitchenOrdersStart,
		PermissionKitchenOrdersMarkReady,
		PermissionKitchenOrdersComplete,
		PermissionKitchenOrdersReopen,

		PermissionPaymentsRead,
		PermissionPaymentsCollect,
		PermissionPaymentsRefund,
		PermissionPaymentsVoid,
		PermissionPaymentsViewReports,

		PermissionTablesRead,
		PermissionTablesCreate,
		PermissionTablesUpdate,
		PermissionTablesDelete,
		PermissionTablesAssign,
		PermissionTablesRelease,
	),

	userdomain.RoleManager: permissionSet(
		PermissionRestaurantRead,
		PermissionRestaurantUpdateProfile,
		PermissionRestaurantUpdateHours,
		PermissionRestaurantUpdateOperations,

		PermissionStaffRead,
		PermissionStaffUpdate,
		PermissionStaffDeactivate,

		PermissionMenuRead,
		PermissionMenuCreate,
		PermissionMenuUpdate,
		PermissionMenuDelete,
		PermissionMenuManageAddons,

		PermissionOrdersRead,
		PermissionOrdersCreate,
		PermissionOrdersUpdate,
		PermissionOrdersCancel,
		PermissionOrdersComplete,

		PermissionKitchenOrdersRead,
		PermissionKitchenOrdersStart,
		PermissionKitchenOrdersMarkReady,
		PermissionKitchenOrdersComplete,
		PermissionKitchenOrdersReopen,

		PermissionPaymentsRead,
		PermissionPaymentsCollect,
		PermissionPaymentsRefund,

		PermissionTablesRead,
		PermissionTablesCreate,
		PermissionTablesUpdate,
		PermissionTablesDelete,
		PermissionTablesAssign,
		PermissionTablesRelease,
	),

	userdomain.RoleStaff: permissionSet(
		PermissionMenuRead,

		PermissionOrdersRead,
		PermissionOrdersCreate,
		PermissionOrdersUpdate,
		PermissionOrdersCancel,

		PermissionKitchenOrdersRead,

		PermissionTablesRead,
		PermissionTablesAssign,
		PermissionTablesRelease,
	),

	userdomain.RoleKitchen: permissionSet(
		PermissionMenuRead,

		PermissionKitchenOrdersRead,
		PermissionKitchenOrdersStart,
		PermissionKitchenOrdersMarkReady,
		PermissionKitchenOrdersComplete,

		PermissionTablesRead,
	),

	userdomain.RoleCashier: permissionSet(
		PermissionMenuRead,

		PermissionOrdersRead,
		PermissionKitchenOrdersRead,

		PermissionPaymentsRead,
		PermissionPaymentsCollect,

		PermissionTablesRead,
		PermissionTablesAssign,
		PermissionTablesRelease,
	),
}
