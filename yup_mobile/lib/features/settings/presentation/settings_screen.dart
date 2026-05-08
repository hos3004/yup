import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import '../../../core/networking/api_client.dart';
import '../../../core/secure_storage/secure_storage_service.dart';
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
          'This will remove all local keys and messages. '
          'You will need to re-register.',
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('Cancel')),
          TextButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('Clear', style: TextStyle(color: Colors.red)),
          ),
        ],
      ),
    );
    if (confirmed != true) return;
    setState(() => _clearing = true);
    try {
      if (_username != null) {
        await widget.storage.clearUserData(_username!);
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
    if (_username == null) return;
    await widget.storage.clearUserData(_username!);
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
                          'Fingerprint comparison only. QR scanning not yet available.',
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
        ],
      ),
    );
  }
}
