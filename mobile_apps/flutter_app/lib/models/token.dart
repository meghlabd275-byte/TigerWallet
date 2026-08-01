/**
 * TigerWallet Token Model
 * Complete token data model
 */

class Token {
  final String address;
  final String symbol;
  final String name;
  final int decimals;
  final String chainId;
  final String logoUrl;
  final double balance;
  final double balanceUSD;
  final double price;
  final bool isNative;
  
  const Token({
    required this.address,
    required this.symbol,
    required this.name,
    required this.decimals,
    required this.chainId,
    this.logoUrl = '',
    this.balance = 0.0,
    this.balanceUSD = 0.0,
    this.price = 0.0,
    this.isNative = false,
  });
  
  factory Token.fromJson(Map<String, dynamic> json) {
    return Token(
      address: json['address'] ?? '',
      symbol: json['symbol'] ?? '',
      name: json['name'] ?? '',
      decimals: json['decimals'] ?? 18,
      chainId: json['chainId'] ?? '',
      logoUrl: json['logoUrl'] ?? '',
      balance: (json['balance'] ?? 0.0).toDouble(),
      balanceUSD: (json['balanceUSD'] ?? 0.0).toDouble(),
      price: (json['price'] ?? 0.0).toDouble(),
      isNative: json['isNative'] ?? false,
    );
  }
  
  Map<String, dynamic> toJson() {
    return {
      'address': address,
      'symbol': symbol,
      'name': name,
      'decimals': decimals,
      'chainId': chainId,
      'logoUrl': logoUrl,
      'balance': balance,
      'balanceUSD': balanceUSD,
      'price': price,
      'isNative': isNative,
    };
  }
  
  Token copyWith({
    String? address,
    String? symbol,
    String? name,
    int? decimals,
    String? chainId,
    String? logoUrl,
    double? balance,
    double? balanceUSD,
    double? price,
    bool? isNative,
  }) {
    return Token(
      address: address ?? this.address,
      symbol: symbol ?? this.symbol,
      name: name ?? this.name,
      decimals: decimals ?? this.decimals,
      chainId: chainId ?? this.chainId,
      logoUrl: logoUrl ?? this.logoUrl,
      balance: balance ?? this.balance,
      balanceUSD: balanceUSD ?? this.balanceUSD,
      price: price ?? this.price,
      isNative: isNative ?? this.isNative,
    );
  }
  
  @override
  bool operator ==(Object other) {
    if (identical(this, other)) return true;
    return other is Token && 
           other.address == address && 
           other.chainId == chainId;
  }
  
  @override
  int get hashCode => address.hashCode ^ chainId.hashCode;
  
  @override
  String toString() {
    return 'Token($symbol - $chainId)';
  }
}

class TokenList {
  final List<Token> tokens;
  
  const TokenList(this.tokens);
  
  factory TokenList.fromJson(List<dynamic> json) {
    return TokenList(
      json.map((e) => Token.fromJson(e)).toList(),
    );
  }
  
  List<Token> getByChain(String chainId) {
    return tokens.where((t) => t.chainId == chainId).toList();
  }
  
  List<Token> search(String query) {
    final q = query.toLowerCase();
    return tokens.where((t) => 
      t.symbol.toLowerCase().contains(q) ||
      t.name.toLowerCase().contains(q)
    ).toList();
  }
}
