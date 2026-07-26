/**
 * TigerWallet Flutter App Constants
 * Network configurations, API endpoints, and app constants
 */

class AppConstants {
  // App Info
  static const String appName = 'TigerWallet';
  static const String appVersion = '1.0.0';
  
  // API Endpoints
  static const String baseURL = 'https://api.tigerwallet.io';
  static const String wsURL = 'wss://ws.tigerwallet.io';
  
  // API Paths
  static const String apiV1 = '/api/v1';
  static const String authPath = '$apiV1/auth';
  static const String walletPath = '$apiV1/wallet';
  static const String swapPath = '$apiV1/swap';
  static const String stakePath = '$apiV1/staking';
  static const String bridgePath = '$apiV1/bridge';
  static const String nftPath = '$apiV1/nft';
  
  // Blockchain Chain IDs
  static const Map<String, int> chainIds = {
    'ethereum': 1,
    'polygon': 137,
    'bsc': 56,
    'arbitrum': 42161,
    'optimism': 10,
    'avalanche': 43114,
    'base': 8453,
    'solana': 0,
    'ton': 0,
    'bitcoin': 0,
  };
  
  // Supported Chains
  static const List<Map<String, dynamic>> supportedChains = [
    {'id': 1, 'name': 'Ethereum', 'symbol': 'ETH', 'type': 'evm', 'rpc': 'https://eth.llamarpc.com'},
    {'id': 56, 'name': 'BNB Smart Chain', 'symbol': 'BNB', 'type': 'evm', 'rpc': 'https://bsc-dataseed.binance.org'},
    {'id': 137, 'name': 'Polygon', 'symbol': 'MATIC', 'type': 'evm', 'rpc': 'https://polygon-rpc.com'},
    {'id': 42161, 'name': 'Arbitrum One', 'symbol': 'ETH', 'type': 'evm', 'rpc': 'https://arb1.arbitrum.io/rpc'},
    {'id': 10, 'name': 'Optimism', 'symbol': 'ETH', 'type': 'evm', 'rpc': 'https://mainnet.optimism.io'},
    {'id': 43114, 'name': 'Avalanche', 'symbol': 'AVAX', 'type': 'evm', 'rpc': 'https://api.avax.network/ext/bc/C/rpc'},
    {'id': 8453, 'name': 'Base', 'symbol': 'ETH', 'type': 'evm', 'rpc': 'https://mainnet.base.org'},
  ];
  
  // Popular Tokens
  static const List<Map<String, dynamic>> popularTokens = [
    {'symbol': 'ETH', 'name': 'Ethereum', 'address': '0x0000000000000000000000000000000000000000', 'decimals': 18, 'chainId': 1},
    {'symbol': 'USDT', 'name': 'Tether USD', 'address': '0xdAC17F958D2ee523a2206206994597C13D831ec7', 'decimals': 6, 'chainId': 1},
    {'symbol': 'USDC', 'name': 'USD Coin', 'address': '0xA0b86991c6218b36c1d19D4a2e9eb0cE3606eB48', 'decimals': 6, 'chainId': 1},
    {'symbol': 'BNB', 'name': 'BNB', 'address': '0x0000000000000000000000000000000000000000', 'decimals': 18, 'chainId': 56},
    {'symbol': 'MATIC', 'name': 'Polygon', 'address': '0x0000000000000000000000000000000000000000', 'decimals': 18, 'chainId': 137},
    {'symbol': 'AVAX', 'name': 'Avalanche', 'address': '0x0000000000000000000000000000000000000000', 'decimals': 18, 'chainId': 43114},
    {'symbol': 'LINK', 'name': 'Chainlink', 'address': '0x514910771AF9Ca656af840dff83E8264EcF986CA', 'decimals': 18, 'chainId': 1},
    {'symbol': 'UNI', 'name': 'Uniswap', 'address': '0x1f9840a85d5af5bf1d1762f925bdaddc4201f984', 'decimals': 18, 'chainId': 1},
  ];
  
  // Transaction Settings
  static const int defaultGasLimit = 21000;
  static const int defaultTokenGasLimit = 65000;
  static const int maxGasLimit = 500000;
  static const int transactionTimeout = 300; // seconds
  
  // Wallet Settings
  static const int bip39WordCount = 24;
  static const String derivationPath = "m/44'/60'/0'/0/0";
  
  // Storage Keys
  static const String keyWalletData = 'wallet_data';
  static const String keyThemeMode = 'theme_mode';
  static const String keySelectedChain = 'selected_chain';
  static const String keyLanguage = 'language';
  static const String keyBiometricEnabled = 'biometric_enabled';
  
  // Limits
  static const double maxTransferAmount = 999999999;
  static const double minTransferAmount = 0.0001;
  
  // Timeouts
  static const int connectionTimeout = 30000;
  static const int receiveTimeout = 30000;
  
  // Pagination
  static const int defaultPageSize = 20;
  static const int maxPageSize = 100;
}

class ChainConfig {
  final int chainId;
  final String name;
  final String symbol;
  final String rpcUrl;
  final String explorerUrl;
  final int decimals;
  final bool isTestnet;
  
  const ChainConfig({
    required this.chainId,
    required this.name,
    required this.symbol,
    required this.rpcUrl,
    required this.explorerUrl,
    required this.decimals,
    this.isTestnet = false,
  });
  
  static const Map<int, ChainConfig> chains = {
    1: ChainConfig(
      chainId: 1,
      name: 'Ethereum',
      symbol: 'ETH',
      rpcUrl: 'https://eth.llamarpc.com',
      explorerUrl: 'https://etherscan.io',
      decimals: 18,
    ),
    56: ChainConfig(
      chainId: 56,
      name: 'BNB Smart Chain',
      symbol: 'BNB',
      rpcUrl: 'https://bsc-dataseed.binance.org',
      explorerUrl: 'https://bscscan.com',
      decimals: 18,
    ),
    137: ChainConfig(
      chainId: 137,
      name: 'Polygon',
      symbol: 'MATIC',
      rpcUrl: 'https://polygon-rpc.com',
      explorerUrl: 'https://polygonscan.com',
      decimals: 18,
    ),
    42161: ChainConfig(
      chainId: 42161,
      name: 'Arbitrum One',
      symbol: 'ETH',
      rpcUrl: 'https://arb1.arbitrum.io/rpc',
      explorerUrl: 'https://arbiscan.io',
      decimals: 18,
    ),
    10: ChainConfig(
      chainId: 10,
      name: 'Optimism',
      symbol: 'ETH',
      rpcUrl: 'https://mainnet.optimism.io',
      explorerUrl: 'https://optimistic.etherscan.io',
      decimals: 18,
    ),
    43114: ChainConfig(
      chainId: 43114,
      name: 'Avalanche',
      symbol: 'AVAX',
      rpcUrl: 'https://api.avax.network/ext/bc/C/rpc',
      explorerUrl: 'https://snowtrace.io',
      decimals: 18,
    ),
    8453: ChainConfig(
      chainId: 8453,
      name: 'Base',
      symbol: 'ETH',
      rpcUrl: 'https://mainnet.base.org',
      explorerUrl: 'https://basescan.org',
      decimals: 18,
    ),
  };
  
  static ChainConfig? getChain(int chainId) => chains[chainId];
  
  static ChainConfig getChainOrDefault(int chainId) => 
      chains[chainId] ?? chains[1]!;
}
