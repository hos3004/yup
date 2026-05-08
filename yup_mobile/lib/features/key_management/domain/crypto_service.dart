import '../../../core/crypto_ffi/crypto_bridge.dart';

class CryptoService {
  final CryptoBridge _bridge;

  CryptoService(this._bridge);

  /// Generate a new Olm account and return identity keys.
  Map<String, dynamic> generateAccount() {
    _bridge.initialize();
    return _bridge.generateAccount();
  }

  /// Get current identity keys.
  Map<String, dynamic> getIdentityKeys() {
    return _bridge.getIdentityKeys();
  }

  /// Generate one-time keys for pre-key bundles.
  List<String> generateOneTimeKeys({int count = 50}) {
    return _bridge.generateOneTimeKeys(count);
  }

  /// Sign a message with the account's Ed25519 key.
  String signMessage(String message) {
    return _bridge.signMessage(message);
  }

  /// Create an outbound Olm session to another user.
  Map<String, dynamic> createOutboundSession(
    String theirIdentityKey,
    String theirOneTimeKey,
  ) {
    return _bridge.createOutboundSession(theirIdentityKey, theirOneTimeKey);
  }

  /// Encrypt a plaintext message using a session.
  Map<String, dynamic> encryptMessage(String sessionId, String plaintext) {
    return _bridge.encryptMessage(sessionId, plaintext);
  }

  /// Create an inbound session from the first (pre-key) message.
  String createInboundSession(
    String theirIdentityKey,
    String ciphertextB64,
  ) {
    return _bridge.createInboundSession(theirIdentityKey, ciphertextB64);
  }

  /// Decrypt a subsequent message using an existing session.
  String decryptMessage(
    String sessionId,
    String ciphertextB64,
    int messageType,
  ) {
    return _bridge.decryptMessage(sessionId, ciphertextB64, messageType);
  }

  /// Compute the fingerprint between our key and another user's key.
  String getFingerprint(String theirIdentityKey) {
    return _bridge.getFingerprint(theirIdentityKey);
  }

  /// Serialize the current account to a JSON pickle string.
  String pickleAccount() {
    return _bridge.pickleAccount();
  }

  /// Restore an account from a JSON pickle string.
  void unpickleAccount(String pickle) {
    _bridge.unpickleAccount(pickle);
  }

  /// Serialize a session to a JSON pickle string.
  String pickleSession(String sessionId) {
    return _bridge.pickleSession(sessionId);
  }

  /// Restore a session from a JSON pickle string.
  void unpickleSession(String sessionId, String pickle) {
    _bridge.unpickleSession(sessionId, pickle);
  }
}
