import 'dart:developer' as developer;

class LogService {
  // Redact Bearer tokens
  static final RegExp _bearerPattern = RegExp(r'Bearer\s+[A-Za-z0-9+/=_-]{10,}', caseSensitive: false);
  // Redact base64-like keys (32+ chars)
  static final RegExp _keyPattern = RegExp(r'[A-Za-z0-9+/]{32,}={0,2}');
  // Redact hex tokens (32+ consecutive hex chars)
  static final RegExp _hexTokenPattern = RegExp(r'[a-f0-9]{32,}', caseSensitive: false);
  // Redact JSON fields containing sensitive data
  static final RegExp _sensitiveJsonField = RegExp(
    r'"(account_pickle|token|auth_token|private|pickle|ciphertext|session_key)"\s*:\s*"(?:[^"\\]|\\.){8,}"',
    caseSensitive: false,
  );

  /// Exposed for testing — applies the same redaction logic.
  static String redactForLog(String message) => _redact(message);

  static String _redact(String message) {
    var result = message;
    // Apply redaction in specific order
    result = result.replaceAllMapped(_bearerPattern, (_) => 'Bearer <REDACTED_TOKEN>');
    result = result.replaceAllMapped(_sensitiveJsonField, (m) {
      final field = m.group(1) ?? 'field';
      return '"$field":"<REDACTED>"';
    });
    result = result.replaceAllMapped(_keyPattern, (_) => '<REDACTED_KEY>');
    result = result.replaceAllMapped(_hexTokenPattern, (_) => '<REDACTED_HEX>');
    return result;
  }

  static void info(String message) {
    developer.log(_redact(message), name: 'YUP', level: 800);
  }

  static void warn(String message) {
    developer.log(_redact(message), name: 'YUP', level: 900);
  }

  static void error(String message, [Object? error, StackTrace? stack]) {
    final redactedMsg = _redact(message);
    final sb = StringBuffer(redactedMsg);
    if (error != null) {
      sb.write(' | error: ');
      sb.write(_redact(error.toString()));
    }
    if (stack != null) {
      sb.write(' | stack: ');
      sb.write(_redact(stack.toString()));
    }
    // Never pass raw error or stackTrace to developer.log — they may contain secrets.
    developer.log(sb.toString(), name: 'YUP', level: 1000);
  }
}
