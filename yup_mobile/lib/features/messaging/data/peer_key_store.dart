import 'dart:convert';
import '../../../core/secure_storage/secure_storage_service.dart';

class PeerKeyInfo {
  final String username;
  final String pinnedIdentityKey;
  final String fingerprint;
  final bool verified;
  final bool keyChanged;
  final int lastSeenAt;

  const PeerKeyInfo({
    required this.username,
    required this.pinnedIdentityKey,
    required this.fingerprint,
    this.verified = false,
    this.keyChanged = false,
    this.lastSeenAt = 0,
  });

  Map<String, dynamic> toJson() => {
    'username': username,
    'pinned_identity_key': pinnedIdentityKey,
    'fingerprint': fingerprint,
    'verified': verified,
    'key_changed': keyChanged,
    'last_seen_at': lastSeenAt,
  };

  static PeerKeyInfo fromJson(Map<String, dynamic> json) => PeerKeyInfo(
    username: json['username'] as String,
    pinnedIdentityKey: json['pinned_identity_key'] as String,
    fingerprint: json['fingerprint'] as String,
    verified: json['verified'] as bool? ?? false,
    keyChanged: json['key_changed'] as bool? ?? false,
    lastSeenAt: json['last_seen_at'] as int? ?? 0,
  );

  PeerKeyInfo copyWith({
    String? pinnedIdentityKey,
    String? fingerprint,
    bool? verified,
    bool? keyChanged,
    int? lastSeenAt,
  }) => PeerKeyInfo(
    username: username,
    pinnedIdentityKey: pinnedIdentityKey ?? this.pinnedIdentityKey,
    fingerprint: fingerprint ?? this.fingerprint,
    verified: verified ?? this.verified,
    keyChanged: keyChanged ?? this.keyChanged,
    lastSeenAt: lastSeenAt ?? this.lastSeenAt,
  );
}

class PeerKeyStore {
  final SecureStorageService _storage;
  final String _username;

  PeerKeyStore(this._storage, this._username);

  static String _storageKey(String username) => 'peer_keys:$username';

  Future<Map<String, PeerKeyInfo>> loadAll() async {
    final raw = await _storage.readRaw(_storageKey(_username));
    if (raw == null || raw.isEmpty) return {};
    try {
      final decoded = jsonDecode(raw) as Map<String, dynamic>;
      return decoded.map((k, v) => MapEntry(k, PeerKeyInfo.fromJson(v as Map<String, dynamic>)));
    } catch (_) {
      return {};
    }
  }

  Future<PeerKeyInfo?> getPeerInfo(String peerUsername) async {
    final all = await loadAll();
    return all[peerUsername];
  }

  Future<void> savePeerInfo(PeerKeyInfo info) async {
    final all = await loadAll();
    all[info.username] = info;
    await _saveAll(all);
  }

  Future<void> removePeerInfo(String peerUsername) async {
    final all = await loadAll();
    all.remove(peerUsername);
    await _saveAll(all);
  }

  Future<void> clearAll() async {
    await _storage.deleteRaw(_storageKey(_username));
  }

  Future<void> _saveAll(Map<String, PeerKeyInfo> all) async {
    final encoded = jsonEncode(all.map((k, v) => MapEntry(k, v.toJson())));
    await _storage.writeRaw(_storageKey(_username), encoded);
  }

  Future<bool> hasKeyChanged({
    required String peerUsername,
    required String newIdentityKey,
    required String newFingerprint,
  }) async {
    final existing = await getPeerInfo(peerUsername);
    if (existing == null) {
      await savePeerInfo(PeerKeyInfo(
        username: peerUsername,
        pinnedIdentityKey: newIdentityKey,
        fingerprint: newFingerprint,
        lastSeenAt: DateTime.now().millisecondsSinceEpoch,
      ));
      return false;
    }
    if (existing.pinnedIdentityKey != newIdentityKey) {
      await savePeerInfo(existing.copyWith(
        pinnedIdentityKey: newIdentityKey,
        fingerprint: newFingerprint,
        keyChanged: true,
        lastSeenAt: DateTime.now().millisecondsSinceEpoch,
      ));
      return true;
    }
    return existing.keyChanged;
  }

  Future<void> acceptNewKey(String peerUsername) async {
    final existing = await getPeerInfo(peerUsername);
    if (existing != null) {
      await savePeerInfo(existing.copyWith(keyChanged: false, verified: false));
    }
  }

  Future<void> markVerified(String peerUsername) async {
    final existing = await getPeerInfo(peerUsername);
    if (existing != null) {
      await savePeerInfo(existing.copyWith(verified: true, keyChanged: false));
    }
  }
}
