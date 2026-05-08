import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import '../../../core/networking/api_client.dart';
import '../../../core/secure_storage/secure_storage_service.dart';
import '../../../core/storage/local_database.dart';
import '../../key_management/domain/crypto_service.dart';

class SettingsScreen extends StatefulWidget {
  final CryptoService cryptoService;
  final SecureStorageService storage;
  final ApiClient api;

  const SettingsScreen({
    super.key,
    required this.cryptoService,
    required this.storage,
    required this.api,
  });

  @override
  State<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends State<SettingsScreen> {
  String? _username;
  String? _curveKey;
  bool _clearing = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    final username = await widget.storage.getStoredUsername();
    if (username != null) {
      final keys = await widget.storage.getIdentityKeys(username);
      setState(() {
        _username = username;
        _curveKey = keys?['curve25519'];
      });
    }
  }

  Future<void> _clearLocalData() async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Clear Local Data?'),
        content: const Text(
          'This will permanently delete ALL local data:\n'
          '- Auth token and identity keys\n'
          '- Account pickle and session data\n'
          '- Encrypted message database\n'
          '- Verification info\n\n'
          'You will need to re-register. Old messages CANNOT be recovered.',
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('Cancel')),
          TextButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('Clear Everything', style: TextStyle(color: Colors.red)),
          ),
        ],
      ),
    );
    if (confirmed != true) return;
    setState(() => _clearing = true);
    try {
      // Delete the encrypted SQLCipher database
      final db = LocalDatabase(widget.storage);
      await db.deleteDatabaseFile();

      // Clear secure storage (keys, sessions, passphrase, etc.)
      if (_username != null) {
        await widget.storage.clearAllUserData(_username!);
      }
      widget.api.clearToken();
      if (mounted) {
        context.go('/register');
      }
    } finally {
      if (mounted) setState(() => _clearing = false);
    }
  }

  Future<void> _logout() async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Log Out?'),
        content: const Text(
          'Logout removes your auth token but keeps your encrypted '
          'message history and keys.\n\n'
          'Use "Clear Local Data" instead if you want to delete everything.',
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('Cancel')),
          TextButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('Log Out'),
          ),
        ],
      ),
    );
    if (confirmed != true) return;
    if (_username == null) return;
    // Logout only removes auth token and active session state
    // Preserves encrypted DB and passphrase
    await widget.storage.deleteRaw('auth_token:$_username');
    await widget.storage.deleteRaw('active_username');
    widget.api.clearToken();
    if (mounted) {
      context.go('/register');
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Settings')),
      body: ListView(
        padding: const EdgeInsets.all(20),
        children: [
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Text('Account', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
                  const SizedBox(height: 12),
                  Text('Username: ${_username ?? "—"}'),
                  const SizedBox(height: 8),
                  Text(
                    'Device Identity Key (Curve25519):',
                    style: TextStyle(fontSize: 12, color: Colors.grey[600]),
                  ),
                  const SizedBox(height: 4),
                  SelectableText(
                    _curveKey ?? '—',
                    style: const TextStyle(fontFamily: 'monospace', fontSize: 11),
                  ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 12),
          Card(
            child: Column(
              children: [
                ListTile(
                  leading: const Icon(Icons.shield_outlined),
                  title: const Text('Verification Info'),
                  subtitle: const Text('How key verification works'),
                  onTap: () {
                    ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(
                        content: Text(
                          'Compare the conversation security fingerprint '
                          'with your contact out-of-band.',
                        ),
                      ),
                    );
                  },
                ),
              ],
            ),
          ),
          const SizedBox(height: 24),
          SizedBox(
            width: double.infinity,
            child: OutlinedButton.icon(
              onPressed: _clearing ? null : _clearLocalData,
              icon: const Icon(Icons.delete_outline),
              label: Text(_clearing ? 'Clearing...' : 'Clear Local Data'),
              style: OutlinedButton.styleFrom(foregroundColor: Colors.red),
            ),
          ),
          const SizedBox(height: 12),
          SizedBox(
            width: double.infinity,
            child: ElevatedButton.icon(
              onPressed: _logout,
              icon: const Icon(Icons.logout),
              label: const Text('Log Out'),
              style: ElevatedButton.styleFrom(foregroundColor: Colors.white, backgroundColor: Colors.red),
            ),
          ),
          const SizedBox(height: 16),
          Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: Colors.amber[50],
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: Colors.amber),
            ),
            child: const Text(
              'Logout preserves encrypted message history.\n'
              'Use "Clear Local Data" to delete everything.',
              style: TextStyle(fontSize: 12, color: Colors.brown),
            ),
          ),
        ],
      ),
    );
  }
}
