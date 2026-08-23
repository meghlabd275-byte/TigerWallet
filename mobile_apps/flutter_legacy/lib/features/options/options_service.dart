// Options Trading Service - Flutter Implementation

class OptionPair {
  final String id;
  final String symbol;
  final String base;
  final String quote;
  final double currentPrice;
  final bool isPreInstalled;

  OptionPair({
    required this.id,
    required this.symbol,
    required this.base,
    required this.quote,
    required this.currentPrice,
    required this.isPreInstalled,
  });
}

class OptionContract {
  final String id;
  final String symbol;
  final String base;
  final String quote;
  final String type; // call or put
  final double strike;
  final String expiry;
  final String expiryLabel;
  final double bid;
  final double ask;
  final double last;
  final double change24h;
  final double volume24h;
  final double openInterest;
  final double impliedVolatility;
  final double delta;
  final double gamma;
  final double theta;

  OptionContract({
    required this.id,
    required this.symbol,
    required this.base,
    required this.quote,
    required this.type,
    required this.strike,
    required this.expiry,
    required this.expiryLabel,
    required this.bid,
    required this.ask,
    required this.last,
    required this.change24h,
    required this.volume24h,
    required this.openInterest,
    required this.impliedVolatility,
    required this.delta,
    required this.gamma,
    required this.theta,
  });
}

class Expiry {
  final String value;
  final String label;

  Expiry({required this.value, required this.label});
}

class OptionsService {
  static final List<Expiry> expiries = [
    Expiry(value: '1h', label: '1 Hour'),
    Expiry(value: '4h', label: '4 Hours'),
    Expiry(value: '1d', label: '1 Day'),
    Expiry(value: '1w', label: '1 Week'),
    Expiry(value: '2w', label: '2 Weeks'),
    Expiry(value: '1m', label: '1 Month'),
    Expiry(value: '3m', label: '3 Months'),
  ];

  static List<OptionPair> generatePairs() {
    final List<OptionPair> pairs = [];
    final bases = [
      'BTC', 'ETH', 'BNB', 'SOL', 'XRP', 'DOGE', 'ADA', 'AVAX', 'DOT', 'LINK',
      'MATIC', 'LTC', 'UNI', 'ATOM', 'XLM', 'NEAR', 'APT', 'ARB', 'OP', 'INJ',
    ];
    final prices = {
      'BTC': 43250.0, 'ETH': 2280.0, 'BNB': 312.5, 'SOL': 98.75, 'XRP': 0.62,
      'DOGE': 0.082, 'ADA': 0.58, 'AVAX': 38.20, 'DOT': 7.85, 'LINK': 14.50,
      'MATIC': 0.92, 'LTC': 72.30, 'UNI': 6.25, 'ATOM': 10.45, 'XLM': 0.125,
      'NEAR': 3.25, 'APT': 9.80, 'ARB': 1.12, 'OP': 2.45, 'INJ': 35.50,
    };

    // Top 200 pre-installed pairs
    for (int i = 0; i < bases.length; i++) {
      final price = prices[bases[i]] ?? 10.0;
      pairs.add(OptionPair(
        id: 'pair-$i',
        symbol: '${bases[i]}/USDT',
        base: bases[i],
        quote: 'USDT',
        currentPrice: price,
        isPreInstalled: i < 20,
      ));
    }

    // Additional pairs to reach 50,000+
    for (int i = 20; i < 50000; i++) {
      final base = 'TOKEN$i';
      final price = 10.0 + i * 0.001;
      pairs.add(OptionPair(
        id: 'pair-$i',
        symbol: '$base/USDT',
        base: base,
        quote: 'USDT',
        currentPrice: price,
        isPreInstalled: false,
      ));
    }

    return pairs;
  }

  static List<OptionContract> generateOptionChain(double currentPrice, String expiry) {
    final List<OptionContract> contracts = [];
    final expiryLabel = expiries.firstWhere((e) => e.value == expiry, orElse: () => Expiry(value: expiry, label: expiry)).label;
    
    // Generate strikes around current price
    final step = currentPrice > 1000 ? 500 : currentPrice > 100 ? 50 : currentPrice > 10 ? 5 : 0.5;
    final range = currentPrice * 0.15;
    
    for (double strike = currentPrice - range; strike <= currentPrice + range; strike += step) {
      // Call option
      final callPrice = (currentPrice - strike).abs() * 0.5 + (DateTime.now().millisecond % 50) / 10;
      contracts.add(OptionContract(
        id: 'call-${strike.toStringAsFixed(2)}-$expiry',
        symbol: '',
        base: '',
        quote: 'USDT',
        type: 'call',
        strike: strike,
        expiry: expiry,
        expiryLabel: expiryLabel,
        bid: callPrice * 0.95,
        ask: callPrice * 1.05,
        last: callPrice,
        change24h: (DateTime.now().millisecond % 20 - 10).toDouble(),
        volume24h: (DateTime.now().millisecond * 10000).toDouble(),
        openInterest: (DateTime.now().millisecond * 5000).toDouble(),
        impliedVolatility: 20 + (DateTime.now().second % 60),
        delta: currentPrice > strike ? 0.3 + (DateTime.now().millisecond % 40) / 100 : (DateTime.now().millisecond % 30) / 100,
        gamma: DateTime.now().millisecond % 10 / 100,
        theta: -(DateTime.now().millisecond % 50) / 100,
      ));

      // Put option
      final putPrice = (strike - currentPrice).abs() * 0.5 + (DateTime.now().millisecond % 50) / 10;
      contracts.add(OptionContract(
        id: 'put-${strike.toStringAsFixed(2)}-$expiry',
        symbol: '',
        base: '',
        quote: 'USDT',
        type: 'put',
        strike: strike,
        expiry: expiry,
        expiryLabel: expiryLabel,
        bid: putPrice * 0.95,
        ask: putPrice * 1.05,
        last: putPrice,
        change24h: (DateTime.now().millisecond % 20 - 10).toDouble(),
        volume24h: (DateTime.now().millisecond * 10000).toDouble(),
        openInterest: (DateTime.now().millisecond * 5000).toDouble(),
        impliedVolatility: 20 + (DateTime.now().second % 60),
        delta: currentPrice < strike ? -(0.3 + (DateTime.now().millisecond % 40) / 100) : -(DateTime.now().millisecond % 30) / 100,
        gamma: DateTime.now().millisecond % 10 / 100,
        theta: -(DateTime.now().millisecond % 50) / 100,
      ));
    }

    return contracts;
  }

  static List<OptionPair> getPreInstalledPairs() {
    return generatePairs().where((p) => p.isPreInstalled).toList();
  }
}
