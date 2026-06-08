import 'dart:convert';

/// Chain Service - Manages blockchain configurations
/// 
/// Supports 40+ chains (EVM + Non-EVM)

class Chain {
  final int chainId;
  final String name;
  final String symbol;
  final String rpcUrl;
  final String explorerUrl;
  final String derivationPath;
  final String chainType;
  final bool isActive;
  final String? iconUrl;

  Chain({
    required this.chainId,
    required this.name,
    required this.symbol,
    required this.rpcUrl,
    required this.explorerUrl,
    required this.derivationPath,
    required this.chainType,
    this.isActive = true,
    this.iconUrl,
  });

  factory Chain.fromJson(Map<String, dynamic> json) {
    return Chain(
      chainId: json['chainId'],
      name: json['name'],
      symbol: json['symbol'],
      rpcUrl: json['rpcUrl'],
      explorerUrl: json['explorerUrl'],
      derivationPath: json['derivationPath'] ?? "m/44'/60'/0'/0/0",
      chainType: json['chainType'] ?? 'evm',
      isActive: json['isActive'] ?? true,
      iconUrl: json['iconUrl'],
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'chainId': chainId,
      'name': name,
      'symbol': symbol,
      'rpcUrl': rpcUrl,
      'explorerUrl': explorerUrl,
      'derivationPath': derivationPath,
      'chainType': chainType,
      'isActive': isActive,
      'iconUrl': iconUrl,
    };
  }
}

/// Chain Service
class ChainService {
  List<Chain> _chains = [];
  bool _isLoaded = false;

  /// Load all supported chains
  Future<void> loadChains() async {
    _chains = _getDefaultChains();
    _isLoaded = true;
  }

  /// Get all supported chains
  List<Chain> getSupportedChains() {
    return _chains.where((c) => c.isActive).toList();
  }

  /// Get chain by ID
  Chain? getChain(int chainId) {
    try {
      return _chains.firstWhere((c) => c.chainId == chainId);
    } catch (_) {
      return null;
    }
  }

  /// Get EVM chains
  List<Chain> getEVMChains() {
    return _chains.where((c) => c.chainType == 'evm' && c.isActive).toList();
  }

  /// Get Non-EVM chains
  List<Chain> getNonEVMChains() {
    return _chains.where((c) => c.chainType != 'evm' && c.isActive).toList();
  }

  /// Get default chains
  List<Chain> _getDefaultChains() {
    return [
      // EVM Chains (20+)
      Chain(
        chainId: 1,
        name: 'Ethereum',
        symbol: 'ETH',
        rpcUrl: 'https://eth.llamarpc.com',
        explorerUrl: 'https://etherscan.io',
        derivationPath: "m/44'/60'/0'/0/0",
        chainType: 'evm',
      ),
      Chain(
        chainId: 56,
        name: 'BNB Chain',
        symbol: 'BNB',
        rpcUrl: 'https://bsc-dataseed.binance.org',
        explorerUrl: 'https://bscscan.com',
        derivationPath: "m/44'/714'/0'/0/0",
        chainType: 'evm',
      ),
      Chain(
        chainId: 137,
        name: 'Polygon',
        symbol: 'MATIC',
        rpcUrl: 'https://polygon-rpc.com',
        explorerUrl: 'https://polygonscan.com',
        derivationPath: "m/44'/966'/0'/0/0",
        chainType: 'evm',
      ),
      Chain(
        chainId: 42161,
        name: 'Arbitrum One',
        symbol: 'ETH',
        rpcUrl: 'https://arb1.arbitrum.io/rpc',
        explorerUrl: 'https://arbiscan.io',
        derivationPath: "m/44'/60'/0'/0/0",
        chainType: 'evm',
      ),
      Chain(
        chainId: 10,
        name: 'Optimism',
        symbol: 'ETH',
        rpcUrl: 'https://mainnet.optimism.io',
        explorerUrl: 'https://optimistic.etherscan.io',
        derivationPath: "m/44'/60'/0'/0/0",
        chainType: 'evm',
      ),
      Chain(
        chainId: 8453,
        name: 'Base',
        symbol: 'ETH',
        rpcUrl: 'https://mainnet.base.org',
        explorerUrl: 'https://basescan.org',
        derivationPath: "m/44'/60'/0'/0/0",
        chainType: 'evm',
      ),
      Chain(
        chainId: 324,
        name: 'zkSync Era',
        symbol: 'ETH',
        rpcUrl: 'https://zksync-era.blockchainrpc.com',
        explorerUrl: 'https://explorer.zksync.io',
        derivationPath: "m/44'/60'/0'/0/0",
        chainType: 'evm',
      ),
      Chain(
        chainId: 59144,
        name: 'Linea',
        symbol: 'ETH',
        rpcUrl: 'https://rpc.linea.build',
        explorerUrl: 'https://lineascan.build',
        derivationPath: "m/44'/60'/0'/0/0",
        chainType: 'evm',
      ),
      Chain(
        chainId: 534352,
        name: 'Scroll',
        symbol: 'ETH',
        rpcUrl: 'https://rpc.scroll.io',
        explorerUrl: 'https://scrollscan.com',
        derivationPath: "m/44'/60'/0'/0/0",
        chainType: 'evm',
      ),
      Chain(
        chainId: 43114,
        name: 'Avalanche',
        symbol: 'AVAX',
        rpcUrl: 'https://api.avax.network',
        explorerUrl: 'https://snowtrace.io',
        derivationPath: "m/44'/60'/0'/0/0",
        chainType: 'evm',
      ),
      Chain(
        chainId: 5000,
        name: 'Mantle',
        symbol: 'MNT',
        rpcUrl: 'https://rpc.mantle.xyz',
        explorerUrl: 'https://explorer.mantle.xyz',
        derivationPath: "m/44'/60'/0'/0/0",
        chainType: 'evm',
      ),
      Chain(
        chainId: 81457,
        name: 'Blast',
        symbol: 'ETH',
        rpcUrl: 'https://rpc.blast.io',
        explorerUrl: 'https://blastscan.io',
        derivationPath: "m/44'/60'/0'/0/0",
        chainType: 'evm',
      ),
      Chain(
        chainId: 100,
        name: 'Gnosis',
        symbol: 'xDAI',
        rpcUrl: 'https://rpc.gnosischain.com',
        explorerUrl: 'https://gnosisscan.io',
        derivationPath: "m/44'/60'/0'/0/0",
        chainType: 'evm',
      ),
      Chain(
        chainId: 250,
        name: 'Fantom',
        symbol: 'FTM',
        rpcUrl: 'https://rpc.fantom.network',
        explorerUrl: 'https://ftmscan.com',
        derivationPath: "m/44'/60'/0'/0/0",
        chainType: 'evm',
      ),
      Chain(
        chainId: 42220,
        name: 'Celo',
        symbol: 'CELO',
        rpcUrl: 'https://rpc.celo.org',
        explorerUrl: 'https://celoscan.io',
        derivationPath: "m/44'/60'/0'/0/0",
        chainType: 'evm',
      ),
      Chain(
        chainId: 8217,
        name: 'Klaytn',
        symbol: 'KLAY',
        rpcUrl: 'https://rpc.klaytn.org',
        explorerUrl: 'https://scope.klaytn.com',
        derivationPath: "m/44'/60'/0'/0/0",
        chainType: 'evm',
      ),
      Chain(
        chainId: 25,
        name: 'Cronos',
        symbol: 'CRO',
        rpcUrl: 'https://rpc-cronos.crypto.org',
        explorerUrl: 'https://cronoscan.com',
        derivationPath: "m/44'/60'/0'/0/0",
        chainType: 'evm',
      ),
      Chain(
        chainId: 1284,
        name: 'Moonbeam',
        symbol: 'GLMR',
        rpcUrl: 'https://rpc.api.moonbeam.network',
        explorerUrl: 'https://moonbeam.moonscan.io',
        derivationPath: "m/44'/60'/0'/0/0",
        chainType: 'evm',
      ),
      Chain(
        chainId: 1285,
        name: 'Moonriver',
        symbol: 'MOVR',
        rpcUrl: 'https://rpc.api.moonriver.moonbeam.network',
        explorerUrl: 'https://moonriver.moonscan.io',
        derivationPath: "m/44'/60'/0'/0/0",
        chainType: 'evm',
      ),
      Chain(
        chainId: 592,
        name: 'Astar',
        symbol: 'ASTR',
        rpcUrl: 'https://rpc.astar.network',
        explorerUrl: 'https://astar.explorer.io',
        derivationPath: "m/44'/60'/0'/0/0",
        chainType: 'evm',
      ),
      // Non-EVM Chains
      Chain(
        chainId: 101,
        name: 'Solana',
        symbol: 'SOL',
        rpcUrl: 'https://api.mainnet-beta.solana.com',
        explorerUrl: 'https://solscan.io',
        derivationPath: "m/44'/501'/0'/0'",
        chainType: 'solana',
      ),
      Chain(
        chainId: 0,
        name: 'Bitcoin',
        symbol: 'BTC',
        rpcUrl: 'https://blockstream.info/api',
        explorerUrl: 'https://mempool.space',
        derivationPath: "m/44'/0'/0'/0/0",
        chainType: 'bitcoin',
      ),
      Chain(
        chainId: 100,
        name: 'Cosmos',
        symbol: 'ATOM',
        rpcUrl: 'https://rpc.cosmos.network',
        explorerUrl: 'https://mintscan.io/cosmos',
        derivationPath: "m/44'/118'/0'/0/0",
        chainType: 'cosmos',
      ),
      Chain(
        chainId: 1100,
        name: 'Sui',
        symbol: 'SUI',
        rpcUrl: 'https://fullnode.mainnet.sui.io',
        explorerUrl: 'https://suiexplorer.com',
        derivationPath: "m/44'/784'/0'/0'/0'",
        chainType: 'sui',
      ),
      Chain(
        chainId: 1101,
        name: 'Aptos',
        symbol: 'APT',
        rpcUrl: 'https://fullnode.mainnet.aptoslabs.com',
        explorerUrl: 'https://aptoscan.com',
        derivationPath: "m/44'/637'/0'/0'/0",
        chainType: 'aptos',
      ),
      Chain(
        chainId: -13,
        name: 'TRON',
        symbol: 'TRX',
        rpcUrl: 'https://api.trongrid.io',
        explorerUrl: 'https://tronscan.org',
        derivationPath: "m/44'/195'/0'/0/0",
        chainType: 'tron',
      ),
      Chain(
        chainId: 2000,
        name: 'NEAR',
        symbol: 'NEAR',
        rpcUrl: 'https://rpc.mainnet.near.org',
        explorerUrl: 'https://nearblocks.io',
        derivationPath: "m/44'/397'/0'/0/0'",
        chainType: 'near',
      ),
      Chain(
        chainId: 0,
        name: 'Polkadot',
        symbol: 'DOT',
        rpcUrl: 'https://rpc.polkadot.io',
        explorerUrl: 'https://polkadot.subscan.io',
        derivationPath: "m/44'/354'/0'/0/0",
        chainType: 'polkadot',
      ),
      Chain(
        chainId: 0,
        name: 'Osmosis',
        symbol: 'OSMO',
        rpcUrl: 'https://osmosis-rpc.polkachu.com',
        explorerUrl: 'https://www.mintscan.io/osmosis',
        derivationPath: "m/44'/118'/0'/0/0",
        chainType: 'cosmos',
      ),
      Chain(
        chainId: 0,
        name: 'Injective',
        symbol: 'INJ',
        rpcUrl: 'https://public.injective.network',
        explorerUrl: 'https://explorer.injective.network',
        derivationPath: "m/44'/690'/0'/0/0",
        chainType: 'cosmos',
      ),
    ];
  }
}