import 'package:network/network.dart';

import '../models/auth_tokens_model.dart';
import '../models/authenticated_user_model.dart';

abstract interface class AuthRemoteDataSource {
  Future<AuthResponseModel> login({
    required String email,
    required String password,
  });

  Future<AuthResponseModel> register({
    required String email,
    required String password,
    required String displayName,
    String? phoneNumber,
  });

  Future<AuthResponseModel> refresh({
    required String refreshToken,
  });

  Future<AuthenticatedUserModel> getCurrentUser();

  Future<void> logout();

  Future<void> forgotPassword({
    required String email,
  });

  Future<void> resetPassword({
    required String token,
    required String newPassword,
  });
}

final class AuthResponseModel {
  const AuthResponseModel({
    required this.tokens,
    required this.user,
  });

  final AuthTokensModel tokens;
  final AuthenticatedUserModel user;

  factory AuthResponseModel.fromJson(
    Map<String, dynamic> json,
  ) {
    final data = _extractData(json);

    return AuthResponseModel(
      tokens: AuthTokensModel.fromJson(
        _asMap(data['tokens']),
      ),
      user: AuthenticatedUserModel.fromJson(
        _asMap(data['user']),
      ),
    );
  }

  static Map<String, dynamic> _extractData(
    Map<String, dynamic> json,
  ) {
    final data = json['data'];

    if (data is Map) {
      return Map<String, dynamic>.from(data);
    }

    return json;
  }

  static Map<String, dynamic> _asMap(Object? value) {
    if (value is Map) {
      return Map<String, dynamic>.from(value);
    }

    throw const FormatException(
      'Expected a JSON object.',
    );
  }
}

final class DioAuthRemoteDataSource
    implements AuthRemoteDataSource {
  const DioAuthRemoteDataSource(
     this._apiClient,
  );

  final ApiClient _apiClient;

  @override
  Future<AuthResponseModel> login({
    required String email,
    required String password,
  }) async {
    final response = await _apiClient.post<Map<String, dynamic>>(
      '/api/v1/auth/login',
      data: {
        'email': email,
        'password': password,
      },
    );

    return AuthResponseModel.fromJson(
      _requireData(response.data),
    );
  }

  @override
Future<AuthResponseModel> register({
  required String email,
  required String password,
  required String displayName,
  String? phoneNumber,
}) async {
  final response = await _apiClient.post<Map<String, dynamic>>(
    '/api/v1/auth/register',
    data: {
      'email': email,
      'password': password,
      'display_name': displayName,
      if (phoneNumber?.trim() case final trimmedPhone?
          when trimmedPhone.isNotEmpty)
        'phone_number': trimmedPhone,
    },
  );

  return AuthResponseModel.fromJson(
    _requireData(response.data),
  );
}

  @override
  Future<AuthResponseModel> refresh({
    required String refreshToken,
  }) async {
    final response = await _apiClient.post<Map<String, dynamic>>(
      '/api/v1/auth/refresh',
      data: {
        'refresh_token': refreshToken,
      },
    );

    return AuthResponseModel.fromJson(
      _requireData(response.data),
    );
  }

  @override
  Future<AuthenticatedUserModel> getCurrentUser() async {
    final response =
        await _apiClient.get<Map<String, dynamic>>(
      '/api/v1/auth/me',
    );

    final json = _requireData(response.data);

    final data = json['data'];

    if (data is Map) {
      return AuthenticatedUserModel.fromJson(
        Map<String, dynamic>.from(data),
      );
    }

    return AuthenticatedUserModel.fromJson(json);
  }

  @override
  Future<void> logout() async {
    await _apiClient.post<void>(
      '/api/v1/auth/logout',
    );
  }

  @override
  Future<void> forgotPassword({
    required String email,
  }) async {
    await _apiClient.post<void>(
      '/api/v1/auth/forgot-password',
      data: {
        'email': email,
      },
    );
  }

  @override
  Future<void> resetPassword({
    required String token,
    required String newPassword,
  }) async {
    await _apiClient.post<void>(
      '/api/v1/auth/reset-password',
      data: {
        'token': token,
        'new_password': newPassword,
      },
    );
  }

  Map<String, dynamic> _requireData(
    Map<String, dynamic>? data,
  ) {
    if (data == null) {
      throw const FormatException(
        'The server returned an empty response.',
      );
    }

    return data;
  }
}