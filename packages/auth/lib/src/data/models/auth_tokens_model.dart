import '../../domain/entities/auth_tokens.dart';

final class AuthTokensModel {
  const AuthTokensModel({
    required this.accessToken,
    required this.refreshToken,
    this.accessTokenExpiresAt,
    this.refreshTokenExpiresAt,
  });

  final String accessToken;
  final String refreshToken;
  final DateTime? accessTokenExpiresAt;
  final DateTime? refreshTokenExpiresAt;

  factory AuthTokensModel.fromJson(Map<String, dynamic> json) {
    return AuthTokensModel(
      accessToken: json['access_token'] as String,
      refreshToken: json['refresh_token'] as String,
      accessTokenExpiresAt: _parseDateTime(
        json['access_token_expires_at'],
      ),
      refreshTokenExpiresAt: _parseDateTime(
        json['refresh_token_expires_at'],
      ),
    );
  }

  AuthTokens toEntity() {
    return AuthTokens(
      accessToken: accessToken,
      refreshToken: refreshToken,
      accessTokenExpiresAt: accessTokenExpiresAt,
      refreshTokenExpiresAt: refreshTokenExpiresAt,
    );
  }

  static DateTime? _parseDateTime(Object? value) {
    if (value is! String) {
      return null;
    }

    return DateTime.tryParse(value)?.toUtc();
  }
}