// NFT Service - Complete NFT Management

import '../../services/api_service.dart';

class NftService {
  final ApiService _api = ApiService.instance;
  
  // Get NFTs for wallet
  Future<List<NFT>> getNfts(String walletAddress, {String? chainId}) async {
    final response = await _api.get('/nft/$walletAddress', queryParams: {
      if (chainId != null) 'chainId': chainId,
    });
    
    if (response.success) {
      return (response.data as List).map((n) => NFT.fromJson(n)).toList();
    }
    return [];
  }
  
  // Get NFT collection
  Future<NFTCollection> getCollection(String contractAddress, String chainId) async {
    final response = await _api.get('/nft/collection/$contractAddress', queryParams: {
      'chainId': chainId,
    });
    
    if (response.success) {
      return NFTCollection.fromJson(response.data);
    }
    throw Exception(response.error);
  }
  
  // Transfer NFT
  Future<String> transferNft({
    required String contractAddress,
    required String tokenId,
    required String toAddress,
    required String fromAddress,
    required String chainId,
  }) async {
    final response = await _api.post('/nft/transfer', body: {
      'contractAddress': contractAddress,
      'tokenId': tokenId,
      'toAddress': toAddress,
      'fromAddress': fromAddress,
      'chainId': chainId,
    });
    
    if (response.success) {
      return response.data['txHash'];
    }
    throw Exception(response.error);
  }
  
  // Get NFT metadata
  Future<NFTMetadata> getMetadata(String contractAddress, String tokenId, String chainId) async {
    final response = await _api.get('/nft/metadata/$contractAddress/$tokenId', queryParams: {
      'chainId': chainId,
    });
    
    if (response.success) {
      return NFTMetadata.fromJson(response.data);
    }
    throw Exception(response.error);
  }
  
  // Get popular collections
  Future<List<NFTCollection>> getPopularCollections({int limit = 10}) async {
    final response = await _api.get('/nft/popular-collections', queryParams: {
      'limit': limit.toString(),
    });
    
    if (response.success) {
      return (response.data as List).map((c) => NFTCollection.fromJson(c)).toList();
    }
    return [];
  }
}

class NFT {
  final String id;
  final String tokenId;
  final String contractAddress;
  final String name;
  final String? description;
  final String imageUrl;
  final String? animationUrl;
  final String chainId;
  final String collectionName;
  final NFTType type;
  final List<NFTAttribute> attributes;
  final String? owner;
  
  NFT({
    required this.id,
    required this.tokenId,
    required this.contractAddress,
    required this.name,
    this.description,
    required this.imageUrl,
    this.animationUrl,
    required this.chainId,
    required this.collectionName,
    required this.type,
    required this.attributes,
    this.owner,
  });
  
  factory NFT.fromJson(Map<String, dynamic> json) {
    return NFT(
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
        orElse: () => NFTType.ERC721,
      ),
      attributes: (json['attributes'] as List?)
          ?.map((a) => NFTAttribute.fromJson(a))
          .toList() ?? [],
      owner: json['owner'],
    );
  }
}

enum NFTType {
  ERC721,
  ERC1155,
  SPL, // Solana
}

class NFTAttribute {
  final String traitType;
  final String value;
  final String? displayType;
  
  NFTAttribute({
    required this.traitType,
    required this.value,
    this.displayType,
  });
  
  factory NFTAttribute.fromJson(Map<String, dynamic> json) {
    return NFTAttribute(
      traitType: json['traitType'],
      value: json['value'],
      displayType: json['displayType'],
    );
  }
}

class NFTMetadata {
  final String name;
  final String? description;
  final String imageUrl;
  final String? animationUrl;
  final List<NFTAttribute> attributes;
  final Map<String, dynamic> extra;
  
  NFTMetadata({
    required this.name,
    this.description,
    required this.imageUrl,
    this.animationUrl,
    required this.attributes,
    required this.extra,
  });
  
  factory NFTMetadata.fromJson(Map<String, dynamic> json) {
    return NFTMetadata(
      name: json['name'],
      description: json['description'],
      imageUrl: json['image'],
      animationUrl: json['animation_url'],
      attributes: (json['attributes'] as List?)
          ?.map((a) => NFTAttribute.fromJson(a))
          .toList() ?? [],
      extra: json,
    );
  }
}

class NFTCollection {
  final String address;
  final String name;
  final String symbol;
  final String description;
  final String imageUrl;
  final String bannerUrl;
  final String chainId;
  final String totalSupply;
  final String holderCount;
  final String floorPrice;
  final String volume24h;
  
  NFTCollection({
    required this.address,
    required this.name,
    required this.symbol,
    required this.description,
    required this.imageUrl,
    required this.bannerUrl,
    required this.chainId,
    required this.totalSupply,
    required this.holderCount,
    required this.floorPrice,
    required this.volume24h,
  });
  
  factory NFTCollection.fromJson(Map<String, dynamic> json) {
    return NFTCollection(
      address: json['address'],
      name: json['name'],
      symbol: json['symbol'],
      description: json['description'],
      imageUrl: json['imageUrl'],
      bannerUrl: json['bannerUrl'],
      chainId: json['chainId'],
      totalSupply: json['totalSupply'],
      holderCount: json['holderCount'],
      floorPrice: json['floorPrice'],
      volume24h: json['volume24h'],
    );
  }
}
