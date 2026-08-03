// Margin Trading Service - Flutter Implementation
// Supports Cross/Isolated Margin, Long/Short, Leverage 1-125x

class MarginPair {
  final String id;
  final String base;
  final String quote;
  final String symbol;
  final double price;
  final double change24h;
  final double volume24h;
  final double high24h;
  final double low24h;
  final double borrowable;
  final double borrowLimit;
  final double currentBorrow;
  final double interestRate;
  final String status;
  final bool isPreInstalled;
  final double minOrderSize;
  final double maxOrderSize;
  final double makerFee;
  final double takerFee;

  MarginPair({
    required this.id,
    required this.base,
    required this.quote,
    required this.symbol,
    required this.price,
    required this.change24h,
    required this.volume24h,
    required this.high24h,
    required this.low24h,
    required this.borrowable,
    required this.borrowLimit,
    required this.currentBorrow,
    required this.interestRate,
    required this.status,
    required this.isPreInstalled,
    required this.minOrderSize,
    required this.maxOrderSize,
    required this.makerFee,
    required this.takerFee,
  });

  factory MarginPair.fromJson(Map<String, dynamic> json) {
    return MarginPair(
      id: json['id'] ?? '',
      base: json['base'] ?? '',
      quote: json['quote'] ?? '',
      symbol: json['symbol'] ?? '',
      price: (json['price'] ?? 0).toDouble(),
      change24h: (json['change24h'] ?? 0).toDouble(),
      volume24h: (json['volume24h'] ?? 0).toDouble(),
      high24h: (json['high24h'] ?? 0).toDouble(),
      low24h: (json['low24h'] ?? 0).toDouble(),
      borrowable: (json['borrowable'] ?? 0).toDouble(),
      borrowLimit: (json['borrowLimit'] ?? 0).toDouble(),
      currentBorrow: (json['currentBorrow'] ?? 0).toDouble(),
      interestRate: (json['interestRate'] ?? 0).toDouble(),
      status: json['status'] ?? 'active',
      isPreInstalled: json['isPreInstalled'] ?? false,
      minOrderSize: (json['minOrderSize'] ?? 0.001).toDouble(),
      maxOrderSize: (json['maxOrderSize'] ?? 1000000).toDouble(),
      makerFee: (json['makerFee'] ?? 0.02).toDouble(),
      takerFee: (json['takerFee'] ?? 0.04).toDouble(),
    );
  }
}

class MarginPosition {
  final String id;
  final String userId;
  final String symbol;
  final String side;
  final double size;
  final double entryPrice;
  final double markPrice;
  final int leverage;
  final double margin;
  final double isolatedMargin;
  final double borrowAmount;
  final String marginMode;
  final double pnl;
  final double pnlPercent;
  final double liquidationPrice;
  final DateTime openTime;

  MarginPosition({
    required this.id,
    required this.userId,
    required this.symbol,
    required this.side,
    required this.size,
    required this.entryPrice,
    required this.markPrice,
    required this.leverage,
    required this.margin,
    required this.isolatedMargin,
    required this.borrowAmount,
    required this.marginMode,
    required this.pnl,
    required this.pnlPercent,
    required this.liquidationPrice,
    required this.openTime,
  });
}

class MarginOrder {
  final String id;
  final String userId;
  final String symbol;
  final String side;
  final String type;
  final double size;
  final double price;
  final double filled;
  final String status;
  final int leverage;
  final String marginMode;
  final double? stopPrice;
  final DateTime createTime;

  MarginOrder({
    required this.id,
    required this.userId,
    required this.symbol,
    required this.side,
    required this.type,
    required this.size,
    required this.price,
    required this.filled,
    required this.status,
    required this.leverage,
    required this.marginMode,
    this.stopPrice,
    required this.createTime,
  });
}

class MarginAccount {
  final String userId;
  final double totalAssets;
  final double totalLiabilities;
  final double netAssets;
  final double availableBalance;
  final double totalBorrowed;
  final double interestAccrued;
  final double marginRatio;
  final double liquidationThreshold;
  final String riskLevel;

  MarginAccount({
    required this.userId,
    required this.totalAssets,
    required this.totalLiabilities,
    required this.netAssets,
    required this.availableBalance,
    required this.totalBorrowed,
    required this.interestAccrued,
    required this.marginRatio,
    required this.liquidationThreshold,
    required this.riskLevel,
  });
}

class BorrowRecord {
  final String id;
  final String userId;
  final String symbol;
  final double amount;
  final double interest;
  final DateTime borrowTime;
  final DateTime? repayTime;
  final String status;

  BorrowRecord({
    required this.id,
    required this.userId,
    required this.symbol,
    required this.amount,
    required this.interest,
    required this.borrowTime,
    this.repayTime,
    required this.status,
  });
}

class MarginTradingService {
  static final Map<String, List<MarginPosition>> _positions = {};
  static final Map<String, List<MarginOrder>> _orders = {};
  static final Map<String, MarginAccount> _accounts = {};
  static final Map<String, List<BorrowRecord>> _borrows = {};

  static List<MarginPair> generatePairs() {
    final List<MarginPair> pairs = [];
    final bases = [
      'BTC', 'ETH', 'BNB', 'SOL', 'XRP', 'DOGE', 'ADA', 'AVAX', 'DOT', 'LINK',
      'MATIC', 'LTC', 'UNI', 'ATOM', 'XLM', 'NEAR', 'APT', 'ARB', 'OP', 'INJ',
    ];
    final quotes = ['USDT', 'USDC', 'BTC', 'ETH'];
    final prices = {
      'BTC': 43250.0, 'ETH': 2280.0, 'BNB': 312.5, 'SOL': 98.75, 'XRP': 0.62,
      'DOGE': 0.082, 'ADA': 0.58, 'AVAX': 38.20, 'DOT': 7.85, 'LINK': 14.50,
      'MATIC': 0.92, 'LTC': 72.30, 'UNI': 6.25, 'ATOM': 10.45, 'XLM': 0.125,
      'NEAR': 3.25, 'APT': 9.80, 'ARB': 1.12, 'OP': 2.45, 'INJ': 35.50,
    };

    int id = 0;
    for (int i = 0; i < bases.length; i++) {
      for (int j = 0; j < quotes.length; j++) {
        final base = bases[i];
        final quote = quotes[j];
        if (base == quote) continue;
        
        final symbol = '$base$quote';
        final price = prices[base] ?? 10.0;
        final borrowable = price * 1000000;
        
        pairs.add(MarginPair(
          id: 'margin_${id++}',
          base: base,
          quote: quote,
          symbol: symbol,
          price: quote == 'BTC' || quote == 'ETH' ? 1.0 / price : price,
          change24h: (DateTime.now().millisecond % 10 - 5).toDouble(),
          volume24h: (DateTime.now().millisecond % 1000000).toDouble(),
          high24h: price * 1.05,
          low24h: price * 0.95,
          borrowable: borrowable,
          borrowLimit: borrowable * 0.8,
          currentBorrow: borrowable * 0.1,
          interestRate: 0.0001,
          status: 'active',
          isPreInstalled: i < 50,
          minOrderSize: 0.001,
          maxOrderSize: 1000000,
          makerFee: 0.02,
          takerFee: 0.04,
        ));
      }
    }
    return pairs;
  }

  static Future<MarginAccount> getAccount(String userId) async {
    if (_accounts.containsKey(userId)) {
      return _accounts[userId]!;
    }
    
    final account = MarginAccount(
      userId: userId,
      totalAssets: 50000.0,
      totalLiabilities: 5000.0,
      netAssets: 45000.0,
      availableBalance: 40000.0,
      totalBorrowed: 5000.0,
      interestAccrued: 0.5,
      marginRatio: 9.0,
      liquidationThreshold: 1.1,
      riskLevel: 'SAFE',
    );
    _accounts[userId] = account;
    return account;
  }

  static Future<List<MarginPosition>> getPositions(String userId) async {
    return _positions[userId] ?? [];
  }

  static Future<List<MarginOrder>> getOrders(String userId) async {
    return _orders[userId] ?? [];
  }

  static Future<List<BorrowRecord>> getBorrowHistory(String userId) async {
    return _borrows[userId] ?? [];
  }

  static Future<MarginOrder> openPosition({
    required String userId,
    required String symbol,
    required String side,
    required double size,
    required double price,
    required int leverage,
    required String marginMode,
  }) async {
    final order = MarginOrder(
      id: 'margin_order_${DateTime.now().millisecondsSinceEpoch}',
      userId: userId,
      symbol: symbol,
      side: side,
      type: 'MARKET',
      size: size,
      price: price,
      filled: 0,
      status: 'PENDING',
      leverage: leverage,
      marginMode: marginMode,
      createTime: DateTime.now(),
    );
    
    _orders[userId] = [...(_orders[userId] ?? []), order];
    return order;
  }

  static Future<MarginOrder> closePosition({
    required String userId,
    required String orderId,
    required double size,
  }) async {
    final orders = _orders[userId] ?? [];
    final index = orders.indexWhere((o) => o.id == orderId);
    if (index != -1) {
      final order = orders[index];
      final closedOrder = MarginOrder(
        id: order.id,
        userId: order.userId,
        symbol: order.symbol,
        side: order.side == 'LONG' ? 'SHORT' : 'LONG',
        type: 'MARKET',
        size: size,
        price: order.price,
        filled: size,
        status: 'FILLED',
        leverage: order.leverage,
        marginMode: order.marginMode,
        createTime: order.createTime,
      );
      orders[index] = closedOrder;
      _orders[userId] = orders;
      return closedOrder;
    }
    throw Exception('Order not found');
  }

  static Future<BorrowRecord> borrow({
    required String userId,
    required String symbol,
    required double amount,
  }) async {
    final record = BorrowRecord(
      id: 'borrow_${DateTime.now().millisecondsSinceEpoch}',
      userId: userId,
      symbol: symbol,
      amount: amount,
      interest: amount * 0.0001,
      borrowTime: DateTime.now(),
      status: 'ACTIVE',
    );
    
    _borrows[userId] = [...(_borrows[userId] ?? []), record];
    return record;
  }

  static Future<BorrowRecord> repay({
    required String userId,
    required String borrowId,
  }) async {
    final borrows = _borrows[userId] ?? [];
    final index = borrows.indexWhere((b) => b.id == borrowId);
    if (index != -1) {
      final borrow = borrows[index];
      final repaid = BorrowRecord(
        id: borrow.id,
        userId: borrow.userId,
        symbol: borrow.symbol,
        amount: borrow.amount,
        interest: borrow.interest,
        borrowTime: borrow.borrowTime,
        repayTime: DateTime.now(),
        status: 'REPAID',
      );
      borrows[index] = repaid;
      _borrows[userId] = borrows;
      return repaid;
    }
    throw Exception('Borrow record not found');
  }

  static Future<MarginOrder> placeOrder({
    required String userId,
    required String symbol,
    required String side,
    required String type,
    required double size,
    required double price,
    int leverage = 1,
    String marginMode = 'CROSS',
    double? stopPrice,
  }) async {
    final order = MarginOrder(
      id: 'margin_order_${DateTime.now().millisecondsSinceEpoch}',
      userId: userId,
      symbol: symbol,
      side: side,
      type: type,
      size: size,
      price: price,
      filled: 0,
      status: 'PENDING',
      leverage: leverage,
      marginMode: marginMode,
      stopPrice: stopPrice,
      createTime: DateTime.now(),
    );
    
    _orders[userId] = [...(_orders[userId] ?? []), order];
    return order;
  }

  static Future<void> cancelOrder(String userId, String orderId) async {
    final orders = _orders[userId] ?? [];
    final index = orders.indexWhere((o) => o.id == orderId);
    if (index != -1) {
      final order = orders[index];
      orders[index] = MarginOrder(
        id: order.id,
        userId: order.userId,
        symbol: order.symbol,
        side: order.side,
        type: order.type,
        size: order.size,
        price: order.price,
        filled: order.filled,
        status: 'CANCELLED',
        leverage: order.leverage,
        marginMode: order.marginMode,
        stopPrice: order.stopPrice,
        createTime: order.createTime,
      );
      _orders[userId] = orders;
    }
  }

  static double calculateLiquidationPrice({
    required double entryPrice,
    required int leverage,
    required String side,
    required double margin,
    required double borrowAmount,
  }) {
    final totalMargin = margin + borrowAmount;
    final positionSize = totalMargin * leverage;
    final liquidationPercent = 1 / leverage;
    
    if (side == 'LONG') {
      return entryPrice * (1 - liquidationPercent);
    } else {
      return entryPrice * (1 + liquidationPercent);
    }
  }

  static double calculatePnL({
    required double entryPrice,
    required double closePrice,
    required double size,
    required String side,
  }) {
    if (side == 'LONG') {
      return (closePrice - entryPrice) * size;
    } else {
      return (entryPrice - closePrice) * size;
    }
  }

  static double calculateMarginRatio({
    required double totalAssets,
    required double totalLiabilities,
  }) {
    if (totalLiabilities == 0) return 999.0;
    return totalAssets / totalLiabilities;
  }
}
