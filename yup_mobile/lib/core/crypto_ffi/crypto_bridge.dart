import 'dart:convert';
import 'dart:ffi';
import 'dart:io';
import 'package:ffi/ffi.dart';

typedef YupGenerateAccountC = Pointer<Utf8> Function();
typedef YupGenerateAccountDart = Pointer<Utf8> Function();

typedef YupGetIdentityKeysC = Pointer<Utf8> Function();
typedef YupGetIdentityKeysDart = Pointer<Utf8> Function();

typedef YupGenerateOneTimeKeysC = Pointer<Utf8> Function(IntPtr count);
typedef YupGenerateOneTimeKeysDart = Pointer<Utf8> Function(int count);

typedef YupSignMessageC = Pointer<Utf8> Function(Pointer<Utf8> msg);
typedef YupSignMessageDart = Pointer<Utf8> Function(Pointer<Utf8> msg);

typedef YupCreateOutboundSessionC = Pointer<Utf8> Function(
  Pointer<Utf8> theirIdentityKey,
  Pointer<Utf8> theirOneTimeKey,
);
typedef YupCreateOutboundSessionDart = Pointer<Utf8> Function(
  Pointer<Utf8> theirIdentityKey,
  Pointer<Utf8> theirOneTimeKey,
);

typedef YupEncryptMessageC = Pointer<Utf8> Function(
  Pointer<Utf8> sessionId,
  Pointer<Utf8> plaintext,
);
typedef YupEncryptMessageDart = Pointer<Utf8> Function(
  Pointer<Utf8> sessionId,
  Pointer<Utf8> plaintext,
);

typedef YupCreateInboundSessionC = Pointer<Utf8> Function(
  Pointer<Utf8> theirIdentityKey,
  Pointer<Utf8> ciphertextB64,
);
typedef YupCreateInboundSessionDart = Pointer<Utf8> Function(
  Pointer<Utf8> theirIdentityKey,
  Pointer<Utf8> ciphertextB64,
);

typedef YupDecryptMessageC = Pointer<Utf8> Function(
  Pointer<Utf8> sessionId,
  Pointer<Utf8> ciphertextB64,
  IntPtr messageType,
);
typedef YupDecryptMessageDart = Pointer<Utf8> Function(
  Pointer<Utf8> sessionId,
  Pointer<Utf8> ciphertextB64,
  int messageType,
);

typedef YupGetFingerprintC = Pointer<Utf8> Function(Pointer<Utf8> theirIdentityKey);
typedef YupGetFingerprintDart = Pointer<Utf8> Function(Pointer<Utf8> theirIdentityKey);

typedef YupPickleAccountC = Pointer<Utf8> Function();
typedef YupPickleAccountDart = Pointer<Utf8> Function();

typedef YupUnpickleAccountC = Pointer<Utf8> Function(Pointer<Utf8> pickle);
typedef YupUnpickleAccountDart = Pointer<Utf8> Function(Pointer<Utf8> pickle);

typedef YupPickleSessionC = Pointer<Utf8> Function(Pointer<Utf8> sessionId);
typedef YupPickleSessionDart = Pointer<Utf8> Function(Pointer<Utf8> sessionId);

typedef YupUnpickleSessionC = Pointer<Utf8> Function(
  Pointer<Utf8> sessionId,
  Pointer<Utf8> pickle,
);
typedef YupUnpickleSessionDart = Pointer<Utf8> Function(
  Pointer<Utf8> sessionId,
  Pointer<Utf8> pickle,
);

typedef YupFreeStringC = Void Function(Pointer<Utf8> s);
typedef YupFreeStringDart = void Function(Pointer<Utf8> s);

class CryptoBridge {
  late final DynamicLibrary _lib;
  late final YupGenerateAccountDart _generateAccount;
  late final YupGetIdentityKeysDart _getIdentityKeys;
  late final YupGenerateOneTimeKeysDart _generateOneTimeKeys;
  late final YupSignMessageDart _signMessage;
  late final YupCreateOutboundSessionDart _createOutboundSession;
  late final YupEncryptMessageDart _encryptMessage;
  late final YupCreateInboundSessionDart _createInboundSession;
  late final YupDecryptMessageDart _decryptMessage;
  late final YupGetFingerprintDart _getFingerprint;
  late final YupPickleAccountDart _pickleAccount;
  late final YupUnpickleAccountDart _unpickleAccount;
  late final YupPickleSessionDart _pickleSession;
  late final YupUnpickleSessionDart _unpickleSession;
  late final YupFreeStringDart _freeString;

  bool _initialized = false;

  void initialize() {
    if (_initialized) return;
    if (Platform.isAndroid) {
      _lib = DynamicLibrary.open('libyup_crypto.so');
    } else if (Platform.isIOS) {
      _lib = DynamicLibrary.process();
    } else if (Platform.isWindows) {
      _lib = DynamicLibrary.open('yup_crypto.dll');
    } else if (Platform.isMacOS) {
      _lib = DynamicLibrary.open('libyup_crypto.dylib');
    } else if (Platform.isLinux) {
      _lib = DynamicLibrary.open('libyup_crypto.so');
    } else {
      throw UnsupportedError('Unsupported platform');
    }

    _generateAccount = _lib
        .lookupFunction<YupGenerateAccountC, YupGenerateAccountDart>('yup_generate_account');
    _getIdentityKeys = _lib
        .lookupFunction<YupGetIdentityKeysC, YupGetIdentityKeysDart>('yup_get_identity_keys');
    _generateOneTimeKeys = _lib
        .lookupFunction<YupGenerateOneTimeKeysC, YupGenerateOneTimeKeysDart>('yup_generate_one_time_keys');
    _signMessage = _lib
        .lookupFunction<YupSignMessageC, YupSignMessageDart>('yup_sign_message');
    _createOutboundSession = _lib
        .lookupFunction<YupCreateOutboundSessionC, YupCreateOutboundSessionDart>('yup_create_outbound_session');
    _encryptMessage = _lib
        .lookupFunction<YupEncryptMessageC, YupEncryptMessageDart>('yup_encrypt_message');
    _createInboundSession = _lib
        .lookupFunction<YupCreateInboundSessionC, YupCreateInboundSessionDart>('yup_create_inbound_session');
    _decryptMessage = _lib
        .lookupFunction<YupDecryptMessageC, YupDecryptMessageDart>('yup_decrypt_message');
    _getFingerprint = _lib
        .lookupFunction<YupGetFingerprintC, YupGetFingerprintDart>('yup_get_fingerprint');
    _pickleAccount = _lib
        .lookupFunction<YupPickleAccountC, YupPickleAccountDart>('yup_pickle_account');
    _unpickleAccount = _lib
        .lookupFunction<YupUnpickleAccountC, YupUnpickleAccountDart>('yup_unpickle_account');
    _pickleSession = _lib
        .lookupFunction<YupPickleSessionC, YupPickleSessionDart>('yup_pickle_session');
    _unpickleSession = _lib
        .lookupFunction<YupUnpickleSessionC, YupUnpickleSessionDart>('yup_unpickle_session');
    _freeString = _lib
        .lookupFunction<YupFreeStringC, YupFreeStringDart>('yup_free_string');

    _initialized = true;
  }

  String _call(Pointer<Utf8> Function() fn) {
    final ptr = fn();
    final result = ptr.toDartString();
    _freeString(ptr);
    return result;
  }

  String _call1<T>(Pointer<Utf8> Function(T) fn, T arg) {
    final ptr = fn(arg);
    final result = ptr.toDartString();
    _freeString(ptr);
    return result;
  }

  String _call2<T1, T2>(Pointer<Utf8> Function(T1, T2) fn, T1 arg1, T2 arg2) {
    final ptr = fn(arg1, arg2);
    final result = ptr.toDartString();
    _freeString(ptr);
    return result;
  }

  String _call3<T1, T2, T3>(
      Pointer<Utf8> Function(T1, T2, T3) fn, T1 arg1, T2 arg2, T3 arg3) {
    final ptr = fn(arg1, arg2, arg3);
    final result = ptr.toDartString();
    _freeString(ptr);
    return result;
  }

  Map<String, dynamic> _parseResult(String raw) {
    if (raw.startsWith('ERR:')) {
      throw Exception(raw.substring(4));
    }
    final json = raw.substring(3);
    if (json.isEmpty) return {};
    return jsonDecode(json) as Map<String, dynamic>;
  }

  String _parseResultString(String raw) {
    if (raw.startsWith('ERR:')) {
      throw Exception(raw.substring(4));
    }
    return raw.substring(3);
  }

  List<dynamic> _parseResultList(String raw) {
    if (raw.startsWith('ERR:')) {
      throw Exception(raw.substring(4));
    }
    return jsonDecode(raw.substring(3)) as List<dynamic>;
  }

  Map<String, dynamic> generateAccount() {
    final raw = _call(_generateAccount);
    return _parseResult(raw);
  }

  Map<String, dynamic> getIdentityKeys() {
    final raw = _call(_getIdentityKeys);
    return _parseResult(raw);
  }

  List<String> generateOneTimeKeys(int count) {
    final ptr = _generateOneTimeKeys(count);
    final raw = ptr.toDartString();
    _freeString(ptr);
    return _parseResultList(raw).cast<String>();
  }

  String signMessage(String message) {
    final msgPtr = message.toNativeUtf8();
    final raw = _call1(_signMessage, msgPtr);
    calloc.free(msgPtr);
    return _parseResultString(raw);
  }

  Map<String, dynamic> createOutboundSession(String theirIdentityKey, String theirOneTimeKey) {
    final idPtr = theirIdentityKey.toNativeUtf8();
    final otkPtr = theirOneTimeKey.toNativeUtf8();
    final raw = _call2(_createOutboundSession, idPtr, otkPtr);
    calloc.free(idPtr);
    calloc.free(otkPtr);
    return _parseResult(raw);
  }

  Map<String, dynamic> encryptMessage(String sessionId, String plaintext) {
    final sidPtr = sessionId.toNativeUtf8();
    final ptPtr = plaintext.toNativeUtf8();
    final raw = _call2(_encryptMessage, sidPtr, ptPtr);
    calloc.free(sidPtr);
    calloc.free(ptPtr);
    return _parseResult(raw);
  }

  Map<String, dynamic> createInboundSession(String theirIdentityKey, String ciphertextB64) {
    final idPtr = theirIdentityKey.toNativeUtf8();
    final ctPtr = ciphertextB64.toNativeUtf8();
    final raw = _call2(_createInboundSession, idPtr, ctPtr);
    calloc.free(idPtr);
    calloc.free(ctPtr);
    return _parseResult(raw);
  }

  String decryptMessage(String sessionId, String ciphertextB64, int messageType) {
    final sidPtr = sessionId.toNativeUtf8();
    final ctPtr = ciphertextB64.toNativeUtf8();
    final raw = _call3(_decryptMessage, sidPtr, ctPtr, messageType);
    calloc.free(sidPtr);
    calloc.free(ctPtr);
    return _parseResultString(raw);
  }

  String getFingerprint(String theirIdentityKey) {
    final idPtr = theirIdentityKey.toNativeUtf8();
    final raw = _call1(_getFingerprint, idPtr);
    calloc.free(idPtr);
    return _parseResultString(raw);
  }

  String pickleAccount() {
    final raw = _call(_pickleAccount);
    return _parseResultString(raw);
  }

  void unpickleAccount(String pickle) {
    final pPtr = pickle.toNativeUtf8();
    final raw = _call1(_unpickleAccount, pPtr);
    calloc.free(pPtr);
    _parseResult(raw); // validates + returns identity keys (discarded)
  }

  String pickleSession(String sessionId) {
    final sidPtr = sessionId.toNativeUtf8();
    final raw = _call1(_pickleSession, sidPtr);
    calloc.free(sidPtr);
    return _parseResultString(raw);
  }

  void unpickleSession(String sessionId, String pickle) {
    final sidPtr = sessionId.toNativeUtf8();
    final pPtr = pickle.toNativeUtf8();
    final raw = _call2(_unpickleSession, sidPtr, pPtr);
    calloc.free(sidPtr);
    calloc.free(pPtr);
    _parseResult(raw); // validates
  }
}
