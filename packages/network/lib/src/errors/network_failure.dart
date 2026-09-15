import 'package:freezed_annotation/freezed_annotation.dart';

part 'network_failure.freezed.dart';

@freezed
sealed class NetworkFailure with _$NetworkFailure {
  const factory NetworkFailure.badRequest({
    String? message,
    Map<String, dynamic>? errors,
  }) = BadRequestFailure;

  const factory NetworkFailure.unauthorized({
    String? message,
  }) = UnauthorizedFailure;

  const factory NetworkFailure.forbidden({
    String? message,
  }) = ForbiddenFailure;

  const factory NetworkFailure.notFound({
    String? message,
  }) = NotFoundFailure;

  const factory NetworkFailure.conflict({
    String? message,
  }) = ConflictFailure;

  const factory NetworkFailure.validation({
    String? message,
    Map<String, dynamic>? errors,
  }) = ValidationFailure;

  const factory NetworkFailure.rateLimited({
    String? message,
    Duration? retryAfter,
  }) = RateLimitedFailure;

  const factory NetworkFailure.server({
    String? message,
    int? statusCode,
  }) = ServerFailure;

  const factory NetworkFailure.network({
    String? message,
  }) = NetworkConnectionFailure;

  const factory NetworkFailure.timeout({
    String? message,
  }) = TimeoutFailure;

  const factory NetworkFailure.serialization({
    String? message,
  }) = SerializationFailure;

  const factory NetworkFailure.unknown({
    String? message,
  }) = UnknownNetworkFailure;
}