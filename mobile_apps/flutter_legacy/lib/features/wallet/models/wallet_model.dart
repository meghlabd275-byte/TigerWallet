// Wallet Model - Complete Wallet Data Structure
// Supports multi-chain wallet addresses from single 24-word seed

import 'package:crypto/crypto.dart';
import 'dart:convert';

class Wallet {
  final String id;
  final String name;
  final String? encryptedMnemonic;
  final Map<String, String> addresses; // chainId -> address
  final Map<String, String> publicKeys; // chainId -> publicKey
  final DateTime createdAt;
  final bool isBackedUp;
  final WalletType type;
  
  Wallet({
    required this.id,
    required this.name,
    this.encryptedMnemonic,
    required this.addresses,
    required this.publicKeys,
    required this.createdAt,
    this.isBackedUp = false,
    this.type = WalletType.hd,
  });
  
  String getAddressForChain(String chainId) {
    return addresses[chainId] ?? '';
  }
  
  String getPublicKeyForChain(String chainId) {
    return publicKeys[chainId] ?? '';
  }
  
  bool hasChain(String chainId) {
    return addresses.containsKey(chainId);
  }
  
  List<String> get connectedChainIds => addresses.keys.toList();
  
  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'name': name,
      'encryptedMnemonic': encryptedMnemonic,
      'addresses': addresses,
      'publicKeys': publicKeys,
      'createdAt': createdAt.toIso8601String(),
      'isBackedUp': isBackedUp,
      'type': type.name,
    };
  }
  
  factory Wallet.fromJson(Map<String, dynamic> json) {
    return Wallet(
      id: json['id'],
      name: json['name'],
      encryptedMnemonic: json['encryptedMnemonic'],
      addresses: Map<String, String>.from(json['addresses']),
      publicKeys: Map<String, String>.from(json['publicKeys']),
      createdAt: DateTime.parse(json['createdAt']),
      isBackedUp: json['isBackedUp'] ?? false,
      type: WalletType.values.firstWhere(
        (e) => e.name == json['type'],
        orElse: () => WalletType.hd,
      ),
    );
  }
  
  Wallet copyWith({
    String? id,
    String? name,
    String? encryptedMnemonic,
    Map<String, String>? addresses,
    Map<String, String>? publicKeys,
    DateTime? createdAt,
    bool? isBackedUp,
    WalletType? type,
  }) {
    return Wallet(
      id: id ?? this.id,
      name: name ?? this.name,
      encryptedMnemonic: encryptedMnemonic ?? this.encryptedMnemonic,
      addresses: addresses ?? this.addresses,
      publicKeys: publicKeys ?? this.publicKeys,
      createdAt: createdAt ?? this.createdAt,
      isBackedUp: isBackedUp ?? this.isBackedUp,
      type: type ?? this.type,
    );
  }
}

enum WalletType {
  hd,           // Hierarchical Deterministic (24-word seed)
  privateKey,    // Single private key import
  hardware,      // Hardware wallet (Ledger, Trezor)
  watchOnly,     // Watch-only (no signing capability)
  multiSig,     // Multi-signature wallet
}

class WalletAddress {
  final String address;
  final String chainId;
  final String? ensName; // Ethereum Name Service
  final bool isContract;
  final String? tokenId; // For NFT
  
  WalletAddress({
    required this.address,
    required this.chainId,
    this.ensName,
    this.isContract = false,
    this.tokenId,
  });
  
  String get displayAddress {
    if (address.length <= 12) return address;
    return '${address.substring(0, 6)}...${address.substring(address.length - 4)}';
  }
  
  Map<String, dynamic> toJson() {
    return {
      'address': address,
      'chainId': chainId,
      'ensName': ensName,
      'isContract': isContract,
      'tokenId': tokenId,
    };
  }
  
  factory WalletAddress.fromJson(Map<String, dynamic> json) {
    return WalletAddress(
      address: json['address'],
      chainId: json['chainId'],
      ensName: json['ensName'],
      isContract: json['isContract'] ?? false,
      tokenId: json['tokenId'],
    );
  }
}

// Transaction Model
class TransactionModel {
  final String id;
  final String hash;
  final String fromAddress;
  final String toAddress;
  final String amount;
  final String tokenSymbol;
  final int decimals;
  final String chainId;
  final TransactionStatus status;
  final TransactionType type;
  final DateTime timestamp;
  final double fee;
  final String? blockNumber;
  final int? confirmations;
  final String? errorMessage;
  
  TransactionModel({
    required this.id,
    required this.hash,
    required this.fromAddress,
    required this.toAddress,
    required this.amount,
    required this.tokenSymbol,
    required this.decimals,
    required this.chainId,
    required this.status,
    required this.type,
    required this.timestamp,
    required this.fee,
    this.blockNumber,
    this.confirmations,
    this.errorMessage,
  });
  
  double get amountDecimal {
    return double.tryParse(amount) ?? 0 / (10 * decimals);
  }
  
  String get displayAmount {
    final value = double.tryParse(amount) ?? 0;
    return '${(value / 1000000000000000000).toStringAsFixed(6)} $tokenSymbol';
  }
  
  bool get isPending => status == TransactionStatus.pending;
  bool get isConfirmed => status == TransactionStatus.confirmed;
  bool get isFailed => status == TransactionStatus.failed;
  
  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'hash': hash,
      'fromAddress': fromAddress,
      'toAddress': toAddress,
      'amount': amount,
      'tokenSymbol': tokenSymbol,
      'decimals': decimals,
      'chainId': chainId,
      'status': status.name,
      'type': type.name,
      'timestamp': timestamp.toIso8601String(),
      'fee': fee,
      'blockNumber': blockNumber,
      'confirmations': confirmations,
      'errorMessage': errorMessage,
    };
  }
  
  factory TransactionModel.fromJson(Map<String, dynamic> json) {
    return TransactionModel(
      id: json['id'],
      hash: json['hash'],
      fromAddress: json['fromAddress'],
      toAddress: json['toAddress'],
      amount: json['amount'],
      tokenSymbol: json['tokenSymbol'],
      decimals: json['decimals'],
      chainId: json['chainId'],
      status: TransactionStatus.values.firstWhere(
        (e) => e.name == json['status'],
        orElse: () => TransactionStatus.pending,
      ),
      type: TransactionType.values.firstWhere(
        (e) => e.name == json['type'],
        orElse: () => TransactionType.transfer,
      ),
      timestamp: DateTime.parse(json['timestamp']),
      fee: (json['fee'] as num).toDouble(),
      blockNumber: json['blockNumber'],
      confirmations: json['confirmations'],
      errorMessage: json['errorMessage'],
    );
  }
}

enum TransactionStatus {
  pending,
  confirmed,
  failed,
  cancelled,
}

enum TransactionType {
  transfer,
  swap,
  stake,
  unstake,
  mint,
  burn,
  approve,
  contractCall,
  nftTransfer,
  bridge,
}
