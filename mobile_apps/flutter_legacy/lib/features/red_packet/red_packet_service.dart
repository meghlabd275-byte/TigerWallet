// Red Packet Service - Flutter Implementation

class RedPacket {
  final String id;
  final String sender;
  final String senderAddress;
  final String token;
  final double amount;
  final int totalCount;
  final int receivedCount;
  final double remainingAmount;
  final int remainingCount;
  final String message;
  final String type; // random or fixed
  final String link;
  final String status;
  final DateTime createTime;
  final DateTime expireTime;

  RedPacket({
    required this.id,
    required this.sender,
    required this.senderAddress,
    required this.token,
    required this.amount,
    required this.totalCount,
    required this.receivedCount,
    required this.remainingAmount,
    required this.remainingCount,
    required this.message,
    required this.type,
    required this.link,
    required this.status,
    required this.createTime,
    required this.expireTime,
  });

  factory RedPacket.create({
    required String sender,
    required String senderAddress,
    required String token,
    required double amount,
    required int totalCount,
    required String type,
    required String message,
  }) {
    final now = DateTime.now();
    final id = 'rp-${now.millisecondsSinceEpoch}';
    return RedPacket(
      id: id,
      sender: sender,
      senderAddress: senderAddress,
      token: token,
      amount: amount,
      totalCount: totalCount,
      receivedCount: 0,
      remainingAmount: amount,
      remainingCount: totalCount,
      message: message,
      type: type,
      link: 'https://tigerwallet.com/redpacket/claim/$id',
      status: 'active',
      createTime: now,
      expireTime: now.add(const Duration(hours: 24)),
    );
  }

  bool get isExpired => DateTime.now().isAfter(expireTime);
  bool get isActive => status == 'active' && !isExpired;
  bool get isCompleted => remainingCount <= 0;
}

class ClaimRecord {
  final String packetId;
  final String claimer;
  final String claimerAddress;
  final double amount;
  final DateTime claimTime;

  ClaimRecord({
    required this.packetId,
    required this.claimer,
    required this.claimerAddress,
    required this.amount,
    required this.claimTime,
  });
}

class RedPacketService {
  static final Map<String, RedPacket> _packets = {};
  static final Map<String, List<ClaimRecord>> _claims = {};

  static RedPacket createPacket({
    required String sender,
    required String senderAddress,
    required String token,
    required double amount,
    required int totalCount,
    required String type,
    required String message,
  }) {
    final packet = RedPacket.create(
      sender: sender,
      senderAddress: senderAddress,
      token: token,
      amount: amount,
      totalCount: totalCount,
      type: type,
      message: message,
    );
    _packets[packet.id] = packet;
    return packet;
  }

  static RedPacket? getPacket(String packetId) {
    return _packets[packetId];
  }

  static RedPacket? getPacketByLink(String link) {
    final id = link.split('/').last;
    return _packets[id];
  }

  static ClaimRecord? claimPacket({
    required String packetId,
    required String claimer,
    required String claimerAddress,
  }) {
    final packet = _packets[packetId];
    if (packet == null) return null;
    if (!packet.isActive) return null;

    double claimAmount;
    if (packet.type == 'random') {
      if (packet.remainingCount == 1) {
        claimAmount = packet.remainingAmount;
      } else {
        claimAmount = (packet.remainingAmount * 2 / packet.remainingCount) * 
            (DateTime.now().millisecond % 100 / 100);
        if (claimAmount > packet.remainingAmount) {
          claimAmount = packet.remainingAmount / 2;
        }
      }
    } else {
      claimAmount = packet.amount / packet.totalCount;
    }

    final claim = ClaimRecord(
      packetId: packetId,
      claimer: claimer,
      claimerAddress: claimerAddress,
      amount: claimAmount,
      claimTime: DateTime.now(),
    );

    _claims[packetId] = [...(_claims[packetId] ?? []), claim];

    // Update packet
    final updatedPacket = RedPacket(
      id: packet.id,
      sender: packet.sender,
      senderAddress: packet.senderAddress,
      token: packet.token,
      amount: packet.amount,
      totalCount: packet.totalCount,
      receivedCount: packet.receivedCount + 1,
      remainingAmount: packet.remainingAmount - claimAmount,
      remainingCount: packet.remainingCount - 1,
      message: packet.message,
      type: packet.type,
      link: packet.link,
      status: packet.remainingCount - 1 <= 0 ? 'completed' : 'active',
      createTime: packet.createTime,
      expireTime: packet.expireTime,
    );
    _packets[packetId] = updatedPacket;

    return claim;
  }

  static List<ClaimRecord> getPacketClaims(String packetId) {
    return _claims[packetId] ?? [];
  }

  static List<RedPacket> getUserPackets(String userId) {
    return _packets.values.where((p) => p.sender == userId).toList();
  }

  static void expirePackets() {
    _packets.forEach((id, packet) {
      if (packet.isActive && packet.isExpired) {
        _packets[id] = RedPacket(
          id: packet.id,
          sender: packet.sender,
          senderAddress: packet.senderAddress,
          token: packet.token,
          amount: packet.amount,
          totalCount: packet.totalCount,
          receivedCount: packet.receivedCount,
          remainingAmount: packet.remainingAmount,
          remainingCount: packet.remainingCount,
          message: packet.message,
          type: packet.type,
          link: packet.link,
          status: 'expired',
          createTime: packet.createTime,
          expireTime: packet.expireTime,
        );
      }
    });
  }
}
