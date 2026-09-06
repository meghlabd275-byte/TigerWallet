/**
 * UserWalletService — full typed client for go/wallet_api (:8443).
 *
 * Every method maps 1:1 to a canonical backend route (see go/wallet_api/main.go).
 * No fabricated data: errors from the backend are propagated, never replaced
 * with fake values. All methods are fail-closed. Separation rule honored:
 * this client only ever calls the wallet_api base URL.
 */

import 'dart:convert';
import 'package:http/http.dart' as http;
import 'package:shared_preferences/shared_preferences.dart';

class ApiException implements Exception {
  final int status;
  final String message;
  ApiException(this.status, this.message);
  @override
  String toString() => 'ApiException($status): $message';
}

class UserWalletService {
  static const String baseUrlKey = 'userwallet_base_url';
  String? _baseUrl;
  String? _token;

  Future<String> baseUrl() async {
    if (_baseUrl != null) return _baseUrl!;
    final prefs = await SharedPreferences.getInstance();
    _baseUrl = prefs.getString(baseUrlKey) ?? 'http://localhost:8443';
    return _baseUrl!;
  }

  Future<void> setBaseUrl(String url) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(baseUrlKey, url);
    _baseUrl = url;
  }

  void setToken(String? token) => _token = token;

  Map<String, String> get _authHeaders => {
        'Content-Type': 'application/json',
        if (_token != null) 'Authorization': 'Bearer $_token',
      };

  Future<Map<String, dynamic>?> _decode(http.Response r) async {
    if (r.statusCode >= 400) {
      String msg = r.body;
      try {
        final j = jsonDecode(r.body);
        if (j is Map && j['error'] != null) msg = j['error'].toString();
      } catch (_) {}
      throw ApiException(r.statusCode, msg);
    }
    if (r.body.isEmpty) return null;
    try {
      final j = jsonDecode(r.body);
      return j is Map<String, dynamic> ? j : {'data': j};
    } catch (_) {
      return {'data': r.body};
    }
  }

  Future<Map<String, dynamic>?> get(String path) async {
    final base = await baseUrl();
    return _decode(await http.get(Uri.parse('$base$path'), headers: _authHeaders));
  }

  Future<Map<String, dynamic>?> post(String path, [Map<String, dynamic>? body]) async {
    final base = await baseUrl();
    return _decode(await http.post(Uri.parse('$base$path'),
        headers: _authHeaders, body: jsonEncode(body ?? {})));
  }

  Future<Map<String, dynamic>?> put(String path, [Map<String, dynamic>? body]) async {
    final base = await baseUrl();
    return _decode(await http.put(Uri.parse('$base$path'),
        headers: _authHeaders, body: jsonEncode(body ?? {})));
  }

  Future<Map<String, dynamic>?> delete(String path) async {
    final base = await baseUrl();
    return _decode(await http.delete(Uri.parse('$base$path'), headers: _authHeaders));
  }

  // ==================== AUTH ====================
  Future<Map<String, dynamic>?> register(String email, String password) =>
      post('/api/v1/auth/register', {'email': email, 'password': password});
  Future<Map<String, dynamic>?> login(String email, String password) =>
      post('/api/v1/auth/login', {'email': email, 'password': password});
  Future<Map<String, dynamic>?> guest() => post('/api/v1/auth/guest');

  // ==================== WALLETS ====================
  Future<Map<String, dynamic>?> createWallet(String password, {String? label}) =>
      post('/api/v1/wallets', {'password': password, 'label': label ?? ''});
  Future<Map<String, dynamic>?> importWallet(String mnemonic, String password, {String? label}) =>
      post('/api/v1/wallets', {'mnemonic': mnemonic, 'password': password, 'label': label ?? ''});
  Future<Map<String, dynamic>?> createWatchOnly(String address, {String? chainId, String? label}) =>
      post('/api/v1/wallets/watch-only', {'address': address, 'chain_id': chainId, 'label': label ?? ''});
  Future<Map<String, dynamic>?> listWallets() => get('/api/v1/wallets');
  Future<Map<String, dynamic>?> getWallet(String id) => get('/api/v1/wallets/$id');
  Future<Map<String, dynamic>?> unlock(String id, String password) =>
      post('/api/v1/wallets/$id/unlock', {'password': password});
  Future<Map<String, dynamic>?> lock(String id) => post('/api/v1/wallets/$id/lock');
  Future<Map<String, dynamic>?> exportEncryptedSeed(String id, String password) =>
      post('/api/v1/wallets/$id/export-encrypted-seed', {'password': password});
  Future<Map<String, dynamic>?> importEncryptedSeed(String encryptedSeed, String password,
          {String? label, int? chainId}) =>
      post('/api/v1/wallets/import-encrypted-seed', {
        'encrypted_seed': encryptedSeed,
        'password': password,
        'label': label ?? '',
        if (chainId != null) 'chain_id': chainId,
      });
  Future<Map<String, dynamic>?> exportKeystore(String id, String password, String exportPassword) =>
      post('/api/v1/keystore/export', {'wallet_id': id, 'password': password, 'export_password': exportPassword});
  Future<Map<String, dynamic>?> importKeystore(String keystoreJson, String password, String? label) =>
      post('/api/v1/keystore/import', {'keystore_json': keystoreJson, 'password': password, 'label': label});

  // ==================== BALANCE / TX ====================
  Future<Map<String, dynamic>?> getBalance(String address, int chainId) =>
      get('/api/v1/balance?address=$address&chain_id=$chainId');
  Future<Map<String, dynamic>?> getTransactions(String address, int chainId) =>
      get('/api/v1/transactions?address=$address&chain_id=$chainId');
  Future<Map<String, dynamic>?> getTransactionReceipt(String txHash, int chainId) =>
      get('/api/v1/transactions/$txHash?chain_id=$chainId');
  Future<Map<String, dynamic>?> simulate(Map<String, dynamic> tx) =>
      post('/api/v1/simulate', tx);

  // ==================== SIGNING / SEND ====================
  Future<Map<String, dynamic>?> send(String walletId, String password, String to, String value, int chainId) =>
      post('/api/v1/send', {'wallet_id': walletId, 'password': password, 'to': to, 'value': value, 'chain_id': chainId});
  Future<Map<String, dynamic>?> sign(String walletId, String message, String password) =>
      post('/api/v1/sign', {'wallet_id': walletId, 'message': message, 'password': password});
  Future<Map<String, dynamic>?> autoSend(String walletId, String password, String to, String value, int chainId) =>
      post('/api/v1/auto-send', {'wallet_id': walletId, 'password': password, 'to': to, 'value': value, 'chain_id': chainId});

  // ==================== NON-EVM ====================
  Future<Map<String, dynamic>?> nonEvmAddress(String walletId, String chainType, int chainId) =>
      post('/api/v1/non_evm/address', {'wallet_id': walletId, 'chain_type': chainType, 'chain_id': chainId});
  Future<Map<String, dynamic>?> nonEvmSign(String walletId, String chainType, int chainId, String message) =>
      post('/api/v1/non_evm/sign', {'wallet_id': walletId, 'chain_type': chainType, 'chain_id': chainId, 'message': message});
  Future<Map<String, dynamic>?> nonEvmSend(String walletId, String chainType, int chainId, String to, String value) =>
      post('/api/v1/non_evm/send', {'wallet_id': walletId, 'chain_type': chainType, 'chain_id': chainId, 'to': to, 'value': value});

  // ==================== CHAINS / TOKENS ====================
  Future<Map<String, dynamic>?> getChains() => get('/api/v1/chains');
  Future<Map<String, dynamic>?> getChain(int id) => get('/api/v1/chains/$id');
  Future<Map<String, dynamic>?> getNetworkStatus() => get('/api/v1/network-status');
  Future<Map<String, dynamic>?> getTokenRegistry({int? chainId}) =>
      get(chainId != null ? '/api/v1/tokens/registry?chain_id=$chainId' : '/api/v1/tokens/registry');
  Future<Map<String, dynamic>?> getToken(int chainId, String symbol) =>
      get('/api/v1/tokens/$chainId/$symbol');

  // ==================== PRICES / GAS / CHART ====================
  Future<Map<String, dynamic>?> getPrice({String? symbols}) =>
      get(symbols != null ? '/api/v1/price?symbols=$symbols' : '/api/v1/price');
  Future<Map<String, dynamic>?> getGas(int chainId) => get('/api/v1/gas?chain_id=$chainId');
  Future<Map<String, dynamic>?> estimateGas(int chainId, Map<String, dynamic> tx) =>
      post('/api/v1/gas/estimate?chain_id=$chainId', tx);
  Future<Map<String, dynamic>?> getChartHistory(String symbol, {String range = '7d'}) =>
      get('/api/v1/chart/history?symbol=$symbol&range=$range');

  // ==================== TERMINAL ====================
  Future<Map<String, dynamic>?> getTerminalKline(String symbol, {int days = 1}) =>
      get('/api/v1/terminal/kline/${Uri.encodeComponent(symbol)}?days=$days');
  Future<Map<String, dynamic>?> getTerminalTicker(String symbol) =>
      get('/api/v1/terminal/ticker/${Uri.encodeComponent(symbol)}');

  // ==================== SWAP / AMM ====================
  Future<Map<String, dynamic>?> getSwapQuote(String from, String to, String amount) =>
      get('/api/v1/swap/quote?from=$from&to=$to&amount=$amount');
  Future<Map<String, dynamic>?> executeSwap(Map<String, dynamic> req) => post('/api/v1/swap/execute', req);
  Future<Map<String, dynamic>?> getAmmQuote(int chainId, String tokenIn, String tokenOut, String amountIn) =>
      get('/api/v1/amm/quote?chain_id=$chainId&token_in=$tokenIn&token_out=$tokenOut&amount_in=$amountIn');
  Future<Map<String, dynamic>?> ammSwap(Map<String, dynamic> req) => post('/api/v1/amm/swap', req);

  // ==================== STAKING ====================
  Future<Map<String, dynamic>?> getStakingQuote({int? chainId, String? token}) => get(
      '/api/v1/staking/quote${chainId != null ? '?chain_id=$chainId' : ''}'
      '${token != null ? '${chainId != null ? "&" : "?"}token=$token' : ''}');
  Future<Map<String, dynamic>?> stakingAction(String action, Map<String, dynamic> req) =>
      post('/api/v1/staking/$action', req);

  // ==================== DEFI / LENDING ====================
  Future<Map<String, dynamic>?> getDefiProtocols() => get('/api/v1/defi/protocols');
  Future<Map<String, dynamic>?> getLendingMarkets() => get('/api/v1/lending/markets');
  Future<Map<String, dynamic>?> lendingAction(String action, Map<String, dynamic> req) =>
      post('/api/v1/lending/$action', req);

  // ==================== BRIDGE ====================
  Future<Map<String, dynamic>?> getBridgeRoutes() => get('/api/v1/bridge/routes');
  Future<Map<String, dynamic>?> getBridgeQuote(String fromChain, String toChain, String token, String amount) =>
      get('/api/v1/bridge/quote?from_chain=$fromChain&to_chain=$toChain&token=$token&amount=$amount');
  Future<Map<String, dynamic>?> bridgeTransfer(Map<String, dynamic> req) => post('/api/v1/bridge/transfer', req);
  Future<Map<String, dynamic>?> getBridgeHistory(String address) =>
      get('/api/v1/bridge/history?address=$address');

  // ==================== TRADING (PERP / MARGIN) ====================
  Future<Map<String, dynamic>?> getPerpetualPositions() => get('/api/v1/perpetual/positions');
  Future<Map<String, dynamic>?> openPerpetualPosition(Map<String, dynamic> req) =>
      post('/api/v1/perpetual/positions', req);
  Future<Map<String, dynamic>?> closePerpetualPosition(String id) =>
      post('/api/v1/perpetual/positions/$id/close');
  Future<Map<String, dynamic>?> getMarginPositions() => get('/api/v1/margin/positions');
  Future<Map<String, dynamic>?> openMarginPosition(Map<String, dynamic> req) =>
      post('/api/v1/margin/positions', req);
  Future<Map<String, dynamic>?> closeMarginPosition(String id) =>
      post('/api/v1/margin/positions/$id/close');

  // ==================== OPTIONS ENGINE (wallet_api /options/*) ====================
  /// GET /options/series — active, unexpired series (filterable by underlying).
  Future<Map<String, dynamic>?> getOptionsSeries({String? underlying}) =>
      get(underlying != null ? '/api/v1/options/series?underlying=${Uri.encodeComponent(underlying)}' : '/api/v1/options/series');
  /// GET /options/quote — live Black-Scholes premium for a series.

  Future<Map<String, dynamic>?> getOptionsQuote(String seriesId) =>
      get('/api/v1/options/quote?series_id=${Uri.encodeComponent(seriesId)}');
  /// GET /options/positions — the caller's open/closed option positions.


  Future<Map<String, dynamic>?> getOptionsPositions() => get('/api/v1/options/positions');
  /// POST /options/positions — open a buy/sell position on a series.



  Future<Map<String, dynamic>?> openOptionsPosition(Map<String, dynamic> req) =>
      post('/api/v1/options/positions', req);
  /// POST /options/positions/:id/close — settle and close an open position.10


  Future<Map<String, dynamic>?> closeOptionsPosition(String id) =>
      post('/api/v1/options/positions/$id/close');

  // ==================== EARN (LAUNCHPOOL / TOKEN SALES) ====================
  Future<Map<String, dynamic>?> getLaunchpool() => get('/api/v1/launchpool');
  Future<Map<String, dynamic>?> getLaunchpoolStakes() => get('/api/v1/launchpool/stakes');
  Future<Map<String, dynamic>?> launchpoolStake(Map<String, dynamic> req) => post('/api/v1/launchpool/stake', req);
  Future<Map<String, dynamic>?> launchpoolUnstake(Map<String, dynamic> req) => post('/api/v1/launchpool/unstake', req);
  Future<Map<String, dynamic>?> getTokenSales() => get('/api/v1/token-sales');
  Future<Map<String, dynamic>?> participateTokenSale(String id, Map<String, dynamic> req) =>
      post('/api/v1/token-sales/$id/participate', req);

  // ==================== SOCIAL / DAO / COPY / P2P / PREDICTION ====================
  Future<Map<String, dynamic>?> getCopyTraders() => get('/api/v1/copytrading/traders');
  Future<Map<String, dynamic>?> followTrader(Map<String, dynamic> req) => post('/api/v1/copytrading/follow', req);
  Future<Map<String, dynamic>?> getCopiers() => get('/api/v1/copytrading/copiers');
  Future<Map<String, dynamic>?> stopCopier(String id) => post('/api/v1/copytrading/copiers/$id/stop');
  Future<Map<String, dynamic>?> getP2PAdverts() => get('/api/v1/p2p/adverts');
  Future<Map<String, dynamic>?> createP2PAdvert(Map<String, dynamic> req) => post('/api/v1/p2p/adverts', req);
  Future<Map<String, dynamic>?> getP2POrders() => get('/api/v1/p2p/orders');
  Future<Map<String, dynamic>?> createP2POrder(Map<String, dynamic> req) => post('/api/v1/p2p/orders', req);
  Future<Map<String, dynamic>?> getDaoProposals() => get('/api/v1/dao/proposals');
  Future<Map<String, dynamic>?> daoVote(String proposalId, Map<String, dynamic> req) =>
      post('/api/v1/dao/proposals/$proposalId/vote', req);
  Future<Map<String, dynamic>?> getDaoDelegates() => get('/api/v1/dao/delegates');
  Future<Map<String, dynamic>?> getPredictionMarkets() => get('/api/v1/prediction/markets');
  Future<Map<String, dynamic>?> predictionBet(String marketId, Map<String, dynamic> req) =>
      post('/api/v1/prediction/$marketId/bet', req);

  // ==================== NFT ====================
  Future<Map<String, dynamic>?> getNfts(String address, int chainId) =>
      get('/api/v1/nfts?address=$address&chain_id=$chainId');
  Future<Map<String, dynamic>?> nftTransfer(Map<String, dynamic> req) => post('/api/v1/nft/transfer', req);

  // ==================== KYC ====================
  Future<Map<String, dynamic>?> kycRegister(Map<String, dynamic> req) => post('/api/v1/kyc/register', req);
  Future<Map<String, dynamic>?> kycSubmit(Map<String, dynamic> req) => post('/api/v1/kyc/submit', req);
  Future<Map<String, dynamic>?> kycStatus() => get('/api/v1/kyc/status');
  Future<Map<String, dynamic>?> kycDocument(Map<String, dynamic> req) => post('/api/v1/kyc/document', req);
  Future<Map<String, dynamic>?> kycSession(String id) => get('/api/v1/kyc/session/$id');

  // ==================== CARDS / RAMP ====================
  Future<Map<String, dynamic>?> getCardBalance(String cardId) => get('/api/v1/cards/$cardId/balance');
  Future<Map<String, dynamic>?> getCardTransactions(String cardId) => get('/api/v1/cards/$cardId/transactions');
  Future<Map<String, dynamic>?> getRampProviders() => get('/api/v1/ramp/providers');
  Future<Map<String, dynamic>?> getRampQuote(String asset, String fiat, String amount) =>
      get('/api/v1/ramp/quote?asset=$asset&fiat=$fiat&amount=$amount');

  // ==================== SECURITY / ENS ====================
  Future<Map<String, dynamic>?> securityCheckUrl(String url) =>
      get('/api/v1/security/check-url?url=${Uri.encodeComponent(url)}');
  Future<Map<String, dynamic>?> securityCheckAddress(String address) =>
      get('/api/v1/security/check-address?address=$address');
  Future<Map<String, dynamic>?> securityScan(String address) =>
      post('/api/v1/security/scan', {'address': address});
  Future<Map<String, dynamic>?> ensResolve(String name) =>
      get('/api/v1/ens/resolve?name=${Uri.encodeComponent(name)}');
  Future<Map<String, dynamic>?> ensLookup(String address) =>
      get('/api/v1/ens/lookup?address=$address');

  // ==================== DAPPS / WALLETCONNECT ====================
  Future<Map<String, dynamic>?> getDapps({String? category}) =>
      get(category != null ? '/api/v1/dapps?category=$category' : '/api/v1/dapps');
  Future<Map<String, dynamic>?> getDapp(String id) => get('/api/v1/dapps/$id');
  Future<Map<String, dynamic>?> getDappCategories() => get('/api/v1/dapps/categories');
  Future<Map<String, dynamic>?> wcPairing(Map<String, dynamic> req) => post('/api/v1/walletconnect/pairing', req);
  Future<Map<String, dynamic>?> wcSessions() => get('/api/v1/walletconnect/sessions');
  Future<Map<String, dynamic>?> wcApprove(String sessionId) => post('/api/v1/walletconnect/$sessionId/approve');
  Future<Map<String, dynamic>?> wcReject(String sessionId) => post('/api/v1/walletconnect/$sessionId/reject');

  // ==================== DEVICES / ADDRESS BOOK / ALERTS / PASSKEY ====================
  Future<Map<String, dynamic>?> getDevices() => get('/api/v1/devices');
  Future<Map<String, dynamic>?> addDevice(Map<String, dynamic> req) => post('/api/v1/devices', req);
  Future<Map<String, dynamic>?> removeDevice(String id) => delete('/api/v1/devices/$id');
  Future<Map<String, dynamic>?> getAddressBook() => get('/api/v1/address-book/contacts');
  Future<Map<String, dynamic>?> addContact(Map<String, dynamic> req) => post('/api/v1/address-book/contacts', req);
  Future<Map<String, dynamic>?> updateContact(String id, Map<String, dynamic> req) =>
      put('/api/v1/address-book/contacts/$id', req);
  Future<Map<String, dynamic>?> removeContact(String id) => delete('/api/v1/address-book/contacts/$id');
  Future<Map<String, dynamic>?> getPriceAlerts() => get('/api/v1/price-alerts');
  Future<Map<String, dynamic>?> createPriceAlert(Map<String, dynamic> req) => post('/api/v1/price-alerts', req);
  Future<Map<String, dynamic>?> updatePriceAlert(String id, Map<String, dynamic> req) =>
      put('/api/v1/price-alerts/$id', req);
  Future<Map<String, dynamic>?> deletePriceAlert(String id) => delete('/api/v1/price-alerts/$id');
  Future<Map<String, dynamic>?> passkeyWallet(Map<String, dynamic> req) => post('/api/v1/passkey/wallet', req);

  // ==================== FEES (public + user) ====================
  Future<Map<String, dynamic>?> getFees() => get('/api/v1/fees');
  Future<Map<String, dynamic>?> getPublicFees() => get('/api/v1/public/fees');
  Future<Map<String, dynamic>?> getPublicFeeTransactions() => get('/api/v1/public/fees/transactions');

  // ============ Wallet & finance plane ============

  /// GET /finance/accounts — multi-chain ledger accounts.
  Future<Map<String, dynamic>?> getFinanceAccounts() => get('/api/v1/finance/accounts');

  /// GET /finance/history — full double-entry ledger history.
  Future<Map<String, dynamic>?> getFinanceHistory({String? currency}) =>
      get('/api/v1/finance/history' + (currency != null ? '?currency=$currency' : ''));

  /// GET /finance/switches — per-token feature switches.
  Future<Map<String, dynamic>?> getFinanceSwitches() => get('/api/v1/finance/switches');

  /// GET /finance/deposit-addresses — deterministic per-user deposit addresses.
  Future<Map<String, dynamic>?> getDepositAddresses() => get('/api/v1/finance/deposit-addresses');
  Future<Map<String, dynamic>?> getDepositAddress(String asset) =>
      get('/api/v1/finance/deposit-addresses/${Uri.encodeComponent(asset)}');
  Future<Map<String, dynamic>?> getDepositAddressQr(String asset) =>
      get('/api/v1/finance/deposit-addresses/${Uri.encodeComponent(asset)}/qr');

  /// POST /finance/withdrawals — risk-scored, HMAC-signed withdrawal request.
  Future<Map<String, dynamic>?> createWithdrawal(
          String currency, String amount, String toAddress) =>
      post('/api/v1/finance/withdrawals',
          {'currency': currency, 'amount': amount, 'to_address': toAddress});

  /// GET /finance/withdrawals — the caller's withdrawal requests.
  Future<Map<String, dynamic>?> getWithdrawals() => get('/api/v1/finance/withdrawals');

  /// GET /finance/convert/rates — admin-managed rate book.
  Future<Map<String, dynamic>?> getConvertRates() => get('/api/v1/finance/convert/rates');

  /// POST /finance/convert — instant conversion at the admin rate.
  Future<Map<String, dynamic>?> financeConvert(String from, String to, String amount) =>
      post('/api/v1/finance/convert',
          {'from_currency': from, 'to_currency': to, 'amount': amount});
  /// GET /finance/convert/history — the caller's settled conversions.



  Future<Map<String, dynamic>?> getConvertHistory() => get('/api/v1/finance/convert/history');

  /// POST /finance/transfer — atomic KYC-gated internal transfer.
  Future<Map<String, dynamic>?> financeTransfer(String toEmail, String currency, String amount) =>
      post('/api/v1/finance/transfer',
          {'to_email': toEmail, 'currency': currency, 'amount': amount});

  /// GET /finance/payment-methods — 881-method / 238-country catalog.
  Future<Map<String, dynamic>?> getPaymentMethods({String? country, String? kind}) {
    final parts = <String>[];
    if (country != null) parts.add('country=$country');
    if (kind != null) parts.add('kind=$kind');
    return get('/api/v1/finance/payment-methods' + (parts.isEmpty ? '' : '?' + parts.join('&')));
  }

  /// GET /finance/p2p/escrow — escrow marketplace (or the caller's orders).
  Future<Map<String, dynamic>?> getEscrowOrders({bool mine = false}) =>
      get('/api/v1/finance/p2p/escrow' + (mine ? '?mine=true' : ''));

  /// POST /finance/p2p/escrow — open a sell order (funds locked, KYC-gated).
  Future<Map<String, dynamic>?> openEscrow(
          String currency, String amount, String fiatCurrency, String fiatAmount,
          String paymentMethodCode, String countryCode) =>
      post('/api/v1/finance/p2p/escrow', {
        'currency': currency,
        'amount': amount,
        'fiat_currency': fiatCurrency,
        'fiat_amount': fiatAmount,
        'payment_method_code': paymentMethodCode,
        'country_code': countryCode,
      });

  /// POST /finance/p2p/escrow/:id/:action — accept/paid/release/dispute/cancel.
  Future<Map<String, dynamic>?> escrowAction(String id, String action, {String? reason}) =>
      post('/api/v1/finance/p2p/escrow/$id/$action', reason != null ? {'reason': reason} : {});


  // ==================== APPROVALS / MULTISIG ====================
  Future<Map<String, dynamic>?> getApprovals(String address, int chainId) =>
      get('/api/v1/approvals?address=$address&chain_id=$chainId');
  /// DELETE /approvals/:id — revoke a token approval.10
  Future<Map<String, dynamic>?> revokeApproval(String id) =>
      delete('/api/v1/approvals/$id');
  Future<Map<String, dynamic>?> multisigWallets() => get('/api/v1/multisig/wallets');
  Future<Map<String, dynamic>?> createMultisig(Map<String, dynamic> req) => post('/api/v1/multisig/wallets', req);

  // ==================== HEALTH ====================
  Future<Map<String, dynamic>?> health() => get('/api/v1/health');
  Future<Map<String, dynamic>?> healthReady() => get('/api/v1/health/ready');
}
