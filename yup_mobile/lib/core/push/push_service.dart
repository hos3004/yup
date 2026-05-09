import 'dart:async';
import 'package:firebase_messaging/firebase_messaging.dart';
import '../networking/api_client.dart';

class PushService {
  final ApiClient _api;
  final _pushTriggerController = StreamController<void>.broadcast();
  Stream<void> get pushTriggers => _pushTriggerController.stream;
  bool _initialized = false;

  PushService(this._api);

  Future<void> initialize() async {
    if (_initialized) return;
    try {
      final messaging = FirebaseMessaging.instance;

      final settings = await messaging.requestPermission(
        alert: true,
        badge: true,
        sound: true,
      );

      if (settings.authorizationStatus == AuthorizationStatus.denied) {
        return;
      }

      final token = await messaging.getToken();
      if (token != null) {
        await _api.registerDeviceToken(token, 'android');
      }

      messaging.onTokenRefresh.listen((newToken) async {
        await _api.registerDeviceToken(newToken, 'android');
      });

      FirebaseMessaging.onMessage.listen(_handleMessage);

      _initialized = true;
    } catch (_) {
      // Firebase or push token unavailable — non-fatal
    }
  }

  void _handleMessage(RemoteMessage message) {
    if (message.data['type'] == 'new_message') {
      _pushTriggerController.add(null);
    }
  }

  void dispose() {
    _pushTriggerController.close();
  }
}
