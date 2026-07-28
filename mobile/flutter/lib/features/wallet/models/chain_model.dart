// Chain Model - Blockchain Network Configuration
// Supports 300+ blockchains with full RPC management

class ChainModel {
  final String id;
  final String name;
  final String symbol;
  final String iconUrl;
  final int chainId; // EVM chain ID
  final String rpcUrl;
  final String explorerUrl;
  final String? explorerApiUrl;
  final ChainType type;
  final bool isTestnet;
  final bool isDefault;
  final String? contractAddress; // For token chains
  final List<TokenOnChain> tokens;
  final ChainConfig config;
  
  ChainModel({
    required this.id,
    required this.name,
    required this.symbol,
    required this.iconUrl,
    required this.chainId,
    required this.rpcUrl,
    required this.explorerUrl,
    this.explorerApiUrl,
    required this.type,
    this.isTestnet = false,
    this.isDefault = false,
    this.contractAddress,
    this.tokens = const [],
    required this.config,
  });
  
  bool get isEVM => type == ChainType.evm;
  bool get isNonEVM => type != ChainType.evm;
  
  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'name': name,
      'symbol': symbol,
      'iconUrl': iconUrl,
      'chainId': chainId,
      'rpcUrl': rpcUrl,
      'explorerUrl': explorerUrl,
      'explorerApiUrl': explorerApiUrl,
      'type': type.name,
      'isTestnet': isTestnet,
      'isDefault': isDefault,
      'contractAddress': contractAddress,
      'tokens': tokens.map((t) => t.toJson()).toList(),
      'config': config.toJson(),
    };
  }
  
  factory ChainModel.fromJson(Map<String, dynamic> json) {
    return ChainModel(
      id: json['id'],
      name: json['name'],
      symbol: json['symbol'],
      iconUrl: json['iconUrl'],
      chainId: json['chainId'],
      rpcUrl: json['rpcUrl'],
      explorerUrl: json['explorerUrl'],
      explorerApiUrl: json['explorerApiUrl'],
      type: ChainType.values.firstWhere(
        (e) => e.name == json['type'],
        orElse: () => ChainType.evm,
      ),
      isTestnet: json['isTestnet'] ?? false,
      isDefault: json['isDefault'] ?? false,
      contractAddress: json['contractAddress'],
      tokens: (json['tokens'] as List?)
              ?.map((t) => TokenOnChain.fromJson(t))
              .toList() ??
          [],
      config: ChainConfig.fromJson(json['config']),
    );
  }
  
  ChainModel copyWith({
    String? id,
    String? name,
    String? symbol,
    String? iconUrl,
    int? chainId,
    String? rpcUrl,
    String? explorerUrl,
    String? explorerApiUrl,
    ChainType? type,
    bool? isTestnet,
    bool? isDefault,
    String? contractAddress,
    List<TokenOnChain>? tokens,
    ChainConfig? config,
  }) {
    return ChainModel(
      id: id ?? this.id,
      name: name ?? this.name,
      symbol: symbol ?? this.symbol,
      iconUrl: iconUrl ?? this.iconUrl,
      chainId: chainId ?? this.chainId,
      rpcUrl: rpcUrl ?? this.rpcUrl,
      explorerUrl: explorerUrl ?? this.explorerUrl,
      explorerApiUrl: explorerApiUrl ?? this.explorerApiUrl,
      type: type ?? this.type,
      isTestnet: isTestnet ?? this.isTestnet,
      isDefault: isDefault ?? this.isDefault,
      contractAddress: contractAddress ?? this.contractAddress,
      tokens: tokens ?? this.tokens,
      config: config ?? this.config,
    );
  }
}

enum ChainType {
  evm,
  bitcoin,
  solana,
  cosmos,
  near,
  algorand,
  cardano,
  substrate,
  ton,
  tron,
  apts,
  sui,
  starknet,
  hedera,
  ripple,
  stellar,
  polkadot,
  other,
}

class ChainConfig {
  final String derivationPath;
  final int addressPrefix;
  final bool supportsEIP1559;
  final bool supportsTokenTransfers;
  final bool supportsNFT;
  final bool supportsStaking;
  final BlockTimeConfig blockTime;
  final GasConfig gasConfig;
  final Map<String, dynamic> extra;
  
  ChainConfig({
    required this.derivationPath,
    required this.addressPrefix,
    this.supportsEIP1559 = false,
    this.supportsTokenTransfers = true,
    this.supportsNFT = false,
    this.supportsStaking = false,
    required this.blockTime,
    required this.gasConfig,
    this.extra = const {},
  });
  
  Map<String, dynamic> toJson() {
    return {
      'derivationPath': derivationPath,
      'addressPrefix': addressPrefix,
      'supportsEIP1559': supportsEIP1559,
      'supportsTokenTransfers': supportsTokenTransfers,
      'supportsNFT': supportsNFT,
      'supportsStaking': supportsStaking,
      'blockTime': blockTime.toJson(),
      'gasConfig': gasConfig.toJson(),
      'extra': extra,
    };
  }
  
  factory ChainConfig.fromJson(Map<String, dynamic> json) {
    return ChainConfig(
      derivationPath: json['derivationPath'],
      addressPrefix: json['addressPrefix'],
      supportsEIP1559: json['supportsEIP1559'] ?? false,
      supportsTokenTransfers: json['supportsTokenTransfers'] ?? true,
      supportsNFT: json['supportsNFT'] ?? false,
      supportsStaking: json['supportsStaking'] ?? false,
      blockTime: BlockTimeConfig.fromJson(json['blockTime']),
      gasConfig: GasConfig.fromJson(json['gasConfig']),
      extra: json['extra'] ?? {},
    );
  }
}

class BlockTimeConfig {
  final int targetTimeSeconds;
  final int confirmationBlocks;
  
  BlockTimeConfig({
    required this.targetTimeSeconds,
    required this.confirmationBlocks,
  });
  
  Map<String, dynamic> toJson() {
    return {
      'targetTimeSeconds': targetTimeSeconds,
      'confirmationBlocks': confirmationBlocks,
    };
  }
  
  factory BlockTimeConfig.fromJson(Map<String, dynamic> json) {
    return BlockTimeConfig(
      targetTimeSeconds: json['targetTimeSeconds'],
      confirmationBlocks: json['confirmationBlocks'],
    );
  }
}

class GasConfig {
  final String gasToken;
  final int decimals;
  final double minGasPrice;
  final double maxGasPrice;
  final double gasLimit;
  
  GasConfig({
    required this.gasToken,
    required this.decimals,
    required this.minGasPrice,
    required this.maxGasPrice,
    required this.gasLimit,
  });
  
  Map<String, dynamic> toJson() {
    return {
      'gasToken': gasToken,
      'decimals': decimals,
      'minGasPrice': minGasPrice,
      'maxGasPrice': maxGasPrice,
      'gasLimit': gasLimit,
    };
  }
  
  factory GasConfig.fromJson(Map<String, dynamic> json) {
    return GasConfig(
      gasToken: json['gasToken'],
      decimals: json['decimals'],
      minGasPrice: (json['minGasPrice'] as num).toDouble(),
      maxGasPrice: (json['maxGasPrice'] as num).toDouble(),
      gasLimit: (json['gasLimit'] as num).toDouble(),
    );
  }
}

class TokenOnChain {
  final String address;
  final String symbol;
  final String name;
  final int decimals;
  final String? iconUrl;
  final double? totalSupply;
  
  TokenOnChain({
    required this.address,
    required this.symbol,
    required this.name,
    required this.decimals,
    this.iconUrl,
    this.totalSupply,
  });
  
  Map<String, dynamic> toJson() {
    return {
      'address': address,
      'symbol': symbol,
      'name': name,
      'decimals': decimals,
      'iconUrl': iconUrl,
      'totalSupply': totalSupply,
    };
  }
  
  factory TokenOnChain.fromJson(Map<String, dynamic> json) {
    return TokenOnChain(
      address: json['address'],
      symbol: json['symbol'],
      name: json['name'],
      decimals: json['decimals'],
      iconUrl: json['iconUrl'],
      totalSupply: (json['totalSupply'] as num?)?.toDouble(),
    );
  }
}
