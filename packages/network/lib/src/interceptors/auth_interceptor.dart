import 'dart:async';

import 'package:dio/dio.dart';

import '../providers/token_provider.dart';

final class AuthInterceptor extends QueuedInterceptor {
  AuthInterceptor({
  required this._dio,
  required this._tokenProvider,
  required this._refreshPath,
});

  final Dio _dio;
  final TokenProvider _tokenProvider;
  final String _refreshPath;

  Future<bool>? _refreshOperation;

  @override
  void onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) {
    final token = _tokenProvider.accessToken;

    if (token != null && token.trim().isNotEmpty) {
      options.headers['Authorization'] = 'Bearer $token';
    }

    handler.next(options);
  }

  @override
  Future<void> onError(
    DioException err,
    ErrorInterceptorHandler handler,
  ) async {
    final request = err.requestOptions;

    final isUnauthorized = err.response?.statusCode == 401;
    final alreadyRetried = request.extra['auth_retry'] == true;
    final isRefreshRequest = request.path == _refreshPath;

    if (!isUnauthorized || alreadyRetried || isRefreshRequest) {
      handler.next(err);
      return;
    }

    final refreshed = await _refreshAccessToken();

    if (!refreshed) {
      await _tokenProvider.clearTokens();
      handler.next(err);
      return;
    }

    final accessToken = _tokenProvider.accessToken;

    if (accessToken == null || accessToken.isEmpty) {
      handler.next(err);
      return;
    }

    final retryOptions = request.copyWith(
      headers: {
        ...request.headers,
        'Authorization': 'Bearer $accessToken',
      },
      extra: {
        ...request.extra,
        'auth_retry': true,
      },
    );

    try {
      final response = await _dio.fetch<dynamic>(retryOptions);
      handler.resolve(response);
    } on DioException catch (retryError) {
      handler.next(retryError);
    }
  }

  Future<bool> _refreshAccessToken() {
    final existingOperation = _refreshOperation;

    if (existingOperation != null) {
      return existingOperation;
    }

    final operation = _performRefresh();

    _refreshOperation = operation;

    operation.whenComplete(() {
      if (identical(_refreshOperation, operation)) {
        _refreshOperation = null;
      }
    });

    return operation;
  }

  Future<bool> _performRefresh() async {
    try {
      return await _tokenProvider.refreshAccessToken();
    } catch (_) {
      return false;
    }
  }
}