import 'dart:convert';
import 'dart:io';
import 'dart:math';
import 'package:sqflite_sqlcipher/sqflite.dart';
import '../../core/secure_storage/secure_storage_service.dart';

class LocalDatabase {
  Database? _db;
  final SecureStorageService _storage;

  LocalDatabase(this._storage);

  static String _dbKey() => 'db_passphrase';

  Future<Database> get database async {
    if (_db != null) return _db!;
    _db = await _init();
    return _db!;
  }

  Future<String> _getOrCreatePassphrase() async {
    final existing = await _storage.readRaw(_dbKey());
    if (existing != null) return existing;
    final bytes = List<int>.generate(32, (_) => Random.secure().nextInt(256));
    final passphrase = base64Encode(bytes);
    await _storage.writeRaw(_dbKey(), passphrase);
    return passphrase;
  }

  Future<Database> _init() async {
    final passphrase = await _getOrCreatePassphrase();
    final dbPath = await getDatabasesPath();
    final path = '$dbPath/yup_messages.db';

    return await openDatabase(
      path,
      password: passphrase,
      version: 1,
      onCreate: (db, version) async {
        await db.execute('''
          CREATE TABLE messages (
            id TEXT PRIMARY KEY,
            sender TEXT NOT NULL,
            recipient TEXT NOT NULL,
            peer_curve_key TEXT NOT NULL,
            text TEXT NOT NULL,
            is_outgoing INTEGER NOT NULL DEFAULT 0,
            status TEXT NOT NULL DEFAULT 'pending',
            created_at INTEGER NOT NULL
          )
        ''');
        await db.execute('''
          CREATE INDEX idx_messages_peer
          ON messages(peer_curve_key, created_at)
        ''');
      },
    );
  }

  Future<void> close() async {
    await _db?.close();
    _db = null;
  }

  /// Deletes the encrypted database file. Call close() first.
  Future<void> deleteDatabaseFile() async {
    await close();
    final dbPath = await getDatabasesPath();
    final path = '$dbPath/yup_messages.db';
    final file = File(path);
    if (await file.exists()) {
      await file.delete();
    }
  }
}
