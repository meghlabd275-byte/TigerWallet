// Crypto Card Service - Flutter Implementation
// Virtual and Physical Crypto Cards with Apple Pay/Google Pay support

class CryptoCard {
  final String id;
  final String userId;
  final String cardNumber;
  final String cardHolder;
  final String expiryDate;
  final String cvv;
  final String type;
  final String network;
  final String status;
  final double dailyLimit;
  final double monthlyLimit;
  final double dailySpent;
  final double monthlySpent;
  final String? virtualCardImage;
  final bool applePayEnabled;
  final bool googlePayEnabled;
  final DateTime createdAt;

  CryptoCard({
    required this.id,
    required this.userId,
    required this.cardNumber,
    required this.cardHolder,
    required this.expiryDate,
    required this.cvv,
    required this.type,
    required this.network,
    required this.status,
    required this.dailyLimit,
    required this.monthlyLimit,
    required this.dailySpent,
    required this.monthlySpent,
    this.virtualCardImage,
    required this.applePayEnabled,
    required this.googlePayEnabled,
    required this.createdAt,
  });

  factory CryptoCard.fromJson(Map<String, dynamic> json) {
    return CryptoCard(
      id: json['id'] ?? '',
      userId: json['userId'] ?? '',
      cardNumber: json['cardNumber'] ?? '',
      cardHolder: json['cardHolder'] ?? '',
      expiryDate: json['expiryDate'] ?? '',
      cvv: json['cvv'] ?? '',
      type: json['type'] ?? 'VIRTUAL',
      network: json['network'] ?? 'VISA',
      status: json['status'] ?? 'ACTIVE',
      dailyLimit: (json['dailyLimit'] ?? 10000).toDouble(),
      monthlyLimit: (json['monthlyLimit'] ?? 100000).toDouble(),
      dailySpent: (json['dailySpent'] ?? 0).toDouble(),
      monthlySpent: (json['monthlySpent'] ?? 0).toDouble(),
      virtualCardImage: json['virtualCardImage'],
      applePayEnabled: json['applePayEnabled'] ?? false,
      googlePayEnabled: json['googlePayEnabled'] ?? false,
      createdAt: json['createdAt'] != null 
        ? DateTime.parse(json['createdAt']) 
        : DateTime.now(),
    );
  }

  String get maskedNumber => '•••• •••• •••• ${cardNumber.substring(cardNumber.length - 4)}';
}

class CardTransaction {
  final String id;
  final String cardId;
  final String userId;
  final double amount;
  final String currency;
  final String merchantName;
  final String merchantCategory;
  final String type;
  final String status;
  final String? receiptUrl;
  final DateTime timestamp;
  final String? location;

  CardTransaction({
    required this.id,
    required this.cardId,
    required this.userId,
    required this.amount,
    required this.currency,
    required this.merchantName,
    required this.merchantCategory,
    required this.type,
    required this.status,
    this.receiptUrl,
    required this.timestamp,
    this.location,
  });
}

class CardFundingSource {
  final String id;
  final String userId;
  final String token;
  final String symbol;
  final double balance;
  final double dailyLimit;
  final double monthlyLimit;
  final bool isDefault;

  CardFundingSource({
    required this.id,
    required this.userId,
    required this.token,
    required this.symbol,
    required this.balance,
    required this.dailyLimit,
    required this.monthlyLimit,
    required this.isDefault,
  });
}

class CardSettings {
  final String userId;
  final bool internationalEnabled;
  final bool onlinePaymentsEnabled;
  final bool contactlessEnabled;
  final double defaultDailyLimit;
  final double defaultMonthlyLimit;
  final List<String> blockedMerchants;

  CardSettings({
    required this.userId,
    required this.internationalEnabled,
    required this.onlinePaymentsEnabled,
    required this.contactlessEnabled,
    required this.defaultDailyLimit,
    required this.defaultMonthlyLimit,
    required this.blockedMerchants,
  });
}

class CardService {
  static final Map<String, List<CryptoCard>> _cards = {};
  static final Map<String, List<CardTransaction>> _transactions = {};
  static final Map<String, List<CardFundingSource>> _fundingSources = {};
  static final Map<String, CardSettings> _settings = {};

  static String _generateCardNumber() {
    final prefix = '4532';
    final random = DateTime.now().millisecondsSinceEpoch.toString();
    return '$prefix${random.substring(0, 12)}';
  }

  static String _generateCVV() {
    return (DateTime.now().millisecondsSinceEpoch % 1000).toString().padLeft(3, '0');
  }

  static String _generateExpiry() {
    final now = DateTime.now();
    final year = (now.year + 3).toString().substring(2);
    final month = (now.month + 1).toString().padLeft(2, '0');
    return '$month/$year';
  }

  static Future<List<CryptoCard>> getCards(String userId) async {
    if (_cards.containsKey(userId)) {
      return _cards[userId]!;
    }
    return [];
  }

  static Future<CryptoCard> createVirtualCard({
    required String userId,
    required String cardHolder,
    required String type,
    required String network,
    double dailyLimit = 10000,
    double monthlyLimit = 100000,
  }) async {
    final card = CryptoCard(
      id: 'card_${DateTime.now().millisecondsSinceEpoch}',
      userId: userId,
      cardNumber: _generateCardNumber(),
      cardHolder: cardHolder,
      expiryDate: _generateExpiry(),
      cvv: _generateCVV(),
      type: type,
      network: network,
      status: 'ACTIVE',
      dailyLimit: dailyLimit,
      monthlyLimit: monthlyLimit,
      dailySpent: 0,
      monthlySpent: 0,
      applePayEnabled: true,
      googlePayEnabled: true,
      createdAt: DateTime.now(),
    );
    
    _cards[userId] = [...(_cards[userId] ?? []), card];
    return card;
  }

  static Future<CryptoCard> createPhysicalCard({
    required String userId,
    required String cardHolder,
    required String shippingAddress,
    double dailyLimit = 10000,
    double monthlyLimit = 100000,
  }) async {
    final card = CryptoCard(
      id: 'card_physical_${DateTime.now().millisecondsSinceEpoch}',
      userId: userId,
      cardNumber: _generateCardNumber(),
      cardHolder: cardHolder,
      expiryDate: _generateExpiry(),
      cvv: _generateCVV(),
      type: 'PHYSICAL',
      network: 'VISA',
      status: 'PENDING_ACTIVATION',
      dailyLimit: dailyLimit,
      monthlyLimit: monthlyLimit,
      dailySpent: 0,
      monthlySpent: 0,
      applePayEnabled: true,
      googlePayEnabled: true,
      createdAt: DateTime.now(),
    );
    
    _cards[userId] = [...(_cards[userId] ?? []), card];
    return card;
  }

  static Future<void> freezeCard(String userId, String cardId) async {
    final cards = _cards[userId] ?? [];
    final index = cards.indexWhere((c) => c.id == cardId);
    if (index != -1) {
      final card = cards[index];
      cards[index] = CryptoCard(
        id: card.id,
        userId: card.userId,
        cardNumber: card.cardNumber,
        cardHolder: card.cardHolder,
        expiryDate: card.expiryDate,
        cvv: card.cvv,
        type: card.type,
        network: card.network,
        status: 'FROZEN',
        dailyLimit: card.dailyLimit,
        monthlyLimit: card.monthlyLimit,
        dailySpent: card.dailySpent,
        monthlySpent: card.monthlySpent,
        virtualCardImage: card.virtualCardImage,
        applePayEnabled: card.applePayEnabled,
        googlePayEnabled: card.googlePayEnabled,
        createdAt: card.createdAt,
      );
      _cards[userId] = cards;
    }
  }

  static Future<void> unfreezeCard(String userId, String cardId) async {
    final cards = _cards[userId] ?? [];
    final index = cards.indexWhere((c) => c.id == cardId);
    if (index != -1) {
      final card = cards[index];
      cards[index] = CryptoCard(
        id: card.id,
        userId: card.userId,
        cardNumber: card.cardNumber,
        cardHolder: card.cardHolder,
        expiryDate: card.expiryDate,
        cvv: card.cvv,
        type: card.type,
        network: card.network,
        status: 'ACTIVE',
        dailyLimit: card.dailyLimit,
        monthlyLimit: card.monthlyLimit,
        dailySpent: card.dailySpent,
        monthlySpent: card.monthlySpent,
        virtualCardImage: card.virtualCardImage,
        applePayEnabled: card.applePayEnabled,
        googlePayEnabled: card.googlePayEnabled,
        createdAt: card.createdAt,
      );
      _cards[userId] = cards;
    }
  }

  static Future<void> terminateCard(String userId, String cardId) async {
    final cards = _cards[userId] ?? [];
    final index = cards.indexWhere((c) => c.id == cardId);
    if (index != -1) {
      final card = cards[index];
      cards[index] = CryptoCard(
        id: card.id,
        userId: card.userId,
        cardNumber: card.cardNumber,
        cardHolder: card.cardHolder,
        expiryDate: card.expiryDate,
        cvv: card.cvv,
        type: card.type,
        network: card.network,
        status: 'TERMINATED',
        dailyLimit: card.dailyLimit,
        monthlyLimit: card.monthlyLimit,
        dailySpent: card.dailySpent,
        monthlySpent: card.monthlySpent,
        virtualCardImage: card.virtualCardImage,
        applePayEnabled: false,
        googlePayEnabled: false,
        createdAt: card.createdAt,
      );
      _cards[userId] = cards;
    }
  }

  static Future<List<CardTransaction>> getTransactions(String userId, {String? cardId}) async {
    final allTxns = _transactions[userId] ?? [];
    if (cardId != null) {
      return allTxns.where((t) => t.cardId == cardId).toList();
    }
    return allTxns;
  }

  static Future<CardTransaction> processPayment({
    required String userId,
    required String cardId,
    required double amount,
    required String currency,
    required String merchantName,
    required String merchantCategory,
  }) async {
    final cards = _cards[userId] ?? [];
    final cardIndex = cards.indexWhere((c) => c.id == cardId);
    
    if (cardIndex == -1) {
      throw Exception('Card not found');
    }
    
    final card = cards[cardIndex];
    if (card.status != 'ACTIVE') {
      throw Exception('Card is not active');
    }
    
    if (card.dailySpent + amount > card.dailyLimit) {
      throw Exception('Daily limit exceeded');
    }
    
    if (card.monthlySpent + amount > card.monthlyLimit) {
      throw Exception('Monthly limit exceeded');
    }
    
    final transaction = CardTransaction(
      id: 'txn_${DateTime.now().millisecondsSinceEpoch}',
      cardId: cardId,
      userId: userId,
      amount: amount,
      currency: currency,
      merchantName: merchantName,
      merchantCategory: merchantCategory,
      type: 'PURCHASE',
      status: 'COMPLETED',
      timestamp: DateTime.now(),
    );
    
    _transactions[userId] = [...(_transactions[userId] ?? []), transaction];
    
    final updatedCard = CryptoCard(
      id: card.id,
      userId: card.userId,
      cardNumber: card.cardNumber,
      cardHolder: card.cardHolder,
      expiryDate: card.expiryDate,
      cvv: card.cvv,
      type: card.type,
      network: card.network,
      status: card.status,
      dailyLimit: card.dailyLimit,
      monthlyLimit: card.monthlyLimit,
      dailySpent: card.dailySpent + amount,
      monthlySpent: card.monthlySpent + amount,
      virtualCardImage: card.virtualCardImage,
      applePayEnabled: card.applePayEnabled,
      googlePayEnabled: card.googlePayEnabled,
      createdAt: card.createdAt,
    );
    cards[cardIndex] = updatedCard;
    _cards[userId] = cards;
    
    return transaction;
  }

  static Future<List<CardFundingSource>> getFundingSources(String userId) async {
    return _fundingSources[userId] ?? [];
  }

  static Future<void> addFundingSource({
    required String userId,
    required String token,
    required String symbol,
    required double balance,
    bool isDefault = false,
  }) async {
    final source = CardFundingSource(
      id: 'fund_${DateTime.now().millisecondsSinceEpoch}',
      userId: userId,
      token: token,
      symbol: symbol,
      balance: balance,
      dailyLimit: 10000,
      monthlyLimit: 100000,
      isDefault: isDefault,
    );
    
    final sources = _fundingSources[userId] ?? [];
    if (isDefault) {
      for (int i = 0; i < sources.length; i++) {
        sources[i] = CardFundingSource(
          id: sources[i].id,
          userId: sources[i].userId,
          token: sources[i].token,
          symbol: sources[i].symbol,
          balance: sources[i].balance,
          dailyLimit: sources[i].dailyLimit,
          monthlyLimit: sources[i].monthlyLimit,
          isDefault: false,
        );
      }
    }
    _fundingSources[userId] = [...sources, source];
  }

  static Future<CardSettings> getCardSettings(String userId) async {
    if (_settings.containsKey(userId)) {
      return _settings[userId]!;
    }
    
    final settings = CardSettings(
      userId: userId,
      internationalEnabled: true,
      onlinePaymentsEnabled: true,
      contactlessEnabled: true,
      defaultDailyLimit: 10000,
      defaultMonthlyLimit: 100000,
      blockedMerchants: [],
    );
    _settings[userId] = settings;
    return settings;
  }

  static Future<void> updateCardSettings(String userId, CardSettings settings) async {
    _settings[userId] = settings;
  }

  static Future<void> setCardLimit({
    required String userId,
    required String cardId,
    required double dailyLimit,
    required double monthlyLimit,
  }) async {
    final cards = _cards[userId] ?? [];
    final index = cards.indexWhere((c) => c.id == cardId);
    if (index != -1) {
      final card = cards[index];
      cards[index] = CryptoCard(
        id: card.id,
        userId: card.userId,
        cardNumber: card.cardNumber,
        cardHolder: card.cardHolder,
        expiryDate: card.expiryDate,
        cvv: card.cvv,
        type: card.type,
        network: card.network,
        status: card.status,
        dailyLimit: dailyLimit,
        monthlyLimit: monthlyLimit,
        dailySpent: card.dailySpent,
        monthlySpent: card.monthlySpent,
        virtualCardImage: card.virtualCardImage,
        applePayEnabled: card.applePayEnabled,
        googlePayEnabled: card.googlePayEnabled,
        createdAt: card.createdAt,
      );
      _cards[userId] = cards;
    }
  }

  static Future<String> getVirtualCardImage(String cardId) async {
    return 'https://api.tigerwallet.com/cards/$cardId/image';
  }

  static Future<void> enableApplePay(String userId, String cardId) async {
    final cards = _cards[userId] ?? [];
    final index = cards.indexWhere((c) => c.id == cardId);
    if (index != -1) {
      final card = cards[index];
      cards[index] = CryptoCard(
        id: card.id,
        userId: card.userId,
        cardNumber: card.cardNumber,
        cardHolder: card.cardHolder,
        expiryDate: card.expiryDate,
        cvv: card.cvv,
        type: card.type,
        network: card.network,
        status: card.status,
        dailyLimit: card.dailyLimit,
        monthlyLimit: card.monthlyLimit,
        dailySpent: card.dailySpent,
        monthlySpent: card.monthlySpent,
        virtualCardImage: card.virtualCardImage,
        applePayEnabled: true,
        googlePayEnabled: card.googlePayEnabled,
        createdAt: card.createdAt,
      );
      _cards[userId] = cards;
    }
  }

  static Future<void> enableGooglePay(String userId, String cardId) async {
    final cards = _cards[userId] ?? [];
    final index = cards.indexWhere((c) => c.id == cardId);
    if (index != -1) {
      final card = cards[index];
      cards[index] = CryptoCard(
        id: card.id,
        userId: card.userId,
        cardNumber: card.cardNumber,
        cardHolder: card.cardHolder,
        expiryDate: card.expiryDate,
        cvv: card.cvv,
        type: card.type,
        network: card.network,
        status: card.status,
        dailyLimit: card.dailyLimit,
        monthlyLimit: card.monthlyLimit,
        dailySpent: card.dailySpent,
        monthlySpent: card.monthlySpent,
        virtualCardImage: card.virtualCardImage,
        applePayEnabled: card.applePayEnabled,
        googlePayEnabled: true,
        createdAt: card.createdAt,
      );
      _cards[userId] = cards;
    }
  }
}
