import 'package:flutter_secure_storage/flutter_secure_storage.dart';

class SecureStorageService {
  final FlutterSecureStorage _storage;

  SecureStorageService()
      : _storage = const FlutterSecureStorage();

  Future<void> storeAuthToken(String username, String token) async {
    await _storage.write(key: 'auth_token:$username', value: token);
  }

  Future<String?> getAuthToken(String username) async {
    return await _storage.read(key: 'auth_token:$username');
  }

  Future<void> storeAccountPickle(String username, String pickle) async {
    await _storage.write(key: 'account_pickle:$username', value: pickle);
  }

  Future<String?> getAccountPickle(String username) async {
    return await _storage.read(key: 'account_pickle:$username');
  }

  Future<void> storeIdentityKeys({
    required String username,
    required String curve25519,
    required String ed25519,
  }) async {
    await Future.wait([
      _storage.write(key: 'identity_curve25519:$username', value: curve25519),
      _storage.write(key: 'identity_ed25519:$username', value: ed25519),
    ]);
  }

  Future<Map<String, String>?> getIdentityKeys(String username) async {
    final results = await Future.wait([
      _storage.read(key: 'identity_curve25519:$username'),
      _storage.read(key: 'identity_ed25519:$username'),
    ]);
    final curve = results[0];
    final ed = results[1];
    if (curve == null || ed == null) return null;
    return {'curve25519': curve, 'ed25519': ed};
  }

  Future<bool> hasExistingAccount(String username) async {
    final pickle = await _storage.read(key: 'account_pickle:$username');
    return pickle != null;
  }

  Future<String?> getStoredUsername() async {
    return await _storage.read(key: 'active_username');
  }

  Future<void> setActiveUsername(String username) async {
    await _storage.write(key: 'active_username', value: username);
  }

  Future<String?> readRaw(String key) async {
    return await _storage.read(key: key);
  }

  Future<void> writeRaw(String key, String value) async {
    await _storage.write(key: key, value: value);
  }

  Future<void> deleteRaw(String key) async {
    await _storage.delete(key: key);
  }

  Future<void> clearUserData(String username) async {
    await Future.wait([
      _storage.delete(key: 'auth_token:$username'),
      _storage.delete(key: 'account_pickle:$username'),
      _storage.delete(key: 'identity_curve25519:$username'),
      _storage.delete(key: 'identity_ed25519:$username'),
      _storage.delete(key: 'sessions:$username'),
      _storage.delete(key: 'active_username'),
    ]);
  }
}
