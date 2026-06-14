import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';

import 'package:cloudbridge/core/protocol/frame.dart';
import 'package:cloudbridge/core/signal/signal_messages.dart';

void main() {
  group('Frame Protocol', () {
    test('encode and decode frame round-trip', () {
      final frame = Frame(
        streamId: StreamID.shell,
        type: FrameType.data,
        payload: Uint8List.fromList('hello world'.codeUnits),
      );

      final encoded = frame.encode();
      final decoded = Frame.decode(encoded);

      expect(decoded.streamId, equals(StreamID.shell));
      expect(decoded.type, equals(FrameType.data));
      expect(decoded.payload, equals('hello world'.codeUnits));
    });

    test('frame header size is 7 bytes', () {
      expect(frameHeaderSize, equals(7));
    });

    test('stream ID constants', () {
      expect(StreamID.control, equals(0x0000));
      expect(StreamID.ssh, equals(0x0001));
      expect(StreamID.shell, equals(0x0002));
      expect(StreamID.rdp, equals(0x0003));
      expect(StreamID.docker, equals(0x0004));
    });
  });

  group('Signal Messages', () {
    test('encode and decode register message', () {
      final msg = SignalMessage(
        type: MsgType.register,
        data: RegisterPayload(
          deviceId: 'test-device',
          deviceName: 'Test PC',
          platform: 'linux',
          version: '0.1.0',
        ).toJson(),
      );

      final encoded = msg.encode();
      final decoded = SignalMessage.decode(encoded);

      expect(decoded.type, equals(MsgType.register));
      expect(decoded.data!['device_id'], equals('test-device'));
    });

    test('message type constants', () {
      expect(MsgType.register, equals('register'));
      expect(MsgType.connectRequest, equals('connect_request'));
      expect(MsgType.sdpOffer, equals('sdp_offer'));
    });
  });
}