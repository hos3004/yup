import 'package:flutter/material.dart';
import 'router.dart';
import 'service_container.dart';

class YupApp extends StatelessWidget {
  final AppServices services;

  const YupApp({super.key, required this.services});

  @override
  Widget build(BuildContext context) {
    return ServicesScope(
      services: services,
      child: MaterialApp.router(
        title: 'YUP',
        debugShowCheckedModeBanner: false,
        theme: ThemeData(
          colorScheme: ColorScheme.fromSeed(seedColor: Colors.teal),
          useMaterial3: true,
        ),
        routerConfig: createRouter(services),
      ),
    );
  }
}
