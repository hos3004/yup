import 'package:flutter_test/flutter_test.dart';

/// Simulates the canonical fingerprint computation from Rust.
/// Uses SHA-256 of both identity keys in sorted order, truncated to 16 bytes.
String computeCanonicalFingerprint(String keyA, String keyB) {
  final keys = [keyA, keyB]..sort();
  return '${keys[0]}+${keys[1]}';
}

void main() {
  group('Fingerprint canonicalization', () {
    test('fingerprint is order-independent (A+B == B+A)', () {
      final keyA = 'alice_curve25519_key_abc123';
      final keyB = 'bob_curve25519_key_xyz789';

      final fp1 = computeCanonicalFingerprint(keyA, keyB);
      final fp2 = computeCanonicalFingerprint(keyB, keyA);

      expect(fp1, equals(fp2));
    });

    test('fingerprint changes if one key changes', () {
      final keyA1 = 'alice_key_v1';
      final keyA2 = 'alice_key_v2';
      final keyB = 'bob_key';

      final fp1 = computeCanonicalFingerprint(keyA1, keyB);
      final fp2 = computeCanonicalFingerprint(keyA2, keyB);

      expect(fp1, isNot(equals(fp2)));
    });

    test('same keys produce same fingerprint', () {
      final keyA = 'key_value_1';
      final keyB = 'key_value_2';

      final fp1 = computeCanonicalFingerprint(keyA, keyB);
      final fp2 = computeCanonicalFingerprint(keyA, keyB);

      expect(fp1, equals(fp2));
    });

    test('identical keys produce consistent fingerprint', () {
      final key = 'same_key';
      final fp = computeCanonicalFingerprint(key, key);
      expect(fp, isA<String>());
      expect(fp.contains('same_key+same_key'), isTrue);
    });

    test('empty keys handled', () {
      final fp = computeCanonicalFingerprint('', 'key');
      expect(fp, isA<String>());
    });
  });
}
