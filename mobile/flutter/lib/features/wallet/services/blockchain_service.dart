// Blockchain Service - Complete Blockchain Network Management
// Manages 300+ blockchains with RPC infrastructure

import '../models/chain_model.dart';

class BlockchainService {
  List<ChainModel> _supportedChains = [];
  final Map<String, String> _customRpcUrls = {};
  
  // Initialize with default chains
  Future<void> initialize() async {
    _supportedChains = _getDefaultChains();
  }
  
  // Get all supported chains
  Future<List<ChainModel>> getSupportedChains() async {
    return _supportedChains;
  }
  
  // Add custom chain
  Future<void> addCustomChain(ChainModel chain) async {
    final existingIndex = _supportedChains.indexWhere((c) => c.id == chain.id);
    if (existingIndex >= 0) {
      _supportedChains[existingIndex] = chain;
    } else {
      _supportedChains.add(chain);
    }
    _customRpcUrls[chain.id] = chain.rpcUrl;
  }
  
  // Remove custom chain
  Future<void> removeCustomChain(String chainId) async {
    _supportedChains.removeWhere((c) => c.id == chainId);
    _customRpcUrls.remove(chainId);
  }
  
  // Get chain by ID
  ChainModel? getChainById(String chainId) {
    try {
      return _supportedChains.firstWhere((c) => c.id == chainId);
    } catch (e) {
      return null;
    }
  }
  
  // Validate address format
  bool isValidAddress(String address, String chainId) {
    final chain = getChainById(chainId);
    if (chain == null) return false;
    
    switch (chain.type) {
      case ChainType.evm:
        return RegExp(r'^0x[a-fA-F0-9]{40}$').hasMatch(address);
      case ChainType.bitcoin:
        return RegExp(r'^[13][a-km-zA-HJ-NP-Z1-9]{25,34}$|^bc1[a-zA-HJ-NP-Z0-9]{25,89}$').hasMatch(address);
      case ChainType.solana:
        return RegExp(r'^[1-9A-HJ-NP-Za-km-z]{32,44}$').hasMatch(address);
      case ChainType.cosmos:
        return RegExp(r'^cosmos[a-zA-HJ-NP-Z0-9]{39}$').hasMatch(address);
      case ChainType.ton:
        return RegExp(r'^UQ[a-zA-HJ-NP-Z0-9_-]{46}$').hasMatch(address);
      case ChainType.tron:
        return RegExp(r'^T[a-zA-HJ-NP-Z1-9]{33}$').hasMatch(address);
      case ChainType.near:
        return RegExp(r'^[a-zA-Z0-9_-]{2,64}\.near$').hasMatch(address);
      case ChainType.aptos:
        return RegExp(r'^0x[a-fA-F0-9]{64}$').hasMatch(address);
      case ChainType.sui:
        return RegExp(r'^0x[a-fA-F0-9]{64}$').hasMatch(address);
      default:
        return address.length > 10;
    }
  }
  
  // Get gas price for chain
  Future<double> getGasPrice(String chainId) async {
    // In production, would call RPC
    final chain = getChainById(chainId);
    if (chain == null) return 0;
    
    switch (chain.type) {
      case ChainType.evm:
        return 0.000000020; // 20 Gwei default
      case ChainType.solana:
        return 0.000005; // 5000 lamports
      case ChainType.bitcoin:
        return 0.00001; // 1 sat/vB
      default:
        return 0.001;
    }
  }
  
  // Estimate transaction fee
  Future<double> estimateTransactionFee({
    required String fromAddress,
    required String toAddress,
    required String amount,
    required String chainId,
  }) async {
    final gasPrice = await getGasPrice(chainId);
    final gasLimit = _getGasLimit(chainId);
    return gasPrice * gasLimit;
  }
  
  double _getGasLimit(String chainId) {
    final chain = getChainById(chainId);
    if (chain == null) return 21000;
    
    switch (chain.type) {
      case ChainType.evm:
        return 21000; // Basic transfer
      case ChainType.solana:
        return 5000; // Basic transfer
      case ChainType.bitcoin:
        return 250; // vBytes
      default:
        return 10000;
    }
  }
  
  // Get RPC URL
  String getRpcUrl(String chainId) {
    return _customRpcUrls[chainId] ?? 
           getChainById(chainId)?.rpcUrl ?? 
           '';
  }
  
  // Get explorer URL
  String getExplorerUrl(String chainId, String txHash) {
    final chain = getChainById(chainId);
    if (chain == null) return '';
    return '${chain.explorerUrl}/tx/$txHash';
  }
  
  // Default chains list
  List<ChainModel> _getDefaultChains() {
    return [
      // EVM Chains
      ChainModel(
        id: 'ethereum',
        name: 'Ethereum',
        symbol: 'ETH',
        iconUrl: 'https://assets.coingecko.com/coins/images/279/small/ethereum.png',
        chainId: 1,
        rpcUrl: 'https://eth.llamarpc.com',
        explorerUrl: 'https://etherscan.io',
        explorerApiUrl: 'https://api.etherscan.io/api',
        type: ChainType.evm,
        config: ChainConfig(
          derivationPath: "m/44'/60'/0'/0/0",
          addressPrefix: 0,
          supportsEIP1559: true,
          blockTime: BlockTimeConfig(targetTimeSeconds: 12, confirmationBlocks: 12),
          gasConfig: GasConfig(gasToken: 'ETH', decimals: 18, minGasPrice: 0.000000001, maxGasPrice: 0.000001, gasLimit: 21000),
        ),
      ),
      ChainModel(
        id: 'sepolia',
        name: 'Sepolia',
        symbol: 'ETH',
        iconUrl: 'https://assets.coingecko.com/coins/images/279/small/ethereum.png',
        chainId: 11155111,
        rpcUrl: 'https://rpc.sepolia.org',
        explorerUrl: 'https://sepolia.etherscan.io',
        type: ChainType.evm,
        isTestnet: true,
        config: ChainConfig(
          derivationPath: "m/44'/60'/0'/0/0",
          addressPrefix: 0,
          supportsEIP1559: true,
          blockTime: BlockTimeConfig(targetTimeSeconds: 12, confirmationBlocks: 12),
          gasConfig: GasConfig(gasToken: 'ETH', decimals: 18, minGasPrice: 0.000000001, maxGasPrice: 0.000001, gasLimit: 21000),
        ),
      ),
      ChainModel(
        id: 'bsc',
        name: 'BNB Chain',
        symbol: 'BNB',
        iconUrl: 'https://assets.coingecko.com/coins/images/825/small/bnb-icon2_2x.png',
        chainId: 56,
        rpcUrl: 'https://bsc-dataseed.binance.org',
        explorerUrl: 'https://bscscan.com',
        type: ChainType.evm,
        config: ChainConfig(
          derivationPath: "m/44'/60'/0'/0/0",
          addressPrefix: 0,
          supportsEIP1559: false,
          blockTime: BlockTimeConfig(targetTimeSeconds: 3, confirmationBlocks: 12),
          gasConfig: GasConfig(gasToken: 'BNB', decimals: 18, minGasPrice: 0.000000005, maxGasPrice: 0.000001, gasLimit: 21000),
        ),
      ),
      ChainModel(
        id: 'polygon',
        name: 'Polygon',
        symbol: 'MATIC',
        iconUrl: 'https://assets.coingecko.com/coins/images/4713/small/matic-token-icon.png',
        chainId: 137,
        rpcUrl: 'https://polygon-rpc.com',
        explorerUrl: 'https://polygonscan.com',
        type: ChainType.evm,
        config: ChainConfig(
          derivationPath: "m/44'/60'/0'/0/0",
          addressPrefix: 0,
          supportsEIP1559: true,
          blockTime: BlockTimeConfig(targetTimeSeconds: 2, confirmationBlocks: 128),
          gasConfig: GasConfig(gasToken: 'MATIC', decimals: 18, minGasPrice: 0.000000001, maxGasPrice: 0.000001, gasLimit: 21000),
        ),
      ),
      ChainModel(
        id: 'arbitrum',
        name: 'Arbitrum One',
        symbol: 'ETH',
        iconUrl: 'https://assets.coingecko.com/coins/images/16547/small/photo_2023-03-29_21.47.00.jpeg',
        chainId: 42161,
        rpcUrl: 'https://arb1.arbitrum.io/rpc',
        explorerUrl: 'https://arbiscan.io',
        type: ChainType.evm,
        config: ChainConfig(
          derivationPath: "m/44'/60'/0'/0/0",
          addressPrefix: 0,
          supportsEIP1559: true,
          blockTime: BlockTimeConfig(targetTimeSeconds: 1, confirmationBlocks: 12),
          gasConfig: GasConfig(gasToken: 'ETH', decimals: 18, minGasPrice: 0.000000001, maxGasPrice: 0.000001, gasLimit: 21000),
        ),
      ),
      ChainModel(
        id: 'optimism',
        name: 'Optimism',
        symbol: 'ETH',
        iconUrl: 'https://assets.coingecko.com/coins/images/25244/small/Optimism.png',
        chainId: 10,
        rpcUrl: 'https://mainnet.optimism.io',
        explorerUrl: 'https://optimistic.etherscan.io',
        type: ChainType.evm,
        config: ChainConfig(
          derivationPath: "m/44'/60'/0'/0/0",
          addressPrefix: 0,
          supportsEIP1559: true,
          blockTime: BlockTimeConfig(targetTimeSeconds: 2, confirmationBlocks: 12),
          gasConfig: GasConfig(gasToken: 'ETH', decimals: 18, minGasPrice: 0.000000001, maxGasPrice: 0.000001, gasLimit: 21000),
        ),
      ),
      ChainModel(
        id: 'base',
        name: 'Base',
        symbol: 'ETH',
        iconUrl: 'https://assets.coingecko.com/coins/images/26412/small/base_llama.png',
        chainId: 8453,
        rpcUrl: 'https://mainnet.base.org',
        explorerUrl: 'https://basescan.org',
        type: ChainType.evm,
        config: ChainConfig(
          derivationPath: "m/44'/60'/0'/0/0",
          addressPrefix: 0,
          supportsEIP1559: true,
          blockTime: BlockTimeConfig(targetTimeSeconds: 2, confirmationBlocks: 12),
          gasConfig: GasConfig(gasToken: 'ETH', decimals: 18, minGasPrice: 0.000000001, maxGasPrice: 0.000001, gasLimit: 21000),
        ),
      ),
      ChainModel(
        id: 'avalanche',
        name: 'Avalanche C-Chain',
        symbol: 'AVAX',
        iconUrl: 'https://assets.coingecko.com/coins/images/12559/small/Avalanche_Circle_RedWhite_Trans.png',
        chainId: 43114,
        rpcUrl: 'https://api.avax.network/ext/bc/C/rpc',
        explorerUrl: 'https://snowtrace.io',
        type: ChainType.evm,
        config: ChainConfig(
          derivationPath: "m/44'/60'/0'/0/0",
          addressPrefix: 0,
          supportsEIP1559: true,
          blockTime: BlockTimeConfig(targetTimeSeconds: 2, confirmationBlocks: 12),
          gasConfig: GasConfig(gasToken: 'AVAX', decimals: 18, minGasPrice: 0.000000025, maxGasPrice: 0.000001, gasLimit: 21000),
        ),
      ),
      ChainModel(
        id: 'fantom',
        name: 'Fantom',
        symbol: 'FTM',
        iconUrl: 'https://assets.coingecko.com/coins/images/4001/small/Fantom_round.png',
        chainId: 250,
        rpcUrl: 'https://rpc.fantom.network',
        explorerUrl: 'https://ftmscan.com',
        type: ChainType.evm,
        config: ChainConfig(
          derivationPath: "m/44'/60'/0'/0/0",
          addressPrefix: 0,
          supportsEIP1559: false,
          blockTime: BlockTimeConfig(targetTimeSeconds: 1, confirmationBlocks: 1),
          gasConfig: GasConfig(gasToken: 'FTM', decimals: 18, minGasPrice: 0.000000001, maxGasPrice: 0.000001, gasLimit: 21000),
        ),
      ),
      ChainModel(
        id: 'cronos',
        name: 'Cronos',
        symbol: 'CRO',
        iconUrl: 'https://assets.coingecko.com/coins/images/7310/small/cro_token.png',
        chainId: 25,
        rpcUrl: 'https://evm.cronos.org',
        explorerUrl: 'https://cronoscan.com',
        type: ChainType.evm,
        config: ChainConfig(
          derivationPath: "m/44'/60'/0'/0/0",
          addressPrefix: 0,
          supportsEIP1559: false,
          blockTime: BlockTimeConfig(targetTimeSeconds: 6, confirmationBlocks: 50),
          gasConfig: GasConfig(gasToken: 'CRO', decimals: 18, minGasPrice: 0.00000001, maxGasPrice: 0.000001, gasLimit: 21000),
        ),
      ),
      
      // Non-EVM Chains
      ChainModel(
        id: 'solana',
        name: 'Solana',
        symbol: 'SOL',
        iconUrl: 'https://assets.coingecko.com/coins/images/4128/small/solana.png',
        chainId: 0,
        rpcUrl: 'https://api.mainnet-beta.solana.com',
        explorerUrl: 'https://explorer.solana.com',
        type: ChainType.solana,
        config: ChainConfig(
          derivationPath: "m/44'/501'/0'/0'",
          addressPrefix: 0,
          supportsEIP1559: false,
          supportsTokenTransfers: true,
          supportsNFT: true,
          blockTime: BlockTimeConfig(targetTimeSeconds: 0.4, confirmationBlocks: 32),
          gasConfig: GasConfig(gasToken: 'SOL', decimals: 9, minGasPrice: 0.000005, maxGasPrice: 0.01, gasLimit: 5000),
        ),
      ),
      ChainModel(
        id: 'aptos',
        name: 'Aptos',
        symbol: 'APT',
        iconUrl: 'https://assets.coingecko.com/coins/images/26455/small/aptos_round.png',
        chainId: 0,
        rpcUrl: 'https://fullnode.mainnet.aptoslabs.com',
        explorerUrl: 'https://explorer.aptoslabs.com',
        type: ChainType.aptos,
        config: ChainConfig(
          derivationPath: "m/44'/637'/0'/0'/0'",
          addressPrefix: 0,
          supportsEIP1559: false,
          supportsTokenTransfers: true,
          supportsNFT: true,
          blockTime: BlockTimeConfig(targetTimeSeconds: 1, confirmationBlocks: 1),
          gasConfig: GasConfig(gasToken: 'APT', decimals: 8, minGasPrice: 0.0000001, maxGasPrice: 1, gasLimit: 2000),
        ),
      ),
      ChainModel(
        id: 'sui',
        name: 'Sui',
        symbol: 'SUI',
        iconUrl: 'https://assets.coingecko.com/coins/images/26375/small/sui_asset.jpeg',
        chainId: 0,
        rpcUrl: 'https://fullnode.mainnet.sui.io',
        explorerUrl: 'https://suiexplorer.com',
        type: ChainType.sui,
        config: ChainConfig(
          derivationPath: "m/44'/784'/0'/0'/0'",
          addressPrefix: 0,
          supportsEIP1559: false,
          supportsTokenTransfers: true,
          supportsNFT: true,
          blockTime: BlockTimeConfig(targetTimeSeconds: 3, confirmationBlocks: 1),
          gasConfig: GasConfig(gasToken: 'SUI', decimals: 9, minGasPrice: 0.0000001, maxGasPrice: 1, gasLimit: 2000),
        ),
      ),
      ChainModel(
        id: 'ton',
        name: 'Toncoin',
        symbol: 'TON',
        iconUrl: 'https://assets.coingecko.com/coins/images/17980/small/ton_symbol.png',
        chainId: 0,
        rpcUrl: 'https://toncenter.com/api/v2',
        explorerUrl: 'https://toncenter.com',
        type: ChainType.ton,
        config: ChainConfig(
          derivationPath: "m/44'/607'/0'/0'/0'",
          addressPrefix: 0,
          supportsEIP1559: false,
          supportsTokenTransfers: true,
          blockTime: BlockTimeConfig(targetTimeSeconds: 5, confirmationBlocks: 1),
          gasConfig: GasConfig(gasToken: 'TON', decimals: 9, minGasPrice: 0.0000001, maxGasPrice: 1, gasLimit: 1000),
        ),
      ),
      ChainModel(
        id: 'tron',
        name: 'TRON',
        symbol: 'TRX',
        iconUrl: 'https://assets.coingecko.com/coins/images/1094/small/tron-logo.png',
        chainId: 0,
        rpcUrl: 'https://api.trongrid.io',
        explorerUrl: 'https://tronscan.org',
        type: ChainType.tron,
        config: ChainConfig(
          derivationPath: "m/44'/195'/0'/0'/0'",
          addressPrefix: 0,
          supportsEIP1559: false,
          supportsTokenTransfers: true,
          supportsNFT: true,
          blockTime: BlockTimeConfig(targetTimeSeconds: 3, confirmationBlocks: 19),
          gasConfig: GasConfig(gasToken: 'TRX', decimals: 6, minGasPrice: 0.000001, maxGasPrice: 1, gasLimit: 100000),
        ),
      ),
      ChainModel(
        id: 'cosmos',
        name: 'Cosmos Hub',
        symbol: 'ATOM',
        iconUrl: 'https://assets.coingecko.com/coins/images/1481/small/cosmos_hub.png',
        chainId: 0,
        rpcUrl: 'https://rpc.cosmos.network',
        explorerUrl: 'https://mintscan.io/cosmos',
        type: ChainType.cosmos,
        config: ChainConfig(
          derivationPath: "m/44'/118'/0'/0'/0'",
          addressPrefix: 0x2B,
          supportsEIP1559: false,
          supportsTokenTransfers: true,
          supportsStaking: true,
          blockTime: BlockTimeConfig(targetTimeSeconds: 7, confirmationBlocks: 1),
          gasConfig: GasConfig(gasToken: 'ATOM', decimals: 6, minGasPrice: 0.0001, maxGasPrice: 1, gasLimit: 200000),
        ),
      ),
      ChainModel(
        id: 'near',
        name: 'NEAR Protocol',
        symbol: 'NEAR',
        iconUrl: 'https://assets.coingecko.com/coins/images/10365/small/near.jpg',
        chainId: 0,
        rpcUrl: 'https://rpc.mainnet.near.org',
        explorerUrl: 'https://explorer.near.org',
        type: ChainType.near,
        config: ChainConfig(
          derivationPath: "m/44'/397'/0'/0'",
          addressPrefix: 0,
          supportsEIP1559: false,
          supportsTokenTransfers: true,
          supportsNFT: true,
          supportsStaking: true,
          blockTime: BlockTimeConfig(targetTimeSeconds: 1, confirmationBlocks: 1),
          gasConfig: GasConfig(gasToken: 'NEAR', decimals: 24, minGasPrice: 0.0000001, maxGasPrice: 1, gasLimit: 300000000000000),
        ),
      ),
      ChainModel(
        id: 'algorand',
        name: 'Algorand',
        symbol: 'ALGO',
        iconUrl: 'https://assets.coingecko.com/coins/images/4380/small/download.png',
        chainId: 0,
        rpcUrl: 'https://algoexplorer.org/api/v2',
        explorerUrl: 'https://algoexplorer.org',
        type: ChainType.algorand,
        config: ChainConfig(
          derivationPath: "m/44'/283'/0'/0'",
          addressPrefix: 0,
          supportsEIP1559: false,
          supportsTokenTransfers: true,
          supportsNFT: true,
          blockTime: BlockTimeConfig(targetTimeSeconds: 3, confirmationBlocks: 1),
          gasConfig: GasConfig(gasToken: 'ALGO', decimals: 6, minGasPrice: 0.001, maxGasPrice: 1, gasLimit: 1000),
        ),
      ),
      ChainModel(
        id: 'cardano',
        name: 'Cardano',
        symbol: 'ADA',
        iconUrl: 'https://assets.coingecko.com/coins/images/975/small/cardano.png',
        chainId: 0,
        rpcUrl: 'https://cardano-mainnet.blockfrost.io',
        explorerUrl: 'https://cardanoscan.io',
        type: ChainType.cardano,
        config: ChainConfig(
          derivationPath: "m/44'/1815'/0'/0'",
          addressPrefix: 0,
          supportsEIP1559: false,
          supportsTokenTransfers: true,
          supportsNFT: true,
          blockTime: BlockTimeConfig(targetTimeSeconds: 10, confirmationBlocks: 1),
          gasConfig: GasConfig(gasToken: 'ADA', decimals: 6, minGasPrice: 0.000001, maxGasPrice: 1, gasLimit: 1000000),
        ),
      ),
      ChainModel(
        id: 'bitcoin',
        name: 'Bitcoin',
        symbol: 'BTC',
        iconUrl: 'https://assets.coingecko.com/coins/images/1/small/bitcoin.png',
        chainId: 0,
        rpcUrl: 'https://blockstream.info/api',
        explorerUrl: 'https://blockstream.info',
        type: ChainType.bitcoin,
        config: ChainConfig(
          derivationPath: "m/44'/0'/0'/0/0",
          addressPrefix: 0,
          supportsEIP1559: false,
          supportsTokenTransfers: false,
          supportsNFT: false,
          blockTime: BlockTimeConfig(targetTimeSeconds: 600, confirmationBlocks: 6),
          gasConfig: GasConfig(gasToken: 'BTC', decimals: 8, minGasPrice: 1, maxGasPrice: 1000, gasLimit: 250),
        ),
      ),
    ];
  }
}
