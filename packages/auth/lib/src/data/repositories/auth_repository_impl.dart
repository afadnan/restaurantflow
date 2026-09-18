import '../../domain/entities/auth_tokens.dart';
import '../../domain/entities/authenticated_user.dart';
import '../../domain/repositories/auth_repository.dart';
import '../datasources/auth_remote_data_source.dart';
import '../datasources/secure_token_storage.dart';

final class AuthRepositoryImpl implements AuthRepository {
  final AuthRemoteDataSource _remoteDataSource;
  final SecureTokenStorage _tokenStorage;

  AuthRepositoryImpl(
    this._remoteDataSource,
    this._tokenStorage,
  );


  AuthenticatedUser? _currentUser;

  @override
  Future<AuthenticatedUser> login({
    required String email,
    required String password,
  }) async {
    final response = await _remoteDataSource.login(
      email: email,
      password: password,
    );

    final tokens = response.tokens.toEntity();
    final user = response.user.toEntity();

    await _tokenStorage.saveTokens(tokens);

    _currentUser = user;

    return user;
  }

  @override
  Future<AuthenticatedUser> register({
    required String email,
    required String password,
    required String displayName,
    String? phoneNumber,
  }) async {
    final response = await _remoteDataSource.register(
      email: email,
      password: password,
      displayName: displayName,
      phoneNumber: phoneNumber,
    );

    final tokens = response.tokens.toEntity();
    final user = response.user.toEntity();

    await _tokenStorage.saveTokens(tokens);

    _currentUser = user;

    return user;
  }

  @override
  Future<AuthTokens?> refreshTokens() async {
    final existingTokens = await _tokenStorage.readTokens();

    if (existingTokens == null ||
        existingTokens.refreshToken.isEmpty) {
      return null;
    }

    final response = await _remoteDataSource.refresh(
      refreshToken: existingTokens.refreshToken,
    );

    final tokens = response.tokens.toEntity();

    await _tokenStorage.saveTokens(tokens);

    _currentUser = response.user.toEntity();

    return tokens;
  }

  @override
  Future<AuthenticatedUser?> restoreSession() async {
    final tokens = await _tokenStorage.readTokens();

    if (tokens == null) {
      return null;
    }

    try {
      final user = await getCurrentUser();

      if (user != null) {
        return user;
      }

      await _tokenStorage.clearTokens();

      return null;
    } catch (_) {
      await _tokenStorage.clearTokens();
      return null;
    }
  }

  @override
  Future<AuthenticatedUser?> getCurrentUser() async {
    if (_currentUser != null) {
      return _currentUser;
    }

    final user = await _remoteDataSource.getCurrentUser();

    _currentUser = user.toEntity();

    return _currentUser;
  }

  @override
  Future<void> logout() async {
    try {
      await _remoteDataSource.logout();
    } finally {
      _currentUser = null;
      await _tokenStorage.clearTokens();
    }
  }

  @override
  Future<void> forgotPassword({
    required String email,
  }) {
    return _remoteDataSource.forgotPassword(
      email: email,
    );
  }

  @override
  Future<void> resetPassword({
    required String token,
    required String newPassword,
  }) {
    return _remoteDataSource.resetPassword(
      token: token,
      newPassword: newPassword,
    );
  }
}