import 'dart:convert';
import '../../../core/secure_storage/secure_storage_service.dart';
import '../../key_management/domain/crypto_service.dart';

class SessionStore {
  final CryptoService _crypto;
  final SecureStorageService _storage;
  final String _username;
  Map<String, _SessionEntry> _sessions = {};

  SessionStore(this._crypto, this._storage, this._username);

  static String _storageKey(String username) => 'sessions:$username';

  Future<void> load() async {
    final raw = await _storage.readRaw(_storageKey(_username));
    if (raw == null || raw.isEmpty) {
      _sessions = {};
      return;
    }
    try {
      final decoded = jsonDecode(raw) as Map<String, dynamic>;
      _sessions = decoded.map((k, v) {
        final entry = v as Map<String, dynamic>;
        return MapEntry(k, _SessionEntry(
          sessionId: entry['session_id'] as String,
          pickle: entry['pickle'] as String,
        ));
      });
      // Restore each session into the Rust bridge
      for (final entry in _sessions.values) {
        _crypto.unpickleSession(entry.sessionId, entry.pickle);
      }
    } catch (_) {
      _sessions = {};
    }
  }

  Future<void> save() async {
    final encoded = jsonEncode(_sessions.map((k, v) => MapEntry(k, {
      'session_id': v.sessionId,
      'pickle': v.pickle,
    })));
    await _storage.writeRaw(_storageKey(_username), encoded);
  }

  String? getSessionId(String peerCurveKey) {
    final entry = _sessions[peerCurveKey];
    return entry?.sessionId;
  }

  Future<String> addSession(String sessionId, String peerCurveKey) async {
    final pickle = _crypto.pickleSession(sessionId);
    _sessions[peerCurveKey] = _SessionEntry(sessionId: sessionId, pickle: pickle);
    await save();
    return sessionId;
  }

  void removeSession(String peerCurveKey) {
    _sessions.remove(peerCurveKey);
  }

  bool hasSession(String peerCurveKey) => _sessions.containsKey(peerCurveKey);

  String? getPeerForSession(String sessionId) {
    for (final entry in _sessions.entries) {
      if (entry.value.sessionId == sessionId) return entry.key;
    }
    return null;
  }
}

class _SessionEntry {
  final String sessionId;
  final String pickle;

  const _SessionEntry({required this.sessionId, required this.pickle});
}
