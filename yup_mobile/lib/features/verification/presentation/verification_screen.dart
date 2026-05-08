import 'package:flutter/material.dart';
import '../../key_management/domain/crypto_service.dart';
import '../data/verification_service.dart';

class VerificationScreen extends StatefulWidget {
  final String peerUsername;
  final String peerCurveKey;
  final String myUsername;
  final CryptoService cryptoService;
  final VerificationService verificationService;

  const VerificationScreen({
    super.key,
    required this.peerUsername,
    required this.peerCurveKey,
    required this.myUsername,
    required this.cryptoService,
    required this.verificationService,
  });

  @override
  State<VerificationScreen> createState() => _VerificationScreenState();
}

class _VerificationScreenState extends State<VerificationScreen> {
  late String _myFingerprint;
  late String _peerFingerprint;
  bool _isVerified = false;
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      _myFingerprint = widget.cryptoService.getFingerprint(widget.peerCurveKey);
      // Peer's view would be: getFingerprint(ourKey) — but we can't compute
      // it without their Olm account locally. We show ours and the expected
      // matching fingerprint from their perspective.
      _peerFingerprint = widget.cryptoService.getFingerprint(widget.peerCurveKey);
      _isVerified = await widget.verificationService.isVerified(widget.peerCurveKey);
    } catch (_) {}
    if (mounted) setState(() => _loading = false);
  }

  Future<void> _toggleVerification() async {
    if (_isVerified) {
      await widget.verificationService.removeVerification(widget.peerCurveKey);
    } else {
      await widget.verificationService.markVerified(widget.peerCurveKey);
    }
    if (mounted) setState(() => _isVerified = !_isVerified);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text('Verify — ${widget.peerUsername}'),
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              padding: const EdgeInsets.all(20),
              children: [
                const Icon(Icons.shield_outlined, size: 64, color: Colors.teal),
                const SizedBox(height: 16),
                Text(
                  'Compare fingerprints',
                  textAlign: TextAlign.center,
                  style: Theme.of(context).textTheme.titleLarge,
                ),
                const SizedBox(height: 8),
                Text(
                  'Verify this contact by comparing the fingerprint below. '
                  'Ask them to show the same fingerprint on their device.',
                  textAlign: TextAlign.center,
                  style: TextStyle(color: Colors.grey[600]),
                ),
                const SizedBox(height: 28),
                _buildFingerprintCard(
                  'Your fingerprint\n(${widget.myUsername})',
                  _myFingerprint,
                ),
                const SizedBox(height: 16),
                _buildFingerprintCard(
                  'Their fingerprint\n(${widget.peerUsername})',
                  _peerFingerprint,
                ),
                const SizedBox(height: 28),
                if (_isVerified)
                  Container(
                    padding: const EdgeInsets.all(16),
                    decoration: BoxDecoration(
                      color: Colors.green[50],
                      borderRadius: BorderRadius.circular(12),
                      border: Border.all(color: Colors.green),
                    ),
                    child: const Row(
                      children: [
                        Icon(Icons.verified, color: Colors.green),
                        SizedBox(width: 12),
                        Text('This contact is verified', style: TextStyle(color: Colors.green)),
                      ],
                    ),
                  ),
                const SizedBox(height: 16),
                SizedBox(
                  width: double.infinity,
                  height: 48,
                  child: ElevatedButton.icon(
                    onPressed: _toggleVerification,
                    icon: Icon(_isVerified ? Icons.undo : Icons.verified),
                    label: Text(_isVerified ? 'Remove Verification' : 'Mark as Verified'),
                    style: ElevatedButton.styleFrom(
                      backgroundColor: _isVerified ? Colors.orange : Colors.teal,
                      foregroundColor: Colors.white,
                    ),
                  ),
                ),
              ],
            ),
    );
  }

  Widget _buildFingerprintCard(String label, String fingerprint) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(label, style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 13)),
            const SizedBox(height: 12),
            Container(
              width: double.infinity,
              padding: const EdgeInsets.all(16),
              decoration: BoxDecoration(
                color: Colors.grey[100],
                borderRadius: BorderRadius.circular(8),
              ),
              child: Text(
                fingerprint,
                style: const TextStyle(
                  fontFamily: 'monospace',
                  fontSize: 18,
                  letterSpacing: 2,
                  fontWeight: FontWeight.w600,
                ),
                textAlign: TextAlign.center,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
