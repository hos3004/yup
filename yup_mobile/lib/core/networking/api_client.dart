import 'dart:convert';
import 'dart:io';

class ApiClient {
  final String baseUrl;
  final HttpClient _client;
  String? _token;

  ApiClient(this.baseUrl) : _client = HttpClient();

  void setToken(String token) {
    _token = token;
  }

  void clearToken() {
    _token = null;
  }

  Map<String, String> get _headers => {
        'Content-Type': 'application/json',
        if (_token != null) 'Authorization': 'Bearer $_token',
      };

  Future<Map<String, dynamic>> registerUser(String username) async {
    final request = await _client.postUrl(Uri.parse('$baseUrl/api/v1/users'));
    request.headers.contentType = ContentType.json;
    request.write(jsonEncode({'username': username}));
    final response = await request.close();
    final body = await response.transform(utf8.decoder).join();
    if (response.statusCode == 201) {
      final data = jsonDecode(body) as Map<String, dynamic>;
      _token = data['auth_token'] as String;
      return data;
    }
    throw HttpException('register failed: ${response.statusCode} $body');
  }

  Future<Map<String, dynamic>> getUser(String username) async {
    final request = await _client.getUrl(Uri.parse('$baseUrl/api/v1/users/$username'));
    final response = await request.close();
    final body = await response.transform(utf8.decoder).join();
    if (response.statusCode == 200) {
      return jsonDecode(body) as Map<String, dynamic>;
    }
    throw HttpException('user not found: $username');
  }

  Future<Map<String, dynamic>> uploadKeys(String username, Map<String, dynamic> keyBundle) async {
    final request = await _client.putUrl(Uri.parse('$baseUrl/api/v1/keys/$username'));
    _headers.forEach((k, v) => request.headers.set(k, v));
    request.write(jsonEncode(keyBundle));
    final response = await request.close();
    final body = await response.transform(utf8.decoder).join();
    if (response.statusCode == 200) {
      return jsonDecode(body) as Map<String, dynamic>;
    }
    throw HttpException('key upload failed: ${response.statusCode} $body');
  }

  Future<Map<String, dynamic>> getKeys(String username) async {
    final request = await _client.getUrl(Uri.parse('$baseUrl/api/v1/keys/$username'));
    _headers.forEach((k, v) => request.headers.set(k, v));
    final response = await request.close();
    final body = await response.transform(utf8.decoder).join();
    if (response.statusCode == 200) {
      return jsonDecode(body) as Map<String, dynamic>;
    }
    throw HttpException('keys not found for: $username');
  }

  Future<Map<String, dynamic>> sendMessage({
    required String recipient,
    required String ciphertext,
    required int messageType,
    required String senderKey,
  }) async {
    final request = await _client.postUrl(Uri.parse('$baseUrl/api/v1/messages'));
    _headers.forEach((k, v) => request.headers.set(k, v));
    request.write(jsonEncode({
      'recipient': recipient,
      'ciphertext': ciphertext,
      'message_type': messageType,
      'sender_key': senderKey,
    }));
    final response = await request.close();
    final body = await response.transform(utf8.decoder).join();
    if (response.statusCode == 201) {
      return jsonDecode(body) as Map<String, dynamic>;
    }
    throw HttpException('send failed: ${response.statusCode} $body');
  }

  Future<List<dynamic>> getMessages() async {
    final request = await _client.getUrl(Uri.parse('$baseUrl/api/v1/messages'));
    _headers.forEach((k, v) => request.headers.set(k, v));
    final response = await request.close();
    final body = await response.transform(utf8.decoder).join();
    if (response.statusCode == 200) {
      return jsonDecode(body) as List<dynamic>;
    }
    throw HttpException('get messages failed: ${response.statusCode} $body');
  }

  Future<Map<String, dynamic>> ackMessage(String messageId) async {
    final request = await _client.postUrl(
      Uri.parse('$baseUrl/api/v1/messages/$messageId/ack'),
    );
    _headers.forEach((k, v) => request.headers.set(k, v));
    final response = await request.close();
    final body = await response.transform(utf8.decoder).join();
    if (response.statusCode == 200) {
      return jsonDecode(body) as Map<String, dynamic>;
    }
    throw HttpException('ack failed: ${response.statusCode} $body');
  }

  Future<List<dynamic>> getSentMessages() async {
    final request = await _client.getUrl(
      Uri.parse('$baseUrl/api/v1/messages/sent'),
    );
    _headers.forEach((k, v) => request.headers.set(k, v));
    final response = await request.close();
    final body = await response.transform(utf8.decoder).join();
    if (response.statusCode == 200) {
      return jsonDecode(body) as List<dynamic>;
    }
    throw HttpException('get sent messages failed: ${response.statusCode} $body');
  }
}
