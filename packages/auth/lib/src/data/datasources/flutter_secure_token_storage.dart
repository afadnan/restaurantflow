import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import '../../domain/entities/auth_tokens.dart';
import 'secure_token_storage.dart';

final class FlutterSecureTokenStorage implements SecureTokenStorage {
  FlutterSecureTokenStorage({
    FlutterSecureStorage? storage,
  }) : _storage = storage ?? const FlutterSecureStorage();

  final FlutterSecureStorage _storage;

  static const _accessTokenKey = 'auth.access_token';
  static const _refreshTokenKey = 'auth.refresh_token';
  static const _accessTokenExpiryKey = 'auth.access_token_expiry';
  static const _refreshTokenExpiryKey = 'auth.refresh_token_expiry';

  @override
  Future<void> saveTokens(AuthTokens tokens) async {
    await Future.wait([
      _storage.write(
        key: _accessTokenKey,
        value: tokens.accessToken,
      ),
      _storage.write(
        key: _refreshTokenKey,
        value: tokens.refreshToken,
      ),
      _storage.write(
        key: _accessTokenExpiryKey,
        value: tokens.accessTokenExpiresAt?.toIso8601String(),
      ),
      _storage.write(
        key: _refreshTokenExpiryKey,
        value: tokens.refreshTokenExpiresAt?.toIso8601String(),
      ),
    ]);
  }

  @override
  Future<AuthTokens?> readTokens() async {
    final values = await Future.wait([
      _storage.read(key: _accessTokenKey),
      _storage.read(key: _refreshTokenKey),
      _storage.read(key: _accessTokenExpiryKey),
      _storage.read(key: _refreshTokenExpiryKey),
    ]);

    final accessToken = values[0];
    final refreshToken = values[1];

    if (accessToken == null ||
        accessToken.isEmpty ||
        refreshToken == null ||
        refreshToken.isEmpty) {
      return null;
    }

    return AuthTokens(
      accessToken: accessToken,
      refreshToken: refreshToken,
      accessTokenExpiresAt: _parseDateTime(values[2]),
      refreshTokenExpiresAt: _parseDateTime(values[3]),
    );
  }

  @override
  Future<void> clearTokens() async {
    await Future.wait([
      _storage.delete(key: _accessTokenKey),
      _storage.delete(key: _refreshTokenKey),
      _storage.delete(key: _accessTokenExpiryKey),
      _storage.delete(key: _refreshTokenExpiryKey),
    ]);
  }

  DateTime? _parseDateTime(String? value) {
    if (value == null || value.isEmpty) {
      return null;
    }

    return DateTime.tryParse(value)?.toUtc();
  }
}