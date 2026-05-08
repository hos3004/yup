import 'package:flutter/material.dart';
import '../core/crypto_ffi/crypto_bridge.dart';
import '../core/networking/api_client.dart';
import '../core/secure_storage/secure_storage_service.dart';
import '../features/key_management/data/device_registration_service.dart';
import '../features/key_management/domain/crypto_service.dart';

class AppServices {
  final SecureStorageService storage;
  final CryptoService crypto;
  final ApiClient api;
  final DeviceRegistrationService registration;

  AppServices._({
    required this.storage,
    required this.crypto,
    required this.api,
    required this.registration,
  });

  static AppServices create({
    String baseUrl = 'http://10.0.2.2:8080',
  }) {
    final storage = SecureStorageService();
    final crypto = CryptoService(CryptoBridge());
    final api = ApiClient(baseUrl);
    final registration = DeviceRegistrationService(api, crypto, storage);
    return AppServices._(
      storage: storage,
      crypto: crypto,
      api: api,
      registration: registration,
    );
  }
}

class ServicesScope extends InheritedWidget {
  final AppServices services;

  const ServicesScope({
    super.key,
    required this.services,
    required super.child,
  });

  static AppServices of(BuildContext context) {
    final scope = context.dependOnInheritedWidgetOfExactType<ServicesScope>();
    assert(scope != null, 'No ServicesScope found in context');
    return scope!.services;
  }

  @override
  bool updateShouldNotify(ServicesScope oldWidget) => services != oldWidget.services;
}

extension BuildContextServices on BuildContext {
  AppServices get services => ServicesScope.of(this);
}
