import 'package:flutter_test/flutter_test.dart';

/// Pure-Dart model of the key change detection logic used in PeerKeyStore.
class TestPeerKeyStore {
  final Map<String, _StoredPeer> _peers = {};

  bool hasKeyChanged(String peerUsername, String newIdentityKey) {
    final existing = _peers[peerUsername];
    if (existing == null) {
      _peers[peerUsername] = _StoredPeer(newIdentityKey, false);
      return false;
    }
    if (existing.identityKey != newIdentityKey) {
      _peers[peerUsername] = _StoredPeer(newIdentityKey, true);
      return true;
    }
    return existing.keyChanged;
  }

  void acceptNewKey(String peerUsername) {
    final existing = _peers[peerUsername];
    if (existing != null) {
      _peers[peerUsername] = _StoredPeer(existing.identityKey, false);
    }
  }
}

class _StoredPeer {
  final String identityKey;
  final bool keyChanged;
  _StoredPeer(this.identityKey, this.keyChanged);
}

void main() {
  group('Key change detection', () {
    late TestPeerKeyStore store;

    setUp(() {
      store = TestPeerKeyStore();
    });

    test('first key pinning returns false (no change)', () {
      final changed = store.hasKeyChanged('bob', 'bob_key_v1');
      expect(changed, isFalse);
    });

    test('same key returns false', () {
      store.hasKeyChanged('bob', 'bob_key_v1');
      final changed = store.hasKeyChanged('bob', 'bob_key_v1');
      expect(changed, isFalse);
    });

    test('different key returns true (key changed)', () {
      store.hasKeyChanged('bob', 'bob_key_v1');
      final changed = store.hasKeyChanged('bob', 'bob_key_v2');
      expect(changed, isTrue);
    });

    test('acceptNewKey resets key_changed flag', () {
      store.hasKeyChanged('bob', 'bob_key_v1');
      store.hasKeyChanged('bob', 'bob_key_v2');
      store.acceptNewKey('bob');
      final changed = store.hasKeyChanged('bob', 'bob_key_v2');
      expect(changed, isFalse);
    });

    test('multiple peers tracked independently', () {
      store.hasKeyChanged('alice', 'alice_key');
      store.hasKeyChanged('bob', 'bob_key');

      expect(store.hasKeyChanged('alice', 'alice_key'), isFalse);
      expect(store.hasKeyChanged('bob', 'bob_key'), isFalse);
      expect(store.hasKeyChanged('alice', 'alice_key_new'), isTrue);
      expect(store.hasKeyChanged('bob', 'bob_key'), isFalse);
    });
  });
}
