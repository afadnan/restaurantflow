import 'package:freezed_annotation/freezed_annotation.dart';

part 'tenant.freezed.dart';

@freezed
abstract class Tenant with _$Tenant {
  const factory Tenant({
    required String id,
    required String name,
    required String slug,
    required TenantStatus status,
    required String timezone,
    required String currency,
  }) = _Tenant;
}

enum TenantStatus {
  active,
  suspended,
  pending,
  archived,
}