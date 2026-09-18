import 'dart:io';

import 'package:dio/dio.dart';

import 'network_failure.dart';

final class NetworkFailureMapper {
  const NetworkFailureMapper();

  NetworkFailure map(Object error) {
    if (error is NetworkFailure) {
      return error;
    }

    if (error is! DioException) {
      return UnknownNetworkFailure(cause: error);
    }

    final response = error.response;
    final statusCode = response?.statusCode;
    final body = _asMap(response?.data);

    final message = _readMessage(body) ?? _fallbackMessage(error);
    final code = _readString(body, 'code');
    final requestId = _readString(body, 'request_id') ??
        _readString(body, 'requestId');

    switch (statusCode) {
      case 400:
        return BadRequestFailure(
          message: message,
          code: code,
          details: _readMap(body, 'details'),
        );

      case 401:
        return UnauthorizedFailure(
          message: message,
          code: code,
        );

      case 403:
        return ForbiddenFailure(
          message: message,
          code: code,
        );

      case 404:
        return NotFoundFailure(
          message: message,
          code: code,
        );

      case 409:
        return ConflictFailure(
          message: message,
          code: code,
          details: _readMap(body, 'details'),
        );

      case 422:
        return ValidationFailure(
          message: message,
          code: code,
          errors: _readValidationErrors(body),
        );

      case 429:
        return RateLimitedFailure(
          message: message,
          code: code,
          retryAfter: _readRetryAfter(response),
        );

      default:
        if (statusCode != null && statusCode >= 500) {
          return ServerFailure(
            message: message,
            code: code,
            statusCode: statusCode,
            requestId: requestId,
          );
        }

        switch (error.type) {
  case DioExceptionType.connectionError:
  case DioExceptionType.connectionTimeout:
  case DioExceptionType.unknown:
    if (error.error is SocketException) {
      return NetworkUnavailableFailure(message: message);
    }

    return NetworkUnavailableFailure(message: message);

  case DioExceptionType.sendTimeout:
  case DioExceptionType.receiveTimeout:
  case DioExceptionType.transformTimeout:
    return TimeoutFailure(message: message);

  case DioExceptionType.badResponse:
    return UnknownNetworkFailure(
      message: message,
      cause: error,
    );

  case DioExceptionType.cancel:
    return UnknownNetworkFailure(
      message: 'The request was cancelled.',
      cause: error,
    );

  case DioExceptionType.badCertificate:
    return NetworkUnavailableFailure(
      message: 'The server certificate could not be verified.',
    );
}}
  }

  Map<String, dynamic> _asMap(Object? value) {
    if (value is Map<String, dynamic>) {
      return value;
    }

    if (value is Map) {
      return Map<String, dynamic>.from(value);
    }

    return const {};
  }

  String? _readMessage(Map<String, dynamic> body) {
    final error = body['error'];

    if (error is Map) {
      final nestedMessage = error['message'];

      if (nestedMessage is String && nestedMessage.trim().isNotEmpty) {
        return nestedMessage;
      }
    }

    final message = body['message'];

    if (message is String && message.trim().isNotEmpty) {
      return message;
    }

    return null;
  }

  String _fallbackMessage(DioException error) {
    return error.message ?? 'A network error occurred.';
  }

  String? _readString(
    Map<String, dynamic> body,
    String key,
  ) {
    final value = body[key];

    return value is String && value.trim().isNotEmpty ? value : null;
  }

  Map<String, dynamic>? _readMap(
    Map<String, dynamic> body,
    String key,
  ) {
    final value = body[key];

    if (value is Map) {
      return Map<String, dynamic>.from(value);
    }

    return null;
  }

  Map<String, List<String>> _readValidationErrors(
    Map<String, dynamic> body,
  ) {
    final rawErrors = body['errors'];

    if (rawErrors is! Map) {
      return const {};
    }

    final result = <String, List<String>>{};

    for (final entry in rawErrors.entries) {
      final field = entry.key.toString();
      final value = entry.value;

      if (value is List) {
        result[field] = value.map((item) => item.toString()).toList();
      } else {
        result[field] = [value.toString()];
      }
    }

    return result;
  }

  Duration? _readRetryAfter(Response<dynamic>? response) {
    final value = response?.headers.value('retry-after');

    if (value == null) {
      return null;
    }

    final seconds = int.tryParse(value);

    if (seconds == null || seconds < 0) {
      return null;
    }

    return Duration(seconds: seconds);
  }
}