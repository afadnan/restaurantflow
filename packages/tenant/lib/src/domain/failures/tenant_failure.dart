import 'package:freezed_annotation/freezed_annotation.dart';

part 'tenant_failure.freezed.dart';

@freezed
sealed class TenantFailure with _$TenantFailure {
  const factory TenantFailure.notAuthenticated() = TenantNotAuthenticated;

  const factory TenantFailure.noTenantSelected() = TenantNoTenantSelected;

  const factory TenantFailure.notFound({
    required String tenantId,
  }) = TenantNotFound;

  const factory TenantFailure.accessDenied({
    required String tenantId,
  }) = TenantAccessDenied;

  const factory TenantFailure.invalidTenant({
    required String message,
  }) = TenantInvalid;

  const factory TenantFailure.network({
    required String message,
  }) = TenantNetworkFailure;

  const factory TenantFailure.server({
    required String message,
    int? statusCode,
  }) = TenantServerFailure;

  const factory TenantFailure.unknown({
    required String message,
  }) = TenantUnknownFailure;
}