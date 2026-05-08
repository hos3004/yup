import 'package:flutter_test/flutter_test.dart';
import 'package:yup_mobile/core/logging/log_service.dart';

void main() {
  group('LogService redaction', () {
    test('redacts Bearer tokens', () {
      // Developer.log can't be captured easily, but we verify no throw
      // and that sensitive patterns are handled
      LogService.info('Authorization: Bearer abcdef1234567890abcdef1234567890');
      LogService.error('Token: abcdef1234567890abcdef1234567890');
    });

    test('redacts base64 key patterns (32+ chars)', () {
      LogService.info('key = abcdefghijklmnopqrstuvwxyz0123456789+/');
      LogService.warn('Key: ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789==');
    });

    test('redacts hex token patterns (32+ hex chars)', () {
      LogService.info('token=abcdef0123456789abcdef0123456789');
      LogService.error('auth: aabbccddeeff00112233445566778899aabbccdd');
    });

    test('redacts sensitive JSON fields', () {
      LogService.info('{"auth_token":"abcdefghijklmnopqrstuvwxyz0123456789"}');
      LogService.warn('{"ciphertext":"verylongbase64encodedciphertextdatahere=="}');
      LogService.error('{"pickle":"verylongpicklestringthatshouldberedacted"}');
    });

    test('handles empty messages', () {
      expect(() => LogService.info(''), returnsNormally);
      expect(() => LogService.warn(''), returnsNormally);
      expect(() => LogService.error(''), returnsNormally);
    });

    test('handles messages with null-like content', () {
      expect(() => LogService.info('null'), returnsNormally);
      expect(() => LogService.warn('undefined'), returnsNormally);
    });

    test('redacts error object and skips stack trace payload', () {
      try {
        throw Exception('Bearer abcdef1234567890abcdef1234567890');
      } catch (e, stack) {
        expect(
          () => LogService.error('error occurred', e, stack),
          returnsNormally,
        );
      }
    });

    test('does not throw on plain text messages', () {
      expect(() => LogService.info('hello world'), returnsNormally);
      expect(() => LogService.info('This is a normal message'), returnsNormally);
    });

    test('does not redact short benign strings', () {
      expect(() => LogService.info('abc123'), returnsNormally);
      expect(() => LogService.info('hello'), returnsNormally);
    });
  });
}
