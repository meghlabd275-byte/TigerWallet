// P2P Trading Service - Flutter Implementation
// Peer-to-peer trading between users

class P2PAdvert {
  final String id;
  final String userId;
  final String username;
  final String avatar;
  final String side;
  final String token;
  final String fiatCurrency;
  final String paymentMethod;
  final double price;
  final double minAmount;
  final double maxAmount;
  final double availableAmount;
  final int ordersCompleted;
  final double completionRate;
  final double avgReleaseTime;
  final bool isOnline;
  final DateTime createTime;

  P2PAdvert({
    required this.id,
    required this.userId,
    required this.username,
    required this.avatar,
    required this.side,
    required this.token,
    required this.fiatCurrency,
    required this.paymentMethod,
    required this.price,
    required this.minAmount,
    required this.maxAmount,
    required this.availableAmount,
    required this.ordersCompleted,
    required this.completionRate,
    required this.avgReleaseTime,
    required this.isOnline,
    required this.createTime,
  });
}

class P2POrder {
  final String id;
  final String advertId;
  final String makerId;
  final String takerId;
  final String side;
  final String token;
  final String fiatCurrency;
  final String paymentMethod;
  final double price;
  final double amount;
  final double fiatAmount;
  final String status;
  final String? chatRoomId;
  final DateTime createTime;
  final DateTime? payTime;
  final DateTime? releaseTime;
  final DateTime? cancelTime;

  P2POrder({
    required this.id,
    required this.advertId,
    required this.makerId,
    required this.takerId,
    required this.side,
    required this.token,
    required this.fiatCurrency,
    required this.paymentMethod,
    required this.price,
    required this.amount,
    required this.fiatAmount,
    required this.status,
    this.chatRoomId,
    required this.createTime,
    this.payTime,
    this.releaseTime,
    this.cancelTime,
  });
}

class P2PChatMessage {
  final String id;
  final String chatRoomId;
  final String senderId;
  final String content;
  final DateTime timestamp;

  P2PChatMessage({
    required this.id,
    required this.chatRoomId,
    required this.senderId,
    required this.content,
    required this.timestamp,
  });
}

class P2PTrade {
  static final Map<String, List<P2PAdvert>> _adverts = {};
  static final Map<String, List<P2POrder>> _orders = {};
  static final Map<String, List<P2PChatMessage>> _messages = {};

  static List<P2PAdvert> generateAdverts({String? token, String? side, String? fiatCurrency}) {
    final List<P2PAdvert> adverts = [];
    
    final users = [
      {'username': 'CryptoTrader1', 'avatar': '🧑‍💼', 'online': true},
      {'username': 'BitSeller', 'avatar': '👨‍💻', 'online': true},
      {'username': 'FastTrade', 'avatar': '⚡', 'online': false},
      {'username': 'P2PPro', 'avatar': '🎯', 'online': true},
      {'username': 'SecureDeal', 'avatar': '🔒', 'online': true},
    ];
    
    final tokens = token != null ? [token] : ['BTC', 'ETH', 'USDT', 'USDC', 'BNB'];
    final currencies = fiatCurrency != null ? [fiatCurrency] : ['USD', 'EUR', 'GBP', 'CNY', 'INR'];
    final payments = ['Bank Transfer', 'PayPal', 'AliPay', 'WeChat Pay', 'UPI', 'Gift Card'];
    
    final basePrices = {'BTC': 43250.0, 'ETH': 2280.0, 'USDT': 1.0, 'USDC': 1.0, 'BNB': 312.5};
    
    int id = 0;
    for (final user in users) {
      for (final tok in tokens) {
        for (final curr in currencies) {
          final priceVariation = (DateTime.now().millisecond % 10 - 5) / 1000.0;
          final basePrice = basePrices[tok] ?? 10.0;
          final isBuy = (id % 2) == 0;
          
          adverts.add(P2PAdvert(
            id: 'advert_${id++}',
            userId: 'user_${users.indexOf(user)}',
            username: user['username'] as String,
            avatar: user['avatar'] as String,
            side: side ?? (isBuy ? 'BUY' : 'SELL'),
            token: tok,
            fiatCurrency: curr,
            paymentMethod: payments[id % payments.length],
            price: basePrice * (1 + priceVariation),
            minAmount: curr == 'USD' ? 10 : 100,
            maxAmount: curr == 'USD' ? 5000 : 50000,
            availableAmount: (id * 0.5 + 1) * basePrice,
            ordersCompleted: 50 + (id * 10),
            completionRate: 95.0 + (id % 5),
            avgReleaseTime: 2.0 + (id % 10),
            isOnline: user['online'] as bool,
            createTime: DateTime.now().subtract(Duration(days: id)),
          ));
        }
      }
    }
    return adverts;
  }

  static Future<List<P2PAdvert>> getAdverts({
    String? token,
    String? side,
    String? fiatCurrency,
    String? paymentMethod,
  }) async {
    var adverts = generateAdverts(token: token, side: side, fiatCurrency: fiatCurrency);
    if (paymentMethod != null) {
      adverts = adverts.where((a) => a.paymentMethod == paymentMethod).toList();
    }
    return adverts;
  }

  static Future<P2PAdvert> createAdvert({
    required String userId,
    required String side,
    required String token,
    required String fiatCurrency,
    required String paymentMethod,
    required double price,
    required double minAmount,
    required double maxAmount,
    required double availableAmount,
  }) async {
    final advert = P2PAdvert(
      id: 'advert_${DateTime.now().millisecondsSinceEpoch}',
      userId: userId,
      username: 'MyUser',
      avatar: '👤',
      side: side,
      token: token,
      fiatCurrency: fiatCurrency,
      paymentMethod: paymentMethod,
      price: price,
      minAmount: minAmount,
      maxAmount: maxAmount,
      availableAmount: availableAmount,
      ordersCompleted: 0,
      completionRate: 100.0,
      avgReleaseTime: 5.0,
      isOnline: true,
      createTime: DateTime.now(),
    );
    _adverts[userId] = [...(_adverts[userId] ?? []), advert];
    return advert;
  }

  static Future<P2POrder> createOrder({
    required String advertId,
    required String takerId,
    required double amount,
  }) async {
    final adverts = generateAdverts();
    final advert = adverts.firstWhere((a) => a.id == advertId, orElse: () => adverts.first);
    
    final order = P2POrder(
      id: 'order_${DateTime.now().millisecondsSinceEpoch}',
      advertId: advertId,
      makerId: advert.userId,
      takerId: takerId,
      side: advert.side == 'BUY' ? 'SELL' : 'BUY',
      token: advert.token,
      fiatCurrency: advert.fiatCurrency,
      paymentMethod: advert.paymentMethod,
      price: advert.price,
      amount: amount,
      fiatAmount: amount * advert.price,
      status: 'PENDING',
      chatRoomId: 'chat_${DateTime.now().millisecondsSinceEpoch}',
      createTime: DateTime.now(),
    );
    _orders[takerId] = [...(_orders[takerId] ?? []), order];
    return order;
  }

  static Future<P2POrder> markAsPaid(String orderId, String userId) async {
    final orders = _orders[userId] ?? [];
    final index = orders.indexWhere((o) => o.id == orderId);
    if (index != -1) {
      final order = orders[index];
      orders[index] = P2POrder(
        id: order.id, advertId: order.advertId, makerId: order.makerId, takerId: order.takerId,
        side: order.side, token: order.token, fiatCurrency: order.fiatCurrency,
        paymentMethod: order.paymentMethod, price: order.price, amount: order.amount,
        fiatAmount: order.fiatAmount, status: 'PAID', chatRoomId: order.chatRoomId,
        createTime: order.createTime, payTime: DateTime.now(), releaseTime: order.releaseTime,
        cancelTime: order.cancelTime,
      );
      _orders[userId] = orders;
      return orders[index];
    }
    throw Exception('Order not found');
  }

  static Future<P2POrder> releaseCrypto(String orderId, String userId) async {
    final orders = _orders[userId] ?? [];
    final index = orders.indexWhere((o) => o.id == orderId);
    if (index != -1) {
      final order = orders[index];
      orders[index] = P2POrder(
        id: order.id, advertId: order.advertId, makerId: order.makerId, takerId: order.takerId,
        side: order.side, token: order.token, fiatCurrency: order.fiatCurrency,
        paymentMethod: order.paymentMethod, price: order.price, amount: order.amount,
        fiatAmount: order.fiatAmount, status: 'COMPLETED', chatRoomId: order.chatRoomId,
        createTime: order.createTime, payTime: order.payTime, releaseTime: DateTime.now(),
        cancelTime: order.cancelTime,
      );
      _orders[userId] = orders;
      return orders[index];
    }
    throw Exception('Order not found');
  }

  static Future<P2POrder> cancelOrder(String orderId, String userId) async {
    final orders = _orders[userId] ?? [];
    final index = orders.indexWhere((o) => o.id == orderId);
    if (index != -1) {
      final order = orders[index];
      orders[index] = P2POrder(
        id: order.id, advertId: order.advertId, makerId: order.makerId, takerId: order.takerId,
        side: order.side, token: order.token, fiatCurrency: order.fiatCurrency,
        paymentMethod: order.paymentMethod, price: order.price, amount: order.amount,
        fiatAmount: order.fiatAmount, status: 'CANCELLED', chatRoomId: order.chatRoomId,
        createTime: order.createTime, payTime: order.payTime, releaseTime: order.releaseTime,
        cancelTime: DateTime.now(),
      );
      _orders[userId] = orders;
      return orders[index];
    }
    throw Exception('Order not found');
  }

  static Future<List<P2POrder>> getOrders(String userId) async {
    return _orders[userId] ?? [];
  }

  static Future<List<P2PChatMessage>> getMessages(String chatRoomId) async {
    return _messages[chatRoomId] ?? [];
  }

  static Future<P2PChatMessage> sendMessage({
    required String chatRoomId,
    required String senderId,
    required String content,
  }) async {
    final message = P2PChatMessage(
      id: 'msg_${DateTime.now().millisecondsSinceEpoch}',
      chatRoomId: chatRoomId,
      senderId: senderId,
      content: content,
      timestamp: DateTime.now(),
    );
    _messages[chatRoomId] = [...(_messages[chatRoomId] ?? []), message];
    return message;
  }
}
