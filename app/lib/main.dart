import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'core/connection/connection_manager.dart';
import 'core/signal/signal_client.dart';
import 'features/devices/device_list_screen.dart';
import 'features/devices/device_list_provider.dart';
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
  String _serverUrl = 'ws://54.39.49.63:10985/signal';
  String _relayUrl = '54.39.49.63:10988';
  String _httpUrl = 'http://54.39.49.63:10985';
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
    // Derive HTTP URL from WebSocket URL
    String httpUrl = url
        .replaceFirst('ws://', 'http://')
        .replaceFirst('wss://', 'https://')
        .replaceAll('/signal', '');

    setState(() {
      _serverUrl = url;
      _httpUrl = httpUrl;
    });

    // Reconnect signal client with new URL
    _signalClient.disconnectClient();
    _signalClient = SignalClient(serverUrl: _serverUrl);
    _connectionManager = ConnectionManager(signalClient: _signalClient);
    _connectionManager.setRelayAddress(_relayUrl);
  }

  void _onRelayUrlChanged(String url) {
    setState(() {
      _relayUrl = url;
    });
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
          signalClient: _signalClient,
          connectionManager: _connectionManager,
          httpUrl: _httpUrl,
          serverUrl: _serverUrl,
          relayUrl: _relayUrl,
          onServerUrlChanged: _onServerUrlChanged,
          onRelayUrlChanged: _onRelayUrlChanged,
        ),
      ),
    );
  }
}

class _HomePage extends StatefulWidget {
  final SignalClient signalClient;
  final ConnectionManager connectionManager;
  final String httpUrl;
  final String serverUrl;
  final String relayUrl;
  final ValueChanged<String> onServerUrlChanged;
  final ValueChanged<String> onRelayUrlChanged;

  const _HomePage({
    required this.signalClient,
    required this.connectionManager,
    required this.httpUrl,
    required this.serverUrl,
    required this.relayUrl,
    required this.onServerUrlChanged,
    required this.onRelayUrlChanged,
  });

  @override
  State<_HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<_HomePage> {
  int _currentIndex = 0;
  late DeviceListProvider _deviceProvider;

  @override
  void initState() {
    super.initState();
    _deviceProvider = DeviceListProvider(baseUrl: widget.httpUrl);
    // Auto-refresh device list on start
    _deviceProvider.refresh();
  }

  @override
  void didUpdateWidget(_HomePage oldWidget) {
    super.didUpdateWidget(oldWidget);
    // Recreate provider when server URL changes
    if (oldWidget.httpUrl != widget.httpUrl) {
      _deviceProvider.dispose();
      _deviceProvider = DeviceListProvider(baseUrl: widget.httpUrl);
      _deviceProvider.refresh();
    }
  }

  @override
  void dispose() {
    _deviceProvider.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final pages = [
      DeviceListScreen(
        signalClient: widget.signalClient,
        connectionManager: widget.connectionManager,
        deviceProvider: _deviceProvider,
      ),
      SettingsScreen(
        currentServerUrl: widget.serverUrl,
        currentRelayUrl: widget.relayUrl,
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
          // Refresh device list when switching to devices tab
          if (index == 0) {
            _deviceProvider.refresh();
          }
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