/// Terminal screen - displays an interactive remote terminal.
library;

import 'dart:async';
import 'dart:typed_data';

import 'package:flutter/material.dart';

import '../../core/connection/connection_manager.dart';
import '../../core/connection/connection_state.dart' as cb;
import '../../core/protocol/frame.dart';

class TerminalScreen extends StatefulWidget {
  final ConnectionManager connectionManager;
  final String deviceId;
  final String deviceName;
  final List<String> protocols;

  const TerminalScreen({
    super.key,
    required this.connectionManager,
    required this.deviceId,
    required this.deviceName,
    required this.protocols,
  });

  @override
  State<TerminalScreen> createState() => _TerminalScreenState();
}

class _TerminalScreenState extends State<TerminalScreen> {
  final List<String> _outputLines = [];
  final ScrollController _scrollController = ScrollController();
  final TextEditingController _inputController = TextEditingController();
  StreamSubscription? _frameSubscription;
  cb.ConnectionState _connectionState = cb.ConnectionState.idle;

  @override
  void initState() {
    super.initState();
    _connectionState = widget.connectionManager.state;
    widget.connectionManager.addStateListener(_onStateChanged);

    // Connect to the device
    widget.connectionManager.connectToDevice(
      widget.deviceId,
      widget.protocols,
    );

    _addOutput('Connecting to ${widget.deviceName}...');
  }

  @override
  void dispose() {
    _frameSubscription?.cancel();
    widget.connectionManager.disconnect();
    _inputController.dispose();
    _scrollController.dispose();
    super.dispose();
  }

  void _onStateChanged(cb.ConnectionState state) {
    setState(() {
      _connectionState = state;
    });

    switch (state) {
      case cb.ConnectionState.connected:
        _addOutput('\nConnected! Terminal ready.');
        _listenToFrames();
        break;
      case cb.ConnectionState.relayConnected:
        _addOutput('\nRelay connected, opening shell...');
        break;
      case cb.ConnectionState.error:
        final error = widget.connectionManager.lastError ?? 'Unknown error';
        _addOutput('\nConnection error: $error');
        if (error.contains('not supported on web')) {
          _addOutput('\n💡 Web 端暂不支持远程终端。');
          _addOutput('请使用 Android/iOS App 获取完整功能。');
        }
        break;
      case cb.ConnectionState.disconnected:
        _addOutput('\nDisconnected.');
        break;
      default:
        break;
    }
  }

  void _listenToFrames() {
    final transport = widget.connectionManager.relayTransport;
    if (transport == null) return;

    _frameSubscription = transport.onFrame.listen((frame) {
      if (frame.streamId == StreamID.shell && frame.type == FrameType.data) {
        final text = String.fromCharCodes(frame.payload);
        _addOutput(text);
      }
    });
  }

  void _addOutput(String text) {
    setState(() {
      _outputLines.add(text);
    });
    // Auto-scroll to bottom
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scrollController.hasClients) {
        _scrollController.animateTo(
          _scrollController.position.maxScrollExtent,
          duration: const Duration(milliseconds: 100),
          curve: Curves.easeOut,
        );
      }
    });
  }

  void _sendCommand(String command) {
    if (command.isEmpty) return;

    final transport = widget.connectionManager.relayTransport;
    if (transport == null || !transport.isConnected) {
      _addOutput('Not connected.');
      return;
    }

    // Send the command + newline
    final data = Uint8List.fromList((command + '\n').codeUnits);
    transport.sendData(StreamID.shell, data);

    // Echo the command locally
    _addOutput('> $command');
    _inputController.clear();
  }

  @override
  Widget build(BuildContext context) {
    final isConnected = _connectionState == cb.ConnectionState.connected ||
        _connectionState == cb.ConnectionState.relayConnected;

    return Scaffold(
      appBar: AppBar(
        title: Text(widget.deviceName),
        centerTitle: true,
        actions: [
          // Connection status indicator
          Padding(
            padding: const EdgeInsets.all(16),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Container(
                  width: 8,
                  height: 8,
                  decoration: BoxDecoration(
                    color: isConnected ? Colors.green : Colors.orange,
                    shape: BoxShape.circle,
                  ),
                ),
                const SizedBox(width: 6),
                Text(
                  _connectionState.displayName,
                  style: Theme.of(context).textTheme.bodySmall,
                ),
              ],
            ),
          ),
          // Disconnect button
          IconButton(
            icon: const Icon(Icons.close),
            onPressed: () {
              widget.connectionManager.disconnect();
              Navigator.pop(context);
            },
          ),
        ],
      ),
      body: Column(
        children: [
          // Terminal output
          Expanded(
            child: Container(
              color: const Color(0xFF1E1E1E),
              child: ListView.builder(
                controller: _scrollController,
                padding: const EdgeInsets.all(12),
                itemCount: _outputLines.length,
                itemBuilder: (context, index) {
                  return SelectableText(
                    _outputLines[index],
                    style: const TextStyle(
                      fontFamily: 'JetBrains Mono',
                      fontSize: 14,
                      color: Color(0xFF00FF00),
                    ),
                  );
                },
              ),
            ),
          ),
          // Input bar
          Container(
            decoration: BoxDecoration(
              color: const Color(0xFF2D2D2D),
              border: Border(
                top: BorderSide(
                  color: Theme.of(context).colorScheme.outlineVariant,
                ),
              ),
            ),
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            child: Row(
              children: [
                const Text(
                  '> ',
                  style: TextStyle(
                    fontFamily: 'JetBrains Mono',
                    fontSize: 14,
                    color: Color(0xFF00FF00),
                  ),
                ),
                Expanded(
                  child: TextField(
                    controller: _inputController,
                    style: const TextStyle(
                      fontFamily: 'JetBrains Mono',
                      fontSize: 14,
                      color: Color(0xFF00FF00),
                    ),
                    decoration: const InputDecoration(
                      hintText: 'Type a command...',
                      hintStyle: TextStyle(color: Color(0xFF666666)),
                      border: InputBorder.none,
                      isDense: true,
                      contentPadding: EdgeInsets.zero,
                    ),
                    onSubmitted: _sendCommand,
                    enabled: isConnected,
                  ),
                ),
                IconButton(
                  icon: const Icon(Icons.send, size: 20),
                  onPressed: isConnected
                      ? () => _sendCommand(_inputController.text)
                      : null,
                  color: isConnected
                      ? const Color(0xFF00FF00)
                      : Colors.grey,
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}