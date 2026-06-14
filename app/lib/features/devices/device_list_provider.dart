/// Device list provider - manages the list of devices from the REST API.
library;

import 'package:flutter/foundation.dart';

import '../../core/api/device_api.dart';
import '../../core/models/device.dart';

class DeviceListProvider extends ChangeNotifier {
  DeviceApi _api;

  List<Device> _devices = [];
  bool _isLoading = false;
  String? _error;

  DeviceListProvider({required String baseUrl}) : _api = DeviceApi(baseUrl: baseUrl);

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
  void updateBaseUrl(String baseUrl) {
    _api.dispose();
    _api = DeviceApi(baseUrl: baseUrl);
    refresh();
  }

  @override
  void dispose() {
    _api.dispose();
    super.dispose();
  }
}