/// IO implementation of Socket connect - used on native platforms.
library;

import 'dart:async';
import 'dart:io';
import 'dart:typed_data';

/// Connects to a TCP socket. Only available on native platforms.
Future<Stream<List<int>>> connectSocket(
  String host,
  int port, {
  Duration? timeout,
}) async {
  final socket = await Socket.connect(host, port, timeout: timeout);
  return socket;
}

/// Send data through a socket.
void socketAdd(dynamic socket, Uint8List data) {
  (socket as Socket).add(data);
}

/// Destroy a socket connection.
void socketDestroy(dynamic socket) {
  (socket as Socket).destroy();
}

/// Get the first element from a socket stream.
Future<Uint8List> socketFirst(dynamic socket) async {
  final data = await (socket as Socket).first;
  return Uint8List.fromList(data);
}

/// Listen to a socket stream.
StreamSubscription<Uint8List> socketListen(
  dynamic socket, {
  required void Function(Uint8List) onData,
  required void Function(dynamic) onError,
  required void Function() onDone,
}) {
  return (socket as Socket).listen(
    (data) => onData(Uint8List.fromList(data)),
    onError: onError,
    onDone: onDone,
  );
}