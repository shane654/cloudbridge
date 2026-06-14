import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'core/connection/connection_manager.dart';
import 'core/signal/signal_client.dart';
import 'features/devices/device_list_screen.dart';
import 'features/settings/settings_screen.dart';

void main() {
  runApp(const CloudBridgeApp());
}

class CloudBridgeApp extends StatefulWidget {
  const CloudBridgeApp({super.key});

  @override
  State<CloudBridgeApp> createState() => _CloudBridgeAppState();
}

class _CloudBridgeAppState extends State<CloudBridgeApp> {
  String _serverUrl = 'ws://localhost:10980/signal';
  String _relayUrl = 'localhost:10988';
  String _httpUrl = 'http://localhost:10980';
  late SignalClient _signalClient;
  late ConnectionManager _connectionManager;
  bool _initialized = false;

  @override
  void initState() {
    super.initState();
    _initSettings();
  }

  Future<void> _initSettings() async {
    final prefs = await SharedPreferences.getInstance();
    setState(() {
      _serverUrl = prefs.getString('server_url') ?? _serverUrl;
      _relayUrl = prefs.getString('relay_url') ?? _relayUrl;
      _httpUrl = prefs.getString('http_url') ?? _httpUrl;
    });

    _signalClient = SignalClient(serverUrl: _serverUrl);
    _connectionManager = ConnectionManager(signalClient: _signalClient);
    _connectionManager.setRelayAddress(_relayUrl);

    setState(() => _initialized = true);
  }

  void _onServerUrlChanged(String url) {
    setState(() {
      _serverUrl = url;
      _httpUrl = url
          .replaceFirst('ws://', 'http://')
          .replaceFirst('wss://', 'https://')
          .replaceAll('/signal', '');
    });
    _signalClient.disconnectClient();
    _signalClient = SignalClient(serverUrl: _serverUrl);
    _connectionManager = ConnectionManager(signalClient: _signalClient);
  }

  void _onRelayUrlChanged(String url) {
    setState(() => _relayUrl = url);
    _connectionManager.setRelayAddress(url);
  }

  @override
  Widget build(BuildContext context) {
    if (!_initialized) {
      return const MaterialApp(
        home: Scaffold(
          body: Center(child: CircularProgressIndicator()),
        ),
      );
    }

    return MultiProvider(
      providers: [
        Provider<SignalClient>.value(value: _signalClient),
        Provider<ConnectionManager>.value(value: _connectionManager),
        Provider<String>.value(value: _httpUrl),
      ],
      child: MaterialApp(
        title: '云桥 CloudBridge',
        debugShowCheckedModeBanner: false,
        theme: ThemeData(
          colorSchemeSeed: const Color(0xFF2563EB),
          useMaterial3: true,
          brightness: Brightness.dark,
        ),
        home: _HomePage(
          serverUrl: _httpUrl,
          signalClient: _signalClient,
          connectionManager: _connectionManager,
          onServerUrlChanged: _onServerUrlChanged,
          onRelayUrlChanged: _onRelayUrlChanged,
          currentServerUrl: _serverUrl,
          currentRelayUrl: _relayUrl,
        ),
      ),
    );
  }
}

class _HomePage extends StatefulWidget {
  final String serverUrl;
  final SignalClient signalClient;
  final ConnectionManager connectionManager;
  final ValueChanged<String> onServerUrlChanged;
  final ValueChanged<String> onRelayUrlChanged;
  final String currentServerUrl;
  final String currentRelayUrl;

  const _HomePage({
    required this.serverUrl,
    required this.signalClient,
    required this.connectionManager,
    required this.onServerUrlChanged,
    required this.onRelayUrlChanged,
    required this.currentServerUrl,
    required this.currentRelayUrl,
  });

  @override
  State<_HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<_HomePage> {
  int _currentIndex = 0;

  @override
  Widget build(BuildContext context) {
    final pages = [
      DeviceListScreen(
        serverUrl: widget.serverUrl,
        signalClient: widget.signalClient,
        connectionManager: widget.connectionManager,
      ),
      SettingsScreen(
        currentServerUrl: widget.currentServerUrl,
        currentRelayUrl: widget.currentRelayUrl,
        onServerUrlChanged: widget.onServerUrlChanged,
        onRelayUrlChanged: widget.onRelayUrlChanged,
      ),
    ];

    return Scaffold(
      body: IndexedStack(
        index: _currentIndex,
        children: pages,
      ),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _currentIndex,
        onDestinationSelected: (index) {
          setState(() => _currentIndex = index);
        },
        destinations: const [
          NavigationDestination(
            icon: Icon(Icons.devices),
            selectedIcon: Icon(Icons.devices),
            label: '设备',
          ),
          NavigationDestination(
            icon: Icon(Icons.settings),
            selectedIcon: Icon(Icons.settings),
            label: '设置',
          ),
        ],
      ),
    );
  }
}