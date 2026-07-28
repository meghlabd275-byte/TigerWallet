// Token Model - Complete Token Data Structure

class TokenModel {
  final String id;
  final String name;
  final String symbol;
  final String address; // Contract address (empty for native)
  final String iconUrl;
  final double balance;
  final int decimals;
  final double balanceUSD;
  final String chainId;
  final String chainName;
  final double price;
  final double priceChange24h;
  final double volume24h;
  final double marketCap;
  final TokenType type;
  final bool isVerified;
  final String? contractAbi;
  
  TokenModel({
    required this.id,
    required this.name,
    required this.symbol,
    required this.address,
    required this.iconUrl,
    required this.balance,
    required this.decimals,
    required this.balanceUSD,
    required this.chainId,
    required this.chainName,
    required this.price,
    required this.priceChange24h,
    required this.volume24h,
    required this.marketCap,
    required this.type,
    this.isVerified = true,
    this.contractAbi,
  });
  
  bool get isNative => address.isEmpty;
  bool get isERC20 => type == TokenType.erc20;
  bool get isNFT => type == TokenType.erc721 || type == TokenType.erc1155;
  
  String get displayBalance {
    if (balance == 0) return '0';
    final value = balance / (10 * decimals);
    if (value < 0.0001) return '< 0.0001';
    return value.toStringAsFixed(6);
  }
  
  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'name': name,
      'symbol': symbol,
      'address': address,
      'iconUrl': iconUrl,
      'balance': balance,
      'decimals': decimals,
      'balanceUSD': balanceUSD,
      'chainId': chainId,
      'chainName': chainName,
      'price': price,
      'priceChange24h': priceChange24h,
      'volume24h': volume24h,
      'marketCap': marketCap,
      'type': type.name,
      'isVerified': isVerified,
      'contractAbi': contractAbi,
    };
  }
  
  factory TokenModel.fromJson(Map<String, dynamic> json) {
    return TokenModel(
      id: json['id'],
      name: json['name'],
      symbol: json['symbol'],
      address: json['address'],
      iconUrl: json['iconUrl'],
      balance: (json['balance'] as num).toDouble(),
      decimals: json['decimals'],
      balanceUSD: (json['balanceUSD'] as num).toDouble(),
      chainId: json['chainId'],
      chainName: json['chainName'],
      price: (json['price'] as num).toDouble(),
      priceChange24h: (json['priceChange24h'] as num).toDouble(),
      volume24h: (json['volume24h'] as num).toDouble(),
      marketCap: (json['marketCap'] as num).toDouble(),
      type: TokenType.values.firstWhere(
        (e) => e.name == json['type'],
        orElse: () => TokenType.erc20,
      ),
      isVerified: json['isVerified'] ?? true,
      contractAbi: json['contractAbi'],
    );
  }
  
  TokenModel copyWith({
    String? id,
    String? name,
    String? symbol,
    String? address,
    String? iconUrl,
    double? balance,
    int? decimals,
    double? balanceUSD,
    String? chainId,
    String? chainName,
    double? price,
    double? priceChange24h,
    double? volume24h,
    double? marketCap,
    TokenType? type,
    bool? isVerified,
    String? contractAbi,
  }) {
    return TokenModel(
      id: id ?? this.id,
      name: name ?? this.name,
      symbol: symbol ?? this.symbol,
      address: address ?? this.address,
      iconUrl: iconUrl ?? this.iconUrl,
      balance: balance ?? this.balance,
      decimals: decimals ?? this.decimals,
      balanceUSD: balanceUSD ?? this.balanceUSD,
      chainId: chainId ?? this.chainId,
      chainName: chainName ?? this.chainName,
      price: price ?? this.price,
      priceChange24h: priceChange24h ?? this.priceChange24h,
      volume24h: volume24h ?? this.volume24h,
      marketCap: marketCap ?? this.marketCap,
      type: type ?? this.type,
      isVerified: isVerified ?? this.isVerified,
      contractAbi: contractAbi ?? this.contractAbi,
    );
  }
}

enum TokenType {
  native,
  erc20,
  erc721,
  erc1155,
  spl, // Solana Program Library
  trc20,
  bep20,
  arc20,
  // Add more as needed
}

// NFT Model
class NFTModel {
  final String id;
  final String tokenId;
  final String contractAddress;
  final String name;
  final String? description;
  final String? imageUrl;
  final String? animationUrl;
  final String chainId;
  final String collectionName;
  final NFTType type;
  final String? attributes;
  final int? quantity;
  
  NFTModel({
    required this.id,
    required this.tokenId,
    required this.contractAddress,
    required this.name,
    this.description,
    this.imageUrl,
    this.animationUrl,
    required this.chainId,
    required this.collectionName,
    required this.type,
    this.attributes,
    this.quantity,
  });
  
  bool get isImage => type == NFTType.image;
  bool get isVideo => type == NFTType.video;
  bool get isAudio => type == NFTType.audio;
  bool get is3D => type == NFTType.model3d;
  
  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'tokenId': tokenId,
      'contractAddress': contractAddress,
      'name': name,
      'description': description,
      'imageUrl': imageUrl,
      'animationUrl': animationUrl,
      'chainId': chainId,
      'collectionName': collectionName,
      'type': type.name,
      'attributes': attributes,
      'quantity': quantity,
    };
  }
  
  factory NFTModel.fromJson(Map<String, dynamic> json) {
    return NFTModel(
      id: json['id'],
      tokenId: json['tokenId'],
      contractAddress: json['contractAddress'],
      name: json['name'],
      description: json['description'],
      imageUrl: json['imageUrl'],
      animationUrl: json['animationUrl'],
      chainId: json['chainId'],
      collectionName: json['collectionName'],
      type: NFTType.values.firstWhere(
        (e) => e.name == json['type'],
        orElse: () => NFTType.image,
      ),
      attributes: json['attributes'],
      quantity: json['quantity'],
    );
  }
}

enum NFTType {
  image,
  video,
  audio,
  model3d,
  application,
}
