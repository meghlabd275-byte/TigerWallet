import 'dart:convert';
import 'package:http/http.dart' as http;

/// NFT Marketplace Service for Flutter App
/// Production-ready NFT marketplace functionality
class NFTMarketplaceService {
  static final NFTMarketplaceService _instance = NFTMarketplaceService._internal();
  factory NFTMarketplaceService() => _instance;
  NFTMarketplaceService._internal();

  final String _baseUrl = 'http://localhost:8443/api/v1/nft';
  
  // Collection Methods
  
  /// Get collection by address
  Future<NFTCollection?> getCollection(String address, {String chain = 'ethereum'}) async {
    final response = await http.get(
      Uri.parse('$_baseUrl/collections/$address?chain=$chain'),
      headers: {'Content-Type': 'application/json'},
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return NFTCollection.fromJson(data);
    }
    return null;
  }
  
  /// Search collections
  Future<List<NFTCollection>> searchCollections(String query, {String chain = 'ethereum', int limit = 20}) async {
    final response = await http.get(
      Uri.parse('$_baseUrl/collections/search?q=$query&chain=$chain&limit=$limit'),
      headers: {'Content-Type': 'application/json'},
    );
    
    if (response.statusCode == 200) {
      final List<dynamic> data = json.decode(response.body);
      return data.map((e) => NFTCollection.fromJson(e)).toList();
    }
    return [];
  }
  
  // NFT Methods
  
  /// Get NFTs for a collection
  Future<List<NFTItem>> getNFTs(String collectionAddress, {String chain = 'ethereum', int limit = 50, int offset = 0}) async {
    final response = await http.get(
      Uri.parse('$_baseUrl/assets?collection=$collectionAddress&chain=$chain&limit=$limit&offset=$offset'),
      headers: {'Content-Type': 'application/json'},
    );
    
    if (response.statusCode == 200) {
      final List<dynamic> data = json.decode(response.body);
      return data.map((e) => NFTItem.fromJson(e)).toList();
    }
    return [];
  }
  
  /// Get NFTs for a wallet
  Future<List<NFTItem>> getUserNFTs(String walletAddress, {String chain = 'ethereum'}) async {
    final response = await http.get(
      Uri.parse('$_baseUrl/owners/$walletAddress?chain=$chain'),
      headers: {'Content-Type': 'application/json'},
    );
    
    if (response.statusCode == 200) {
      final List<dynamic> data = json.decode(response.body);
      return data.map((e) => NFTItem.fromJson(e)).toList();
    }
    return [];
  }
  
  // Listing Methods
  
  /// Get listings for a collection
  Future<List<NFTListing>> getListings(String collectionAddress, {String chain = 'ethereum', int limit = 50}) async {
    final response = await http.get(
      Uri.parse('$_baseUrl/listings?collection=$collectionAddress&chain=$chain&limit=$limit'),
      headers: {'Content-Type': 'application/json'},
    );
    
    if (response.statusCode == 200) {
      final List<dynamic> data = json.decode(response.body);
      return data.map((e) => NFTListing.fromJson(e)).toList();
    }
    return [];
  }
  
  /// Get user's listings
  Future<List<NFTListing>> getUserListings(String walletAddress, {String chain = 'ethereum'}) async {
    final response = await http.get(
      Uri.parse('$_baseUrl/listings/$walletAddress?chain=$chain'),
      headers: {'Content-Type': 'application/json'},
    );
    
    if (response.statusCode == 200) {
      final List<dynamic> data = json.decode(response.body);
      return data.map((e) => NFTListing.fromJson(e)).toList();
    }
    return [];
  }
  
  // Trading Methods
  
  /// Create a listing (sell NFT)
  Future<String> createListing({
    required String walletAddress,
    required String collectionAddress,
    required String tokenId,
    required double price,
    String priceToken = 'ETH',
    String chain = 'ethereum',
  }) async {
    final response = await http.post(
      Uri.parse('$_baseUrl/listings'),
      headers: {'Content-Type': 'application/json'},
      body: json.encode({
        'wallet_address': walletAddress,
        'collection_address': collectionAddress,
        'token_id': tokenId,
        'price': price,
        'price_token': priceToken,
        'chain': chain,
      }),
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return data['listingId'];
    }
    throw Exception('Failed to create listing');
  }
  
  /// Cancel a listing
  Future<bool> cancelListing(String walletAddress, String listingId, {String chain = 'ethereum'}) async {
    final response = await http.delete(
      Uri.parse('$_baseUrl/listings/$listingId'),
      headers: {'Content-Type': 'application/json'},
      body: json.encode({
        'wallet_address': walletAddress,
        'chain': chain,
      }),
    );
    
    return response.statusCode == 200;
  }
  
  /// Buy NFT (fulfill listing)
  Future<String> buyNFT(String buyerAddress, String listingId, {String chain = 'ethereum'}) async {
    final response = await http.post(
      Uri.parse('$_baseUrl/purchase'),
      headers: {'Content-Type': 'application/json'},
      body: json.encode({
        'buyer_address': buyerAddress,
        'listing_id': listingId,
        'chain': chain,
      }),
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return data['txHash'];
    }
    throw Exception('Purchase failed');
  }
  
  // Offer Methods
  
  /// Make an offer
  Future<String> makeOffer({
    required String makerAddress,
    required String collectionAddress,
    required String tokenId,
    required double price,
    String priceToken = 'ETH',
    required int expirationTime,
    String chain = 'ethereum',
  }) async {
    final response = await http.post(
      Uri.parse('$_baseUrl/offers'),
      headers: {'Content-Type': 'application/json'},
      body: json.encode({
        'maker_address': makerAddress,
        'collection_address': collectionAddress,
        'token_id': tokenId,
        'price': price,
        'price_token': priceToken,
        'expiration_time': expirationTime,
        'chain': chain,
      }),
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return data['offerId'];
    }
    throw Exception('Failed to create offer');
  }
  
  /// Accept an offer
  Future<String> acceptOffer(String sellerAddress, String offerId, {String chain = 'ethereum'}) async {
    final response = await http.post(
      Uri.parse('$_baseUrl/offers/$offerId/accept'),
      headers: {'Content-Type': 'application/json'},
      body: json.encode({
        'seller_address': sellerAddress,
        'chain': chain,
      }),
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return data['txHash'];
    }
    throw Exception('Failed to accept offer');
  }
  
  // Trade History
  
  /// Get trade history for a collection
  Future<List<NFTTrade>> getCollectionTrades(String collectionAddress, {String chain = 'ethereum', int limit = 50}) async {
    final response = await http.get(
      Uri.parse('$_baseUrl/trades?collection=$collectionAddress&chain=$chain&limit=$limit'),
      headers: {'Content-Type': 'application/json'},
    );
    
    if (response.statusCode == 200) {
      final List<dynamic> data = json.decode(response.body);
      return data.map((e) => NFTTrade.fromJson(e)).toList();
    }
    return [];
  }
}

// Models

class NFTCollection {
  final String id;
  final String address;
  final String name;
  final String symbol;
  final String? description;
  final String? imageUrl;
  final String? bannerUrl;
  final double floorPrice;
  final int totalSupply;
  final int owners;
  final double volume24h;
  final String chain;

  NFTCollection({
    required this.id,
    required this.address,
    required this.name,
    required this.symbol,
    this.description,
    this.imageUrl,
    this.bannerUrl,
    required this.floorPrice,
    required this.totalSupply,
    required this.owners,
    required this.volume24h,
    required this.chain,
  });

  factory NFTCollection.fromJson(Map<String, dynamic> json) {
    return NFTCollection(
      id: json['id'] ?? '',
      address: json['address'] ?? '',
      name: json['name'] ?? '',
      symbol: json['symbol'] ?? '',
      description: json['description'],
      imageUrl: json['imageUrl'],
      bannerUrl: json['bannerUrl'],
      floorPrice: (json['floorPrice'] ?? 0).toDouble(),
      totalSupply: json['totalSupply'] ?? 0,
      owners: json['owners'] ?? 0,
      volume24h: (json['volume24h'] ?? 0).toDouble(),
      chain: json['chain'] ?? '',
    );
  }
}

class NFTItem {
  final String id;
  final String tokenId;
  final String collectionAddress;
  final String name;
  final String? description;
  final String? imageUrl;
  final String? animationUrl;
  final List<NFTAttribute>? attributes;
  final String? owner;
  final double? price;
  final String chain;

  NFTItem({
    required this.id,
    required this.tokenId,
    required this.collectionAddress,
    required this.name,
    this.description,
    this.imageUrl,
    this.animationUrl,
    this.attributes,
    this.owner,
    this.price,
    required this.chain,
  });

  factory NFTItem.fromJson(Map<String, dynamic> json) {
    return NFTItem(
      id: json['id'] ?? '',
      tokenId: json['tokenId'] ?? '',
      collectionAddress: json['collectionAddress'] ?? '',
      name: json['name'] ?? '',
      description: json['description'],
      imageUrl: json['imageUrl'],
      animationUrl: json['animationUrl'],
      attributes: json['attributes'] != null 
          ? (json['attributes'] as List).map((e) => NFTAttribute.fromJson(e)).toList()
          : null,
      owner: json['owner'],
      price: json['price']?.toDouble(),
      chain: json['chain'] ?? '',
    );
  }
}

class NFTAttribute {
  final String traitType;
  final String value;

  NFTAttribute({required this.traitType, required this.value});

  factory NFTAttribute.fromJson(Map<String, dynamic> json) {
    return NFTAttribute(
      traitType: json['traitType'] ?? '',
      value: json['value'] ?? '',
    );
  }
}

class NFTListing {
  final String id;
  final String tokenId;
  final String collectionAddress;
  final String seller;
  final double price;
  final String priceToken;
  final int expirationTime;
  final String chain;
  final String? imageUrl;
  final String? name;

  NFTListing({
    required this.id,
    required this.tokenId,
    required this.collectionAddress,
    required this.seller,
    required this.price,
    required this.priceToken,
    required this.expirationTime,
    required this.chain,
    this.imageUrl,
    this.name,
  });

  factory NFTListing.fromJson(Map<String, dynamic> json) {
    return NFTListing(
      id: json['id'] ?? '',
      tokenId: json['tokenId'] ?? '',
      collectionAddress: json['collectionAddress'] ?? '',
      seller: json['seller'] ?? '',
      price: (json['price'] ?? 0).toDouble(),
      priceToken: json['priceToken'] ?? 'ETH',
      expirationTime: json['expirationTime'] ?? 0,
      chain: json['chain'] ?? '',
      imageUrl: json['imageUrl'],
      name: json['name'],
    );
  }
}

class NFTTrade {
  final String id;
  final String tokenId;
  final String collectionAddress;
  final String buyer;
  final String seller;
  final double price;
  final int timestamp;
  final String txHash;
  final String chain;

  NFTTrade({
    required this.id,
    required this.tokenId,
    required this.collectionAddress,
    required this.buyer,
    required this.seller,
    required this.price,
    required this.timestamp,
    required this.txHash,
    required this.chain,
  });

  factory NFTTrade.fromJson(Map<String, dynamic> json) {
    return NFTTrade(
      id: json['id'] ?? '',
      tokenId: json['tokenId'] ?? '',
      collectionAddress: json['collectionAddress'] ?? '',
      buyer: json['buyer'] ?? '',
      seller: json['seller'] ?? '',
      price: (json['price'] ?? 0).toDouble(),
      timestamp: json['timestamp'] ?? 0,
      txHash: json['txHash'] ?? '',
      chain: json['chain'] ?? '',
    );
  }
}
