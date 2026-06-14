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
  void dispose() {
    _serverUrlController.dispose();
    _relayUrlController.dispose();
    super.dispose();
  }

  Future<void> _saveSettings() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('server_url', _serverUrlController.text);
    await prefs.setString('relay_url', _relayUrlController.text);

    widget.onServerUrlChanged(_serverUrlController.text);
    widget.onRelayUrlChanged(_relayUrlController.text);

    setState(() => _saved = true);
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('设置已保存')),
    );
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
              '设置已保存，请重新连接',
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