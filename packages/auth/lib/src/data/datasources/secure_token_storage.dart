import '../../domain/entities/auth_tokens.dart';

abstract interface class SecureTokenStorage {
  Future<void> saveTokens(AuthTokens tokens);

  Future<AuthTokens?> readTokens();

  Future<void> clearTokens();
}