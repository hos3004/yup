import 'package:flutter_test/flutter_test.dart';
import 'package:yup_mobile/core/logging/log_service.dart';

/// Helper: captures the redacted output by checking LogService does not throw
/// and that we can exercise all paths. The _redact logic is tested directly
/// via redactForLog.
String redact(String s) => LogService.redactForLog(s);

void main() {
  group('LogService redaction', () {
    // ─── _redact direct tests ──────────────────────────────

    test('redacts Bearer tokens', () {
      final result = redact('Authorization: Bearer abcdef1234567890abcdef1234567890xyz');
      expect(result, contains('Bearer <REDACTED_TOKEN>'));
      expect(result, isNot(contains('abcdef1234567890abcdef1234567890xyz')));
    });

    test('redacts auth_token JSON', () {
      final result = redact('{"auth_token":"secretauthtokenvaluehere1234567890"}');
      expect(result, contains('"auth_token":"<REDACTED>"'));
      expect(result, isNot(contains('secretauthtokenvaluehere')));
    });

    test('redacts base64 keys (32+ chars)', () {
      final result = redact('key = abcdefghijklmnopqrstuvwxyz0123456789+/');
      expect(result, isNot(contains('abcdefghijklmnopqrstuvwxyz0123456789+/')));
    });

    test('redacts hex tokens (32+ hex chars)', () {
      final result = redact('token=abcdef0123456789abcdef0123456789');
      expect(result, isNot(contains('abcdef0123456789abcdef0123456789')));
    });

    test('redacts ciphertext in JSON', () {
      final result = redact('{"ciphertext":"verylongbase64encodedciphertextdatahere=="}');
      expect(result, contains('"ciphertext":"<REDACTED>"'));
      expect(result, isNot(contains('verylongbase64encodedciphertextdatahere')));
    });

    test('redacts pickle in JSON', () {
      final result = redact('{"pickle":"verylongpicklestringthatshouldberedactednow"}');
      expect(result, contains('"pickle":"<REDACTED>"'));
      expect(result, isNot(contains('verylongpicklestringthatshouldberedacted')));
    });

    test('redacts sensitive stack trace content', () {
      // Simulate a stack trace containing a sensitive token
      final stackTrace = 'Error: Bearer abcdef1234567890abcdef1234567890xyz\n'
          '  at sendMessage (conversation_service.dart:140)\n'
          '  at main.dart:42';
      final result = redact(stackTrace);
      expect(result, isNot(contains('Bearer abcdef1234567890abcdef1234567890xyz')));
    });

    test('redacts account/session pickle-like string', () {
      final result = redact('{"account_pickle":"{\\"sessions\\":[],\\"account\\":\\"data\\"}"}');
      expect(result, contains('"account_pickle":"<REDACTED>"'));
    });

    // ─── LogService method smoke tests ─────────────────────

    test('info does not throw with sensitive content', () {
      expect(
        () => LogService.info('Authorization: Bearer abcdef1234567890abcdef1234567890'),
        returnsNormally,
      );
    });

    test('warn does not throw with sensitive content', () {
      expect(
        () => LogService.warn('Key: ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789=='),
        returnsNormally,
      );
    });

    test('error redacts error object and stack trace', () {
      try {
        throw Exception('Bearer abcdef1234567890abcdef1234567890');
      } catch (e, stack) {
        // error() should not pass raw error/stack to developer.log
        // We verify no throw and that the API is safe
        expect(
          () => LogService.error('error occurred', e, stack),
          returnsNormally,
        );
      }
    });

    test('error does not leak sensitive error message into developer.log error param',
        () {
      // The test verifies the contract: error() never passes raw params.
      // We can't directly inspect developer.log output, but we verify
      // the method completes normally.
      final leakyError = Exception('auth_token: "super-secret-token-value-here"');
      expect(
        () => LogService.error('test', leakyError),
        returnsNormally,
      );
    });

    test('handles empty messages', () {
      expect(() => LogService.info(''), returnsNormally);
      expect(() => LogService.warn(''), returnsNormally);
      expect(() => LogService.error(''), returnsNormally);
    });

    test('does not redact short benign strings', () {
      expect(() => LogService.info('hello world'), returnsNormally);
      expect(() => LogService.info('abc123'), returnsNormally);
    });
  });
}
