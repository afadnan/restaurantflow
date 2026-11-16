package authorization

// Permission represents one permitted business action.
type Permission string

const (
	// Restaurant permissions.
	PermissionRestaurantRead              Permission = "restaurant.read"
	PermissionRestaurantCreate            Permission = "restaurant.create"
	PermissionRestaurantUpdateProfile     Permission = "restaurant.update_profile"
	PermissionRestaurantUpdateHours       Permission = "restaurant.update_hours"
	PermissionRestaurantUpdateOperations  Permission = "restaurant.update_operations"
	PermissionRestaurantUpdateSecurity    Permission = "restaurant.update_security"
	PermissionRestaurantDeactivate        Permission = "restaurant.deactivate"
	PermissionRestaurantTransferOwnership Permission = "restaurant.transfer_ownership"
	PermissionRestaurantDelete            Permission = "restaurant.delete"

	// Staff permissions.
	PermissionStaffRead       Permission = "staff.read"
	PermissionStaffCreate     Permission = "staff.create"
	PermissionStaffUpdate     Permission = "staff.update"
	PermissionStaffDeactivate Permission = "staff.deactivate"
	PermissionStaffAssignRole Permission = "staff.assign_role"
	PermissionStaffDelete     Permission = "staff.delete"

	// Menu permissions.
	PermissionMenuRead         Permission = "menu.read"
	PermissionMenuCreate       Permission = "menu.create"
	PermissionMenuUpdate       Permission = "menu.update"
	PermissionMenuDelete       Permission = "menu.delete"
	PermissionMenuManageAddons Permission = "menu.manage_addons"

	// Order permissions.
	PermissionOrdersRead     Permission = "orders.read"
	PermissionOrdersCreate   Permission = "orders.create"
	PermissionOrdersUpdate   Permission = "orders.update"
	PermissionOrdersCancel   Permission = "orders.cancel"
	PermissionOrdersComplete Permission = "orders.complete"

	// Kitchen permissions.
	PermissionKitchenOrdersRead      Permission = "kitchen_orders.read"
	PermissionKitchenOrdersStart     Permission = "kitchen_orders.start"
	PermissionKitchenOrdersMarkReady Permission = "kitchen_orders.mark_ready"
	PermissionKitchenOrdersComplete  Permission = "kitchen_orders.complete"
	PermissionKitchenOrdersReopen    Permission = "kitchen_orders.reopen"

	// Payment permissions.
	PermissionPaymentsRead        Permission = "payments.read"
	PermissionPaymentsCollect     Permission = "payments.collect"
	PermissionPaymentsRefund      Permission = "payments.refund"
	PermissionPaymentsVoid        Permission = "payments.void"
	PermissionPaymentsViewReports Permission = "payments.view_reports"

	// Table permissions.
	PermissionTablesRead    Permission = "tables.read"
	PermissionTablesCreate  Permission = "tables.create"
	PermissionTablesUpdate  Permission = "tables.update"
	PermissionTablesDelete  Permission = "tables.delete"
	PermissionTablesAssign  Permission = "tables.assign"
	PermissionTablesRelease Permission = "tables.release"
)
