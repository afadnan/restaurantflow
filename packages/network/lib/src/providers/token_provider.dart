abstract interface class TokenProvider {
  String? get accessToken;

  Future<bool> refreshAccessToken();

  Future<void> clearTokens();
}