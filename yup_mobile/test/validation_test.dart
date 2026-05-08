import 'package:flutter_test/flutter_test.dart';

bool isValidUsername(String username) {
  if (username.length < 3 || username.length > 32) return false;
  for (final c in username.codeUnits) {
    final char = String.fromCharCode(c);
    if (!RegExp(r'^[a-zA-Z0-9_-]$').hasMatch(char)) return false;
  }
  return true;
}

void main() {
  group('Username validation', () {
    test('rejects empty string', () {
      expect(isValidUsername(''), false);
    });

    test('rejects too short (2 chars)', () {
      expect(isValidUsername('ab'), false);
    });

    test('accepts minimum length (3 chars)', () {
      expect(isValidUsername('abc'), true);
    });

    test('accepts alphanumeric with underscore and hyphen', () {
      expect(isValidUsername('user_name-123'), true);
    });

    test('rejects spaces', () {
      expect(isValidUsername('user name'), false);
    });

    test('rejects special characters', () {
      expect(isValidUsername('user@name!'), false);
    });

    test('accepts maximum length (32 chars)', () {
      expect(isValidUsername('a' * 32), true);
    });

    test('rejects too long (33 chars)', () {
      expect(isValidUsername('a' * 33), false);
    });

    test('rejects Turkish characters', () {
      expect(isValidUsername('kullanıcı'), false);
    });
  });
}
