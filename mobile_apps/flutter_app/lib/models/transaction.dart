/**
 * TigerWallet Transaction Model
 * Complete transaction data model
 */

enum TransactionStatus {
  pending,
  confirmed,
  failed,
}

enum TransactionType {
  send,
  receive,
  swap,
  approve,
  contractInteraction,
  nftTransfer,
  unknown,
}

class Transaction {
  final String txHash;
  final String from;
  final String to;
  final String tokenAddress;
  final String tokenSymbol;
  final double amount;
  final double fee;
  final int timestamp;
  final TransactionStatus status;
  final TransactionType type;
  final String chainId;
  final String blockNumber;
  final int confirmations;
  final String? errorMessage;
  
  const Transaction({
    required this.txHash,
    required this.from,
    required this.to,
    required this.tokenAddress,
    required this.tokenSymbol,
    required this.amount,
    required this.fee,
    required this.timestamp,
    required this.status,
    required this.type,
    required this.chainId,
    this.blockNumber = '',
    this.confirmations = 0,
    this.errorMessage,
  });
  
  factory Transaction.fromJson(Map<String, dynamic> json) {
    return Transaction(
      txHash: json['txHash'] ?? '',
      from: json['from'] ?? '',
      to: json['to'] ?? '',
      tokenAddress: json['tokenAddress'] ?? '',
      tokenSymbol: json['tokenSymbol'] ?? '',
      amount: (json['amount'] ?? 0.0).toDouble(),
      fee: (json['fee'] ?? 0.0).toDouble(),
      timestamp: json['timestamp'] ?? 0,
      status: _parseStatus(json['status']),
      type: _parseType(json['type']),
      chainId: json['chainId'] ?? '',
      blockNumber: json['blockNumber'] ?? '',
      confirmations: json['confirmations'] ?? 0,
      errorMessage: json['errorMessage'],
    );
  }
  
  Map<String, dynamic> toJson() {
    return {
      'txHash': txHash,
      'from': from,
      'to': to,
      'tokenAddress': tokenAddress,
      'tokenSymbol': tokenSymbol,
      'amount': amount,
      'fee': fee,
      'timestamp': timestamp,
      'status': status.name,
      'type': type.name,
      'chainId': chainId,
      'blockNumber': blockNumber,
      'confirmations': confirmations,
      'errorMessage': errorMessage,
    };
  }
  
  static TransactionStatus _parseStatus(dynamic status) {
    switch (status?.toString().toLowerCase()) {
      case 'confirmed':
        return TransactionStatus.confirmed;
      case 'failed':
        return TransactionStatus.failed;
      default:
        return TransactionStatus.pending;
    }
  }
  
  static TransactionType _parseType(dynamic type) {
    switch (type?.toString().toLowerCase()) {
      case 'send':
        return TransactionType.send;
      case 'receive':
        return TransactionType.receive;
      case 'swap':
        return TransactionType.swap;
      case 'approve':
        return TransactionType.approve;
      case 'contract_interaction':
      case 'contractinteraction':
        return TransactionType.contractInteraction;
      case 'nft_transfer':
      case 'nfttransfer':
        return TransactionType.nftTransfer;
      default:
        return TransactionType.unknown;
    }
  }
  
  bool get isPending => status == TransactionStatus.pending;
  bool get isConfirmed => status == TransactionStatus.confirmed;
  bool get isFailed => status == TransactionStatus.failed;
  
  DateTime get dateTime => DateTime.fromMillisecondsSinceEpoch(timestamp * 1000);
  
  String get shortFrom => '${from.substring(0, 6)}...${from.substring(from.length - 4)}';
  String get shortTo => '${to.substring(0, 6)}...${to.substring(to.length - 4)}';
  
  String get explorerUrl {
    // Would use chain service to get URL
    return 'https://etherscan.io/tx/$txHash';
  }
}

class TransactionList {
  final List<Transaction> transactions;
  
  const TransactionList(this.transactions);
  
  factory TransactionList.fromJson(List<dynamic> json) {
    return TransactionList(
      json.map((e) => Transaction.fromJson(e)).toList(),
    );
  }
  
  List<Transaction> getByChain(String chainId) {
    return transactions.where((t) => t.chainId == chainId).toList();
  }
  
  List<Transaction> getByStatus(TransactionStatus status) {
    return transactions.where((t) => t.status == status).toList();
  }
  
  List<Transaction> getByType(TransactionType type) {
    return transactions.where((t) => t.type == type).toList();
  }
  
  List<Transaction> getRecent({int limit = 20}) {
    final sorted = List<Transaction>.from(transactions)
      ..sort((a, b) => b.timestamp.compareTo(a.timestamp));
    return sorted.take(limit).toList();
  }
}
