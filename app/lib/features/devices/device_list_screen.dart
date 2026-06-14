/// Device list screen - shows registered devices and allows connecting.
library;

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api/device_api.dart';
import '../../core/connection/connection_manager.dart';
import '../../core/models/device.dart';
import '../../core/signal/signal_client.dart';
import '../terminal/terminal_screen.dart';
import 'device_list_provider.dart';

class DeviceListScreen extends StatefulWidget {
  final String serverUrl;
  final SignalClient signalClient;
  final ConnectionManager connectionManager;

  const DeviceListScreen({
    super.key,
    required this.serverUrl,
    required this.signalClient,
    required this.connectionManager,
  });

  @override
  State<DeviceListScreen> createState() => _DeviceListScreenState();
}

class _DeviceListScreenState extends State<DeviceListScreen> {
  late DeviceListProvider _provider;
  late DeviceApi _api;

  @override
  void initState() {
    super.initState();
    _api = DeviceApi(baseUrl: widget.serverUrl);
    _provider = DeviceListProvider(serverUrl: widget.serverUrl);
    _provider.refresh();

    // Auto-refresh every 5 seconds
    Future.doWhile(() async {
      await Future.delayed(const Duration(seconds: 5));
      if (mounted) {
        _provider.refresh();
        return true;
      }
      return false;
    });
  }

  @override
  void dispose() {
    _provider.dispose();
    _api.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return ChangeNotifierProvider.value(
      value: _provider,
      child: Scaffold(
        appBar: AppBar(
          title: const Text('云桥 CloudBridge'),
          centerTitle: true,
          actions: [
            IconButton(
              icon: const Icon(Icons.settings),
              onPressed: () => _navigateToSettings(context),
            ),
          ],
        ),
        body: Consumer<DeviceListProvider>(
          builder: (context, provider, child) {
            if (provider.isLoading && provider.devices.isEmpty) {
              return const Center(child: CircularProgressIndicator());
            }

            if (provider.error != null) {
              return Center(
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    const Icon(Icons.error_outline, size: 48, color: Colors.red),
                    const SizedBox(height: 16),
                    Text('连接失败', style: Theme.of(context).textTheme.titleMedium),
                    const SizedBox(height: 8),
                    Text(provider.error!, style: Theme.of(context).textTheme.bodySmall),
                    const SizedBox(height: 16),
                    FilledButton.icon(
                      onPressed: provider.refresh,
                      icon: const Icon(Icons.refresh),
                      label: const Text('重试'),
                    ),
                  ],
                ),
              );
            }

            if (provider.devices.isEmpty) {
              return Center(
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    const Icon(Icons.devices, size: 64, color: Colors.grey),
                    const SizedBox(height: 16),
                    Text('没有在线设备', style: Theme.of(context).textTheme.titleMedium),
                    const SizedBox(height: 8),
                    Text('请确保 Agent 已启动并注册到服务器',
                        style: Theme.of(context).textTheme.bodySmall),
                    const SizedBox(height: 16),
                    FilledButton.icon(
                      onPressed: provider.refresh,
                      icon: const Icon(Icons.refresh),
                      label: const Text('刷新'),
                    ),
                  ],
                ),
              );
            }

            return RefreshIndicator(
              onRefresh: provider.refresh,
              child: ListView.builder(
                padding: const EdgeInsets.all(16),
                itemCount: provider.devices.length,
                itemBuilder: (context, index) {
                  final device = provider.devices[index];
                  return _DeviceCard(
                    device: device,
                    onTap: () => _connectToDevice(context, device),
                  );
                },
              ),
            );
          },
        ),
        floatingActionButton: FloatingActionButton(
          onPressed: _provider.refresh,
          child: const Icon(Icons.refresh),
        ),
      ),
    );
  }

  void _connectToDevice(BuildContext context, Device device) {
    // Navigate to terminal screen with connection
    Navigator.push(
      context,
      MaterialPageRoute(
        builder: (context) => TerminalScreen(
          connectionManager: widget.connectionManager,
          deviceId: device.id,
          deviceName: device.name,
          protocols: const ['shell'],
        ),
      ),
    );
  }

  void _navigateToSettings(BuildContext context) {
    // TODO: Navigate to settings screen
  }
}

class _DeviceCard extends StatelessWidget {
  final Device device;
  final VoidCallback onTap;

  const _DeviceCard({required this.device, required this.onTap});

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Row(
            children: [
              // Platform icon
              CircleAvatar(
                radius: 24,
                backgroundColor: device.online
                    ? Theme.of(context).colorScheme.primaryContainer
                    : Theme.of(context).colorScheme.surfaceVariant,
                child: Text(
                  device.platformIcon,
                  style: const TextStyle(fontSize: 24),
                ),
              ),
              const SizedBox(width: 16),
              // Device info
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      device.name,
                      style: Theme.of(context).textTheme.titleMedium,
                    ),
                    const SizedBox(height: 4),
                    Text(
                      '${device.platform} • ${device.version}',
                      style: Theme.of(context).textTheme.bodySmall,
                    ),
                    if (device.lastSeen != null)
                      Text(
                        '最后在线: ${_formatTime(device.lastSeen!)}',
                        style: Theme.of(context).textTheme.bodySmall?.copyWith(
                              color: Theme.of(context).colorScheme.outline,
                            ),
                      ),
                  ],
                ),
              ),
              // Status indicator
              Column(
                crossAxisAlignment: CrossAxisAlignment.end,
                children: [
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                    decoration: BoxDecoration(
                      color: device.online
                          ? Colors.green.withOpacity(0.1)
                          : Colors.grey.withOpacity(0.1),
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Container(
                          width: 8,
                          height: 8,
                          decoration: BoxDecoration(
                            color: device.online ? Colors.green : Colors.grey,
                            shape: BoxShape.circle,
                          ),
                        ),
                        const SizedBox(width: 6),
                        Text(
                          device.online ? '在线' : '离线',
                          style: TextStyle(
                            fontSize: 12,
                            color: device.online ? Colors.green : Colors.grey,
                            fontWeight: FontWeight.w500,
                          ),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(height: 4),
                  Icon(
                    Icons.chevron_right,
                    color: Theme.of(context).colorScheme.outline,
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }

  String _formatTime(DateTime time) {
    final now = DateTime.now();
    final diff = now.difference(time);
    if (diff.inMinutes < 1) return '刚刚';
    if (diff.inHours < 1) return '${diff.inMinutes}分钟前';
    if (diff.inDays < 1) return '${diff.inHours}小时前';
    return '${diff.inDays}天前';
  }
}