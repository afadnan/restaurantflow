import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:tenant/src/domain/entities/tenant.dart';

part 'tenant_model.freezed.dart';
part 'tenant_model.g.dart';

@freezed
abstract class TenantModel with _$TenantModel {
  const factory TenantModel({
    required String id,
    required String name,
    required String slug,
    required TenantStatus status,
    required String timezone,
    required String currency,
  }) = _TenantModel;

  factory TenantModel.fromJson(Map<String, dynamic> json) =>
      _$TenantModelFromJson(json);
}

extension TenantModelMapper on TenantModel {
  Tenant toDomain() {
    return Tenant(
      id: id,
      name: name,
      slug: slug,
      status: status,
      timezone: timezone,
      currency: currency,
    );
  }
}