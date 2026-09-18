import '../../domain/entities/authenticated_user.dart';

final class AuthenticatedUserModel {
  const AuthenticatedUserModel({
    required this.id,
    required this.email,
    required this.displayName,
    required this.roles,
    required this.permissions,
    this.phoneNumber,
  });

  final String id;
  final String email;
  final String displayName;
  final String? phoneNumber;
  final List<String> roles;
  final List<String> permissions;

  factory AuthenticatedUserModel.fromJson(
    Map<String, dynamic> json,
  ) {
    return AuthenticatedUserModel(
      id: json['id'] as String,
      email: json['email'] as String,
      displayName: json['display_name'] as String,
      phoneNumber: json['phone_number'] as String?,
      roles: _stringList(json['roles']),
      permissions: _stringList(json['permissions']),
    );
  }

  AuthenticatedUser toEntity() {
    return AuthenticatedUser(
      id: id,
      email: email,
      displayName: displayName,
      phoneNumber: phoneNumber,
      roles: List.unmodifiable(roles),
      permissions: List.unmodifiable(permissions),
    );
  }

  static List<String> _stringList(Object? value) {
    if (value is! List) {
      return const [];
    }

    return value
        .whereType<String>()
        .toList(growable: false);
  }
}