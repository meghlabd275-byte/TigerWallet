// Fiat On-Ramp Service - Flutter Implementation
// Buy crypto with fiat currency

class FiatProvider {
  final String id;
  final String name;
  final String logo;
  final List<String> supportedFiat;
  final List<String> supportedCrypto;
  final double minAmount;
  final double maxAmount;
  final double feePercent;
  final String processingTime;
  final List<String> paymentMethods;
  final bool isAvailable;

  FiatProvider({
    required this.id,
    required this.name,
    required this.logo,
    required this.supportedFiat,
    required this.supportedCrypto,
    required this.minAmount,
    required this.maxAmount,
    required this.feePercent,
    required this.processingTime,
    required this.paymentMethods,
    required this.isAvailable,
  });
}

class FiatOrder {
  final String id;
  final String userId;
  final String providerId;
  final String providerName;
  final String side;
  final String fiatCurrency;
  final String cryptoCurrency;
  final double fiatAmount;
  final double cryptoAmount;
  final double exchangeRate;
  final double fee;
  final String paymentMethod;
  final String status;
  final String? walletAddress;
  final String? txHash;
  final String? bankReference;
  final DateTime createTime;
  final DateTime? confirmTime;
  final DateTime? completeTime;

  FiatOrder({
    required this.id,
    required this.userId,
    required this.providerId,
    required this.providerName,
    required this.side,
    required this.fiatCurrency,
    required this.cryptoCurrency,
    required this.fiatAmount,
    required this.cryptoAmount,
    required this.exchangeRate,
    required this.fee,
    required this.paymentMethod,
    required this.status,
    this.walletAddress,
    this.txHash,
    this.bankReference,
    required this.createTime,
    this.confirmTime,
    this.completeTime,
  });
}

class FiatRampService {
  static final Map<String, List<FiatOrder>> _orders = {};
  
  static List<FiatProvider> getProviders() {
    return [
      FiatProvider(
        id: 'provider_1',
        name: 'MoonPay',
        logo: '🌙',
        supportedFiat: ['USD', 'EUR', 'GBP', 'AUD', 'CAD'],
        supportedCrypto: ['BTC', 'ETH', 'USDT', 'USDC', 'BNB', 'SOL', 'XRP', 'ADA'],
        minAmount: 30,
        maxAmount: 50000,
        feePercent: 2.5,
        processingTime: '5-30 minutes',
        paymentMethods: ['Bank Card', 'Bank Transfer', 'Apple Pay', 'Google Pay'],
        isAvailable: true,
      ),
      FiatProvider(
        id: 'provider_2',
        name: 'Simplex',
        logo: '💳',
        supportedFiat: ['USD', 'EUR', 'GBP'],
        supportedCrypto: ['BTC', 'ETH', 'USDT', 'USDC'],
        minAmount: 50,
        maxAmount: 25000,
        feePercent: 3.5,
        processingTime: '10-60 minutes',
        paymentMethods: ['Bank Card', 'Apple Pay', 'Google Pay'],
        isAvailable: true,
      ),
      FiatProvider(
        id: 'provider_3',
        name: 'Transak',
        logo: '🔄',
        supportedFiat: ['USD', 'EUR', 'GBP', 'INR', 'CNY'],
        supportedCrypto: ['BTC', 'ETH', 'USDT', 'MATIC', 'AVAX', 'SOL'],
        minAmount: 20,
        maxAmount: 100000,
        feePercent: 2.0,
        processingTime: '15-45 minutes',
        paymentMethods: ['Bank Transfer', 'Credit/Debit Card', 'UPI', 'WeChat Pay'],
        isAvailable: true,
      ),
      FiatProvider(
        id: 'provider_4',
        name: 'OnRamper',
        logo: '📱',
        supportedFiat: ['USD', 'EUR', 'GBP', 'AUD', 'JPY'],
        supportedCrypto: ['BTC', 'ETH', 'USDT', 'BNB', 'ADA', 'DOT'],
        minAmount: 25,
        maxAmount: 75000,
        feePercent: 1.8,
        processingTime: '5-20 minutes',
        paymentMethods: ['Bank Card', 'Apple Pay', 'Samsung Pay'],
        isAvailable: true,
      ),
    ];
  }

  static List<FiatProvider> getProvidersByFiat(String fiatCurrency) {
    return getProviders().where((p) => p.supportedFiat.contains(fiatCurrency)).toList();
  }

  static List<FiatProvider> getProvidersByCrypto(String cryptoCurrency) {
    return getProviders().where((p) => p.supportedCrypto.contains(cryptoCurrency)).toList();
  }

  static double calculateExchangeRate({
    required String providerId,
    required String fiatCurrency,
    required String cryptoCurrency,
    required double fiatAmount,
  }) {
    final provider = getProviders().firstWhere((p) => p.id == providerId);
    final baseRate = _getBaseRate(cryptoCurrency);
    final fee = fiatAmount * (provider.feePercent / 100);
    final netAmount = fiatAmount - fee;
    return netAmount / baseRate;
  }

  static double _getBaseRate(String crypto) {
    final rates = {
      'BTC': 43250.0, 'ETH': 2280.0, 'USDT': 1.0, 'USDC': 1.0,
      'BNB': 312.5, 'SOL': 98.75, 'XRP': 0.62, 'ADA': 0.58,
      'MATIC': 0.92, 'AVAX': 38.2, 'DOT': 7.85, 'LINK': 14.50,
    };
    return rates[crypto] ?? 10.0;
  }

  static Future<FiatOrder> createBuyOrder({
    required String userId,
    required String providerId,
    required String fiatCurrency,
    required String cryptoCurrency,
    required double fiatAmount,
    required String paymentMethod,
    required String walletAddress,
  }) async {
    final providers = getProviders();
    final provider = providers.firstWhere((p) => p.id == providerId, orElse: () => providers.first);
    
    final exchangeRate = calculateExchangeRate(
      providerId: providerId,
      fiatCurrency: fiatCurrency,
      cryptoCurrency: cryptoCurrency,
      fiatAmount: fiatAmount,
    );
    
    final cryptoAmount = (fiatAmount * (1 - provider.feePercent / 100)) / exchangeRate;
    final fee = fiatAmount * (provider.feePercent / 100);
    
    final order = FiatOrder(
      id: 'fiat_order_${DateTime.now().millisecondsSinceEpoch}',
      userId: userId,
      providerId: providerId,
      providerName: provider.name,
      side: 'BUY',
      fiatCurrency: fiatCurrency,
      cryptoCurrency: cryptoCurrency,
      fiatAmount: fiatAmount,
      cryptoAmount: cryptoAmount,
      exchangeRate: exchangeRate,
      fee: fee,
      paymentMethod: paymentMethod,
      status: 'PENDING',
      walletAddress: walletAddress,
      createTime: DateTime.now(),
    );
    
    _orders[userId] = [...(_orders[userId] ?? []), order];
    return order;
  }

  static Future<FiatOrder> createSellOrder({
    required String userId,
    required String providerId,
    required String fiatCurrency,
    required String cryptoCurrency,
    required double cryptoAmount,
    required String paymentMethod,
  }) async {
    final providers = getProviders();
    final provider = providers.firstWhere((p) => p.id == providerId, orElse: () => providers.first);
    
    final baseRate = _getBaseRate(cryptoCurrency);
    final fiatAmount = cryptoAmount * baseRate * (1 - provider.feePercent / 100);
    
    final order = FiatOrder(
      id: 'fiat_order_${DateTime.now().millisecondsSinceEpoch}',
      userId: userId,
      providerId: providerId,
      providerName: provider.name,
      side: 'SELL',
      fiatCurrency: fiatCurrency,
      cryptoCurrency: cryptoCurrency,
      fiatAmount: fiatAmount,
      cryptoAmount: cryptoAmount,
      exchangeRate: baseRate,
      fee: fiatAmount * (provider.feePercent / 100),
      paymentMethod: paymentMethod,
      status: 'PENDING',
      createTime: DateTime.now(),
    );
    
    _orders[userId] = [...(_orders[userId] ?? []), order];
    return order;
  }

  static Future<FiatOrder?> getOrder(String userId, String orderId) async {
    final orders = _orders[userId] ?? [];
    try {
      return orders.firstWhere((o) => o.id == orderId);
    } catch (e) {
      return null;
    }
  }

  static Future<List<FiatOrder>> getOrders(String userId) async {
    return _orders[userId] ?? [];
  }

  static Future<FiatOrder> confirmPayment(String userId, String orderId, String bankReference) async {
    final orders = _orders[userId] ?? [];
    final index = orders.indexWhere((o) => o.id == orderId);
    
    if (index != -1) {
      final order = orders[index];
      orders[index] = FiatOrder(
        id: order.id, userId: order.userId, providerId: order.providerId,
        providerName: order.providerName, side: order.side, fiatCurrency: order.fiatCurrency,
        cryptoCurrency: order.cryptoCurrency, fiatAmount: order.fiatAmount,
        cryptoAmount: order.cryptoAmount, exchangeRate: order.exchangeRate,
        fee: order.fee, paymentMethod: order.paymentMethod, status: 'AWAITING_CONFIRMATION',
        walletAddress: order.walletAddress, bankReference: bankReference,
        createTime: order.createTime, confirmTime: DateTime.now(),
        completeTime: order.completeTime,
      );
      _orders[userId] = orders;
      return orders[index];
    }
    throw Exception('Order not found');
  }

  static Future<FiatOrder> completeOrder(String userId, String orderId, String txHash) async {
    final orders = _orders[userId] ?? [];
    final index = orders.indexWhere((o) => o.id == orderId);
    
    if (index != -1) {
      final order = orders[index];
      orders[index] = FiatOrder(
        id: order.id, userId: order.userId, providerId: order.providerId,
        providerName: order.providerName, side: order.side, fiatCurrency: order.fiatCurrency,
        cryptoCurrency: order.cryptoCurrency, fiatAmount: order.fiatAmount,
        cryptoAmount: order.cryptoAmount, exchangeRate: order.exchangeRate,
        fee: order.fee, paymentMethod: order.paymentMethod, status: 'COMPLETED',
        walletAddress: order.walletAddress, txHash: txHash, bankReference: order.bankReference,
        createTime: order.createTime, confirmTime: order.confirmTime,
        completeTime: DateTime.now(),
      );
      _orders[userId] = orders;
      return orders[index];
    }
    throw Exception('Order not found');
  }

  static Future<FiatOrder> cancelOrder(String userId, String orderId) async {
    final orders = _orders[userId] ?? [];
    final index = orders.indexWhere((o) => o.id == orderId);
    
    if (index != -1) {
      final order = orders[index];
      orders[index] = FiatOrder(
        id: order.id, userId: order.userId, providerId: order.providerId,
        providerName: order.providerName, side: order.side, fiatCurrency: order.fiatCurrency,
        cryptoCurrency: order.cryptoCurrency, fiatAmount: order.fiatAmount,
        cryptoAmount: order.cryptoAmount, exchangeRate: order.exchangeRate,
        fee: order.fee, paymentMethod: order.paymentMethod, status: 'CANCELLED',
        walletAddress: order.walletAddress, txHash: order.txHash,
        bankReference: order.bankReference, createTime: order.createTime,
        confirmTime: order.confirmTime, completeTime: DateTime.now(),
      );
      _orders[userId] = orders;
      return orders[index];
    }
    throw Exception('Order not found');
  }

  static List<String> getSupportedFiatCurrencies() {
    return ['USD', 'EUR', 'GBP', 'AUD', 'CAD', 'JPY', 'CNY', 'INR', 'KRW', 'BRL'];
  }

  static List<String> getSupportedCryptoCurrencies() {
    return ['BTC', 'ETH', 'USDT', 'USDC', 'BNB', 'SOL', 'XRP', 'ADA', 'MATIC', 'AVAX', 'DOT', 'LINK'];
  }
}
