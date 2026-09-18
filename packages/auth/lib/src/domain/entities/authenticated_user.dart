import 'package:equatable/equatable.dart';

final class AuthenticatedUser extends Equatable {
  const AuthenticatedUser({
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

  bool hasRole(String role) {
    return roles.contains(role);
  }

  bool hasPermission(String permission) {
    return permissions.contains(permission);
  }

  @override
  List<Object?> get props => [
        id,
        email,
        displayName,
        phoneNumber,
        roles,
        permissions,
      ];
}