import '../../../core/crypto_ffi/crypto_bridge.dart';

class CryptoService {
  final CryptoBridge _bridge;

  CryptoService(this._bridge);

  Map<String, dynamic> generateAccount() {
    _bridge.initialize();
    return _bridge.generateAccount();
  }

  Map<String, dynamic> getIdentityKeys() {
    return _bridge.getIdentityKeys();
  }

  List<String> generateOneTimeKeys({int count = 50}) {
    return _bridge.generateOneTimeKeys(count);
  }

  String signMessage(String message) {
    return _bridge.signMessage(message);
  }

  Map<String, dynamic> createOutboundSession(
    String theirIdentityKey,
    String theirOneTimeKey,
  ) {
    return _bridge.createOutboundSession(theirIdentityKey, theirOneTimeKey);
  }

  Map<String, dynamic> encryptMessage(String sessionId, String plaintext) {
    return _bridge.encryptMessage(sessionId, plaintext);
  }

  Map<String, dynamic> createInboundSession(
    String theirIdentityKey,
    String ciphertextB64,
  ) {
    return _bridge.createInboundSession(theirIdentityKey, ciphertextB64);
  }

  String decryptMessage(
    String sessionId,
    String ciphertextB64,
    int messageType,
  ) {
    return _bridge.decryptMessage(sessionId, ciphertextB64, messageType);
  }

  String getFingerprint(String theirIdentityKey) {
    return _bridge.getFingerprint(theirIdentityKey);
  }

  String pickleAccount() {
    return _bridge.pickleAccount();
  }

  void unpickleAccount(String pickle) {
    _bridge.unpickleAccount(pickle);
  }

  String pickleSession(String sessionId) {
    return _bridge.pickleSession(sessionId);
  }

  void unpickleSession(String sessionId, String pickle) {
    _bridge.unpickleSession(sessionId, pickle);
  }
}
