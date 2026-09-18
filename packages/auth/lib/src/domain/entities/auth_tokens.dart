import 'package:equatable/equatable.dart';

final class AuthTokens extends Equatable {
  const AuthTokens({
    required this.accessToken,
    required this.refreshToken,
    this.accessTokenExpiresAt,
    this.refreshTokenExpiresAt,
  });

  final String accessToken;
  final String refreshToken;
  final DateTime? accessTokenExpiresAt;
  final DateTime? refreshTokenExpiresAt;

  bool get isAccessTokenExpired {
    final expiry = accessTokenExpiresAt;

    if (expiry == null) {
      return false;
    }

    return DateTime.now().toUtc().isAfter(expiry);
  }

  @override
  List<Object?> get props => [
        accessToken,
        refreshToken,
        accessTokenExpiresAt,
        refreshTokenExpiresAt,
      ];
}