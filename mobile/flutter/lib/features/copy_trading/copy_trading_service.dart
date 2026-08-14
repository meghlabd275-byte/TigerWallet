// Copy Trading Service - Flutter Implementation

import 'dart:convert';
import 'package:http/http.dart' as http;
import '../../core/constants/app_constants.dart';

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
  // copy_trading_service (go/copy_trading_service) listens on :8006.
  static const String _copyTradingServiceUrl =
      String.fromEnvironment('COPY_TRADING_SERVICE_URL',
          defaultValue: 'http://localhost:8006');

  static String _authToken() {
    // Reuse the wallet_api JWT stored by the auth flow, if present.
    return '';
  }

  /// Fetch real traders from the copy_trading backend. Never fabricates —
  /// returns an empty list on failure.
  static Future<List<Trader>> fetchTraders() async {
    try {
      final token = _authToken();
      final res = await http.get(
        Uri.parse('$_copyTradingServiceUrl/api/v1/copytrading/traders'),
        headers: {
          'Content-Type': 'application/json',
          if (token.isNotEmpty) 'Authorization': 'Bearer $token',
        },
      ).timeout(AppConstants.apiTimeout);
      if (res.statusCode != 200) return [];
      final body = jsonDecode(res.body) as Map<String, dynamic>;
      final list = body['traders'] as List<dynamic>? ?? [];
      return list.map((raw) {
        final t = raw as Map<String, dynamic>;
        final win = double.tryParse('${t['win_rate'] ?? ''}') ?? 0.0;
        final pnl = double.tryParse('${t['pnl_pct'] ?? ''}') ?? 0.0;
        final followers = (t['followers'] as num?)?.toInt() ?? 0;
        return Trader(
          id: '${t['id'] ?? ''}',
          username: '${t['name'] ?? t['address'] ?? 'trader'}',
          avatar: '',
          winRate: win,
          totalPnL: 0.0,
          pnlPercent: pnl,
          followers: followers,
          copyCount: 0,
          tradingPair: '',
          monthlyPnL: 0.0,
          weeklyPnL: 0.0,
          dailyPnL: 0.0,
          maxDrawdown: 0.0,
          avgHoldingTime: '',
          riskLevel: 'medium',
          isFollowing: false,
          isPreInstalled: false,
        );
      }).toList();
    } catch (_) {
      // Fail closed — no fabricated traders.
      return [];
    }
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
