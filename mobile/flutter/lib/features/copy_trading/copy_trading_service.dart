// Copy Trading Service - Flutter Implementation

class Trader {
  final String id;
  final String username;
  final String avatar;
  final double winRate;
  final double totalPnL;
  final double pnlPercent;
  final int followers;
  final int copyCount;
  final String tradingPair;
  final double monthlyPnL;
  final double weeklyPnL;
  final double dailyPnL;
  final double maxDrawdown;
  final String avgHoldingTime;
  final String riskLevel;
  final bool isFollowing;
  final bool isPreInstalled;

  Trader({
    required this.id,
    required this.username,
    required this.avatar,
    required this.winRate,
    required this.totalPnL,
    required this.pnlPercent,
    required this.followers,
    required this.copyCount,
    required this.tradingPair,
    required this.monthlyPnL,
    required this.weeklyPnL,
    required this.dailyPnL,
    required this.maxDrawdown,
    required this.avgHoldingTime,
    required this.riskLevel,
    required this.isFollowing,
    required this.isPreInstalled,
  });
}

class CopyPosition {
  final String id;
  final String traderId;
  final String traderName;
  final String userId;
  final String symbol;
  final String side;
  final double size;
  final double entryPrice;
  final double currentPrice;
  final double pnl;
  final double pnlPercent;
  final DateTime openTime;
  final String status;

  CopyPosition({
    required this.id,
    required this.traderId,
    required this.traderName,
    required this.userId,
    required this.symbol,
    required this.side,
    required this.size,
    required this.entryPrice,
    required this.currentPrice,
    required this.pnl,
    required this.pnlPercent,
    required this.openTime,
    required this.status,
  });
}

class CopySettings {
  final String userId;
  final double copyAmount;
  final int copyLeverage;
  final double stopLossPercent;
  final double takeProfitPercent;
  final bool autoCopy;

  CopySettings({
    required this.userId,
    required this.copyAmount,
    required this.copyLeverage,
    required this.stopLossPercent,
    required this.takeProfitPercent,
    required this.autoCopy,
  });
}

class CopyTradingService {
  static List<Trader> generateTraders() {
    final List<Trader> traders = [];
    
    // Pre-installed top traders
    final preInstalledTraders = [
      {'username': 'CryptoWhale', 'avatar': '🐋', 'winRate': 78.5, 'totalPnL': 125000.0, 'pnlPercent': 156.2, 'followers': 15234, 'copyCount': 4521, 'pair': 'BTC/USDT', 'monthly': 12.5, 'weekly': 3.2, 'daily': 0.8, 'drawdown': -8.5, 'time': '2h 30m', 'risk': 'medium'},
      {'username': 'DeFiMaster', 'avatar': '🎯', 'winRate': 82.3, 'totalPnL': 98500.0, 'pnlPercent': 142.8, 'followers': 12456, 'copyCount': 3890, 'pair': 'ETH/USDT', 'monthly': 15.2, 'weekly': 4.1, 'daily': 1.2, 'drawdown': -6.2, 'time': '4h 15m', 'risk': 'low'},
      {'username': 'AltSeason', 'avatar': '🚀', 'winRate': 71.2, 'totalPnL': 87000.0, 'pnlPercent': 198.5, 'followers': 8923, 'copyCount': 2156, 'pair': 'SOL/USDT', 'monthly': 22.5, 'weekly': 8.3, 'daily': 2.1, 'drawdown': -12.8, 'time': '1h 45m', 'risk': 'high'},
      {'username': 'GridTrader', 'avatar': '📊', 'winRate': 85.1, 'totalPnL': 67800.0, 'pnlPercent': 98.3, 'followers': 6543, 'copyCount': 1890, 'pair': 'BNB/USDT', 'monthly': 8.2, 'weekly': 2.1, 'daily': 0.5, 'drawdown': -4.2, 'time': '6h 20m', 'risk': 'low'},
      {'username': 'MomentumKing', 'avatar': '👑', 'winRate': 75.8, 'totalPnL': 54200.0, 'pnlPercent': 125.6, 'followers': 9876, 'copyCount': 2567, 'pair': 'DOGE/USDT', 'monthly': 18.5, 'weekly': 5.2, 'daily': 1.5, 'drawdown': -15.2, 'time': '0h 45m', 'risk': 'high'},
      {'username': 'SwingTrader', 'avatar': '🌊', 'winRate': 68.5, 'totalPnL': 42500.0, 'pnlPercent': 88.2, 'followers': 5432, 'copyCount': 1234, 'pair': 'XRP/USDT', 'monthly': 10.2, 'weekly': 2.8, 'daily': 0.3, 'drawdown': -9.5, 'time': '12h 30m', 'risk': 'medium'},
      {'username': 'BotMaster', 'avatar': '🤖', 'winRate': 88.2, 'totalPnL': 38900.0, 'pnlPercent': 72.5, 'followers': 4321, 'copyCount': 987, 'pair': 'AVAX/USDT', 'monthly': 6.8, 'weekly': 1.5, 'daily': 0.2, 'drawdown': -3.2, 'time': '8h 00m', 'risk': 'low'},
      {'username': 'NanoGainer', 'avatar': '💎', 'winRate': 73.2, 'totalPnL': 31500.0, 'pnlPercent': 145.8, 'followers': 7654, 'copyCount': 1876, 'pair': 'PEPE/USDT', 'monthly': 25.2, 'weekly': 9.5, 'daily': 3.2, 'drawdown': -18.5, 'time': '0h 30m', 'risk': 'high'},
      {'username': 'StableTrader', 'avatar': '🛡️', 'winRate': 91.2, 'totalPnL': 28900.0, 'pnlPercent': 52.3, 'followers': 3210, 'copyCount': 654, 'pair': 'LINK/USDT', 'monthly': 4.2, 'weekly': 1.0, 'daily': 0.1, 'drawdown': -2.1, 'time': '24h 00m', 'risk': 'low'},
      {'username': 'FlashBoys', 'avatar': '⚡', 'winRate': 76.8, 'totalPnL': 24500.0, 'pnlPercent': 168.5, 'followers': 5678, 'copyCount': 1432, 'pair': 'MATIC/USDT', 'monthly': 14.5, 'weekly': 4.8, 'daily': 1.8, 'drawdown': -11.2, 'time': '1h 15m', 'risk': 'high'},
    ];

    for (int i = 0; i < preInstalledTraders.length; i++) {
      final t = preInstalledTraders[i];
      traders.add(Trader(
        id: 'trader-${i + 1}',
        username: t['username'] as String,
        avatar: t['avatar'] as String,
        winRate: t['winRate'] as double,
        totalPnL: t['totalPnL'] as double,
        pnlPercent: t['pnlPercent'] as double,
        followers: t['followers'] as int,
        copyCount: t['copyCount'] as int,
        tradingPair: t['pair'] as String,
        monthlyPnL: t['monthly'] as double,
        weeklyPnL: t['weekly'] as double,
        dailyPnL: t['daily'] as double,
        maxDrawdown: t['drawdown'] as double,
        avgHoldingTime: t['time'] as String,
        riskLevel: t['risk'] as String,
        isFollowing: false,
        isPreInstalled: true,
      ));
    }

    // Generate additional traders to reach 500+
    final avatars = ['🐵', '🦊', '🦁', '🐯', '🐲', '🐍', '🐴', '🦄', '🐝', '🦋'];
    final pairs = ['BTC/USDT', 'ETH/USDT', 'BNB/USDT', 'SOL/USDT', 'XRP/USDT'];
    final risks = ['low', 'medium', 'high'];

    for (int i = 0; i < 500; i++) {
      traders.add(Trader(
        id: 'trader-${i + 100}',
        username: 'Trader${i + 100}',
        avatar: avatars[i % avatars.length],
        winRate: 60 + (i % 30).toDouble(),
        totalPnL: 1000 + (i * 200).toDouble(),
        pnlPercent: 20 + (i % 200).toDouble(),
        followers: 100 + (i * 20),
        copyCount: 50 + (i * 10),
        tradingPair: pairs[i % pairs.length],
        monthlyPnL: (i % 30 - 5).toDouble(),
        weeklyPnL: (i % 10 - 2).toDouble(),
        dailyPnL: (i % 3 - 1).toDouble(),
        maxDrawdown: -(2 + (i % 20)).toDouble(),
        avgHoldingTime: '${i % 24}h ${i % 60}m',
        riskLevel: risks[i % 3],
        isFollowing: false,
        isPreInstalled: false,
      ));
    }

    return traders;
  }

  static List<Trader> getTopTraders() {
    return generateTraders().where((t) => t.isPreInstalled).toList();
  }

  static List<Trader> filterByRisk(List<Trader> traders, String risk) {
    if (risk == 'all') return traders;
    return traders.where((t) => t.riskLevel == risk).toList();
  }

  static List<Trader> sortByPnL(List<Trader> traders) {
    final sorted = List<Trader>.from(traders);
    sorted.sort((a, b) => b.totalPnL.compareTo(a.totalPnL));
    return sorted;
  }
}
