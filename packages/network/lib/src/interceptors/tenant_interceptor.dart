import 'package:dio/dio.dart';

import '../providers/tenant_provider.dart';

final class TenantInterceptor extends Interceptor {
  TenantInterceptor({
  required this._tenantProvider,
});

  final TenantProvider _tenantProvider;

  @override
  void onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) {
    final tenantId = _tenantProvider.currentTenantId;

    if (tenantId != null && tenantId.trim().isNotEmpty) {
      options.headers['X-Tenant-ID'] = tenantId;
    }

    handler.next(options);
  }
}