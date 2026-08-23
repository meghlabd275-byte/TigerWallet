// Wallet Service - Core Wallet Operations
//
// Delegates ALL key management, signing and broadcast to the canonical
// TigerWallet Go wallet-api backend (go/wallet_api, port 8443), which
// performs REAL BIP-39 mnemonic generation, BIP-32/44 HD key derivation
// (secp256k1), AES-256-GCM seed encryption backed by PostgreSQL, real
// EVM transaction signing + eth_sendRawTransaction broadcast, and real
// on-chain balance/token/tx/NFT fetching.
//
// No local fake crypto: no XOR "encryption", no sha256-as-KDF, no
// fabricated addresses/tx hashes, no all-zero private keys, no
// hand-rolled mnemonic wordlist. The backend is the single signer.

import 'dart:convert';
import 'package:http/http.dart' as http;
import '../models/wallet_model.dart';
import '../models/token_model.dart';

class WalletService {
  bool _isInitialized = false;
  Wallet? _currentWallet;
  String? _mnemonic; // session-only (memory); never persisted in plaintext
  String? _authToken; // JWT from backend
  String? _userId;

  // Configurable backend URL. Defaults to the local wallet-api (port 8443)
  // matching the web app and browser extension. Override for production
  // via --dart-define=WALLET_API_URL=https://api.tigerwallet.io.
  static const String backendUrl =
      String.fromEnvironment('WALLET_API_URL', defaultValue: 'http://localhost:8443');

  bool get isInitialized => _isInitialized;
  Wallet? get currentWallet => _currentWallet;

  // Map the app's chain identifiers to the wallet-api numeric chain IDs.
  static const Map<String, int> _chainIdByName = {
    'ethereum': 1,
    'sepolia': 11155111,
    'bsc': 56,
    'polygon': 137,
    'arbitrum': 42161,
    'optimism': 10,
    'base': 8453,
    'avalanche': 43114,
    'fantom': 250,
    'cronos': 25,
    'celo': 42220,
    'gnosis': 100,
    'moonbeam': 1284,
    'kava': 2222,
    'linea': 59144,
    'mantle': 5000,
    'opbnb': 204,
  };

  static int _resolveChainId(String chainId) {
    final n = int.tryParse(chainId);
    if (n != null) return n;
    return _chainIdByName[chainId] ?? 1;
  }

  Future<void> initialize() async {
    _isInitialized = true;
  }

  // Check if a wallet already exists on the backend for this user.
  Future<bool> hasExistingWallet() async {
    if (_authToken == null) return false;
    final resp = await _get('/api/v1/wallets');
    if (!resp.ok) return false;
    final wallets = resp.body['wallets'] as List?;
    return wallets != null && wallets.isNotEmpty;
  }

  // Register a new backend account. The backend hashes the password (bcrypt)
  // and issues a JWT. Required before creating a wallet.
  Future<void> register({
    required String email,
    required String username,
    required String password,
  }) async {
    final resp = await _post('/api/v1/auth/register', {
      'email': email,
      'username': username,
      'password': password,
    });
    if (!resp.ok) throw Exception(resp.error ?? 'registration failed');
    _authToken = resp.body['token'] as String?;
    _userId = resp.body['user_id']?.toString();
  }

  // Log in to an existing backend account.
  Future<void> login({required String email, required String password}) async {
    final resp = await _post('/api/v1/auth/login', {
      'email': email,
      'password': password,
    });
    if (!resp.ok) throw Exception(resp.error ?? 'login failed');
    _authToken = resp.body['token'] as String?;
    _userId = resp.body['user_id']?.toString();
  }

  // Generate mnemonic (24 words) via the backend by creating a wallet on
  // chain 1 (Ethereum) with 256-bit entropy. The backend returns the
  // mnemonic ONCE; we keep it in memory for the session only.
  Future<String> generateMnemonic() async {
    if (_authToken == null) {
      throw Exception('Not authenticated: register or login first');
    }
    final resp = await _post('/api/v1/wallets', {
      'password': 'temp-session-pw',
      'chain_id': 1,
      'entropy_bits': 256,
      'label': 'Main Wallet',
    });
    if (!resp.ok) throw Exception(resp.error ?? 'mnemonic generation failed');
    final mnemonic = resp.body['mnemonic'] as String?;
    if (mnemonic == null || mnemonic.isEmpty) {
      throw Exception('backend did not return a mnemonic');
    }
    return mnemonic;
  }

  // Create wallet: ask the backend to generate a real mnemonic, derive the
  // EVM address, AES-256-GCM-encrypt the seed and persist it. The password
  // protects the encrypted seed on the backend.
  Future<Wallet> createWallet(String password) async {
    if (_authToken == null) {
      throw Exception('Not authenticated: register or login first');
    }
    final resp = await _post('/api/v1/wallets', {
      'password': password,
      'chain_id': 1,
      'entropy_bits': 256,
      'label': 'Main Wallet',
    });
    if (!resp.ok) throw Exception(resp.error ?? 'wallet creation failed');
    _mnemonic = resp.body['mnemonic'] as String?;
    final address = (resp.body['address'] as String?) ?? '';
    _currentWallet = Wallet(
      id: (resp.body['id'] as String?) ?? '',
      name: (resp.body['label'] as String?) ?? 'Main Wallet',
      encryptedMnemonic: null, // seed encrypted on backend, not stored client-side
      addresses: {'ethereum': address},
      publicKeys: const {},
      createdAt: DateTime.now(),
      isBackedUp: false,
      type: WalletType.hd,
    );
    return _currentWallet!;
  }

  // Import wallet from mnemonic: the backend validates the BIP-39 checksum,
  // re-derives the address and persists the encrypted seed.
  Future<Wallet> importWallet(String mnemonic, String password) async {
    if (_authToken == null) {
      throw Exception('Not authenticated: register or login first');
    }
    final resp = await _post('/api/v1/wallets', {
      'mnemonic': mnemonic.trim(),
      'password': password,
      'chain_id': 1,
      'label': 'Imported Wallet',
    });
    if (!resp.ok) throw Exception(resp.error ?? 'mnemonic import failed');
    _mnemonic = mnemonic.trim();
    final address = (resp.body['address'] as String?) ?? '';
    _currentWallet = Wallet(
      id: (resp.body['id'] as String?) ?? '',
      name: 'Imported Wallet',
      encryptedMnemonic: null,
      addresses: {'ethereum': address},
      publicKeys: const {},
      createdAt: DateTime.now(),
      isBackedUp: true,
      type: WalletType.hd,
    );
    return _currentWallet!;
  }

  // Import wallet from a raw private key. The backend's wallet store is
  // mnemonic/seed-based, so a bare private key cannot be safely persisted
  // there. We refuse rather than fabricate an address or store the key
  // insecurely. Users should import via their 12/24-word mnemonic instead.
  Future<Wallet> importWalletFromPrivateKey(String privateKey, String password) async {
    if (!isValidPrivateKey(privateKey)) {
      throw Exception('Invalid private key format');
    }
    throw Exception(
      'Private-key import is not supported. Import via your 12/24-word '
      'mnemonic so the backend can derive and encrypt the seed.',
    );
  }

  // Unlock wallet: re-load the persisted wallet list from the backend. The
  // password is verified by the backend when signing/sending; here we just
  // load the address metadata. Returns the first wallet or null.
  Future<Wallet?> unlockWallet(String password) async {
    if (_authToken == null) return null;
    final resp = await _get('/api/v1/wallets');
    if (!resp.ok) return null;
    final wallets = resp.body['wallets'] as List?;
    if (wallets == null || wallets.isEmpty) return null;
    final w = wallets.first as Map<String, dynamic>;
    _currentWallet = Wallet(
      id: (w['id'] as String?) ?? '',
      name: (w['label'] as String?) ?? 'Wallet',
      encryptedMnemonic: null,
      addresses: {'ethereum': (w['address'] as String?) ?? ''},
      publicKeys: const {},
      createdAt: DateTime.now(),
      isBackedUp: true,
      type: WalletType.hd,
    );
    return _currentWallet;
  }

  // Lock wallet
  void lockWallet() {
    _currentWallet = null;
    _mnemonic = null;
  }

  // Get mnemonic (only available in-memory immediately after create/import).
  Future<String> getMnemonic(String password) async {
    if (_mnemonic == null || _mnemonic!.isEmpty) {
      throw Exception(
        'Mnemonic is only available immediately after create/import. '
        'Re-import the mnemonic to view it again.',
      );
    }
    return _mnemonic!;
  }

  // Send transaction via the backend: real EVM signing + broadcast.
  // `amount` is in ether (native) for native transfers.
  Future<String> sendTransaction({
    required String toAddress,
    required String amount,
    required String tokenAddress,
    required String chainId,
    String? password,
  }) async {
    if (_currentWallet == null) throw Exception('Wallet not unlocked');
    if (password == null || password.isEmpty) {
      throw Exception('Password required to sign and broadcast');
    }
    if (tokenAddress.isNotEmpty && tokenAddress != '0x') {
      // ERC-20 transfer needs encoded transfer(to, amount) calldata, which the
      // backend accepts via the `data` field once a token-send helper exists.
      // Refuse rather than fake a transfer.
      throw Exception('ERC-20 transfer not yet supported; use native send');
    }
    final resp = await _post('/api/v1/send', {
      'wallet_id': _currentWallet!.id,
      'password': password,
      'to': toAddress,
      'value': amount,
      'chain_id': _resolveChainId(chainId),
    });
    if (!resp.ok) throw Exception(resp.error ?? 'send failed');
    final txHash = resp.body['tx_hash'] as String?;
    if (txHash == null || txHash.isEmpty) throw Exception('no tx hash returned');
    return txHash;
  }

  // Sign an arbitrary message (EIP-191 personal_sign) via the backend.
  Future<String> signMessage(String message, String password) async {
    if (_currentWallet == null) throw Exception('Wallet not unlocked');
    final resp = await _post('/api/v1/sign', {
      'wallet_id': _currentWallet!.id,
      'password': password,
      'message': message,
    });
    if (!resp.ok) throw Exception(resp.error ?? 'signing failed');
    return (resp.body['signature'] as String?) ?? '';
  }

  // Get token balances from the backend's real on-chain fetchers.
  Future<List<TokenModel>> getTokenBalances() async {
    if (_currentWallet == null) return [];
    final address = _currentWallet!.getAddressForChain('ethereum');
    if (address.isEmpty) return [];
    final resp = await _get('/api/v1/tokens', {'address': address, 'chain_id': '1'});
    if (!resp.ok) return [];
    final tokens = resp.body['tokens'] as List? ?? [];
    return tokens.map<TokenModel>((t) {
      final m = t as Map<String, dynamic>;
      return TokenModel(
        id: (m['contract'] as String?) ?? (m['symbol'] as String?) ?? '',
        name: (m['name'] as String?) ?? '',
        symbol: (m['symbol'] as String?) ?? '',
        address: (m['contract'] as String?) ?? '',
        iconUrl: '',
        balance: (m['balance_f'] as num?)?.toDouble() ?? 0,
        decimals: (m['decimals'] as num?)?.toInt() ?? 18,
        balanceUSD: (m['usd_value'] as num?)?.toDouble() ?? 0,
        chainId: 'ethereum',
        chainName: 'Ethereum',
        price: (m['usd_price'] as num?)?.toDouble() ?? 0,
        priceChange24h: 0,
        volume24h: 0,
        marketCap: 0,
        type: TokenType.erc20,
      );
    }).toList();
  }

  // Get transaction history from the backend's real indexer (Etherscan).
  Future<List<TransactionModel>> getTransactionHistory(String chainId) async {
    if (_currentWallet == null) return [];
    final address = _currentWallet!.getAddressForChain('ethereum');
    if (address.isEmpty) return [];
    final resp = await _get('/api/v1/transactions', {
      'address': address,
      'chain_id': _resolveChainId(chainId).toString(),
    });
    if (!resp.ok) return [];
    final txs = resp.body['transactions'] as List? ?? [];
    return txs.map<TransactionModel>((t) {
      final m = t as Map<String, dynamic>;
      return TransactionModel(
        id: (m['hash'] as String?) ?? '',
        hash: (m['hash'] as String?) ?? '',
        fromAddress: (m['from'] as String?) ?? '',
        toAddress: (m['to'] as String?) ?? '',
        amount: (m['value'] as String?) ?? '0',
        tokenSymbol: (m['token_symbol'] as String?) ?? 'ETH',
        decimals: 18,
        chainId: chainId,
        status: _mapTxStatus((m['status'] as String?) ?? 'pending'),
        type: TransactionType.transfer,
        timestamp: DateTime.fromMillisecondsSinceEpoch(
          ((m['timestamp'] as num?)?.toInt() ?? 0) * 1000,
          isUtc: true,
        ),
        fee: 0,
        confirmations: null,
        errorMessage: null,
      );
    }).toList();
  }

  TransactionStatus _mapTxStatus(String s) {
    switch (s) {
      case 'confirmed':
      case 'success':
        return TransactionStatus.confirmed;
      case 'failed':
        return TransactionStatus.failed;
      case 'cancelled':
        return TransactionStatus.cancelled;
      default:
        return TransactionStatus.pending;
    }
  }

  bool isValidMnemonic(String mnemonic) {
    final words = mnemonic.trim().split(RegExp(r'\s+'));
    if (words.length != 12 && words.length != 24) return false;
    // Full wordlist + checksum validation is performed by the backend on
    // import; here we only sanity-check word count and lowercase charset.
    for (final word in words) {
      if (!RegExp(r'^[a-z]+$').hasMatch(word.toLowerCase())) return false;
    }
    return true;
  }

  bool isValidPrivateKey(String privateKey) {
    var k = privateKey;
    if (k.startsWith('0x')) k = k.substring(2);
    return RegExp(r'^[0-9a-fA-F]{64}$').hasMatch(k);
  }

  // ---- HTTP helpers ----

  Future<_Resp> _get(String path, [Map<String, String>? query]) async {
    final uri = Uri.parse('$backendUrl$path').replace(queryParameters: query);
    final headers = <String, String>{'Accept': 'application/json'};
    if (_authToken != null) headers['Authorization'] = 'Bearer $_authToken';
    try {
      final r = await http.get(uri, headers: headers).timeout(const Duration(seconds: 30));
      return _Resp.from(r);
    } catch (e) {
      return _Resp.err(e.toString());
    }
  }

  Future<_Resp> _post(String path, Map<String, dynamic> body) async {
    final uri = Uri.parse('$backendUrl$path');
    final headers = <String, String>{
      'Content-Type': 'application/json',
      'Accept': 'application/json',
    };
    if (_authToken != null) headers['Authorization'] = 'Bearer $_authToken';
    try {
      final r = await http
          .post(uri, headers: headers, body: jsonEncode(body))
          .timeout(const Duration(seconds: 30));
      return _Resp.from(r);
    } catch (e) {
      return _Resp.err(e.toString());
    }
  }
}

class _Resp {
  final bool ok;
  final Map<String, dynamic> body;
  final String? error;
  _Resp(this.ok, this.body, [this.error]);
  factory _Resp.from(http.Response r) {
    try {
      final data = jsonDecode(r.body);
      if (r.statusCode >= 200 && r.statusCode < 300) {
        return _Resp(true, (data as Map).cast<String, dynamic>());
      }
      final msg = (data is Map ? (data['error'] ?? data['message']) : null)?.toString() ??
          'request failed';
      return _Resp(false, const {}, msg);
    } catch (_) {
      if (r.statusCode >= 200 && r.statusCode < 300) return _Resp(true, const {});
      return _Resp(false, const {}, r.reasonPhrase ?? 'request failed');
    }
  }
  factory _Resp.err(String e) => _Resp(false, const {}, e);
}
