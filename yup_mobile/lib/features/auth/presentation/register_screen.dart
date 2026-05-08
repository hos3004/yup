import 'package:flutter/material.dart';
import '../../key_management/data/device_registration_service.dart';

class RegisterScreen extends StatefulWidget {
  final DeviceRegistrationService registrationService;
  final void Function(String username, String curve25519, String ed25519) onRegistered;

  const RegisterScreen({
    super.key,
    required this.registrationService,
    required this.onRegistered,
  });

  @override
  State<RegisterScreen> createState() => _RegisterScreenState();
}

class _RegisterScreenState extends State<RegisterScreen> {
  final _controller = TextEditingController();
  bool _loading = false;
  bool _showRegister = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _tryRestore();
  }

  Future<void> _tryRestore() async {
    setState(() => _loading = true);
    try {
      final result = await widget.registrationService.tryRestoreLastSession();
      if (result != null && mounted) {
        widget.onRegistered(result.username, result.curve25519, result.ed25519);
        return;
      }
    } catch (_) {}
    if (mounted) {
      setState(() {
        _loading = false;
        _showRegister = true;
      });
    }
  }

  Future<void> _register() async {
    final username = _controller.text.trim();
    if (username.length < 3) {
      setState(() => _error = 'Username must be at least 3 characters');
      return;
    }
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final result = await widget.registrationService.register(username);
      if (mounted) {
        widget.onRegistered(result.username, result.curve25519, result.ed25519);
      }
    } catch (e) {
      if (mounted) setState(() => _error = e.toString());
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('YUP — Register')),
      body: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Text(
              _loading ? 'Setting up...' : 'Choose your username',
              style: Theme.of(context).textTheme.headlineSmall,
            ),
            if (_loading) ...[
              const SizedBox(height: 24),
              const CircularProgressIndicator(),
            ],
            if (_showRegister) ...[
              const SizedBox(height: 24),
              TextField(
                controller: _controller,
                decoration: InputDecoration(
                  labelText: 'Username',
                  hintText: 'At least 3 characters',
                  errorText: _error,
                  border: const OutlineInputBorder(),
                ),
                textInputAction: TextInputAction.done,
                onSubmitted: (_) => _register(),
              ),
              const SizedBox(height: 16),
              SizedBox(
                width: double.infinity,
                height: 48,
                child: ElevatedButton(
                  onPressed: _loading ? null : _register,
                  child: const Text('Register'),
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
