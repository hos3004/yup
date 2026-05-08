import '../../../core/networking/api_client.dart';
import '../../../core/secure_storage/secure_storage_service.dart';

class AuthRepository {
  final ApiClient _api;
  final SecureStorageService _storage;

  AuthRepository(this._api, this._storage);

  Future<Map<String, dynamic>> registerUser(String username) async {
    return await _api.registerUser(username);
  }

  Future<Map<String, dynamic>> getUser(String username) async {
    return await _api.getUser(username);
  }

  Future<void> storeAuthToken(String username, String token) async {
    await _storage.storeAuthToken(username, token);
  }

  Future<String?> getAuthToken(String username) async {
    return await _storage.getAuthToken(username);
  }

  Future<String?> getStoredUsername() async {
    return await _storage.getStoredUsername();
  }

  Future<void> setActiveUsername(String username) async {
    await _storage.setActiveUsername(username);
  }

  Future<void> clearAuth(String username) async {
    await _storage.clearUserData(username);
  }
}
