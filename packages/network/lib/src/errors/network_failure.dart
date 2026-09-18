import 'package:equatable/equatable.dart';

sealed class NetworkFailure extends Equatable {
  const NetworkFailure();

  @override
  List<Object?> get props => [];
}

final class BadRequestFailure extends NetworkFailure {
  const BadRequestFailure({
    this.message = 'The request was invalid.',
    this.code,
    this.details,
  });

  final String message;
  final String? code;
  final Map<String, dynamic>? details;

  @override
  List<Object?> get props => [message, code, details];
}

final class UnauthorizedFailure extends NetworkFailure {
  const UnauthorizedFailure({
    this.message = 'Authentication is required.',
    this.code,
  });

  final String message;
  final String? code;

  @override
  List<Object?> get props => [message, code];
}

final class ForbiddenFailure extends NetworkFailure {
  const ForbiddenFailure({
    this.message = 'You do not have permission to perform this action.',
    this.code,
  });

  final String message;
  final String? code;

  @override
  List<Object?> get props => [message, code];
}

final class NotFoundFailure extends NetworkFailure {
  const NotFoundFailure({
    this.message = 'The requested resource was not found.',
    this.code,
  });

  final String message;
  final String? code;

  @override
  List<Object?> get props => [message, code];
}

final class ConflictFailure extends NetworkFailure {
  const ConflictFailure({
    this.message = 'The request conflicts with the current state.',
    this.code,
    this.details,
  });

  final String message;
  final String? code;
  final Map<String, dynamic>? details;

  @override
  List<Object?> get props => [message, code, details];
}

final class ValidationFailure extends NetworkFailure {
  const ValidationFailure({
    this.message = 'One or more fields are invalid.',
    this.code,
    this.errors = const {},
  });

  final String message;
  final String? code;
  final Map<String, List<String>> errors;

  @override
  List<Object?> get props => [message, code, errors];
}

final class RateLimitedFailure extends NetworkFailure {
  const RateLimitedFailure({
    this.message = 'Too many requests. Please try again later.',
    this.code,
    this.retryAfter,
  });

  final String message;
  final String? code;
  final Duration? retryAfter;

  @override
  List<Object?> get props => [message, code, retryAfter];
}

final class ServerFailure extends NetworkFailure {
  const ServerFailure({
    this.message = 'The server encountered an unexpected error.',
    this.code,
    this.statusCode,
    this.requestId,
  });

  final String message;
  final String? code;
  final int? statusCode;
  final String? requestId;

  @override
  List<Object?> get props => [
        message,
        code,
        statusCode,
        requestId,
      ];
}

final class NetworkUnavailableFailure extends NetworkFailure {
  const NetworkUnavailableFailure({
    this.message = 'Unable to connect to the server.',
  });

  final String message;

  @override
  List<Object?> get props => [message];
}

final class TimeoutFailure extends NetworkFailure {
  const TimeoutFailure({
    this.message = 'The request timed out.',
  });

  final String message;

  @override
  List<Object?> get props => [message];
}

final class SerializationFailure extends NetworkFailure {
  const SerializationFailure({
    this.message = 'The server response could not be processed.',
    this.cause,
  });

  final String message;
  final Object? cause;

  @override
  List<Object?> get props => [message, cause];
}

final class UnknownNetworkFailure extends NetworkFailure {
  const UnknownNetworkFailure({
    this.message = 'An unexpected network error occurred.',
    this.cause,
  });

  final String message;
  final Object? cause;

  @override
  List<Object?> get props => [message, cause];
}