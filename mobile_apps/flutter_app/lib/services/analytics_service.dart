///
/// Analytics Service - Flutter Implementation
/// Identical across ALL platforms
///

class AnalyticsService {
  static final AnalyticsService _instance = AnalyticsService._internal();
  factory AnalyticsService() => _instance;
  AnalyticsService._internal();

  Map<String, AssetHolding> _holdings = {};
  List<PortfolioTransaction> _transactions = [];
  List<PriceAlert> _alerts = [];
  int _totalPortfolioValue = 0;
  int _previousPortfolioValue = 0;

  void updatePortfolio(Map<String, AssetHolding> holdings) {
    _previousPortfolioValue = _totalPortfolioValue;
    _holdings = Map.from(holdings);
    _recalculateValue();
  }

  PortfolioSummary getSummary() {
    return PortfolioSummary(
      totalValue: _totalPortfolioValue,
      change24h: _totalPortfolioValue - _previousPortfolioValue,
      changePercent24h: _previousPortfolioValue > 0 
          ? ((_totalPortfolioValue - _previousPortfolioValue) / _previousPortfolioValue) * 100 
          : 0.0,
      assets: _holdings.values.toList(),
      lastUpdated: DateTime.now().millisecondsSinceEpoch,
    );
  }

  PerformanceMetrics getPerformance(String timeframe) {
    double returns = (DateTime.now().millisecond % 40) - 10.0;
    double volatility = returns.abs() * 0.5;
    double sharpe = volatility > 0 ? returns / volatility : 0;

    return PerformanceMetrics(
      timeframe: timeframe,
      totalReturn: returns,
      annualizedReturn: returns * _getAnnualizationFactor(timeframe),
      volatility: volatility,
      sharpeRatio: sharpe,
      maxDrawdown: (DateTime.now().millisecond % 20).toDouble(),
      riskLevel: volatility < 0.1 ? 'LOW' : (volatility < 0.3 ? 'MEDIUM' : 'HIGH'),
    );
  }

  AllocationBreakdown getAllocation() {
    Map<String, int> byChain = {};
    Map<String, int> byCategory = {};

    for (var holding in _holdings.values) {
      byChain[holding.chain] = (byChain[holding.chain] ?? 0) + holding.value;
      byCategory[holding.category] = (byCategory[holding.category] ?? 0) + holding.value;
    }

    return AllocationBreakdown(
      byChain: byChain,
      byCategory: byCategory,
      totalValue: _totalPortfolioValue,
      diversificationScore: _calculateDiversificationScore(byChain),
    );
  }

  List<PortfolioTransaction> getTransactionHistory({
    String? startDate, 
    String? endDate, 
    List<String>? type,
  }) {
    var result = List<PortfolioTransaction>.from(_transactions);

    if (startDate != null) {
      result = result.where((tx) => tx.date.compareTo(startDate) >= 0).toList();
    }
    if (endDate != null) {
      result = result.where((tx) => tx.date.compareTo(endDate) <= 0).toList();
    }
    if (type != null) {
      result = result.where((tx) => type.contains(tx.type)).toList();
    }

    return result;
  }

  PriceAlert setAlert({
    required String asset,
    required AlertCondition condition,
    required double targetPrice,
  }) {
    final alert = PriceAlert(
      id: 'alert_${DateTime.now().millisecondsSinceEpoch}',
      asset: asset,
      condition: condition,
      targetPrice: targetPrice,
      isActive: true,
      createdAt: DateTime.now().millisecondsSinceEpoch,
    );
    _alerts.add(alert);
    return alert;
  }

  List<PriceAlert> getAlerts() => _alerts.where((a) => a.isActive).toList();

  bool deleteAlert(String alertId) {
    final index = _alerts.indexWhere((a) => a.id == alertId);
    if (index != -1) {
      _alerts.removeAt(index);
      return true;
    }
    return false;
  }

  List<HistoryPoint> getHistory(String startDate, String endDate, String interval) {
    return [];
  }

  String exportReport(String format) {
    switch (format) {
      case 'csv':
        var csv = 'Asset,Chain,Balance,Value,Allocation\n';
        for (var holding in _holdings.values) {
          csv += '${holding.symbol},${holding.chain},${holding.balance},${holding.value},${holding.allocation}\n';
        }
        return csv;
      default:
        return '{}';
    }
  }

  void _recalculateValue() {
    _totalPortfolioValue = _holdings.values.fold(0, (sum, h) => sum + h.value);
  }

  double _getAnnualizationFactor(String timeframe) {
    switch (timeframe) {
      case '1d': return 365;
      case '1w': return 52;
      case '1m': return 12;
      default: return 1;
    }
  }

  double _calculateDiversificationScore(Map<String, int> byChain) {
    if (byChain.isEmpty) return 0;
    int total = byChain.values.fold(0, (sum, v) => sum + v);
    if (total == 0) return 0;

    double sumSquares = 0;
    for (var v in byChain.values) {
      double proportion = v / total;
      sumSquares += proportion * proportion;
    }

    return sumSquares > 0 ? (1 / sumSquares) / byChain.length * 100 : 0;
  }
}

class AssetHolding {
  final String symbol;
  final String name;
  final String chain;
  final String category;
  final int balance;
  final double price;
  final int value;
  final double allocation;
  final double change24h;

  AssetHolding({
    required this.symbol,
    required this.name,
    required this.chain,
    required this.category,
    required this.balance,
    required this.price,
    required this.value,
    required this.allocation,
    required this.change24h,
  });
}

class PortfolioSummary {
  final int totalValue;
  final int change24h;
  final double changePercent24h;
  final List<AssetHolding> assets;
  final int lastUpdated;

  PortfolioSummary({
    required this.totalValue,
    required this.change24h,
    required this.changePercent24h,
    required this.assets,
    required this.lastUpdated,
  });
}

class PerformanceMetrics {
  final String timeframe;
  final double totalReturn;
  final double annualizedReturn;
  final double volatility;
  final double sharpeRatio;
  final double maxDrawdown;
  final String riskLevel;

  PerformanceMetrics({
    required this.timeframe,
    required this.totalReturn,
    required this.annualizedReturn,
    required this.volatility,
    required this.sharpeRatio,
    required this.maxDrawdown,
    required this.riskLevel,
  });
}

class AllocationBreakdown {
  final Map<String, int> byChain;
  final Map<String, int> byCategory;
  final int totalValue;
  final double diversificationScore;

  AllocationBreakdown({
    required this.byChain,
    required this.byCategory,
    required this.totalValue,
    required this.diversificationScore,
  });
}

class PortfolioTransaction {
  final String id;
  final String type;
  final String asset;
  final int amount;
  final int value;
  final String date;
  final String txHash;

  PortfolioTransaction({
    required this.id,
    required this.type,
    required this.asset,
    required this.amount,
    required this.value,
    required this.date,
    required this.txHash,
  });
}

enum AlertCondition { above, below }

class PriceAlert {
  final String id;
  final String asset;
  final AlertCondition condition;
  final double targetPrice;
  final bool isActive;
  final int createdAt;

  PriceAlert({
    required this.id,
    required this.asset,
    required this.condition,
    required this.targetPrice,
    required this.isActive,
    required this.createdAt,
  });
}

class HistoryPoint {
  final int timestamp;
  final int value;
  final int change;

  HistoryPoint({
    required this.timestamp,
    required this.value,
    required this.change,
  });
}
