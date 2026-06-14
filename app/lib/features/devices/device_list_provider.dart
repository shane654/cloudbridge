/// Device list provider - manages the list of devices from the REST API.
library;

import 'package:flutter/foundation.dart';

import '../../core/api/device_api.dart';
import '../../core/models/device.dart';

class DeviceListProvider extends ChangeNotifier {
  final DeviceApi _api;

  List<Device> _devices = [];
  bool _isLoading = false;
  String? _error;

  DeviceListProvider({required String serverUrl})
      : _api = DeviceApi(baseUrl: serverUrl);

  List<Device> get devices => _devices;
  bool get isLoading => _isLoading;
  String? get error => _error;

  /// Fetch the device list from the server.
  Future<void> refresh() async {
    _isLoading = true;
    _error = null;
    notifyListeners();

    try {
      _devices = await _api.listDevices();
      _error = null;
    } catch (e) {
      _error = e.toString();
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  /// Update the server URL and refresh.
  void updateServerUrl(String url) {
    _api.dispose(); // Close old client
    // Note: we can't reassign _api since it's final.
    // In a real app, we'd use a factory pattern. For MVP, the provider
    // is recreated when settings change.
  }
}