/// Device list screen - shows registered devices and allows connecting.
library;

import 'package:flutter/material.dart';

import '../../core/connection/connection_manager.dart';
import '../../core/models/device.dart';
import '../../core/signal/signal_client.dart';
import '../terminal/terminal_screen.dart';
import 'device_list_provider.dart';

class DeviceListScreen extends StatefulWidget {
  final SignalClient signalClient;
  final ConnectionManager connectionManager;
  final DeviceListProvider deviceProvider;

  const DeviceListScreen({
    super.key,
    required this.signalClient,
    required this.connectionManager,
    required this.deviceProvider,
  });

  @override
  State<DeviceListScreen> createState() => _DeviceListScreenState();
}

class _DeviceListScreenState extends State<DeviceListScreen> {
  SignalConnectionState _signalState = SignalConnectionState.disconnected;

  @override
  void initState() {
    super.initState();
    _signalState = widget.signalClient.state;
    widget.signalClient.addStateListener(_onSignalStateChanged);
    _connectSignalIfNeeded();
  }

  void _onSignalStateChanged(SignalConnectionState state) {
    if (mounted) {
      setState(() => _signalState = state);
    }
  }

  @override
  void dispose() {
    widget.signalClient.removeStateListener(_onSignalStateChanged);
    super.dispose();
  }

  Future<void> _connectSignalIfNeeded() async {
    if (widget.signalClient.state == SignalConnectionState.disconnected ||
        widget.signalClient.state == SignalConnectionState.error) {
      try {
        await widget.signalClient.connect();
        widget.signalClient.register(
          deviceId: 'app-${DateTime.now().millisecondsSinceEpoch}',
          deviceName: 'CloudBridge App',
          platform: 'web',
          version: '0.1.0',
        );
        widget.signalClient.startHeartbeat();
      } catch (e) {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('信令连接失败: $e')),
          );
        }
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: widget.deviceProvider,
      builder: (context, _) {
        final provider = widget.deviceProvider;

        // Signal connection color
        final signalColor = switch (_signalState) {
          SignalConnectionState.registered => Colors.green,
          SignalConnectionState.connected => Colors.orange,
          SignalConnectionState.connecting => Colors.orange,
          _ => Colors.grey,
        };

        return Scaffold(
          appBar: AppBar(
            title: const Text('云桥 CloudBridge'),
            centerTitle: true,
            actions: [
              Padding(
                padding: const EdgeInsets.only(right: 8),
                child: Icon(Icons.circle, size: 12, color: signalColor),
              ),
              IconButton(
                icon: const Icon(Icons.refresh),
                onPressed: () {
                  provider.refresh();
                  _connectSignalIfNeeded();
                },
              ),
            ],
          ),
          body: _buildBody(provider),
          floatingActionButton: FloatingActionButton(
            onPressed: () {
              provider.refresh();
              _connectSignalIfNeeded();
            },
            child: const Icon(Icons.refresh),
          ),
        );
      },
    );
  }

  Widget _buildBody(DeviceListProvider provider) {
    if (provider.isLoading && provider.devices.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }

    if (provider.error != null) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(32),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              const Icon(Icons.error_outline, size: 48, color: Colors.red),
              const SizedBox(height: 16),
              const Text('连接失败', style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
              const SizedBox(height: 8),
              Text(
                provider.error!,
                style: const TextStyle(fontSize: 12, color: Colors.grey),
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 8),
              const Text(
                '请检查设置中的服务器地址是否正确',
                style: TextStyle(fontSize: 12, color: Colors.grey),
              ),
              const SizedBox(height: 16),
              FilledButton.icon(
                onPressed: () {
                  provider.refresh();
                  _connectSignalIfNeeded();
                },
                icon: const Icon(Icons.refresh),
                label: const Text('重试'),
              ),
            ],
          ),
        ),
      );
    }

    if (provider.devices.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(32),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              const Icon(Icons.devices, size: 64, color: Colors.grey),
              const SizedBox(height: 16),
              const Text('没有在线设备', style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
              const SizedBox(height: 8),
              const Text(
                '请确保 Agent 已启动并注册到服务器\n在设置中配置正确的服务器地址',
                textAlign: TextAlign.center,
                style: TextStyle(fontSize: 14, color: Colors.grey),
              ),
              const SizedBox(height: 16),
              FilledButton.icon(
                onPressed: () {
                  provider.refresh();
                  _connectSignalIfNeeded();
                },
                icon: const Icon(Icons.refresh),
                label: const Text('刷新'),
              ),
            ],
          ),
        ),
      );
    }

    return RefreshIndicator(
      onRefresh: () async {
        provider.refresh();
      },
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
  }

  void _connectToDevice(BuildContext context, Device device) {
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