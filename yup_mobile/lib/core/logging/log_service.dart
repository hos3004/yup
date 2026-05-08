import 'dart:developer' as developer;

class LogService {
  static final RegExp _keyPattern = RegExp(r'[A-Za-z0-9+/]{32,}={0,2}');
  static final RegExp _tokenPattern = RegExp(r'[a-f0-9]{64}');

  static String _redact(String message) {
    var result = message;
    result = result.replaceAllMapped(_keyPattern, (_) => '<REDACTED_KEY>');
    result = result.replaceAllMapped(_tokenPattern, (_) => '<REDACTED_TOKEN>');
    return result;
  }

  static void info(String message) {
    developer.log(_redact(message), name: 'YUP', level: 800);
  }

  static void warn(String message) {
    developer.log(_redact(message), name: 'YUP', level: 900);
  }

  static void error(String message, [Object? error, StackTrace? stack]) {
    developer.log(
      _redact(message),
      name: 'YUP',
      level: 1000,
      error: error,
      stackTrace: stack,
    );
  }
}
