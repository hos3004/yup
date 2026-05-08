import 'package:sqflite_sqlcipher/sqflite.dart';
import '../../features/messaging/domain/conversation_service.dart' show MessageItem;
import 'local_database.dart';

class MessageDao {
  final LocalDatabase _db;

  MessageDao(this._db);

  Future<void> insertMessage(MessageItem msg, String username, String peerCurveKey) async {
    final db = await _db.database;
    await db.insert('messages', {
      'id': msg.id,
      'sender': msg.sender,
      'recipient': username,
      'peer_curve_key': peerCurveKey,
      'text': msg.text,
      'is_outgoing': msg.isOutgoing ? 1 : 0,
      'status': msg.status,
      'created_at': msg.createdAt.millisecondsSinceEpoch,
    }, conflictAlgorithm: ConflictAlgorithm.replace);
  }

  Future<void> updateStatus(String messageId, String status) async {
    final db = await _db.database;
    await db.update(
      'messages',
      {'status': status},
      where: 'id = ?',
      whereArgs: [messageId],
    );
  }

  Future<List<MessageItem>> getConversation({
    required String username,
    required String peerCurveKey,
    int limit = 100,
  }) async {
    final db = await _db.database;
    final rows = await db.query(
      'messages',
      where: 'peer_curve_key = ? AND recipient = ?',
      whereArgs: [peerCurveKey, username],
      orderBy: 'created_at ASC',
      limit: limit,
    );
    return rows.map((r) => MessageItem(
      id: r['id'] as String,
      sender: r['sender'] as String,
      text: r['text'] as String,
      isOutgoing: (r['is_outgoing'] as int) == 1,
      status: r['status'] as String,
      createdAt: DateTime.fromMillisecondsSinceEpoch(r['created_at'] as int),
    )).toList();
  }

  Future<void> deleteConversation(String peerCurveKey) async {
    final db = await _db.database;
    await db.delete('messages', where: 'peer_curve_key = ?', whereArgs: [peerCurveKey]);
  }

  Future<int> messageCount() async {
    final db = await _db.database;
    final result = await db.rawQuery('SELECT COUNT(*) as c FROM messages');
    return result.first['c'] as int;
  }
}
