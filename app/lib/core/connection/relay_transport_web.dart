/// Web stub for Socket operations - throws UnsupportedError.
/// This file is used on web platform where TCP Socket is unavailable.
library;

import 'dart:async';
import 'dart:typed_data';

/// Throws UnsupportedError on web platform.
Future<Stream<List<int>>> connectSocket(
  String host,
  int port, {
  Duration? timeout,
}) async {
  throw UnsupportedError(
    'TCP Socket is not available on web platform. '
    'Please use the Android/iOS app for terminal access.',
  );
}

void socketAdd(dynamic socket, Uint8List data) {
  throw UnsupportedError('TCP Socket not available on web');
}

void socketDestroy(dynamic socket) {
  throw UnsupportedError('TCP Socket not available on web');
}

Future<Uint8List> socketFirst(dynamic socket) async {
  throw UnsupportedError('TCP Socket not available on web');
}

StreamSubscription<Uint8List> socketListen(
  dynamic socket, {
  required void Function(Uint8List) onData,
  required void Function(dynamic) onError,
  required void Function() onDone,
}) {
  throw UnsupportedError('TCP Socket not available on web');
}