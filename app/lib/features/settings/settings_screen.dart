/// Settings screen - configure server addresses.
library;

import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';

class SettingsScreen extends StatefulWidget {
  final String currentServerUrl;
  final String currentRelayUrl;
  final ValueChanged<String> onServerUrlChanged;
  final ValueChanged<String> onRelayUrlChanged;

  const SettingsScreen({
    super.key,
    required this.currentServerUrl,
    required this.currentRelayUrl,
    required this.onServerUrlChanged,
    required this.onRelayUrlChanged,
  });

  @override
  State<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends State<SettingsScreen> {
  late TextEditingController _serverUrlController;
  late TextEditingController _relayUrlController;
  bool _saved = false;

  @override
  void initState() {
    super.initState();
    _serverUrlController = TextEditingController(text: widget.currentServerUrl);
    _relayUrlController = TextEditingController(text: widget.currentRelayUrl);
  }

  @override
  void didUpdateWidget(SettingsScreen oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.currentServerUrl != widget.currentServerUrl) {
      _serverUrlController.text = widget.currentServerUrl;
    }
    if (oldWidget.currentRelayUrl != widget.currentRelayUrl) {
      _relayUrlController.text = widget.currentRelayUrl;
    }
  }

  @override
  void dispose() {
    _serverUrlController.dispose();
    _relayUrlController.dispose();
    super.dispose();
  }

  Future<void> _saveSettings() async {
    final serverUrl = _serverUrlController.text.trim();
    final relayUrl = _relayUrlController.text.trim();

    if (serverUrl.isEmpty) return;

    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('server_url', serverUrl);
    await prefs.setString('relay_url', relayUrl);

    // Also save the HTTP URL derived from WebSocket URL
    final httpUrl = serverUrl
        .replaceFirst('ws://', 'http://')
        .replaceFirst('wss://', 'https://')
        .replaceAll('/signal', '');
    await prefs.setString('http_url', httpUrl);

    widget.onServerUrlChanged(serverUrl);
    widget.onRelayUrlChanged(relayUrl);

    setState(() => _saved = true);
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('设置已保存，请返回设备页刷新')),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('设置'),
        centerTitle: true,
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          // Connection guide
          Card(
            color: Theme.of(context).colorScheme.primaryContainer.withOpacity(0.3),
            child: const Padding(
              padding: EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Icon(Icons.info_outline, size: 20),
                      SizedBox(width: 8),
                      Text('快速开始', style: TextStyle(fontWeight: FontWeight.bold)),
                    ],
                  ),
                  SizedBox(height: 8),
                  Text('1. 在服务器上启动 cloudbridge-server'),
                  Text('2. 在远程设备上启动 cloudbridge-agent'),
                  Text('3. 在下方填写服务器 IP 地址'),
                  Text('4. 返回设备页刷新即可看到在线设备'),
                ],
              ),
            ),
          ),
          const SizedBox(height: 24),
          // Server section
          Text(
            '服务器配置',
            style: Theme.of(context).textTheme.titleMedium?.copyWith(
                  color: Theme.of(context).colorScheme.primary,
                ),
          ),
          const SizedBox(height: 16),
          TextField(
            controller: _serverUrlController,
            decoration: const InputDecoration(
              labelText: '信令服务器地址',
              hintText: 'ws://192.168.1.100:10980/signal',
              prefixIcon: Icon(Icons.dns),
              border: OutlineInputBorder(),
              helperText: 'WebSocket 地址，包含端口号和 /signal 路径',
            ),
          ),
          const SizedBox(height: 16),
          TextField(
            controller: _relayUrlController,
            decoration: const InputDecoration(
              labelText: 'Relay 中继地址',
              hintText: '192.168.1.100:10988',
              prefixIcon: Icon(Icons.swap_horiz),
              border: OutlineInputBorder(),
              helperText: '仅填 IP:端口，不含协议前缀',
            ),
          ),
          const SizedBox(height: 24),
          FilledButton.icon(
            onPressed: _saveSettings,
            icon: const Icon(Icons.save),
            label: const Text('保存设置'),
          ),
          if (_saved) ...[
            const SizedBox(height: 8),
            Text(
              '设置已保存，请返回设备页刷新',
              style: Theme.of(context).textTheme.bodySmall?.copyWith(
                    color: Colors.green,
                  ),
            ),
          ],
          const SizedBox(height: 32),
          // About section
          Text(
            '关于',
            style: Theme.of(context).textTheme.titleMedium?.copyWith(
                  color: Theme.of(context).colorScheme.primary,
                ),
          ),
          const SizedBox(height: 16),
          const Card(
            child: Padding(
              padding: EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('云桥 CloudBridge', style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
                  SizedBox(height: 8),
                  Text('版本 0.1.0 (MVP)'),
                  SizedBox(height: 4),
                  Text('通过手机连接远程设备'),
                  SizedBox(height: 4),
                  Text('支持直连和中继模式'),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}