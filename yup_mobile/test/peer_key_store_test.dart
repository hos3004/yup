import 'package:flutter_test/flutter_test.dart';
import 'package:yup_mobile/core/secure_storage/secure_storage_service.dart';
import 'package:yup_mobile/features/messaging/data/peer_key_store.dart';

/// A real-SecureStorageService-like implementation backed by in-memory map.
class InMemorySecureStorage extends SecureStorageService {
  final Map<String, String> _store = {};

  InMemorySecureStorage() : super.testable();

  @override
  Future<void> writeRaw(String key, String value) async {
    _store[key] = value;
  }

  @override
  Future<String?> readRaw(String key) async {
    return _store[key];
  }

  @override
  Future<void> deleteRaw(String key) async {
    _store.remove(key);
  }
}

void main() {
  group('PeerKeyStore real behavior', () {
    late InMemorySecureStorage storage;
    late PeerKeyStore peerStore;

    setUp(() {
      storage = InMemorySecureStorage();
      peerStore = PeerKeyStore(storage, 'alice');
    });

    test('first key pin returns false (no change)', () async {
      final changed = await peerStore.hasKeyChanged(
        peerUsername: 'bob',
        newIdentityKey: 'bob_key_v1',
        newFingerprint: 'fp_v1',
      );
      expect(changed, isFalse);

      // Verify it was stored
      final info = await peerStore.getPeerInfo('bob');
      expect(info, isNotNull);
      expect(info!.pinnedIdentityKey, equals('bob_key_v1'));
    });

    test('same key returns false (no change)', () async {
      await peerStore.hasKeyChanged(
        peerUsername: 'bob', newIdentityKey: 'bob_key_v1', newFingerprint: 'fp_v1',
      );
      final changed = await peerStore.hasKeyChanged(
        peerUsername: 'bob', newIdentityKey: 'bob_key_v1', newFingerprint: 'fp_v1',
      );
      expect(changed, isFalse);
    });

    test('changed key sets key_changed flag', () async {
      await peerStore.hasKeyChanged(
        peerUsername: 'bob', newIdentityKey: 'bob_key_v1', newFingerprint: 'fp_v1',
      );
      final changed = await peerStore.hasKeyChanged(
        peerUsername: 'bob', newIdentityKey: 'bob_key_v2', newFingerprint: 'fp_v2',
      );
      expect(changed, isTrue);

      final info = await peerStore.getPeerInfo('bob');
      expect(info!.keyChanged, isTrue);
    });

    test('acceptNewKey updates pin and resets key_changed', () async {
      await peerStore.hasKeyChanged(
        peerUsername: 'bob', newIdentityKey: 'bob_key_v1', newFingerprint: 'fp_v1',
      );
      await peerStore.hasKeyChanged(
        peerUsername: 'bob', newIdentityKey: 'bob_key_v2', newFingerprint: 'fp_v2',
      );

      await peerStore.acceptNewKey('bob');

      final info = await peerStore.getPeerInfo('bob');
      expect(info!.keyChanged, isFalse);
      expect(info.pinnedIdentityKey, equals('bob_key_v2'));
    });

    test('multiple peers tracked independently', () async {
      await peerStore.hasKeyChanged(
        peerUsername: 'alice', newIdentityKey: 'alice_key', newFingerprint: 'fp_a',
      );
      await peerStore.hasKeyChanged(
        peerUsername: 'bob', newIdentityKey: 'bob_key', newFingerprint: 'fp_b',
      );

      // Alice changes
      final aliceChanged = await peerStore.hasKeyChanged(
        peerUsername: 'alice', newIdentityKey: 'alice_key_new', newFingerprint: 'fp_a2',
      );
      expect(aliceChanged, isTrue);

      // Bob unchanged
      final bobChanged = await peerStore.hasKeyChanged(
        peerUsername: 'bob', newIdentityKey: 'bob_key', newFingerprint: 'fp_b',
      );
      expect(bobChanged, isFalse);
    });

    test('clearAll removes all peer data', () async {
      await peerStore.hasKeyChanged(
        peerUsername: 'bob', newIdentityKey: 'k', newFingerprint: 'fp',
      );
      await peerStore.clearAll();
      final all = await peerStore.loadAll();
      expect(all, isEmpty);
    });
  });
}
