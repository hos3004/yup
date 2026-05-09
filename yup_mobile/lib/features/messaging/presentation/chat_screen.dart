import 'dart:async';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import '../../../core/networking/api_client.dart';
import '../../../core/push/push_service.dart';
import '../../../core/secure_storage/secure_storage_service.dart';
import '../../../core/storage/local_database.dart';
import '../../../core/storage/message_dao.dart';
import '../../key_management/domain/crypto_service.dart';
import '../../verification/data/verification_service.dart';
import '../../verification/presentation/verification_screen.dart';
import '../data/peer_key_store.dart';
import '../data/session_store.dart';
import '../domain/conversation_service.dart';

class ChatScreen extends StatefulWidget {
  final String username;
  final String myCurveKey;
  final CryptoService cryptoService;
  final ApiClient apiClient;
  final SecureStorageService secureStorage;
  final PushService? pushService;

  const ChatScreen({
    super.key,
    required this.username,
    required this.myCurveKey,
    required this.cryptoService,
    required this.apiClient,
    required this.secureStorage,
    this.pushService,
  });

  @override
  State<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends State<ChatScreen> {
  late final ConversationService _conversation;
  late final SessionStore _sessionStore;
  late final PeerKeyStore _peerKeyStore;
  late final VerificationService _verificationService;
  late final LocalDatabase _localDatabase;
  late final MessageDao _messageDao;
  final bool _isVerified = false;
  final _recipientController = TextEditingController();
  final _messageController = TextEditingController();
  final _messages = <MessageItem>[];
  StreamSubscription? _msgSub;
  StreamSubscription? _errSub;
  StreamSubscription? _keyChangedSub;
  String? _error;
  String? _activeRecipient;
  bool _connecting = false;

  @override
  void initState() {
    super.initState();
    _initServices();
  }

  Future<void> _initServices() async {
    _localDatabase = LocalDatabase(widget.secureStorage);
    _messageDao = MessageDao(_localDatabase);

    _sessionStore = SessionStore(
      widget.cryptoService,
      widget.secureStorage,
      widget.username,
    );
    await _sessionStore.load();

    _peerKeyStore = PeerKeyStore(widget.secureStorage, widget.username);

    _conversation = ConversationService(
      apiClient: widget.apiClient,
      cryptoService: widget.cryptoService,
      sessionStore: _sessionStore,
      peerKeyStore: _peerKeyStore,
      messageDao: _messageDao,
      username: widget.username,
      pushTriggers: widget.pushService?.pushTriggers,
    );
    _conversation.initialize(widget.myCurveKey);

    _verificationService = VerificationService(widget.secureStorage, widget.username);

    _msgSub = _conversation.messages.listen((msgs) {
      if (mounted) {
        setState(() {
          _messages.clear();
          _messages.addAll(msgs);
        });
      }
    });

    _errSub = _conversation.errors.listen((err) {
      if (mounted) setState(() => _error = err);
    });

    _keyChangedSub = _conversation.keyChangedEvents.listen((peerUsername) {
      if (mounted) _showKeyChangedWarning(peerUsername);
    });

    _conversation.startPolling();
  }

  Future<void> _showKeyChangedWarning(String peerUsername) async {
    final action = await showDialog<String>(
      context: context,
      barrierDismissible: false,
      builder: (ctx) => AlertDialog(
        title: const Icon(Icons.warning_amber_rounded, size: 48, color: Colors.orange),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Text(
              'Security key changed',
              style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 12),
            Text(
              'The security key for $peerUsername has changed. '
              'This may happen if they reinstalled the app or changed devices, '
              'but it could also indicate a security risk.',
              textAlign: TextAlign.center,
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, 'cancel'),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(ctx, 'verify'),
            child: const Text('View Verification'),
          ),
          ElevatedButton(
            onPressed: () => Navigator.pop(ctx, 'accept'),
            child: const Text('Accept New Key'),
          ),
        ],
      ),
    );

    if (action == 'cancel') {
      setState(() => _activeRecipient = null);
    } else if (action == 'verify') {
      final peerKey = _conversation.recipientCurveKey;
      if (peerKey != null && mounted) {
        Navigator.of(context).push(
          MaterialPageRoute(
            builder: (_) => VerificationScreen(
              peerUsername: peerUsername,
              peerCurveKey: peerKey,
              myUsername: widget.username,
              cryptoService: widget.cryptoService,
              verificationService: _verificationService,
            ),
          ),
        );
      }
    } else if (action == 'accept') {
      await _peerKeyStore.acceptNewKey(peerUsername);
      // Retry connecting with the new key
      _connect();
    }
  }

  Future<void> _connect() async {
    final recipient = _recipientController.text.trim();
    if (recipient.isEmpty) return;

    setState(() {
      _connecting = true;
      _error = null;
      _activeRecipient = recipient;
    });

    try {
      await _conversation.startConversation(recipient);
    } on KeyChangedException {
      // Warning dialog is shown via stream, no additional error needed
    } catch (e) {
      if (mounted) setState(() => _error = e.toString());
    } finally {
      if (mounted) setState(() => _connecting = false);
    }
  }

  Future<void> _send() async {
    final text = _messageController.text.trim();
    if (text.isEmpty || _activeRecipient == null) return;

    _messageController.clear();
    await _conversation.sendMessage(_activeRecipient!, text);
  }

  Future<void> _showVerification() async {
    final peerKey = _conversation.recipientCurveKey;
    if (peerKey == null) return;

    Navigator.of(context).push(
      MaterialPageRoute(
        builder: (_) => VerificationScreen(
          peerUsername: _activeRecipient!,
          peerCurveKey: peerKey,
          myUsername: widget.username,
          cryptoService: widget.cryptoService,
          verificationService: _verificationService,
        ),
      ),
    );
  }

  @override
  void dispose() {
    _msgSub?.cancel();
    _errSub?.cancel();
    _keyChangedSub?.cancel();
    _conversation.dispose();
    _recipientController.dispose();
    _messageController.dispose();
    super.dispose();
  }

  IconData _statusIcon(String status) {
    switch (status) {
      case 'delivered':
        return Icons.done_all;
      case 'received':
        return Icons.done_all;
      default:
        return Icons.access_time;
    }
  }

  Color _statusColor(String status) {
    switch (status) {
      case 'delivered':
        return Colors.blue;
      case 'received':
        return Colors.green;
      default:
        return Colors.grey;
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text('YUP — ${widget.username}'),
        actions: [
          Padding(
            padding: const EdgeInsets.only(right: 8),
            child: Icon(Icons.check_circle, color: Colors.green, size: 20),
          ),
          IconButton(
            icon: const Icon(Icons.settings),
            onPressed: () => context.push('/settings'),
            tooltip: 'Settings',
          ),
        ],
      ),
      body: Column(
        children: [
          _buildConnectPanel(),
          if (_error != null)
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
              child: Text(_error!, style: const TextStyle(color: Colors.red, fontSize: 12)),
            ),
          Expanded(child: _buildMessageList()),
          _buildInputBar(),
        ],
      ),
    );
  }

  Widget _buildConnectPanel() {
    return Container(
      padding: const EdgeInsets.all(12),
      color: Colors.grey[100],
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: TextField(
                  controller: _recipientController,
                  decoration: InputDecoration(
                    labelText: _activeRecipient ?? 'Recipient username',
                    border: const OutlineInputBorder(),
                    isDense: true,
                  ),
                ),
              ),
              const SizedBox(width: 8),
              if (_activeRecipient != null)
                IconButton(
                  icon: Icon(
                    _isVerified ? Icons.verified : Icons.verified_outlined,
                    color: _isVerified ? Colors.green : Colors.grey,
                  ),
                  onPressed: _showVerification,
                  tooltip: 'Verify contact',
                ),
              ElevatedButton(
                onPressed: _connecting ? null : _connect,
                child: _connecting
                    ? const SizedBox(width: 16, height: 16, child: CircularProgressIndicator(strokeWidth: 2))
                    : Text(_activeRecipient != null ? 'Switch' : 'Connect'),
              ),
            ],
          ),
          if (_activeRecipient != null)
            Padding(
              padding: const EdgeInsets.only(top: 4),
              child: Text(
                'Chatting with: $_activeRecipient',
                style: const TextStyle(fontSize: 12, color: Colors.teal),
              ),
            ),
        ],
      ),
    );
  }

  Widget _buildMessageList() {
    if (_messages.isEmpty) {
      return Center(
        child: Text(
          _activeRecipient == null
              ? 'Enter a username and connect to start chatting'
              : 'No messages yet. Send a message!',
          style: TextStyle(color: Colors.grey[500]),
        ),
      );
    }

    return ListView.builder(
      padding: const EdgeInsets.all(8),
      itemCount: _messages.length,
      itemBuilder: (context, index) {
        final msg = _messages[index];
        final isOutgoing = msg.sender == widget.username;

        return Align(
          alignment: isOutgoing ? Alignment.centerRight : Alignment.centerLeft,
          child: Container(
            margin: const EdgeInsets.symmetric(vertical: 3),
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            constraints: BoxConstraints(maxWidth: MediaQuery.of(context).size.width * 0.75),
            decoration: BoxDecoration(
              color: isOutgoing ? Colors.teal[50] : Colors.grey[200],
              borderRadius: BorderRadius.circular(12),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                if (!isOutgoing)
                  Padding(
                    padding: const EdgeInsets.only(bottom: 2),
                    child: Text(msg.sender, style: const TextStyle(fontSize: 11, fontWeight: FontWeight.bold)),
                  ),
                Text(msg.text),
                const SizedBox(height: 2),
                Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Text(
                      _formatTime(msg.createdAt),
                      style: const TextStyle(fontSize: 10, color: Colors.grey),
                    ),
                    if (isOutgoing) ...[
                      const SizedBox(width: 4),
                      Icon(_statusIcon(msg.status), size: 14, color: _statusColor(msg.status)),
                    ],
                  ],
                ),
              ],
            ),
          ),
        );
      },
    );
  }

  Widget _buildInputBar() {
    return Container(
      padding: const EdgeInsets.all(8),
      decoration: BoxDecoration(
        color: Colors.white,
        boxShadow: [BoxShadow(color: Colors.black12, offset: const Offset(0, -1), blurRadius: 4)],
      ),
      child: Row(
        children: [
          Expanded(
            child: TextField(
              controller: _messageController,
              decoration: const InputDecoration(
                hintText: 'Type a message...',
                border: OutlineInputBorder(),
                isDense: true,
              ),
              onSubmitted: (_) => _send(),
            ),
          ),
          const SizedBox(width: 8),
          IconButton(
            icon: const Icon(Icons.send),
            onPressed: _activeRecipient != null ? _send : null,
          ),
        ],
      ),
    );
  }

  String _formatTime(DateTime dt) {
    final now = DateTime.now();
    if (dt.day == now.day && dt.month == now.month && dt.year == now.year) {
      return '${dt.hour.toString().padLeft(2, '0')}:${dt.minute.toString().padLeft(2, '0')}';
    }
    return '${dt.day}/${dt.month} ${dt.hour.toString().padLeft(2, '0')}:${dt.minute.toString().padLeft(2, '0')}';
  }
}
