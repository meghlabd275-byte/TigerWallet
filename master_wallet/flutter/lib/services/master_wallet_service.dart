/**
 * MasterWalletService - Flutter Implementation
 *
 * Thin REST client over the canonical Go backend (:8450). All key material,
 * signing, and balance resolution happen server-side (real secp256k1 signing +
 * RPC broadcast via /api/v1/master-wallet/:id/sign). There is NO in-memory
 * wallet, NO local signing, and NO fabricated data.
 *
 * See master_wallet/CANONICAL_API_CONTRACT.md for the full contract.
 */

import 'dart:convert';
import 'package:http/http.dart' as http;

class MasterWalletService {
  // Canonical backend base URL (port 8450). Configurable via the
  // MASTER_WALLET_API_URL environment / build flag.
  static const String API_BASE = String.fromEnvironment(
    'MASTER_WALLET_API_URL',
    defaultValue: 'http://localhost:8450',
  );

  static const String _apiV1 = '$API_BASE/api/v1';

  // Chain IDs (metadata only - the backend is the source of truth for balances).
  static const int CHAIN_ETHEREUM = 1;
  static const int CHAIN_BSC = 56;
  static const int CHAIN_POLYGON = 137;
  static const int CHAIN_ARBITRUM = 42161;
  static const int CHAIN_OPTIMISM = 10;
  static const int CHAIN_AVALANCHE = 43114;

  String? _token;

  MasterWalletService({String? token}) : _token = token;

  void setToken(String? token) => _token = token;

  Map<String, String> get _headers => {
        'Content-Type': 'application/json',
        if (_token != null) 'Authorization': 'Bearer $_token',
      };

  Exception _error(http.Response r) =>
      Exception('backend error ${r.statusCode}: ${r.body}');

  // ==================== Wallets ====================

  /// List master wallets for the authenticated user.
  Future<List<WalletData>> getWallets() async {
    final r = await http.get(Uri.parse('$_apiV1/master-wallet'), headers: _headers);
    if (r.statusCode != 200) throw _error(r);
    final body = jsonDecode(r.body) as Map<String, dynamic>;
    final wallets = body['wallets'] as List? ?? [];
    return wallets.map((w) => WalletData.fromJson(w as Map<String, dynamic>)).toList();
  }

  /// Create a master wallet on the backend. The mnemonic is generated and
  /// returned ONCE server-side (real BIP-39); it is never stored locally.
  Future<WalletResult> createWallet({
    required String name,
    required String password,
    int chainId = CHAIN_ETHEREUM,
  }) async {
    try {
      final r = await http.post(
        Uri.parse('$_apiV1/master-wallet'),
        headers: _headers,
        body: jsonEncode({
          'name': name,
          'password': password,
          'chain_id': chainId,
        }),
      );
      if (r.statusCode != 200 && r.statusCode != 201) {
        return WalletResult(success: false, error: 'backend ${r.statusCode}: ${r.body}');
      }
      final data = jsonDecode(r.body) as Map<String, dynamic>;
      final walletId = (data['wallet_id'] ?? data['id']) as String? ?? '';
      return WalletResult(
        success: true,
        walletId: walletId,
        address: data['address'] as String?,
        mnemonic: data['mnemonic'] as String?,
      );
    } catch (e) {
      return WalletResult(success: false, error: e.toString());
    }
  }

  /// Fetch a single master wallet by id.
  Future<WalletData> getWallet(String walletId) async {
    final r = await http.get(
      Uri.parse('$_apiV1/master-wallet/$walletId'),
      headers: _headers,
    );
    if (r.statusCode != 200) throw _error(r);
    return WalletData.fromJson(jsonDecode(r.body) as Map<String, dynamic>);
  }

  Future<bool> deleteWallet(String walletId) async {
    final r = await http.delete(
      Uri.parse('$_apiV1/master-wallet/$walletId'),
      headers: _headers,
    );
    return r.statusCode == 200 || r.statusCode == 204;
  }

  // ==================== Balance (real RPC, server-side) ====================

  /// Get native balance from the backend's GET /balance endpoint (real RPC).
  Future<BalanceResult> getBalance(String walletId, int chainId) async {
    try {
      final r = await http.get(
        Uri.parse('$_apiV1/master-wallet/$walletId/balance'),
        headers: _headers,
      );
      if (r.statusCode != 200) {
        return BalanceResult(success: false, error: 'backend ${r.statusCode}: ${r.body}');
      }
      final data = jsonDecode(r.body) as Map<String, dynamic>;
      final native = (data['native'] as Map<String, dynamic>?) ?? {};
      return BalanceResult(
        success: true,
        balance: _toDouble(native['balance']),
        symbol: native['symbol'] as String? ?? '',
        decimals: 18,
        usdValue: _toDouble(data['usd_value']),
      );
    } catch (e) {
      return BalanceResult(success: false, error: e.toString());
    }
  }

  /// Get token balances from the same balance endpoint (the backend resolves
  /// live ERC-20 balances for tracked tokens). The [tokenAddress] filter is
  /// applied client-side against the returned token list.
  Future<TokenBalanceResult> getTokenBalance(
    String walletId,
    int chainId,
    String tokenAddress,
  ) async {
    try {
      final r = await http.get(
        Uri.parse('$_apiV1/master-wallet/$walletId/balance'),
        headers: _headers,
      );
      if (r.statusCode != 200) {
        return TokenBalanceResult(
          success: false,
          error: 'backend ${r.statusCode}: ${r.body}',
        );
      }
      final data = jsonDecode(r.body) as Map<String, dynamic>;
      final tokens = (data['tokens'] as List?) ?? const [];
      for (final t in tokens) {
        final m = t as Map<String, dynamic>;
        final contract = (m['contract'] ?? m['address']) as String? ?? '';
        if (contract.toLowerCase() == tokenAddress.toLowerCase()) {
          return TokenBalanceResult(
            success: true,
            balance: _toStr(m['balance']),
            symbol: m['symbol'] as String? ?? 'TOKEN',
            decimals: (m['decimals'] as num?)?.toInt() ?? 18,
          );
        }
      }
      return TokenBalanceResult(
        success: false,
        error: 'Token not tracked for this wallet',
      );
    } catch (e) {
      return TokenBalanceResult(success: false, error: e.toString());
    }
  }

  // ==================== Send / Sign (real, server-side) ====================

  /// Send a transaction via the backend's POST /sign endpoint. The backend
  /// decrypts the wallet seed with [password], derives the secp256k1 key,
  /// builds + signs the EIP-1559 tx, broadcasts via eth_sendRawTransaction,
  /// and returns the REAL node transaction hash.
  Future<TransactionResult> sendTransaction({
    required String walletId,
    required int chainId,
    required String toAddress,
    required BigInt amount,
    required String password,
    String? token,
  }) async {
    try {
      final r = await http.post(
        Uri.parse('$_apiV1/master-wallet/$walletId/sign'),
        headers: _headers,
        body: jsonEncode({
          'to': toAddress,
          'amount': _formatAmount(amount),
          'password': password,
          if (token != null) 'token': token,
        }),
      );
      if (r.statusCode != 200) {
        return TransactionResult(
          success: false,
          error: 'backend ${r.statusCode}: ${r.body}',
        );
      }
      final data = jsonDecode(r.body) as Map<String, dynamic>;
      return TransactionResult(
        success: true,
        txHash: data['transaction_hash'] as String?,
        from: data['from'] as String?,
        to: toAddress,
        amount: _formatAmount(amount),
      );
    } catch (e) {
      return TransactionResult(success: false, error: e.toString());
    }
  }

  // ==================== Transactions ====================

  Future<List<Map<String, dynamic>>> getTransactions(String walletId) async {
    final r = await http.get(
      Uri.parse('$_apiV1/master-wallet/$walletId/transactions'),
      headers: _headers,
    );
    if (r.statusCode != 200) throw _error(r);
    final body = jsonDecode(r.body) as Map<String, dynamic>;
    final txs = body['transactions'] as List? ?? [];
    return txs.cast<Map<String, dynamic>>();
  }

  // ==================== Helpers ====================

  double _toDouble(dynamic v) {
    if (v is num) return v.toDouble();
    if (v is String) return double.tryParse(v) ?? 0;
    return 0;
  }

  String _toStr(dynamic v) => v?.toString() ?? '0';

  String _formatAmount(BigInt amount) {
    // The backend accepts human-readable amounts (e.g. "0.5"). Convert wei.
    final whole = amount ~/ BigInt.from(10).pow(18);
    final frac = amount.remainder(BigInt.from(10).pow(18));
    if (frac == BigInt.zero) return whole.toString();
    final fracStr = frac.toString().padLeft(18, '0').replaceAll(RegExp(r'0+$'), '');
    return '$whole.$fracStr';
  }
}

// ==================== Data Classes ====================

class ChainConfig {
  final int id;
  final String name;
  final String symbol;
  final String rpcUrl;
  final String explorerUrl;
  final int decimals;
  final bool isEVM;

  ChainConfig({
    required this.id,
    required this.name,
    required this.symbol,
    required this.rpcUrl,
    required this.explorerUrl,
    required this.decimals,
    required this.isEVM,
  });
}

class WalletData {
  final String id;
  final String address;
  final String publicKey;
  final String name;
  final int chainId;
  final int createdAt;

  WalletData({
    required this.id,
    required this.address,
    this.publicKey = '',
    this.name = '',
    this.chainId = 1,
    this.createdAt = 0,
  });

  factory WalletData.fromJson(Map<String, dynamic> json) {
    return WalletData(
      id: json['id'] as String? ?? '',
      address: json['address'] as String? ?? '',
      publicKey: json['public_key'] as String? ?? '',
      name: json['name'] as String? ?? '',
      chainId: (json['chain_id'] as num?)?.toInt() ?? 1,
      createdAt: json['created_at'] is int
          ? json['created_at'] as int
          : (json['created_at'] != null
              ? DateTime.tryParse(json['created_at'].toString())
                      ?.millisecondsSinceEpoch ??
                  0
              : 0),
    );
  }
}

class WalletResult {
  final bool success;
  final String? walletId;
  final String? address;
  final String? mnemonic;
  final String? error;

  WalletResult({
    required this.success,
    this.walletId,
    this.address,
    this.mnemonic,
    this.error,
  });
}

class BalanceResult {
  final bool success;
  final double balance;
  final String symbol;
  final int decimals;
  final double usdValue;
  final String? error;

  BalanceResult({
    required this.success,
    required this.balance,
    required this.symbol,
    required this.decimals,
    this.usdValue = 0,
    this.error,
  });
}

class TokenBalanceResult {
  final bool success;
  final String balance;
  final String symbol;
  final int decimals;
  final String? error;

  TokenBalanceResult({
    required this.success,
    required this.balance,
    required this.symbol,
    required this.decimals,
    this.error,
  });
}

class TransactionResult {
  final bool success;
  final String? txHash;
  final String? from;
  final String? to;
  final String? amount;
  final String? error;

  TransactionResult({
    required this.success,
    this.txHash,
    this.from,
    this.to,
    this.amount,
    this.error,
  });
}
