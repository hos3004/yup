import '../../../core/networking/api_client.dart';
import '../../../core/secure_storage/secure_storage_service.dart';
import '../domain/crypto_service.dart';

class RegistrationResult {
  final String username;
  final String curve25519;
  final String ed25519;
  final String authToken;

  const RegistrationResult({
    required this.username,
    required this.curve25519,
    required this.ed25519,
    required this.authToken,
  });
}

class DeviceRegistrationService {
  final ApiClient _api;
  final CryptoService _crypto;
  final SecureStorageService _storage;

  DeviceRegistrationService(this._api, this._crypto, this._storage);

  /// Full first-time registration flow.
  Future<RegistrationResult> register(String username) async {
    final regData = await _api.registerUser(username);
    final authToken = regData['auth_token'] as String;

    final keys = _crypto.generateAccount();
    final curve25519 = keys['curve25519'] as String;
    final ed25519 = keys['ed25519'] as String;

    final otks = _crypto.generateOneTimeKeys(count: 50);
    final signature = _crypto.signMessage(curve25519);

    await _api.uploadKeys(username, {
      'curve_key': curve25519,
      'ed_key': ed25519,
      'one_time_keys': otks,
      'signature': signature,
    });

    final accountPickle = _crypto.pickleAccount();

    await Future.wait([
      _storage.storeAuthToken(username, authToken),
      _storage.storeAccountPickle(username, accountPickle),
      _storage.storeIdentityKeys(
        username: username,
        curve25519: curve25519,
        ed25519: ed25519,
      ),
      _storage.setActiveUsername(username),
    ]);

    return RegistrationResult(
      username: username,
      curve25519: curve25519,
      ed25519: ed25519,
      authToken: authToken,
    );
  }

  /// Restore an existing account from secure storage.
  Future<RegistrationResult?> tryRestore(String username) async {
    final token = await _storage.getAuthToken(username);
    final pickle = await _storage.getAccountPickle(username);
    final keys = await _storage.getIdentityKeys(username);

    if (token == null || pickle == null || keys == null) return null;

    _api.setToken(token);
    _crypto.unpickleAccount(pickle);

    return RegistrationResult(
      username: username,
      curve25519: keys['curve25519']!,
      ed25519: keys['ed25519']!,
      authToken: token,
    );
  }

  /// Restore the most recent session if available.
  Future<RegistrationResult?> tryRestoreLastSession() async {
    final username = await _storage.getStoredUsername();
    if (username == null) return null;
    return tryRestore(username);
  }
}
