import 'package:network/network.dart';

import 'secure_token_storage.dart';

final class AuthTokenProvider implements TokenProvider {
  AuthTokenProvider({
    required this._tokenStorage,
  required this._refresh,
  });

  final SecureTokenStorage _tokenStorage;
  final Future<bool> Function() _refresh;

  String? _accessToken;

  @override
  String? get accessToken => _accessToken;

  Future<void> initialize() async {
    final tokens = await _tokenStorage.readTokens();
    _accessToken = tokens?.accessToken;
  }

  Future<void> updateAccessToken(String accessToken) async {
    _accessToken = accessToken;
  }

  @override
  Future<bool> refreshAccessToken() async {
    final refreshed = await _refresh();

    if (!refreshed) {
      return false;
    }

    final tokens = await _tokenStorage.readTokens();
    _accessToken = tokens?.accessToken;

    return _accessToken != null && _accessToken!.isNotEmpty;
  }

  @override
  Future<void> clearTokens() async {
    _accessToken = null;
    await _tokenStorage.clearTokens();
  }
}