import 'package:flutter_test/flutter_test.dart';

/// Pure-Dart model of the SecureStorageService clear data behavior.
class TestSecureStorage {
  final Map<String, String> _store = {};

  Future<void> write(String key, String value) async => _store[key] = value;
  Future<String?> read(String key) async => _store[key];
  Future<void> delete(String key) async => _store.remove(key);

  Future<void> clearAllUserData(String username) async {
    await Future.wait([
      delete('auth_token:$username'),
      delete('account_pickle:$username'),
      delete('identity_curve25519:$username'),
      delete('identity_ed25519:$username'),
      delete('sessions:$username'),
      delete('peer_keys:$username'),
      delete('verified_contacts:$username'),
      delete('db_passphrase'),
      delete('active_username'),
    ]);
  }
}

/// Simulates LocalDatabase.deleteDatabaseFile
class TestLocalDatabase {
  bool databaseExists = true;
  bool _closed = false;

  Future<void> close() async {
    _closed = true;
  }

  Future<void> deleteDatabaseFile() async {
    await close();
    // Simulate file deletion
    databaseExists = false;
  }

  bool get wasClosed => _closed;
}

void main() {
  group('Clear Local Data', () {
    late TestSecureStorage storage;

    setUp(() async {
      storage = TestSecureStorage();
      await storage.write('auth_token:alice', 'tok123');
      await storage.write('account_pickle:alice', 'pickle123');
      await storage.write('identity_curve25519:alice', 'curve123');
      await storage.write('identity_ed25519:alice', 'ed123');
      await storage.write('sessions:alice', 'session_data');
      await storage.write('peer_keys:alice', 'peer_data');
      await storage.write('verified_contacts:alice', 'verified_data');
      await storage.write('db_passphrase', 'passphrase123');
      await storage.write('active_username', 'alice');
    });

    test('clearAllUserData removes auth token', () async {
      await storage.clearAllUserData('alice');
      expect(await storage.read('auth_token:alice'), isNull);
    });

    test('clearAllUserData removes account pickle', () async {
      await storage.clearAllUserData('alice');
      expect(await storage.read('account_pickle:alice'), isNull);
    });

    test('clearAllUserData removes identity keys', () async {
      await storage.clearAllUserData('alice');
      expect(await storage.read('identity_curve25519:alice'), isNull);
      expect(await storage.read('identity_ed25519:alice'), isNull);
    });

    test('clearAllUserData removes DB passphrase', () async {
      await storage.clearAllUserData('alice');
      expect(await storage.read('db_passphrase'), isNull);
    });

    test('clearAllUserData removes active username', () async {
      await storage.clearAllUserData('alice');
      expect(await storage.read('active_username'), isNull);
    });

    test('clearAllUserData removes sessions and peer data', () async {
      await storage.clearAllUserData('alice');
      expect(await storage.read('sessions:alice'), isNull);
      expect(await storage.read('peer_keys:alice'), isNull);
      expect(await storage.read('verified_contacts:alice'), isNull);
    });

    test('clearAllUserData does not affect other users', () async {
      await storage.write('auth_token:bob', 'bob_token');
      await storage.clearAllUserData('alice');
      expect(await storage.read('auth_token:bob'), equals('bob_token'));
    });

    test('deleteDatabaseFile closes DB and marks file deleted', () async {
      final db = TestLocalDatabase();
      expect(db.databaseExists, isTrue);
      expect(db.wasClosed, isFalse);

      await db.deleteDatabaseFile();

      expect(db.wasClosed, isTrue);
      expect(db.databaseExists, isFalse);
    });
  });
}
