import 'dart:async';
import '../../../core/networking/api_client.dart';
import '../../../core/storage/message_dao.dart';
import '../../key_management/domain/crypto_service.dart';
import '../data/peer_key_store.dart';
import '../data/session_store.dart';

class MessageItem {
  final String id;
  final String sender;
  final String text;
  final bool isOutgoing;
  final String status;
  final DateTime createdAt;

  const MessageItem({
    required this.id,
    required this.sender,
    required this.text,
    required this.isOutgoing,
    this.status = 'pending',
    required this.createdAt,
  });

  MessageItem copyWith({String? status}) {
    return MessageItem(
      id: id,
      sender: sender,
      text: text,
      isOutgoing: isOutgoing,
      status: status ?? this.status,
      createdAt: createdAt,
    );
  }
}

class ConversationService {
  final ApiClient _api;
  final CryptoService _crypto;
  final SessionStore _sessionStore;
  final PeerKeyStore _peerKeyStore;
  final MessageDao _messageDao;
  final String _username;
  final List<MessageItem> _messages = [];
  Timer? _pollTimer;
  String? _sessionId;
  String? _myCurveKey;
  String? _recipientCurveKey;
  String? _recipientOtk;

  final _messageController = StreamController<List<MessageItem>>.broadcast();
  final _errorController = StreamController<String>.broadcast();
  final _keyChangedController = StreamController<String>.broadcast();

  Stream<List<MessageItem>> get messages => _messageController.stream;
  Stream<String> get errors => _errorController.stream;
  Stream<String> get keyChangedEvents => _keyChangedController.stream;

  List<MessageItem> get currentMessages => List.unmodifiable(_messages);

  ConversationService({
    required ApiClient apiClient,
    required CryptoService cryptoService,
    required SessionStore sessionStore,
    required PeerKeyStore peerKeyStore,
    required MessageDao messageDao,
    required String username,
  })  : _api = apiClient,
        _crypto = cryptoService,
        _sessionStore = sessionStore,
        _peerKeyStore = peerKeyStore,
        _messageDao = messageDao,
        _username = username;

  String? get recipientCurveKey => _recipientCurveKey;

  void initialize(String myCurveKey) {
    _myCurveKey = myCurveKey;
  }

  Future<String> startConversation(String recipientUsername) async {
    try {
      final theirBundle = await _api.getKeys(recipientUsername);
      final theirKey = theirBundle['curve_key'] as String;
      final theirOtks = theirBundle['one_time_keys'] as List<dynamic>;

      _recipientCurveKey = theirKey;

      // Check for key change
      final fingerprint = _crypto.getFingerprint(theirKey);
      final changed = await _peerKeyStore.hasKeyChanged(
        peerUsername: recipientUsername,
        newIdentityKey: theirKey,
        newFingerprint: fingerprint,
      );
      if (changed) {
        _keyChangedController.add(recipientUsername);
        // Do not proceed with session creation — caller must handle warning
        throw KeyChangedException(recipientUsername);
      }

      // Load conversation history from local DB
      final history = await _messageDao.getConversation(
        username: _username,
        peerCurveKey: theirKey,
      );
      if (history.isNotEmpty) {
        _messages.clear();
        _messages.addAll(history);
        _messageController.add(List.from(_messages));
      }

      // Check for existing session
      final existingSessionId = _sessionStore.getSessionId(theirKey);
      if (existingSessionId != null) {
        _sessionId = existingSessionId;
        return existingSessionId;
      }

      if (theirOtks.isEmpty) {
        throw Exception('$recipientUsername has no one-time keys');
      }

      _recipientOtk = theirOtks.first as String;

      final sessionInfo = _crypto.createOutboundSession(_recipientCurveKey!, _recipientOtk!);
      _sessionId = sessionInfo['session_id'] as String;

      await _sessionStore.addSession(_sessionId!, _recipientCurveKey!);

      return _sessionId!;
    } on KeyChangedException {
      rethrow;
    } catch (e) {
      _errorController.add('Session error: $e');
      rethrow;
    }
  }

  Future<void> sendMessage(String recipientUsername, String text) async {
    if (_sessionId == null || _recipientCurveKey == null) {
      await startConversation(recipientUsername);
    }

    try {
      final encrypted = _crypto.encryptMessage(_sessionId!, text);
      final ct = encrypted['ciphertext'] as String;
      final msgType = encrypted['message_type'] as int;

      final env = await _api.sendMessage(
        recipient: recipientUsername,
        ciphertext: ct,
        messageType: msgType,
        senderKey: _myCurveKey!,
      );

      final msg = MessageItem(
        id: env['id'] as String,
        sender: _username,
        text: text,
        isOutgoing: true,
        status: 'pending',
        createdAt: DateTime.now(),
      );

      _messages.add(msg);
      _messageController.add(List.from(_messages));

      await _messageDao.insertMessage(msg, _username, _recipientCurveKey!);
    } catch (e) {
      _errorController.add('Send error: $e');
    }
  }

  Future<void> pollIncoming() async {
    try {
      final envs = await _api.getMessages();
      for (final env in envs) {
        final envMap = env as Map<String, dynamic>;
        final msgId = envMap['id'] as String;
        final sender = envMap['sender_username'] as String;
        final ct = envMap['ciphertext'] as String;
        final msgType = envMap['message_type'] as int;
        final senderKey = envMap['sender_curve_key'] as String? ?? '';
        final status = envMap['status'] as String? ?? 'delivered';

        if (_messages.any((m) => m.id == msgId)) continue;

        String plaintext;
        final existingSessionId = _sessionStore.getSessionId(senderKey);
        if (existingSessionId != null) {
          plaintext = _crypto.decryptMessage(existingSessionId, ct, msgType);
          _sessionId ??= existingSessionId;
        } else {
          // Returns {session_id, plaintext}
          final inboundResult = _crypto.createInboundSession(senderKey, ct);
          final inboundSessionId = inboundResult['session_id'] as String;
          plaintext = inboundResult['plaintext'] as String;
          _sessionId = inboundSessionId;
          // Store the actual inbound session keyed by sender's curve key
          await _sessionStore.addSession(inboundSessionId, senderKey);
        }

        final msg = MessageItem(
          id: msgId,
          sender: sender,
          text: plaintext,
          isOutgoing: false,
          status: status,
          createdAt: DateTime.now(),
        );

        _messages.add(msg);
        _messageController.add(List.from(_messages));

        if (_recipientCurveKey != null) {
          await _messageDao.insertMessage(msg, _username, _recipientCurveKey!);
        }

        try {
          await _api.ackMessage(msgId);
        } catch (_) {}
      }
    } catch (_) {}
  }

  Future<void> pollSentStatus() async {
    try {
      final sentEnvs = await _api.getSentMessages();
      final updatedStatuses = <String, String>{};
      for (final env in sentEnvs) {
        final envMap = env as Map<String, dynamic>;
        final id = envMap['id'] as String;
        final status = envMap['status'] as String;
        updatedStatuses[id] = status;
      }

      bool changed = false;
      for (int i = 0; i < _messages.length; i++) {
        if (_messages[i].isOutgoing) {
          final newStatus = updatedStatuses[_messages[i].id];
          if (newStatus != null && newStatus != _messages[i].status) {
            _messages[i] = _messages[i].copyWith(status: newStatus);
            await _messageDao.updateStatus(_messages[i].id, newStatus);
            changed = true;
          }
        }
      }
      if (changed) {
        _messageController.add(List.from(_messages));
      }
    } catch (_) {}
  }

  void startPolling() {
    _pollTimer?.cancel();
    pollIncoming();
    pollSentStatus();
    _pollTimer = Timer.periodic(const Duration(seconds: 3), (_) {
      pollIncoming();
      pollSentStatus();
    });
  }

  void stopPolling() {
    _pollTimer?.cancel();
    _pollTimer = null;
  }

  void dispose() {
    stopPolling();
    _messageController.close();
    _errorController.close();
    _keyChangedController.close();
  }
}

class KeyChangedException implements Exception {
  final String peerUsername;
  KeyChangedException(this.peerUsername);

  @override
  String toString() => 'Key changed for $peerUsername';
}
