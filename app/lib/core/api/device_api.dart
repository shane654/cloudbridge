/// REST API client for CloudBridge server.
library;

import 'dart:convert';

import 'package:http/http.dart' as http;

import '../models/device.dart';
import '../models/session.dart';

/// Client for the CloudBridge REST API.
class DeviceApi {
  final String baseUrl;
  final http.Client _client;

  DeviceApi({required this.baseUrl, http.Client? client})
      : _client = client ?? http.Client();

  /// Fetch the list of registered devices.
  Future<List<Device>> listDevices() async {
    final response = await _client.get(
      Uri.parse('$baseUrl/api/v1/devices'),
      headers: {'Content-Type': 'application/json'},
    );

    if (response.statusCode != 200) {
      throw Exception('Failed to list devices: ${response.statusCode}');
    }

    final List<dynamic> jsonList = jsonDecode(response.body) as List<dynamic>;
    return jsonList
        .map((json) => Device.fromJson(json as Map<String, dynamic>))
        .toList();
  }

  /// Fetch a specific device by ID.
  Future<Device> getDevice(String deviceId) async {
    final response = await _client.get(
      Uri.parse('$baseUrl/api/v1/devices/$deviceId'),
      headers: {'Content-Type': 'application/json'},
    );

    if (response.statusCode != 200) {
      throw Exception('Failed to get device: ${response.statusCode}');
    }

    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return Device.fromJson(json);
  }

  /// Fetch the list of active sessions.
  Future<List<Session>> listSessions() async {
    final response = await _client.get(
      Uri.parse('$baseUrl/api/v1/sessions'),
      headers: {'Content-Type': 'application/json'},
    );

    if (response.statusCode != 200) {
      throw Exception('Failed to list sessions: ${response.statusCode}');
    }

    final List<dynamic> jsonList = jsonDecode(response.body) as List<dynamic>;
    return jsonList
        .map((json) => Session.fromJson(json as Map<String, dynamic>))
        .toList();
  }

  /// Fetch server statistics.
  Future<Map<String, dynamic>> getStats() async {
    final response = await _client.get(
      Uri.parse('$baseUrl/api/v1/stats'),
      headers: {'Content-Type': 'application/json'},
    );

    if (response.statusCode != 200) {
      throw Exception('Failed to get stats: ${response.statusCode}');
    }

    return jsonDecode(response.body) as Map<String, dynamic>;
  }

  /// Health check.
  Future<bool> healthCheck() async {
    try {
      final response = await _client.get(
        Uri.parse('$baseUrl/health'),
      );
      return response.statusCode == 200;
    } catch (_) {
      return false;
    }
  }

  void dispose() {
    _client.close();
  }
}