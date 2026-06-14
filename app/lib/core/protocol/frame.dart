/// Frame protocol for CloudBridge multiplexed stream transport.
/// Aligned with Go's internal/protocol/frame.go
library;

import 'dart:typed_data';

/// Stream IDs for well-known channels
class StreamID {
  static const int control = 0x0000;
  static const int ssh = 0x0001;
  static const int shell = 0x0002;
  static const int rdp = 0x0003;
  static const int docker = 0x0004;
  static const int vnc = 0x0005;
}

/// Frame types
class FrameType {
  static const int data = 0x01;
  static const int openStream = 0x02;
  static const int closeStream = 0x03;
  static const int windowUpdate = 0x04;
  static const int ping = 0x05;
  static const int pong = 0x06;
}

/// Frame header size: StreamID(2) + Type(1) + Length(4) = 7 bytes
const int frameHeaderSize = 7;

/// Maximum payload size: 64 KiB
const int maxPayloadSize = 64 * 1024;

/// A frame in the CloudBridge wire protocol.
///
/// Wire format:
/// +----------+--------+-----------+----------+
/// | StreamID | Type   | Length    | Payload  |
/// | 2 bytes  | 1 byte | 4 bytes   | N bytes  |
/// +----------+--------+-----------+----------+
class Frame {
  final int streamId;
  final int type;
  final Uint8List payload;

  Frame({
    required this.streamId,
    required this.type,
    required this.payload,
  });

  /// Encode the frame to its wire format.
  Uint8List encode() {
    if (payload.length > maxPayloadSize) {
      throw ArgumentError(
          'Payload too large: ${payload.length} > $maxPayloadSize');
    }

    final buf = Uint8List(frameHeaderSize + payload.length);
    final view = ByteData.view(buf.buffer);

    // StreamID (2 bytes, big-endian)
    view.setUint16(0, streamId);
    // Type (1 byte)
    buf[2] = type;
    // Length (4 bytes, big-endian)
    view.setUint32(3, payload.length);

    // Payload
    buf.setRange(frameHeaderSize, frameHeaderSize + payload.length, payload);

    return buf;
  }

  /// Decode a frame from its wire format.
  static Frame decode(Uint8List data) {
    if (data.length < frameHeaderSize) {
      throw FormatException('Frame too short: ${data.length} < $frameHeaderSize');
    }

    final view = ByteData.view(data.buffer, data.offsetInBytes, data.length);
    final streamId = view.getUint16(0);
    final type = data[2];
    final length = view.getUint32(3);

    if (data.length < frameHeaderSize + length) {
      throw FormatException(
          'Frame truncated: expected $length bytes payload, got ${data.length - frameHeaderSize}');
    }
    if (length > maxPayloadSize) {
      throw FormatException('Payload too large: $length > $maxPayloadSize');
    }

    final payload = Uint8List.sublistView(
        data, frameHeaderSize, frameHeaderSize + length);

    return Frame(streamId: streamId, type: type, payload: payload);
  }

  @override
  String toString() => 'Frame(streamId=$streamId, type=$type, payloadLen=${payload.length})';
}

/// Read a frame from a byte buffer, returning the frame and remaining bytes.
/// Returns null if the buffer doesn't contain a complete frame.
FrameReadResult? tryReadFrame(Uint8List buffer) {
  if (buffer.length < frameHeaderSize) {
    return null; // Not enough data for header
  }

  final view = ByteData.view(buffer.buffer, buffer.offsetInBytes, buffer.length);
  final length = view.getUint32(3);

  if (buffer.length < frameHeaderSize + length) {
    return null; // Not enough data for payload
  }

  final frameData = Uint8List.sublistView(buffer, 0, frameHeaderSize + length);
  final frame = Frame.decode(frameData);
  final remaining = buffer.length - frameHeaderSize - length;

  return FrameReadResult(
    frame: frame,
    consumedBytes: frameHeaderSize + length,
    remainingBytes: remaining,
  );
}

class FrameReadResult {
  final Frame frame;
  final int consumedBytes;
  final int remainingBytes;

  FrameReadResult({
    required this.frame,
    required this.consumedBytes,
    required this.remainingBytes,
  });
}