// Convert Service - Flutter Implementation (Like Binance Convert)

class ConvertPair {
  final String from;
  final String to;
  final double rate;
  final double inverseRate;
  final double fee;
  final bool enabled;

  ConvertPair({
    required this.from,
    required this.to,
    required this.rate,
    required this.inverseRate,
    required this.fee,
    required this.enabled,
  });
}

class ConvertOrder {
  final String id;
  final String userId;
  final String fromToken;
  final String toToken;
  final double fromAmount;
  final double toAmount;
  final double rate;
  final double fee;
  final String status;
  final DateTime createTime;

  ConvertOrder({
    required this.id,
    required this.userId,
    required this.fromToken,
    required this.toToken,
    required this.fromAmount,
    required this.toAmount,
    required this.rate,
    required this.fee,
    required this.status,
    required this.createTime,
  });
}

class Token {
  final String symbol;
  final String name;
  final double balance;
  final String icon;

  Token({
    required this.symbol,
    required this.name,
    required this.balance,
    required this.icon,
  });
}

class ConvertService {
  static final Map<String, ConvertPair> _pairs = {};
  static final Map<String, List<ConvertOrder>> _orders = {};
  static final Map<String, Map<String, double>> _balances = {};

  static void _initializePairs() {
    if (_pairs.isNotEmpty) return;

    final pairData = [
      {'from': 'BTC', 'to': 'USDT', 'rate': 43250.0},
      {'from': 'ETH', 'to': 'USDT', 'rate': 2280.0},
      {'from': 'BNB', 'to': 'USDT', 'rate': 312.5},
      {'from': 'SOL', 'to': 'USDT', 'rate': 98.75},
      {'from': 'XRP', 'to': 'USDT', 'rate': 0.62},
      {'from': 'DOGE', 'to': 'USDT', 'rate': 0.082},
      {'from': 'ADA', 'to': 'USDT', 'rate': 0.58},
      {'from': 'AVAX', 'to': 'USDT', 'rate': 38.20},
      {'from': 'DOT', 'to': 'USDT', 'rate': 7.85},
      {'from': 'LINK', 'to': 'USDT', 'rate': 14.50},
      {'from': 'MATIC', 'to': 'USDT', 'rate': 0.92},
      {'from': 'LTC', 'to': 'USDT', 'rate': 72.30},
      {'from': 'UNI', 'to': 'USDT', 'rate': 6.25},
      {'from': 'ATOM', 'to': 'USDT', 'rate': 10.45},
      {'from': 'XLM', 'to': 'USDT', 'rate': 0.125},
      {'from': 'NEAR', 'to': 'USDT', 'rate': 3.25},
      {'from': 'USDC', 'to': 'USDT', 'rate': 1.0001},
      {'from': 'USDT', 'to': 'USDC', 'rate': 0.9999},
    ];

    for (final p in pairData) {
      final from = p['from'] as String;
      final to = p['to'] as String;
      final rate = p['rate'] as double;
      final key = '${from}_$to';
      _pairs[key] = ConvertPair(
        from: from,
        to: to,
        rate: rate,
        inverseRate: 1 / rate,
        fee: 0.1,
        enabled: true,
      );
    }
  }

  static List<Token> getAvailableTokens() {
    _initializePairs();
    return [
      Token(symbol: 'USDT', name: 'Tether USD', balance: 50000, icon: '💵'),
      Token(symbol: 'USDC', name: 'USD Coin', balance: 25000, icon: '💳'),
      Token(symbol: 'BTC', name: 'Bitcoin', balance: 0.5, icon: '₿'),
      Token(symbol: 'ETH', name: 'Ethereum', balance: 5, icon: 'Ξ'),
      Token(symbol: 'BNB', name: 'BNB', balance: 50, icon: '⬡'),
      Token(symbol: 'SOL', name: 'Solana', balance: 100, icon: '◎'),
      Token(symbol: 'XRP', name: 'Ripple', balance: 10000, icon: '✕'),
      Token(symbol: 'DOGE', name: 'Dogecoin', balance: 100000, icon: 'Ð'),
      Token(symbol: 'ADA', name: 'Cardano', balance: 5000, icon: '₳'),
      Token(symbol: 'AVAX', name: 'Avalanche', balance: 500, icon: '🔺'),
    ];
  }

  static ConvertPair? getPair(String from, String to) {
    _initializePairs();
    final key = '${from}_$to';
    return _pairs[key];
  }

  static (double, double)? getRate(String from, String to) {
    _initializePairs();
    
    // Try direct pair
    final directKey = '${from}_$to';
    if (_pairs.containsKey(directKey) && _pairs[directKey]!.enabled) {
      return (_pairs[directKey]!.rate, _pairs[directKey]!.fee);
    }

    // Try reverse pair
    final reverseKey = '${to}_$from';
    if (_pairs.containsKey(reverseKey) && _pairs[reverseKey]!.enabled) {
      return (_pairs[reverseKey]!.inverseRate, _pairs[reverseKey]!.fee);
    }

    // Try through USDT as intermediate
    final fromUSDTKey = '${from}_USDT';
    final toUSDTKey = 'USDT_$to';

    if (_pairs.containsKey(fromUSDTKey) && _pairs.containsKey(toUSDTKey)) {
      final fromPair = _pairs[fromUSDTKey]!;
      final toPair = _pairs[toUSDTKey]!;
      if (fromPair.enabled && toPair.enabled) {
        final combinedRate = fromPair.rate * toPair.rate;
        final combinedFee = (fromPair.fee + toPair.fee) / 2;
        return (combinedRate, combinedFee);
      }
    }

    return null;
  }

  static ConvertOrder? convert({
    required String userId,
    required String fromToken,
    required String toToken,
    required double fromAmount,
  }) {
    final rateData = getRate(fromToken, toToken);
    if (rateData == null) return null;

    final rate = rateData.$1;
    final feePercent = rateData.$2;
    final fee = fromAmount * feePercent / 100;
    final netAmount = fromAmount - fee;
    final toAmount = netAmount * rate;

    final now = DateTime.now();
    final order = ConvertOrder(
      id: 'convert-${now.millisecondsSinceEpoch}',
      userId: userId,
      fromToken: fromToken,
      toToken: toToken,
      fromAmount: fromAmount,
      toAmount: toAmount,
      rate: rate,
      fee: fee,
      status: 'completed',
      createTime: now,
    );

    _orders[userId] = [...(_orders[userId] ?? []), order];
    return order;
  }

  static List<ConvertOrder> getUserOrders(String userId) {
    return _orders[userId] ?? [];
  }

  static Map<String, double> getUserBalance(String userId) {
    return _balances[userId] ?? {
      'USDT': 50000.0,
      'USDC': 25000.0,
      'BTC': 0.5,
      'ETH': 5.0,
    };
  }
}
