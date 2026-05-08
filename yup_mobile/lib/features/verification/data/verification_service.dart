import 'dart:convert';
import '../../../core/secure_storage/secure_storage_service.dart';

class VerificationService {
  final SecureStorageService _storage;
  final String _username;

  VerificationService(this._storage, this._username);

  static String _storageKey(String username) => 'verified_contacts:$username';

  Future<Set<String>> getVerifiedContacts() async {
    final raw = await _storage.readRaw(_storageKey(_username));
    if (raw == null || raw.isEmpty) return {};
    try {
      final list = jsonDecode(raw) as List<dynamic>;
      return list.map((e) => e as String).toSet();
    } catch (_) {
      return {};
    }
  }

  Future<bool> isVerified(String peerCurveKey) async {
    final verified = await getVerifiedContacts();
    return verified.contains(peerCurveKey);
  }

  Future<void> markVerified(String peerCurveKey) async {
    final verified = await getVerifiedContacts();
    verified.add(peerCurveKey);
    await _storage.writeRaw(
      _storageKey(_username),
      jsonEncode(verified.toList()),
    );
  }

  Future<void> removeVerification(String peerCurveKey) async {
    final verified = await getVerifiedContacts();
    verified.remove(peerCurveKey);
    await _storage.writeRaw(
      _storageKey(_username),
      jsonEncode(verified.toList()),
    );
  }

  Future<void> clearAll() async {
    await _storage.deleteRaw(_storageKey(_username));
  }
}
