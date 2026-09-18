import '../entities/auth_tokens.dart';
import '../entities/authenticated_user.dart';

abstract interface class AuthRepository {
  Future<AuthenticatedUser> login({
    required String email,
    required String password,
  });

  Future<AuthenticatedUser> register({
    required String email,
    required String password,
    required String displayName,
    String? phoneNumber,
  });

  Future<void> logout();

  Future<AuthenticatedUser?> restoreSession();

  Future<AuthTokens?> refreshTokens();

  Future<void> forgotPassword({
    required String email,
  });

  Future<void> resetPassword({
    required String token,
    required String newPassword,
  });

  Future<AuthenticatedUser?> getCurrentUser();
}