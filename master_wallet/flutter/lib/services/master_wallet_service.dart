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

  /// Create a transaction RECORD (distinct from [sendTransaction], which
  /// POSTs to /sign and signs+broadcasts on the backend). This POSTs to the
  /// canonical POST /master-wallet/:id/transactions route with body
  /// {to, value, data, chain_id} and returns the recorded transaction object.
  Future<Map<String, dynamic>> createTransactionRecord({
    required String masterId,
    required String to,
    required String value,
    required String data,
    required int chainId,
  }) async {
    final r = await http.post(
      Uri.parse('$_apiV1/master-wallet/$masterId/transactions'),
      headers: _headers,
      body: jsonEncode({
        'to': to,
        'value': value,
        'data': data,
        'chain_id': chainId,
      }),
    );
    if (r.statusCode != 200 && r.statusCode != 201) throw _error(r);
    return jsonDecode(r.body) as Map<String, dynamic>;
  }

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

  /// Approve a pending transaction via the canonical
  /// POST /master-wallet/:id/transactions/:tid/approve route.
  Future<bool> approveTransaction(String walletId, String txId) async {
    final r = await http.post(
      Uri.parse('$_apiV1/master-wallet/$walletId/transactions/$txId/approve'),
      headers: _headers,
    );
    if (r.statusCode != 200) throw _error(r);
    return true;
  }

  /// Reject a pending transaction via the canonical
  /// POST /master-wallet/:id/transactions/:tid/reject route.
  Future<bool> rejectTransaction(String walletId, String txId) async {
    final r = await http.post(
      Uri.parse('$_apiV1/master-wallet/$walletId/transactions/$txId/reject'),
      headers: _headers,
    );
    if (r.statusCode != 200) throw _error(r);
    return true;
  }

  // ==================== UserWallet: EVM chain management ====================

  /// GET /master-wallet/:id/user-chains/evm → {chains: [...]} (or raw list).
  Future<List<Map<String, dynamic>>> listUserEVMChains(String walletId) async {
    final r = await http.get(
      Uri.parse('$_apiV1/master-wallet/$walletId/user-chains/evm'),
      headers: _headers,
    );
    if (r.statusCode != 200) throw _error(r);
    final body = jsonDecode(r.body);
    final List list;
    if (body is List) {
      list = body;
    } else {
      final m = body as Map<String, dynamic>;
      list = (m['chains'] as List? ?? m['data'] as List? ?? const []) as List;
    }
    return list.cast<Map<String, dynamic>>();
  }

  /// POST /master-wallet/:id/user-chains/evm
  /// body: {chain_id, name, symbol, rpc_url, explorer_url, decimals, derivation_path}
  Future<Map<String, dynamic>> addUserEVMChain(
    String walletId,
    Map<String, dynamic> chain,
  ) async {
    final r = await http.post(
      Uri.parse('$_apiV1/master-wallet/$walletId/user-chains/evm'),
      headers: _headers,
      body: jsonEncode(chain),
    );
    if (r.statusCode != 200 && r.statusCode != 201) throw _error(r);
    return jsonDecode(r.body) as Map<String, dynamic>;
  }

  /// PUT /master-wallet/:id/user-chains/evm/:chainId
  Future<Map<String, dynamic>> updateUserEVMChain(
    String walletId,
    String chainId,
    Map<String, dynamic> chain,
  ) async {
    final r = await http.put(
      Uri.parse('$_apiV1/master-wallet/$walletId/user-chains/evm/$chainId'),
      headers: _headers,
      body: jsonEncode(chain),
    );
    if (r.statusCode != 200) throw _error(r);
    return jsonDecode(r.body) as Map<String, dynamic>;
  }

  /// DELETE /master-wallet/:id/user-chains/evm/:chainId
  Future<bool> removeUserEVMChain(String walletId, String chainId) async {
    final r = await http.delete(
      Uri.parse('$_apiV1/master-wallet/$walletId/user-chains/evm/$chainId'),
      headers: _headers,
    );
    if (r.statusCode != 200 && r.statusCode != 204) throw _error(r);
    return true;
  }

  // ==================== UserWallet: Non-EVM chain management ====================

  /// GET /master-wallet/:id/user-chains/nonevm → {chains: [...]} (or raw list).
  Future<List<Map<String, dynamic>>> listUserNonEVMChains(String walletId) async {
    final r = await http.get(
      Uri.parse('$_apiV1/master-wallet/$walletId/user-chains/nonevm'),
      headers: _headers,
    );
    if (r.statusCode != 200) throw _error(r);
    final body = jsonDecode(r.body);
    final List list;
    if (body is List) {
      list = body;
    } else {
      final m = body as Map<String, dynamic>;
      list = (m['chains'] as List? ?? m['data'] as List? ?? const []) as List;
    }
    return list.cast<Map<String, dynamic>>();
  }

  /// POST /master-wallet/:id/user-chains/nonevm
  /// body: {chain_id, name, symbol, chain_type, rpc_url, derivation_path, address_prefix}
  Future<Map<String, dynamic>> addUserNonEVMChain(
    String walletId,
    Map<String, dynamic> chain,
  ) async {
    final r = await http.post(
      Uri.parse('$_apiV1/master-wallet/$walletId/user-chains/nonevm'),
      headers: _headers,
      body: jsonEncode(chain),
    );
    if (r.statusCode != 200 && r.statusCode != 201) throw _error(r);
    return jsonDecode(r.body) as Map<String, dynamic>;
  }

  /// PUT /master-wallet/:id/user-chains/nonevm/:chainId
  Future<Map<String, dynamic>> updateUserNonEVMChain(
    String walletId,
    String chainId,
    Map<String, dynamic> chain,
  ) async {
    final r = await http.put(
      Uri.parse('$_apiV1/master-wallet/$walletId/user-chains/nonevm/$chainId'),
      headers: _headers,
      body: jsonEncode(chain),
    );
    if (r.statusCode != 200) throw _error(r);
    return jsonDecode(r.body) as Map<String, dynamic>;
  }

  /// DELETE /master-wallet/:id/user-chains/nonevm/:chainId
  Future<bool> removeUserNonEVMChain(String walletId, String chainId) async {
    final r = await http.delete(
      Uri.parse('$_apiV1/master-wallet/$walletId/user-chains/nonevm/$chainId'),
      headers: _headers,
    );
    if (r.statusCode != 200 && r.statusCode != 204) throw _error(r);
    return true;
  }

  // ==================== UserWallet: Token management ====================

  /// GET /master-wallet/:id/user-tokens?chain_id= → {tokens: [...]} (or raw list).
  Future<List<Map<String, dynamic>>> listUserTokens(
    String walletId, {
    String? chainId,
  }) async {
    final uri = Uri.parse('$_apiV1/master-wallet/$walletId/user-tokens');
    final r = await http.get(
      chainId == null ? uri : uri.replace(queryParameters: {'chain_id': chainId}),
      headers: _headers,
    );
    if (r.statusCode != 200) throw _error(r);
    final body = jsonDecode(r.body);
    final List list;
    if (body is List) {
      list = body;
    } else {
      final m = body as Map<String, dynamic>;
      list = (m['tokens'] as List? ?? m['data'] as List? ?? const []) as List;
    }
    return list.cast<Map<String, dynamic>>();
  }

  /// POST /master-wallet/:id/user-tokens
  /// body: {chain_id, contract_address, symbol, name, decimals, is_native}
  Future<Map<String, dynamic>> addUserToken(
    String walletId,
    Map<String, dynamic> token,
  ) async {
    final r = await http.post(
      Uri.parse('$_apiV1/master-wallet/$walletId/user-tokens'),
      headers: _headers,
      body: jsonEncode(token),
    );
    if (r.statusCode != 200 && r.statusCode != 201) throw _error(r);
    return jsonDecode(r.body) as Map<String, dynamic>;
  }

  /// PUT /master-wallet/:id/user-tokens/:tokenId
  Future<Map<String, dynamic>> updateUserToken(
    String walletId,
    String tokenId,
    Map<String, dynamic> token,
  ) async {
    final r = await http.put(
      Uri.parse('$_apiV1/master-wallet/$walletId/user-tokens/$tokenId'),
      headers: _headers,
      body: jsonEncode(token),
    );
    if (r.statusCode != 200) throw _error(r);
    return jsonDecode(r.body) as Map<String, dynamic>;
  }

  /// DELETE /master-wallet/:id/user-tokens/:tokenId
  Future<bool> removeUserToken(String walletId, String tokenId) async {
    final r = await http.delete(
      Uri.parse('$_apiV1/master-wallet/$walletId/user-tokens/$tokenId'),
      headers: _headers,
    );
    if (r.statusCode != 200 && r.statusCode != 204) throw _error(r);
    return true;
  }

  // ==================== UserWallet: Address derivation ====================

  /// POST /master-wallet/:id/derive-user-address
  /// body: {mnemonic, chain_id, chain_type, derivation_path, account_index}
  Future<Map<String, dynamic>> deriveUserAddress(
    String walletId,
    Map<String, dynamic> request,
  ) async {
    final r = await http.post(
      Uri.parse('$_apiV1/master-wallet/$walletId/derive-user-address'),
      headers: _headers,
      body: jsonEncode(request),
    );
    if (r.statusCode != 200 && r.statusCode != 201) throw _error(r);
    return jsonDecode(r.body) as Map<String, dynamic>;
  }

  /// GET /master-wallet/:id/user-wallet-addresses → {addresses: [...]} (or raw list).
  Future<List<Map<String, dynamic>>> listUserWalletAddresses(
    String walletId,
  ) async {
    final r = await http.get(
      Uri.parse('$_apiV1/master-wallet/$walletId/user-wallet-addresses'),
      headers: _headers,
    );
    if (r.statusCode != 200) throw _error(r);
    final body = jsonDecode(r.body);
    final List list;
    if (body is List) {
      list = body;
    } else {
      final m = body as Map<String, dynamic>;
      list = (m['addresses'] as List? ?? m['data'] as List? ?? const []) as List;
    }
    return list.cast<Map<String, dynamic>>();
  }

  // ==================== UserWallet: Auto-sign ====================

  /// POST /master-wallet/:id/auto-sign-transaction
  /// body: {mnemonic, chain_id, chain_type, tx_type, to_address, value, token_address}
  Future<Map<String, dynamic>> autoSignTransaction(
    String walletId,
    Map<String, dynamic> request,
  ) async {
    final r = await http.post(
      Uri.parse('$_apiV1/master-wallet/$walletId/auto-sign-transaction'),
      headers: _headers,
      body: jsonEncode(request),
    );
    if (r.statusCode != 200 && r.statusCode != 201) throw _error(r);
    return jsonDecode(r.body) as Map<String, dynamic>;
  }

  /// GET /master-wallet/:id/auto-sign-logs → {logs: [...]} (or raw list).
  Future<List<Map<String, dynamic>>> listAutoSignLogs(String walletId) async {
    final r = await http.get(
      Uri.parse('$_apiV1/master-wallet/$walletId/auto-sign-logs'),
      headers: _headers,
    );
    if (r.statusCode != 200) throw _error(r);
    final body = jsonDecode(r.body);
    final List list;
    if (body is List) {
      list = body;
    } else {
      final m = body as Map<String, dynamic>;
      list = (m['logs'] as List? ?? m['data'] as List? ?? const []) as List;
    }
    return list.cast<Map<String, dynamic>>();
  }

  // ==================== UserWallet: Feature flags ====================

  /// GET /master-wallet/:id/feature-flags → {flags: [...]} (or raw list).
  Future<List<Map<String, dynamic>>> listFeatureFlags(String walletId) async {
    final r = await http.get(
      Uri.parse('$_apiV1/master-wallet/$walletId/feature-flags'),
      headers: _headers,
    );
    if (r.statusCode != 200) throw _error(r);
    final body = jsonDecode(r.body);
    final List list;
    if (body is List) {
      list = body;
    } else {
      final m = body as Map<String, dynamic>;
      list = (m['flags'] as List? ?? m['data'] as List? ?? const []) as List;
    }
    return list.cast<Map<String, dynamic>>();
  }

  /// POST /master-wallet/:id/feature-flags
  /// body: {flag_key, flag_value, description, is_enabled}
  Future<Map<String, dynamic>> addFeatureFlag(
    String walletId,
    Map<String, dynamic> flag,
  ) async {
    final r = await http.post(
      Uri.parse('$_apiV1/master-wallet/$walletId/feature-flags'),
      headers: _headers,
      body: jsonEncode(flag),
    );
    if (r.statusCode != 200 && r.statusCode != 201) throw _error(r);
    return jsonDecode(r.body) as Map<String, dynamic>;
  }

  /// PUT /master-wallet/:id/feature-flags/:flagId
  Future<Map<String, dynamic>> updateFeatureFlag(
    String walletId,
    String flagId,
    Map<String, dynamic> flag,
  ) async {
    final r = await http.put(
      Uri.parse('$_apiV1/master-wallet/$walletId/feature-flags/$flagId'),
      headers: _headers,
      body: jsonEncode(flag),
    );
    if (r.statusCode != 200) throw _error(r);
    return jsonDecode(r.body) as Map<String, dynamic>;
  }

  /// DELETE /master-wallet/:id/feature-flags/:flagId
  Future<bool> removeFeatureFlag(String walletId, String flagId) async {
    final r = await http.delete(
      Uri.parse('$_apiV1/master-wallet/$walletId/feature-flags/$flagId'),
      headers: _headers,
    );
    if (r.statusCode != 200 && r.statusCode != 204) throw _error(r);
    return true;
  }

  // ==================== Fees ====================

  /// GET /master-wallet/:id/fees → {fees: [...]}
  Future<List<Map<String, dynamic>>> getFees(String walletId) async {
    final r = await http.get(
      Uri.parse('$_apiV1/master-wallet/$walletId/fees'),
      headers: _headers,
    );
    if (r.statusCode != 200) throw _error(r);
    final body = jsonDecode(r.body) as Map<String, dynamic>;
    final fees = body['fees'] as List? ?? [];
    return fees.cast<Map<String, dynamic>>();
  }

  /// POST /master-wallet/:id/fees — body: {fee_type, fee_percentage?, fee_fixed?, is_active?}
  Future<Map<String, dynamic>> createFee(
    String walletId,
    Map<String, dynamic> fee,
  ) async {
    final r = await http.post(
      Uri.parse('$_apiV1/master-wallet/$walletId/fees'),
      headers: _headers,
      body: jsonEncode(fee),
    );
    if (r.statusCode != 200 && r.statusCode != 201) throw _error(r);
    return jsonDecode(r.body) as Map<String, dynamic>;
  }

  /// DELETE /master-wallet/:id/fees/:fid
  Future<bool> deleteFee(String walletId, String feeId) async {
    final r = await http.delete(
      Uri.parse('$_apiV1/master-wallet/$walletId/fees/$feeId'),
      headers: _headers,
    );
    if (r.statusCode != 200 && r.statusCode != 204) throw _error(r);
    return true;
  }

  // ==================== Auto-Sign Rules ====================

  /// GET /master-wallet/:id/auto-sign → {auto_sign_rules: [...]}
  Future<List<Map<String, dynamic>>> getAutoSignRules(String walletId) async {
    final r = await http.get(
      Uri.parse('$_apiV1/master-wallet/$walletId/auto-sign'),
      headers: _headers,
    );
    if (r.statusCode != 200) throw _error(r);
    final body = jsonDecode(r.body) as Map<String, dynamic>;
    final rules = body['auto_sign_rules'] as List? ??
        body['rules'] as List? ??
        const [];
    return rules.cast<Map<String, dynamic>>();
  }

  /// POST /master-wallet/:id/auto-sign — body: {name, rule_type, conditions?, max_amount?, is_active?}
  Future<Map<String, dynamic>> createAutoSignRule(
    String walletId,
    Map<String, dynamic> rule,
  ) async {
    final r = await http.post(
      Uri.parse('$_apiV1/master-wallet/$walletId/auto-sign'),
      headers: _headers,
      body: jsonEncode(rule),
    );
    if (r.statusCode != 200 && r.statusCode != 201) throw _error(r);
    return jsonDecode(r.body) as Map<String, dynamic>;
  }

  /// DELETE /master-wallet/:id/auto-sign/:rid
  Future<bool> deleteAutoSignRule(String walletId, String ruleId) async {
    final r = await http.delete(
      Uri.parse('$_apiV1/master-wallet/$walletId/auto-sign/$ruleId'),
      headers: _headers,
    );
    if (r.statusCode != 200 && r.statusCode != 204) throw _error(r);
    return true;
  }

  // ==================== Sub Wallets ====================

  /// GET /master-wallet/:id/sub-wallets → {sub_wallets: [...]}
  Future<List<Map<String, dynamic>>> getSubWallets(String walletId) async {
    final r = await http.get(
      Uri.parse('$_apiV1/master-wallet/$walletId/sub-wallets'),
      headers: _headers,
    );
    if (r.statusCode != 200) throw _error(r);
    final body = jsonDecode(r.body) as Map<String, dynamic>;
    final subWallets = body['sub_wallets'] as List? ?? const [];
    return subWallets.cast<Map<String, dynamic>>();
  }

  /// POST /master-wallet/:id/sub-wallets — body: {name, password, chain_id}
  Future<Map<String, dynamic>> createSubWallet(
    String walletId,
    String name,
    String password,
    int chainId,
  ) async {
    final r = await http.post(
      Uri.parse('$_apiV1/master-wallet/$walletId/sub-wallets'),
      headers: _headers,
      body: jsonEncode({
        'name': name,
        'password': password,
        'chain_id': chainId,
      }),
    );
    if (r.statusCode != 200 && r.statusCode != 201) throw _error(r);
    return jsonDecode(r.body) as Map<String, dynamic>;
  }

  /// GET /master-wallet/:id/sub-wallets/:sid/balance → {native: {balance, symbol}, ...}
  Future<Map<String, dynamic>> getSubWalletBalance(
    String walletId,
    String subWalletId,
  ) async {
    final r = await http.get(
      Uri.parse(
        '$_apiV1/master-wallet/$walletId/sub-wallets/$subWalletId/balance',
      ),
      headers: _headers,
    );
    if (r.statusCode != 200) throw _error(r);
    return jsonDecode(r.body) as Map<String, dynamic>;
  }

  /// POST /master-wallet/:id/sub-wallets/:sid/transfer — body: {to, amount, password, token?}
  Future<Map<String, dynamic>> transferSubWallet({
    required String walletId,
    required String subWalletId,
    required String to,
    required String amount,
    required String password,
    String? token,
  }) async {
    final r = await http.post(
      Uri.parse(
        '$_apiV1/master-wallet/$walletId/sub-wallets/$subWalletId/transfer',
      ),
      headers: _headers,
      body: jsonEncode({
        'to': to,
        'amount': amount,
        'password': password,
        if (token != null) 'token': token,
      }),
    );
    if (r.statusCode != 200 && r.statusCode != 201) throw _error(r);
    return jsonDecode(r.body) as Map<String, dynamic>;
  }

  // ==================== Users ====================

  /// GET /master-wallet/:id/users → {users: [...]}
  Future<List<Map<String, dynamic>>> getUsers(String walletId) async {
    final r = await http.get(
      Uri.parse('$_apiV1/master-wallet/$walletId/users'),
      headers: _headers,
    );
    if (r.statusCode != 200) throw _error(r);
    final body = jsonDecode(r.body) as Map<String, dynamic>;
    final users = body['users'] as List? ?? const [];
    return users.cast<Map<String, dynamic>>();
  }

  /// POST /master-wallet/:id/users — body: {email, password, name?, role?}
  Future<Map<String, dynamic>> createUser(
    String walletId,
    Map<String, dynamic> user,
  ) async {
    final r = await http.post(
      Uri.parse('$_apiV1/master-wallet/$walletId/users'),
      headers: _headers,
      body: jsonEncode(user),
    );
    if (r.statusCode != 200 && r.statusCode != 201) throw _error(r);
    return jsonDecode(r.body) as Map<String, dynamic>;
  }

  /// DELETE /master-wallet/:id/users/:uid
  Future<bool> deleteUser(String walletId, String userId) async {
    final r = await http.delete(
      Uri.parse('$_apiV1/master-wallet/$walletId/users/$userId'),
      headers: _headers,
    );
    if (r.statusCode != 200 && r.statusCode != 204) throw _error(r);
    return true;
  }

  // ==================== Notifications ====================

  /// GET /master-wallet/:id/notifications → {notifications: [...]}
  Future<List<Map<String, dynamic>>> getNotifications(String walletId) async {
    final r = await http.get(
      Uri.parse('$_apiV1/master-wallet/$walletId/notifications'),
      headers: _headers,
    );
    if (r.statusCode != 200) throw _error(r);
    final body = jsonDecode(r.body) as Map<String, dynamic>;
    final notifs = body['notifications'] as List? ?? const [];
    return notifs.cast<Map<String, dynamic>>();
  }

  /// POST /master-wallet/:id/notifications — body:
  /// {notification_type, title, message, category?, priority?, channel?, user_id?, data?}
  Future<Map<String, dynamic>> createNotification(
    String walletId,
    Map<String, dynamic> notification,
  ) async {
    final r = await http.post(
      Uri.parse('$_apiV1/master-wallet/$walletId/notifications'),
      headers: _headers,
      body: jsonEncode(notification),
    );
    if (r.statusCode != 200 && r.statusCode != 201) throw _error(r);
    return jsonDecode(r.body) as Map<String, dynamic>;
  }

  // ==================== Webhooks ====================

  /// GET /master-wallet/:id/webhooks → {webhooks: [...]}
  Future<List<Map<String, dynamic>>> getWebhooks(String walletId) async {
    final r = await http.get(
      Uri.parse('$_apiV1/master-wallet/$walletId/webhooks'),
      headers: _headers,
    );
    if (r.statusCode != 200) throw _error(r);
    final body = jsonDecode(r.body) as Map<String, dynamic>;
    final webhooks = body['webhooks'] as List? ?? const [];
    return webhooks.cast<Map<String, dynamic>>();
  }

  /// POST /master-wallet/:id/webhooks — body: {name, url, events, retry_count?}
  Future<Map<String, dynamic>> createWebhook(
    String walletId,
    Map<String, dynamic> webhook,
  ) async {
    final r = await http.post(
      Uri.parse('$_apiV1/master-wallet/$walletId/webhooks'),
      headers: _headers,
      body: jsonEncode(webhook),
    );
    if (r.statusCode != 200 && r.statusCode != 201) throw _error(r);
    return jsonDecode(r.body) as Map<String, dynamic>;
  }

  /// DELETE /master-wallet/:id/webhooks/:wid
  Future<bool> deleteWebhook(String walletId, String webhookId) async {
    final r = await http.delete(
      Uri.parse('$_apiV1/master-wallet/$walletId/webhooks/$webhookId'),
      headers: _headers,
    );
    if (r.statusCode != 200 && r.statusCode != 204) throw _error(r);
    return true;
  }

  // ==================== Analytics ====================

  /// GET /master-wallet/:id/analytics/transactions → {by_status: {...}}
  Future<Map<String, dynamic>> getAnalyticsTransactions(
    String walletId,
  ) async {
    final r = await http.get(
      Uri.parse('$_apiV1/master-wallet/$walletId/analytics/transactions'),
      headers: _headers,
    );
    if (r.statusCode != 200) throw _error(r);
    return jsonDecode(r.body) as Map<String, dynamic>;
  }

  /// GET /master-wallet/:id/analytics/volume → {total_volume, transaction_count}
  Future<VolumeAnalytics> getVolumeAnalytics(String walletId) async {
    final r = await http.get(
      Uri.parse('$_apiV1/master-wallet/$walletId/analytics/volume'),
      headers: _headers,
    );
    if (r.statusCode != 200) throw _error(r);
    final data = jsonDecode(r.body) as Map<String, dynamic>;
    return VolumeAnalytics(
      totalVolume: _toDouble(data['total_volume']),
      transactionCount: (data['transaction_count'] as num?)?.toInt() ?? 0,
    );
  }

  /// GET /master-wallet/:id/analytics/wallets → {master_wallets, sub_wallets, users}
  Future<Map<String, dynamic>> getAnalyticsWallets(String walletId) async {
    final r = await http.get(
      Uri.parse('$_apiV1/master-wallet/$walletId/analytics/wallets'),
      headers: _headers,
    );
    if (r.statusCode != 200) throw _error(r);
    return jsonDecode(r.body) as Map<String, dynamic>;
  }

  // ==================== Public (no auth) ====================

  /// GET /api/v1/chains → {chains: [...]} (public, no auth).
  Future<List<Map<String, dynamic>>> getChains() async {
    final r = await http.get(
      Uri.parse('$_apiV1/chains'),
      headers: {'Content-Type': 'application/json'},
    );
    if (r.statusCode != 200) throw _error(r);
    final body = jsonDecode(r.body) as Map<String, dynamic>;
    final chains = body['chains'] as List? ?? const [];
    return chains.cast<Map<String, dynamic>>();
  }

  /// GET /health → {status, service, version, port, time} (public, no auth).
  Future<Map<String, dynamic>> health() async {
    final r = await http.get(
      Uri.parse('$API_BASE/health'),
      headers: {'Content-Type': 'application/json'},
    );
    if (r.statusCode != 200) throw _error(r);
    return jsonDecode(r.body) as Map<String, dynamic>;
  }

  /// GET /api/v1/transactions/history?address=&chain_id= →
  /// {address, chain_id, transactions: [...]} (public, no auth).
  Future<List<Map<String, dynamic>>> getTransactionHistory({
    required String address,
    int chainId = CHAIN_ETHEREUM,
  }) async {
    final uri = Uri.parse('$_apiV1/transactions/history').replace(
      queryParameters: {'address': address, 'chain_id': chainId.toString()},
    );
    final r = await http.get(
      uri,
      headers: {'Content-Type': 'application/json'},
    );
    if (r.statusCode != 200) throw _error(r);
    final body = jsonDecode(r.body) as Map<String, dynamic>;
    final txs = body['transactions'] as List? ?? const [];
    return txs.cast<Map<String, dynamic>>();
  }

  // ==================== Master wallet mutation ====================

  /// PUT /master-wallet/:id — partial update. Only the supplied fields are
  /// sent; null fields are omitted from the body so the backend leaves them
  /// untouched. Returns the backend's {id, updated} result.
  Future<Map<String, dynamic>> updateMasterWallet(
    String masterId, {
    String? name,
    bool? isActive,
    double? dailyLimit,
    double? perTransactionLimit,
    Map<String, dynamic>? metadata,
  }) async {
    final body = <String, dynamic>{};
    if (name != null) body['name'] = name;
    if (isActive != null) body['is_active'] = isActive;
    if (dailyLimit != null) body['daily_limit'] = dailyLimit;
    if (perTransactionLimit != null) {
      body['per_transaction_limit'] = perTransactionLimit;
    }
    if (metadata != null) body['metadata'] = metadata;
    final r = await http.put(
      Uri.parse('$_apiV1/master-wallet/$masterId'),
      headers: _headers,
      body: jsonEncode(body),
    );
    if (r.statusCode != 200) throw _error(r);
    return jsonDecode(r.body) as Map<String, dynamic>;
  }

  // ==================== Transactions (single) ====================

  /// GET /master-wallet/:id/transactions/:tid → {transaction: {...}}.
  Future<Map<String, dynamic>> getTransaction(
    String masterId,
    String txId,
  ) async {
    final r = await http.get(
      Uri.parse('$_apiV1/master-wallet/$masterId/transactions/$txId'),
      headers: _headers,
    );
    if (r.statusCode != 200) throw _error(r);
    final body = jsonDecode(r.body) as Map<String, dynamic>;
    // Unwrap the {transaction: {...}} envelope when present; otherwise return
    // the raw object so callers get the transaction payload either way.
    final tx = body['transaction'];
    if (tx is Map<String, dynamic>) return tx;
    return body;
  }

  // ==================== Multisig ====================

  /// GET /master-wallet/:id/multisig/wallets/:wid →
  /// {multisig_wallet: {id, name, owners, threshold, chain_id, address,
  /// pending_transactions?}}.
  Future<MultisigWalletDetail> getMultisigWalletDetail(
    String masterId,
    String walletId,
  ) async {
    final r = await http.get(
      Uri.parse('$_apiV1/master-wallet/$masterId/multisig/wallets/$walletId'),
      headers: _headers,
    );
    if (r.statusCode != 200) throw _error(r);
    final body = jsonDecode(r.body) as Map<String, dynamic>;
    final mw = body['multisig_wallet'];
    if (mw is Map<String, dynamic>) {
      return MultisigWalletDetail.fromJson(mw);
    }
    // Tolerate an unwrapped payload.
    return MultisigWalletDetail.fromJson(body);
  }

  // ==================== Passkeys ====================

  /// POST /master-wallet/:id/passkey/register — body {credential_id(base64url),
  /// public_key(base64url SPKI), sign_count(int), transports(List<String>),
  /// label} → {passkey_id, credential_id, registered:bool}.
  Future<Map<String, dynamic>> registerPasskey(
    String masterId,
    String credentialId,
    String publicKey,
    int signCount,
    List<String> transports,
    String label,
  ) async {
    final r = await http.post(
      Uri.parse('$_apiV1/master-wallet/$masterId/passkey/register'),
      headers: _headers,
      body: jsonEncode({
        'credential_id': credentialId,
        'public_key': publicKey,
        'sign_count': signCount,
        'transports': transports,
        'label': label,
      }),
    );
    if (r.statusCode != 200 && r.statusCode != 201) throw _error(r);
    return jsonDecode(r.body) as Map<String, dynamic>;
  }

  /// GET /master-wallet/:id/passkey/credentials →
  /// {passkeys: [{id, credential_id, sign_count, transports, label,
  /// created_at, updated_at}]}.
  Future<List<PasskeyCredential>> listPasskeys(String masterId) async {
    final r = await http.get(
      Uri.parse('$_apiV1/master-wallet/$masterId/passkey/credentials'),
      headers: _headers,
    );
    if (r.statusCode != 200) throw _error(r);
    final body = jsonDecode(r.body) as Map<String, dynamic>;
    final passkeys = body['passkeys'] as List? ?? const [];
    return passkeys
        .map((p) => PasskeyCredential.fromJson(p as Map<String, dynamic>))
        .toList();
  }

  /// DELETE /master-wallet/:id/passkey/credentials/:credId → 204.
  Future<bool> deletePasskey(String masterId, String credId) async {
    final r = await http.delete(
      Uri.parse('$_apiV1/master-wallet/$masterId/passkey/credentials/$credId'),
      headers: _headers,
    );
    if (r.statusCode != 200 && r.statusCode != 204) throw _error(r);
    return true;
  }

  /// POST /master-wallet/:id/passkey/verify-assertion — body
  /// {credential_id, authenticator_data, client_data_json, signature}
  /// (all base64url) → {verified:bool, credential_id}.
  Future<Map<String, dynamic>> verifyPasskeyAssertion(
    String masterId,
    String credentialId,
    String authData,
    String clientDataJson,
    String signature,
  ) async {
    final r = await http.post(
      Uri.parse('$_apiV1/master-wallet/$masterId/passkey/verify-assertion'),
      headers: _headers,
      body: jsonEncode({
        'credential_id': credentialId,
        'authenticator_data': authData,
        'client_data_json': clientDataJson,
        'signature': signature,
      }),
    );
    if (r.statusCode != 200) throw _error(r);
    return jsonDecode(r.body) as Map<String, dynamic>;
  }

  // ==================== Helpers ====================

  // ==================== Two-Party Gate ====================

  /// POST /api/v1/master-wallet/:id/withdrawal-request — two-party gate.
  /// Body {to_address, amount_wei, currency?, chain_id?} → {withdrawal_id, status}.
  Future<WithdrawalRequestResult?> requestWithdrawal(
    String masterId,
    String toAddress,
    String amountWei, {
    String? currency,
    int? chainId,
  }) async {
    try {
      final r = await http.post(
        Uri.parse('$_apiV1/master-wallet/$masterId/withdrawal-request'),
        headers: _headers,
        body: jsonEncode({
          'to_address': toAddress,
          'amount_wei': amountWei,
          if (currency != null) 'currency': currency,
          if (chainId != null) 'chain_id': chainId,
        }),
      );
      if (r.statusCode != 200 && r.statusCode != 201) return null;
      final data = jsonDecode(r.body) as Map<String, dynamic>;
      return WithdrawalRequestResult.fromJson(data);
    } catch (e) {
      return null;
    }
  }

  /// POST /api/v1/master-wallet/:id/revenue-payout — executes a two-party
  /// gate payout. Body {to, amount, password, gas_limit?, withdrawal_id}
  /// → {transaction_hash, status, withdrawal_id?, from?, chain_id?}.
  Future<RevenuePayoutResult?> revenuePayout(
    String masterId,
    String to,
    String amount,
    String password, {
    int? gasLimit,
    required String withdrawalId,
  }) async {
    try {
      final r = await http.post(
        Uri.parse('$_apiV1/master-wallet/$masterId/revenue-payout'),
        headers: _headers,
        body: jsonEncode({
          'to': to,
          'amount': amount,
          'password': password,
          'withdrawal_id': withdrawalId,
          if (gasLimit != null) 'gas_limit': gasLimit,
        }),
      );
      if (r.statusCode != 200 && r.statusCode != 201) return null;
      final data = jsonDecode(r.body) as Map<String, dynamic>;
      return RevenuePayoutResult.fromJson(data);
    } catch (e) {
      return null;
    }
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

class VolumeAnalytics {
  final double totalVolume;
  final int transactionCount;

  VolumeAnalytics({
    required this.totalVolume,
    required this.transactionCount,
  });
}

/// Multisig wallet detail returned by
/// GET /master-wallet/:id/multisig/wallets/:wid.
class MultisigWalletDetail {
  final String id;
  final String name;
  final List<String> owners;
  final int threshold;
  final int chainId;
  final String address;
  final List<Map<String, dynamic>> pendingTransactions;

  MultisigWalletDetail({
    required this.id,
    required this.name,
    required this.owners,
    required this.threshold,
    required this.chainId,
    required this.address,
    this.pendingTransactions = const [],
  });

  factory MultisigWalletDetail.fromJson(Map<String, dynamic> json) {
    final owners = (json['owners'] as List? ?? const [])
        .map((o) => o.toString())
        .toList();
    final pending = json['pending_transactions'];
    final pendingList = pending is List
        ? pending
            .map((t) => t is Map<String, dynamic> ? t : <String, dynamic>{})
            .toList()
            .cast<Map<String, dynamic>>()
        : const <Map<String, dynamic>>[];
    return MultisigWalletDetail(
      id: json['id'] as String? ?? '',
      name: json['name'] as String? ?? '',
      owners: owners,
      threshold: (json['threshold'] as num?)?.toInt() ?? 0,
      chainId: (json['chain_id'] as num?)?.toInt() ?? 0,
      address: json['address'] as String? ?? '',
      pendingTransactions: pendingList,
    );
  }
}

/// Passkey credential returned by
/// GET /master-wallet/:id/passkey/credentials.
class PasskeyCredential {
  final String id;
  final String credentialId;
  final int signCount;
  final List<String> transports;
  final String label;
  final int createdAt;
  final int updatedAt;

  PasskeyCredential({
    required this.id,
    required this.credentialId,
    required this.signCount,
    required this.transports,
    required this.label,
    required this.createdAt,
    required this.updatedAt,
  });

  factory PasskeyCredential.fromJson(Map<String, dynamic> json) {
    final transports = (json['transports'] as List? ?? const [])
        .map((t) => t.toString())
        .toList();
    return PasskeyCredential(
      id: json['id'] as String? ?? '',
      credentialId: json['credential_id'] as String? ?? '',
      signCount: (json['sign_count'] as num?)?.toInt() ?? 0,
      transports: transports,
      label: json['label'] as String? ?? '',
      createdAt: _parseTimestamp(json['created_at']),
      updatedAt: _parseTimestamp(json['updated_at']),
    );
  }

  static int _parseTimestamp(dynamic v) {
    if (v is int) return v;
    if (v == null) return 0;
    return DateTime.tryParse(v.toString())?.millisecondsSinceEpoch ?? 0;
  }
}

/// POST /api/v1/master-wallet/:id/withdrawal-request → {withdrawal_id, status}
class WithdrawalRequestResult {
  final String withdrawalId;
  final String status;

  WithdrawalRequestResult({
    required this.withdrawalId,
    required this.status,
  });

  factory WithdrawalRequestResult.fromJson(Map<String, dynamic> json) {
    return WithdrawalRequestResult(
      withdrawalId: json['withdrawal_id'] as String? ?? '',
      status: json['status'] as String? ?? '',
    );
  }
}

/// POST /api/v1/master-wallet/:id/revenue-payout →
/// {transaction_hash, status, withdrawal_id?, from?, chain_id?}
class RevenuePayoutResult {
  final String transactionHash;
  final String status;
  final String? withdrawalId;
  final String? from;
  final int? chainId;

  RevenuePayoutResult({
    required this.transactionHash,
    required this.status,
    this.withdrawalId,
    this.from,
    this.chainId,
  });

  factory RevenuePayoutResult.fromJson(Map<String, dynamic> json) {
    return RevenuePayoutResult(
      transactionHash: json['transaction_hash'] as String? ?? '',
      status: json['status'] as String? ?? '',
      withdrawalId: json['withdrawal_id'] as String?,
      from: json['from'] as String?,
      chainId: (json['chain_id'] as num?)?.toInt(),
    );
  }
}
