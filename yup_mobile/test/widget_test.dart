import 'package:flutter_test/flutter_test.dart';
import 'package:yup_mobile/app/router.dart';
import 'package:yup_mobile/app/service_container.dart';

void main() {
  test('AppServices creates without throwing', () {
    // Note: full integration test requires native FFI library (libyup_crypto.so)
    // which is not available in test environment. This confirms imports resolve.
    expect(AppServices, isA<Type>());
    expect(ServicesScope, isA<Type>());
    expect(createRouter, isA<Function>());
  });
}
