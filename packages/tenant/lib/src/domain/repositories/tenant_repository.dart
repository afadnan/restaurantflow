import 'package:tenant/src/domain/entities/tenant.dart';

abstract interface class TenantRepository {
  Future<Tenant> getCurrentTenant();

  Future<Tenant> getTenant(String tenantId);

  Future<List<Tenant>> getAvailableTenants();

  Future<Tenant> switchTenant(String tenantId);

  Future<void> clearTenant();
}