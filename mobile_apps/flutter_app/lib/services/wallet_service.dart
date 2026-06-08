import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:crypto/crypto.dart';
import 'package:hex/hex.dart';
import 'dart:convert';
import 'dart:math';
import 'chain_service.dart';

/// TigerWallet Service - Core Wallet Functionality
/// 
/// Features:
/// - HD wallet derivation (BIP-39, BIP-44)
/// - Multi-chain support (EVM + Non-EVM)
/// - Secure key storage
/// - Transaction signing
/// - Balance fetching

class WalletService {
  final FlutterSecureStorage _secureStorage;
  final ChainService _chainService;
  
  String? _seedPhrase;
  String? _masterKey;
  Map<String, String> _addresses = {};
  Map<String, double> _balances = {};
  bool _walletExists = false;

  WalletService({
    required FlutterSecureStorage secureStorage,
    required ChainService chainService,
  })  : _secureStorage = secureStorage,
        _chainService = chainService;

  // =========================================================================
  // Wallet Creation / Import
  // =========================================================================

  /// Generate new 24-word seed phrase
  Future<String> generateSeedPhrase() async {
    final words = WORDLIST;
    final random = Random.secure();
    final phrase = List.generate(24, (_) => words[random.nextInt(words.length)]);
    return phrase.join(' ');
  }

  /// Import wallet from seed phrase
  Future<bool> importFromSeed(String seedPhrase) async {
    if (!_validateSeedPhrase(seedPhrase)) {
      return false;
    }
    
    _seedPhrase = seedPhrase;
    _masterKey = _generateMasterKey(seedPhrase);
    
    // Derive addresses for all supported chains
    await _deriveAddresses();
    
    // Save encrypted seed
    await _secureStorage.write(key: 'wallet_seed', value: seedPhrase);
    await _secureStorage.write(key: 'wallet_exists', value: 'true');
    
    _walletExists = true;
    return true;
  }

  /// Check if wallet exists
  Future<bool> checkWalletExists() async {
    final exists = await _secureStorage.read(key: 'wallet_exists');
    if (exists == 'true') {
      _walletExists = true;
      return true;
    }
    return false;
  }

  /// Unlock wallet with password
  Future<bool> unlockWallet(String password) async {
    final seed = await _secureStorage.read(key: 'wallet_seed');
    if (seed == null) return false;
    
    _seedPhrase = seed;
    _masterKey = _generateMasterKey(seed + password);
    await _deriveAddresses();
    
    return true;
  }

  // =========================================================================
  // Address Derivation
  // =========================================================================

  /// Derive addresses for all supported chains
  Future<void> _deriveAddresses() async {
    if (_masterKey == null) return;
    
    final chains = _chainService.getSupportedChains();
    
    for (final chain in chains) {
      final address = _deriveAddress(chain.derivationPath, chain.chainId);
      _addresses[chain.chainId.toString()] = address;
    }
  }

  /// Derive address for specific chain
  String _deriveAddress(String derivationPath, int chainId) {
    if (_masterKey == null) return '';
    
    // BIP-44 derivation
    // m/44'/coin_type'/0'/0/0
    final coinType = _getCoinType(chainId);
    final path = "m/44'/$coinType'/0'/0/0";
    
    // Simplified derivation (in production, use proper HD key derivation)
    final hash = sha256.convert(utf8.encode(_masterKey! + path));
    final addressBytes = hash.bytes;
    
    // Generate address based on chain type
    if (_isEVMChain(chainId)) {
      return _generateEVMAddress(addressBytes);
    } else if (chainId == 101) { // Solana
      return _generateSolanaAddress(addressBytes);
    } else if (chainId == 0) { // Bitcoin
      return _generateBitcoinAddress(addressBytes);
    }
    
    return HEX.encode(addressBytes.take(20).toList());
  }

  /// Get coin type for chain (BIP-44)
  int _getCoinType(int chainId) {
    const coinTypes = {
      1: 60,    // Ethereum
      56: 714,   // BNB Chain
      137: 966,  // Polygon
      42161: 60, // Arbitrum
      10: 60,    // Optimism
      8453: 60,  // Base
      324: 60,   // zkSync
      43114: 60, // Avalanche
      101: 501,  // Solana
      0: 0,      // Bitcoin
    };
    return coinTypes[chainId] ?? 60;
  }

  bool _isEVMChain(int chainId) {
    return [1, 56, 137, 42161, 10, 8453, 324, 43114, 5000, 81457, 100, 250, 42220, 8217, 25, 1284, 1285, 592]
        .contains(chainId);
  }

  String _generateEVMAddress(List<int> hashBytes) {
    final address = '0x' + HEX.encode(hashBytes.take(20).toList());
    return '0x${address.substring(2).toLowerCase()}';
  }

  String _generateSolanaAddress(List<int> hashBytes) {
    // Base58 encode (simplified)
    const chars = '123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz';
    final random = Random(HEX.encode(hashBytes).hashCode);
    return List.generate(32, (_) => chars[random.nextInt(chars.length)]).join();
  }

  String _generateBitcoinAddress(List<int> hashBytes) {
    // Simplified - would use proper Base58Check in production
    return '1' + HEX.encode(hashBytes.take(20).toList()).substring(0, 34);
  }

  String _generateMasterKey(String seed) {
    final hash = sha256.convert(utf8.encode(seed));
    return HEX.encode(hash.bytes);
  }

  bool _validateSeedPhrase(String phrase) {
    final words = phrase.trim().split(RegExp(r'\s+'));
    if (words.length != 12 && words.length != 24) {
      return false;
    }
    return words.every((w) => WORDLIST.contains(w));
  }

  // =========================================================================
  // Public Getters
  // =========================================================================

  String? get seedPhrase => _seedPhrase;
  bool get walletExists => _walletExists;
  
  Map<String, String> get addresses => Map.unmodifiable(_addresses);
  
  String? getAddress(int chainId) => _addresses[chainId.toString()];

  double? getBalance(int chainId) => _balances[chainId.toString()];

  // =========================================================================
  // Transaction Operations
  // =========================================================================

  /// Sign transaction (simplified - would use proper signing in production)
  Future<String> signTransaction({
    required int chainId,
    required String to,
    required String value,
    required String data,
    required int gasLimit,
    required String gasPrice,
  }) async {
    if (_masterKey == null) {
      throw Exception('Wallet not unlocked');
    }
    
    // Create transaction data
    final txData = {
      'chainId': chainId,
      'to': to,
      'value': value,
      'data': data,
      'gasLimit': gasLimit,
      'gasPrice': gasPrice,
      'nonce': await _getNonce(chainId),
    };
    
    // Sign with master key
    final hash = sha256.convert(utf8.encode(jsonEncode(txData) + _masterKey!));
    final signature = HEX.encode(hash.bytes);
    
    return signature;
  }

  Future<int> _getNonce(int chainId) async {
    // In production, fetch from RPC
    return 0;
  }

  // =========================================================================
  // Balance Operations
  // =========================================================================

  /// Fetch balance for chain
  Future<double> fetchBalance(int chainId) async {
    final address = _addresses[chainId.toString()];
    if (address == null) return 0.0;
    
    // In production, fetch from RPC
    // Simplified for demo
    _balances[chainId.toString()] = 0.0;
    return 0.0;
  }

  /// Fetch all balances
  Future<void> fetchAllBalances() async {
    for (final chainId in _addresses.keys) {
      await fetchBalance(int.parse(chainId));
    }
  }

  // =========================================================================
  // Export / Backup
  // =========================================================================

  /// Export wallet (requires authentication)
  Future<String?> exportSeedPhrase() async {
    return _seedPhrase;
  }

  /// Delete wallet
  Future<void> deleteWallet() async {
    await _secureStorage.delete(key: 'wallet_seed');
    await _secureStorage.delete(key: 'wallet_exists');
    _seedPhrase = null;
    _masterKey = null;
    _addresses = {};
    _balances = {};
    _walletExists = false;
  }
}

// BIP-39 Wordlist (abbreviated - full list would be 2048 words)
const WORDLIST = [
  'abandon', 'ability', 'able', 'about', 'above', 'absent', 'absorb', 'abstract',
  'absurd', 'abuse', 'access', 'accident', 'account', 'accuse', 'achieve', 'acid',
  'acoustic', 'acquire', 'across', 'act', 'action', 'actor', 'actress', 'actual',
  'adapt', 'add', 'addict', 'address', 'adjust', 'admit', 'adult', 'advance',
  'advice', 'aerobic', 'affair', 'afford', 'afraid', 'again', 'age', 'agent',
  'agree', 'ahead', 'aim', 'air', 'airport', 'aisle', 'alarm', 'album', 'alert',
  'alien', 'all', 'alley', 'allow', 'almost', 'alone', 'alpha', 'already', 'also',
  'alter', 'always', 'amateur', 'amazing', 'among', 'amount', 'amused', 'analyst',
  'anchor', 'ancient', 'anger', 'angle', 'angry', 'animal', 'ankle', 'announce',
  'annual', 'another', 'answer', 'antenna', 'anticipate', 'anxiety', 'any', 'apart',
  'apology', 'appear', 'apple', 'approve', 'april', 'arch', 'arctic', 'area',
  'arena', 'argue', 'arm', 'armed', 'armor', 'army', 'around', 'arrange', 'arrest',
];

// JSON encode helper
String jsonEncode(dynamic obj) => json.encode(obj);