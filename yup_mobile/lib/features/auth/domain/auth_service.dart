import '../../../core/networking/api_client.dart';
import '../../../core/secure_storage/secure_storage_service.dart';
import '../../key_management/data/device_registration_service.dart';
import '../data/auth_repository.dart';

class AuthResult {
  final String username;
  final String curve25519;
  final String ed25519;
  final String authToken;

  const AuthResult({
    required this.username,
    required this.curve25519,
    required this.ed25519,
    required this.authToken,
  });
}

class AuthService {
  final AuthRepository _authRepo;
  final DeviceRegistrationService _registration;

  AuthService({
    required ApiClient api,
    required SecureStorageService storage,
    required DeviceRegistrationService registration,
  })  : _authRepo = AuthRepository(api, storage),
        _registration = registration;

  Future<AuthResult> register(String username) async {
    final result = await _registration.register(username);
    return AuthResult(
      username: result.username,
      curve25519: result.curve25519,
      ed25519: result.ed25519,
      authToken: result.authToken,
    );
  }

  Future<AuthResult?> tryRestoreLastSession() async {
    final result = await _registration.tryRestoreLastSession();
    if (result == null) return null;
    return AuthResult(
      username: result.username,
      curve25519: result.curve25519,
      ed25519: result.ed25519,
      authToken: result.authToken,
    );
  }

  Future<String?> getStoredUsername() async {
    return await _authRepo.getStoredUsername();
  }

  Future<void> logout(String username) async {
    await _authRepo.clearAuth(username);
  }
}
