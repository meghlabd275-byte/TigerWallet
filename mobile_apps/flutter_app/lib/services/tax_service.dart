///
/// Tax Service - Flutter Implementation
/// Identical across ALL platforms
///

class TaxService {
  static final TaxService _instance = TaxService._internal();
  factory TaxService() => _instance;
  TaxService._internal();

  List<TaxTransaction> _transactions = [];
  Map<String, List<TaxLot>> _taxLots = {};
  List<IncomeEvent> _incomeEvents = [];
  
  String _jurisdiction = 'US';
  CostBasisMethod _costBasisMethod = CostBasisMethod.fifo;

  bool setJurisdiction(String jurisdictionCode) {
    _jurisdiction = jurisdictionCode;
    return true;
  }

  bool setCostBasisMethod(CostBasisMethod method) {
    _costBasisMethod = method;
    return true;
  }

  void addTransaction(TaxTransaction tx) {
    _transactions.add(tx);
  }

  TaxReport calculateGains() {
    int shortTermGains = 0;
    int shortTermLosses = 0;
    int longTermGains = 0;
    int longTermLosses = 0;
    int totalIncome = 0;

    for (var event in _incomeEvents) {
      totalIncome += event.fairMarketValue;
    }

    return TaxReport(
      year: 2024,
      shortTermGains: shortTermGains,
      shortTermLosses: shortTermLosses,
      longTermGains: longTermGains,
      longTermLosses: longTermLosses,
      totalIncome: totalIncome,
      totalTransactions: _transactions.length,
      jurisdiction: _jurisdiction,
      costBasisMethod: _costBasisMethod,
    );
  }

  List<TaxLot> getAvailableLots(String asset) {
    return _taxLots[asset]?.where((lot) => lot.remainingAmount > 0).toList() ?? [];
  }

  void addIncomeEvent(IncomeEvent event) {
    _incomeEvents.add(event);

    final lot = TaxLot(
      id: 'lot_${DateTime.now().millisecondsSinceEpoch}',
      asset: event.asset,
      amount: event.amount,
      remainingAmount: event.amount,
      costBasis: 0,
      fairMarketValue: event.fairMarketValue,
      acquisitionDate: event.date,
      isLongTerm: false,
    );

    _taxLots[event.asset] = [...(_taxLots[event.asset] ?? []), lot];
  }

  String exportCSV() {
    var csv = 'Date,Type,Asset,Amount,Cost Basis,Proceeds,Gain/Loss,Exchange\n';
    for (var tx in _transactions) {
      csv += '${tx.date},${tx.type},${tx.asset},${tx.amount},${tx.costBasis},${tx.proceeds},${tx.gainLoss},${tx.exchange}\n';
    }
    return csv;
  }
}

enum CostBasisMethod { fifo, lifo, hifo }
enum TransactionType { buy, sell, transfer, swap, stake, unstake, mint, burn }

class TaxTransaction {
  final String id;
  final TransactionType type;
  final String date;
  final String asset;
  final int amount;
  final int price;
  final int costBasis;
  final int proceeds;
  final int gainLoss;
  final String exchange;
  final String txHash;

  TaxTransaction({
    required this.id,
    required this.type,
    required this.date,
    required this.asset,
    required this.amount,
    required this.price,
    required this.costBasis,
    required this.proceeds,
    required this.gainLoss,
    required this.exchange,
    required this.txHash,
  });
}

class TaxLot {
  final String id;
  final String asset;
  final int amount;
  int remainingAmount;
  final int costBasis;
  final int fairMarketValue;
  final String acquisitionDate;
  final bool isLongTerm;

  TaxLot({
    required this.id,
    required this.asset,
    required this.amount,
    required this.remainingAmount,
    required this.costBasis,
    required this.fairMarketValue,
    required this.acquisitionDate,
    required this.isLongTerm,
  });
}

class IncomeEvent {
  final String id;
  final String type;
  final String asset;
  final int amount;
  final int fairMarketValue;
  final String date;

  IncomeEvent({
    required this.id,
    required this.type,
    required this.asset,
    required this.amount,
    required this.fairMarketValue,
    required this.date,
  });
}

class TaxReport {
  final int year;
  final int shortTermGains;
  final int shortTermLosses;
  final int longTermGains;
  final int longTermLosses;
  final int totalIncome;
  final int totalTransactions;
  final String jurisdiction;
  final CostBasisMethod costBasisMethod;

  TaxReport({
    required this.year,
    required this.shortTermGains,
    required this.shortTermLosses,
    required this.longTermGains,
    required this.longTermLosses,
    required this.totalIncome,
    required this.totalTransactions,
    required this.jurisdiction,
    required this.costBasisMethod,
  });
}
