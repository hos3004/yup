import 'package:flutter/material.dart';
import 'app/app.dart';
import 'app/service_container.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  final services = AppServices.create();
  runApp(YupApp(services: services));
}
