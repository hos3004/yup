import 'package:go_router/go_router.dart';
import '../features/auth/presentation/register_screen.dart';
import '../features/messaging/presentation/chat_screen.dart';
import '../features/settings/presentation/settings_screen.dart';
import '../features/verification/data/verification_service.dart';
import '../features/verification/presentation/verification_screen.dart';
import 'service_container.dart';

GoRouter createRouter(AppServices services) {
  return GoRouter(
    initialLocation: '/register',
    routes: [
      GoRoute(
        path: '/register',
        builder: (context, state) => RegisterScreen(
          registrationService: services.registration,
          onRegistered: (username, curve25519, ed25519) {
            context.go('/chat', extra: {
              'username': username,
              'curve25519': curve25519,
              'ed25519': ed25519,
            });
          },
        ),
      ),
      GoRoute(
        path: '/chat',
        builder: (context, state) {
          final extra = state.extra as Map<String, String>;
          return ChatScreen(
            username: extra['username']!,
            myCurveKey: extra['curve25519']!,
            cryptoService: services.crypto,
            apiClient: services.api,
            secureStorage: services.storage,
          );
        },
      ),
      GoRoute(
        path: '/verification',
        builder: (context, state) {
          final extra = state.extra as Map<String, dynamic>;
          return VerificationScreen(
            peerUsername: extra['peerUsername'] as String,
            peerCurveKey: extra['peerCurveKey'] as String,
            myUsername: extra['myUsername'] as String,
            cryptoService: services.crypto,
            verificationService: extra['verificationService'] as VerificationService,
          );
        },
      ),
      GoRoute(
        path: '/settings',
        builder: (context, state) => SettingsScreen(
          cryptoService: services.crypto,
          storage: services.storage,
          api: services.api,
        ),
      ),
    ],
  );
}
