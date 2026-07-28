// Wallet Provider - Complete Wallet State Management
// Manages all wallet operations including multi-chain support

import 'package:flutter/foundation.dart';
import '../models/wallet_model.dart';
import '../models/token_model.dart';
import '../models/chain_model.dart';
import '../services/wallet_service.dart';
import '../services/blockchain_service.dart';

class WalletProvider extends ChangeNotifier {
  final WalletService _walletService;
  final BlockchainService _blockchainService;
  
  // Wallet State
  bool _isLoading = false;
  bool _isInitialized = false;
  String? _error;
  Wallet? _currentWallet;
  List<ChainModel> _connectedChains = [];
  List<TokenModel> _tokens = [];
  double _totalBalance = 0.0;
  double _totalBalanceUSD = 0.0;
  double _change24h = 0.0;
  
  // Getters
  bool get isLoading => _isLoading;
  bool get isInitialized => _isInitialized;
  String? get error => _error;
  Wallet? get currentWallet => _currentWallet;
  List<ChainModel> get connectedChains => _connectedChains;
  List<TokenModel> get tokens => _tokens;
  double get totalBalance => _totalBalance;
  double get totalBalanceUSD => _totalBalanceUSD;
  double get change24h => _change24h;
  
  WalletProvider({
    required WalletService walletService,
    required BlockchainService blockchainService,
  })  : _walletService = walletService,
        _blockchainService = blockchainService;
  
  // Initialize wallet
  Future<void> initialize() async {
    _isLoading = true;
    notifyListeners();
    
    try {
      await _walletService.initialize();
      _connectedChains = await _blockchainService.getSupportedChains();
      _isInitialized = true;
      _error = null;
    } catch (e) {
      _error = e.toString();
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }
  
  // Check if wallet exists
  Future<bool> hasExistingWallet() async {
    return await _walletService.hasExistingWallet();
  }
  
  // Create new wallet
  Future<Wallet> createWallet(String password) async {
    _isLoading = true;
    _error = null;
    notifyListeners();
    
    try {
      _currentWallet = await _walletService.createWallet(password);
      await _loadWalletData();
      _error = null;
      return _currentWallet!;
    } catch (e) {
      _error = e.toString();
      rethrow;
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }
  
  // Import wallet from mnemonic
  Future<Wallet> importWallet(String mnemonic, String password) async {
    _isLoading = true;
    _error = null;
    notifyListeners();
    
    try {
      _currentWallet = await _walletService.importWallet(mnemonic, password);
      await _loadWalletData();
      _error = null;
      return _currentWallet!;
    } catch (e) {
      _error = e.toString();
      rethrow;
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }
  
  // Import wallet from private key
  Future<Wallet> importWalletFromPrivateKey(String privateKey, String password) async {
    _isLoading = true;
    _error = null;
    notifyListeners();
    
    try {
      _currentWallet = await _walletService.importWalletFromPrivateKey(privateKey, password);
      await _loadWalletData();
      _error = null;
      return _currentWallet!;
    } catch (e) {
      _error = e.toString();
      rethrow;
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }
  
  // Unlock wallet with password
  Future<bool> unlockWallet(String password) async {
    _isLoading = true;
    _error = null;
    notifyListeners();
    
    try {
      _currentWallet = await _walletService.unlockWallet(password);
      await _loadWalletData();
      _error = null;
      return true;
    } catch (e) {
      _error = e.toString();
      return false;
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }
  
  // Lock wallet
  void lockWallet() {
    _walletService.lockWallet();
    _currentWallet = null;
    _tokens = [];
    _totalBalance = 0;
    _totalBalanceUSD = 0;
    notifyListeners();
  }
  
  // Get mnemonic backup
  Future<String> getMnemonic(String password) async {
    return await _walletService.getMnemonic(password);
  }
  
  // Send transaction
  Future<String> sendTransaction({
    required String toAddress,
    required String amount,
    required String tokenAddress,
    required String chainId,
  }) async {
    _isLoading = true;
    _error = null;
    notifyListeners();
    
    try {
      final txHash = await _walletService.sendTransaction(
        toAddress: toAddress,
        amount: amount,
        tokenAddress: tokenAddress,
        chainId: chainId,
      );
      await refreshBalances();
      return txHash;
    } catch (e) {
      _error = e.toString();
      rethrow;
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }
  
  // Get transaction history
  Future<List<TransactionModel>> getTransactionHistory(String chainId) async {
    return await _walletService.getTransactionHistory(chainId);
  }
  
  // Add custom chain
  Future<void> addCustomChain(ChainModel chain) async {
    await _blockchainService.addCustomChain(chain);
    _connectedChains = await _blockchainService.getSupportedChains();
    notifyListeners();
  }
  
  // Remove custom chain
  Future<void> removeCustomChain(String chainId) async {
    await _blockchainService.removeCustomChain(chainId);
    _connectedChains = await _blockchainService.getSupportedChains();
    notifyListeners();
  }
  
  // Select chain
  void selectChain(ChainModel chain) {
    // Filter tokens for selected chain
    // This would trigger a rebuild
    notifyListeners();
  }
  
  // Refresh balances
  Future<void> refreshBalances() async {
    try {
      _tokens = await _walletService.getTokenBalances();
      _calculateTotalBalance();
      _error = null;
    } catch (e) {
      _error = e.toString();
    }
    notifyListeners();
  }
  
  // Private methods
  Future<void> _loadWalletData() async {
    await refreshBalances();
  }
  
  void _calculateTotalBalance() {
    _totalBalance = _tokens.fold(0.0, (sum, token) => sum + token.balance);
    _totalBalanceUSD = _tokens.fold(0.0, (sum, token) => sum + token.balanceUSD);
    // Calculate 24h change
    _change24h = _calculate24hChange();
  }
  
  double _calculate24hChange() {
    // In production, this would calculate from actual price changes
    if (_totalBalanceUSD == 0) return 0;
    
    // Mock calculation - in production would fetch from price API
    return 2.5; // 2.5% change
  }
  
  // Get address for specific chain
  String getAddressForChain(String chainId) {
    if (_currentWallet == null) return '';
    return _currentWallet!.getAddressForChain(chainId);
  }
  
  // Verify address
  bool isValidAddress(String address, String chainId) {
    return _blockchainService.isValidAddress(address, chainId);
  }
  
  // Get gas price
  Future<double> getGasPrice(String chainId) async {
    return await _blockchainService.getGasPrice(chainId);
  }
  
  // Estimate transaction fee
  Future<double> estimateTransactionFee({
    required String fromAddress,
    required String toAddress,
    required String amount,
    required String chainId,
  }) async {
    return await _blockchainService.estimateTransactionFee(
      fromAddress: fromAddress,
      toAddress: toAddress,
      amount: amount,
      chainId: chainId,
    );
  }
}
