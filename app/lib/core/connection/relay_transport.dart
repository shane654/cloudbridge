/// Relay transport client for CloudBridge.
/// Connects to the relay server via TCP, performs binary handshake,
/// and sends/receives frames.
library;

import 'dart:async';
import 'dart:io';
import 'dart:typed_data';

import '../protocol/frame.dart' show Frame, FrameType, FrameReadResult, StreamID, frameHeaderSize, maxPayloadSize, tryReadFrame;

/// Relay handshake constants (aligned with Go's internal/relay/handshake.go)
const String _handshakeMagic = 'CBLD';
const int _handshakeVersion = 0x01;
const int _peerTypeInitiator = 0x01;
const int _peerTypeResponder = 0x02;

/// Transport for relay connections.
class RelayTransport {
  final String address;
  final String sessionId;
  final bool isInitiator;

  Socket? _socket;
  StreamSubscription<Uint8List>? _subscription;
  final _frameController = StreamController<Frame>.broadcast();
  final _buffer = BytesBuilder();
  bool _connected = false;

  RelayTransport({
    required this.address,
    required this.sessionId,
    this.isInitiator = true,
  });

  /// Whether the transport is connected.
  bool get isConnected => _connected;

  /// Stream of incoming frames.
  Stream<Frame> get onFrame => _frameController.stream;

  /// Connect to the relay server and perform handshake.
  Future<void> connect() async {
    if (_connected) return;

    _socket = await Socket.connect(
      _parseHost(address),
      _parsePort(address),
      timeout: const Duration(seconds: 10),
    );

    // Perform handshake
    _writeHandshake();

    // Read handshake acknowledgment
    await _readHandshakeAck();

    _connected = true;

    // Start reading frames
    _subscription = _socket!.listen(
      _onData,
      onError: _onError,
      onDone: _onDone,
    );
  }

  /// Send a frame over the relay connection.
  void sendFrame(Frame frame) {
    if (!_connected || _socket == null) return;
    _socket!.add(frame.encode());
  }

  /// Send raw data on a specific stream.
  void sendData(int streamId, Uint8List data) {
    final frame = Frame(
      streamId: streamId,
      type: FrameType.data,
      payload: data,
    );
    sendFrame(frame);
  }

  /// Open a new stream.
  void openStream(int streamId) {
    final frame = Frame(
      streamId: streamId,
      type: FrameType.openStream,
      payload: Uint8List(0),
    );
    sendFrame(frame);
  }

  /// Close a stream.
  void closeStream(int streamId) {
    final frame = Frame(
      streamId: streamId,
      type: FrameType.closeStream,
      payload: Uint8List(0),
    );
    sendFrame(frame);
  }

  /// Disconnect from the relay server.
  Future<void> disconnect() async {
    _connected = false;
    await _subscription?.cancel();
    _socket?.destroy();
    _socket = null;
    if (!_frameController.isClosed) {
      await _frameController.close();
    }
  }

  void _writeHandshake() {
    final idBytes = Uint8List.fromList(sessionId.codeUnits);
    final buf = BytesBuilder();

    // Magic (4 bytes)
    buf.add(Uint8List.fromList(_handshakeMagic.codeUnits));
    // Version (1 byte)
    buf.addByte(_handshakeVersion);
    // Session ID length (2 bytes, big-endian)
    buf.addByte((idBytes.length >> 8) & 0xFF);
    buf.addByte(idBytes.length & 0xFF);
    // Session ID
    buf.add(idBytes);
    // Peer type (1 byte)
    buf.addByte(isInitiator ? _peerTypeInitiator : _peerTypeResponder);

    _socket!.add(buf.takeBytes());
  }

  Future<void> _readHandshakeAck() async {
    // Read 5 bytes: magic(4) + status(1)
    final header = await _readExactly(5);

    final magic = String.fromCharCodes(header.sublist(0, 4));
    if (magic != _handshakeMagic) {
      throw FormatException('Invalid relay handshake magic: $magic');
    }

    final status = header[4];
    if (status == 0x00) {
      // OK
      return;
    }

    // Error: read error message
    final msgLenBuf = await _readExactly(2);
    final msgLen = (msgLenBuf[0] << 8) | msgLenBuf[1];
    final msgBuf = await _readExactly(msgLen);
    final msg = String.fromCharCodes(msgBuf);

    throw Exception('Relay handshake failed: $msg');
  }

  Future<Uint8List> _readExactly(int count) async {
    final data = <int>[];
    while (data.length < count) {
      if (_socket == null) {
        throw Exception('Socket disconnected during handshake');
      }
      // Wait for data
      final chunk = await _socket!.first;
      data.addAll(chunk);
    }
    return Uint8List.fromList(data.sublist(0, count));
  }

  void _onData(Uint8List data) {
    _buffer.add(data);
    _processBuffer();
  }

  void _processBuffer() {
    while (true) {
      final result = tryReadFrame(Uint8List.fromList(_buffer.takeBytes()));
      if (result == null) {
        // Not enough data, wait for more
        // (buffer was consumed by takeBytes, we need to put it back)
        break;
      }

      // Re-add remaining bytes to buffer
      final allBytes = _buffer.takeBytes();
      final remaining = allBytes.sublist(result.consumedBytes);
      if (remaining.isNotEmpty) {
        _buffer.add(remaining);
      }

      _frameController.add(result.frame);

      if (result.remainingBytes == 0) {
        break;
      }
    }
  }

  void _onError(dynamic error) {
    _connected = false;
    _frameController.addError(error);
  }

  void _onDone() {
    _connected = false;
    _frameController.close();
  }

  String _parseHost(String addr) {
    final parts = addr.split(':');
    return parts[0];
  }

  int _parsePort(String addr) {
    final parts = addr.split(':');
    return parts.length > 1 ? int.parse(parts[1]) : 10988;
  }
}