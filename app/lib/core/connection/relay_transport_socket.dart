/// Default socket stub - will be overridden by platform-specific imports.
library;

import 'dart:async';
import 'dart:typed_data';

/// Default stub that throws UnsupportedError.
/// This should never be called directly - the conditional import
/// will select the correct implementation.
Future<Stream<List<int>>> connectSocket(
  String host,
  int port, {
  Duration? timeout,
}) async {
  throw UnsupportedError('Socket not available on this platform');
}

void socketAdd(dynamic socket, Uint8List data) {
  throw UnsupportedError('Socket not available on this platform');
}

void socketDestroy(dynamic socket) {
  throw UnsupportedError('Socket not available on this platform');
}

Future<Uint8List> socketFirst(dynamic socket) async {
  throw UnsupportedError('Socket not available on this platform');
}

StreamSubscription<Uint8List> socketListen(
  dynamic socket, {
  required void Function(Uint8List) onData,
  required void Function(dynamic) onError,
  required void Function() onDone,
}) {
  throw UnsupportedError('Socket not available on this platform');
}