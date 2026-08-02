/**
 * TigerWallet - Complete Wallet Service
 * Production-ready wallet operations with all blockchain support
 */

import { walletApi, transactionApi, chainApi, swapApi } from './api';

// ============================================================================
// Types
// ============================================================================

export interface WalletCreationRequest {
  name: string;
  chain: string;
  password?: string;
}

export interface WalletImportRequest {
  seedPhrase: string;
  chain: string;
  name: string;
  password?: string;
}

export interface SendTransactionRequest {
  walletId: string;
  to: string;
  amount: string;
  tokenAddress?: string;
  gasPrice?: string;
}

export interface SwapRequest {
  walletId: string;
  fromToken: string;
  toToken: string;
  fromAmount: string;
  slippage?: number;
}

export interface StakeRequest {
  walletId: string;
  chain: string;
  validator: string;
  amount: string;
}

export interface BridgeRequest {
  walletId: string;
  fromChain: string;
  toChain: string;
  token: string;
  amount: string;
}

// ============================================================================
// Wallet Service
// ============================================================================

export class WalletService {
  /**
   * Create a new wallet with 24-word seed phrase
   */
  static async createWallet(request: WalletCreationRequest) {
    try {
      const wallet = await walletApi.createWallet(request.chain, request.name);
      return {
        success: true,
        wallet,
        seedPhrase: wallet.seedPhrase // Only returned once on creation
      };
    } catch (error) {
      console.error('Failed to create wallet:', error);
      throw error;
    }
  }

  /**
   * Import existing wallet from seed phrase
   */
  static async importWallet(request: WalletImportRequest) {
    try {
      const wallet = await walletApi.importWallet(
        request.seedPhrase,
        request.chain,
        request.name
      );
      return { success: true, wallet };
    } catch (error) {
      console.error('Failed to import wallet:', error);
      throw error;
    }
  }

  /**
   * Get all wallets for current user
   */
  static async getWallets() {
    try {
      const wallets = await walletApi.getWallets();
      return wallets;
    } catch (error) {
      console.error('Failed to get wallets:', error);
      throw error;
    }
  }

  /**
   * Get wallet by ID with full balance details
   */
  static async getWalletDetails(walletId: string) {
    try {
      const [wallet, balance, transactions] = await Promise.all([
        walletApi.getWallet(walletId),
        walletApi.getBalance(walletId),
        walletApi.getTransactions(walletId)
      ]);

      return {
        ...wallet,
        balance,
        recentTransactions: transactions.slice(0, 10)
      };
    } catch (error) {
      console.error('Failed to get wallet details:', error);
      throw error;
    }
  }

  /**
   * Get all supported chains
   */
  static async getSupportedChains() {
    try {
      return await chainApi.getChains();
    } catch (error) {
      console.error('Failed to get chains:', error);
      // Return default chains
      return this.getDefaultChains();
    }
  }

  /**
   * Add custom chain (for master wallet)
   */
  static async addCustomChain(chainData: any) {
    try {
      return await chainApi.addChain(chainData);
    } catch (error) {
      console.error('Failed to add chain:', error);
      throw error;
    }
  }

  /**
   * Get default chains
   */
  static getDefaultChains() {
    return [
      { id: 'ethereum', name: 'Ethereum', symbol: 'ETH', chainId: 1, isEVM: true },
      { id: 'polygon', name: 'Polygon', symbol: 'MATIC', chainId: 137, isEVM: true },
      { id: 'bsc', name: 'BNB Chain', symbol: 'BNB', chainId: 56, isEVM: true },
      { id: 'arbitrum', name: 'Arbitrum', symbol: 'ETH', chainId: 42161, isEVM: true },
      { id: 'optimism', name: 'Optimism', symbol: 'ETH', chainId: 10, isEVM: true },
      { id: 'avalanche', name: 'Avalanche', symbol: 'AVAX', chainId: 43114, isEVM: true },
      { id: 'base', name: 'Base', symbol: 'ETH', chainId: 8453, isEVM: true },
      { id: 'solana', name: 'Solana', symbol: 'SOL', chainId: 0, isEVM: false },
      { id: 'ton', name: 'TON', symbol: 'TON', chainId: 0, isEVM: false },
      { id: 'tron', name: 'TRON', symbol: 'TRX', chainId: 0, isEVM: false },
      { id: 'bitcoin', name: 'Bitcoin', symbol: 'BTC', chainId: 0, isEVM: false },
    ];
  }
}

// ============================================================================
// Transaction Service
// ============================================================================

export class TransactionService {
  /**
   * Send transaction
   */
  static async send(request: SendTransactionRequest) {
    try {
      // First estimate gas
      const gasEstimate = await transactionApi.estimateGas(
        '', // from will be derived from wallet
        request.to,
        request.amount,
        request.tokenAddress
      );

      // Execute transaction
      const result = await transactionApi.send(
        request.walletId,
        request.to,
        request.amount,
        request.tokenAddress || '0x0000000000000000000000000000000000000000',
        request.gasPrice || gasEstimate.gasPrice
      );

      return {
        success: true,
        hash: result.hash,
        nonce: result.nonce,
        gasEstimate
      };
    } catch (error) {
      console.error('Failed to send transaction:', error);
      throw error;
    }
  }

  /**
   * Get transaction status
   */
  static async getStatus(txHash: string) {
    try {
      return await transactionApi.getStatus(txHash);
    } catch (error) {
      console.error('Failed to get transaction status:', error);
      throw error;
    }
  }

  /**
   * Cancel pending transaction
   */
  static async cancel(walletId: string, nonce: number) {
    try {
      return await transactionApi.cancel(walletId, nonce);
    } catch (error) {
      console.error('Failed to cancel transaction:', error);
      throw error;
    }
  }

  /**
   * Get transaction history
   */
  static async getHistory(walletId: string, page = 1, limit = 50) {
    try {
      return await walletApi.getTransactions(walletId, page, limit);
    } catch (error) {
      console.error('Failed to get transaction history:', error);
      throw error;
    }
  }

  /**
   * Estimate gas for transaction
   */
  static async estimateGas(from: string, to: string, amount: string, tokenAddress?: string) {
    try {
      return await transactionApi.estimateGas(from, to, amount, tokenAddress);
    } catch (error) {
      console.error('Failed to estimate gas:', error);
      // Return default estimates
      return {
        gasLimit: '21000',
        gasPrice: '1000000000',
        totalFee: '0.000021'
      };
    }
  }
}

// ============================================================================
// Swap Service
// ============================================================================

export class SwapService {
  /**
   * Get swap quote from DEX aggregator
   */
  static async getQuote(fromToken: string, toToken: string, amount: string, slippage = 0.5) {
    try {
      return await swapApi.getQuote(fromToken, toToken, amount, slippage);
    } catch (error) {
      console.error('Failed to get swap quote:', error);
      throw error;
    }
  }

  /**
   * Execute swap
   */
  static async execute(request: SwapRequest) {
    try {
      // Get quote first
      const quote = await this.getQuote(
        request.fromToken,
        request.toToken,
        request.fromAmount,
        request.slippage
      );

      // Execute swap
      const result = await swapApi.execute(
        request.walletId,
        request.fromToken,
        request.toToken,
        request.fromAmount,
        quote.toAmount,
        quote.route
      );

      return {
        success: true,
        hash: result.hash,
        quote
      };
    } catch (error) {
      console.error('Failed to execute swap:', error);
      throw error;
    }
  }

  /**
   * Get supported tokens for swap
   */
  static async getSupportedTokens(chain: string) {
    try {
      return await swapApi.getTokens(chain);
    } catch (error) {
      console.error('Failed to get tokens:', error);
      return this.getDefaultTokens();
    }
  }

  /**
   * Get default tokens
   */
  static getDefaultTokens() {
    return [
      { symbol: 'ETH', name: 'Ethereum', decimals: 18, address: '', chain: 'ethereum' },
      { symbol: 'USDT', name: 'Tether USD', decimals: 6, address: '0xdAC17F958D2ee523a2206206994597C13D831ec7', chain: 'ethereum' },
      { symbol: 'USDC', name: 'USD Coin', decimals: 6, address: '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48', chain: 'ethereum' },
      { symbol: 'BNB', name: 'BNB', decimals: 18, address: '', chain: 'bsc' },
      { symbol: 'MATIC', name: 'Polygon', decimals: 18, address: '0x7D1AfA7B718fb893dB30A3aBc0Cfc608AaCfeBB0', chain: 'polygon' },
      { symbol: 'AVAX', name: 'Avalanche', decimals: 18, address: '0xB31f66AA3C1e785363F0875A1B74E27b85FD66c7', chain: 'avalanche' },
      { symbol: 'SOL', name: 'Solana', decimals: 9, address: '', chain: 'solana' },
      { symbol: 'BTC', name: 'Bitcoin', decimals: 8, address: '', chain: 'bitcoin' },
    ];
  }
}

// ============================================================================
// Staking Service
// ============================================================================

export class StakingService {
  /**
   * Get staking positions
   */
  static async getPositions(walletId: string) {
    try {
      const response = await fetch(`/api/v1/staking/positions?wallet_id=${walletId}`);
      return await response.json();
    } catch (error) {
      console.error('Failed to get staking positions:', error);
      return [];
    }
  }

  /**
   * Stake tokens
   */
  static async stake(request: StakeRequest) {
    try {
      const response = await fetch('/api/v1/staking/stake', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(request)
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to stake:', error);
      throw error;
    }
  }

  /**
   * Unstake tokens
   */
  static async unstake(positionId: string) {
    try {
      const response = await fetch(`/api/v1/staking/unstake`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ positionId })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to unstake:', error);
      throw error;
    }
  }

  /**
   * Get validators
   */
  static async getValidators(chain: string) {
    try {
      const response = await fetch(`/api/v1/staking/validators?chain=${chain}`);
      return await response.json();
    } catch (error) {
      console.error('Failed to get validators:', error);
      return [];
    }
  }

  /**
   * Get staking rewards
   */
  static async getRewards(walletId: string) {
    try {
      const response = await fetch(`/api/v1/staking/rewards?wallet_id=${walletId}`);
      return await response.json();
    } catch (error) {
      console.error('Failed to get rewards:', error);
      return [];
    }
  }
}

// ============================================================================
// Bridge Service
// ============================================================================

export class BridgeService {
  /**
   * Get bridge quote
   */
  static async getQuote(fromChain: string, toChain: string, token: string, amount: string) {
    try {
      const response = await fetch('/api/v1/bridge/quote', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ fromChain, toChain, token, amount })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to get bridge quote:', error);
      throw error;
    }
  }

  /**
   * Execute bridge
   */
  static async execute(request: BridgeRequest) {
    try {
      const response = await fetch('/api/v1/bridge/execute', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(request)
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to execute bridge:', error);
      throw error;
    }
  }

  /**
   * Get bridge status
   */
  static async getStatus(bridgeTxId: string) {
    try {
      const response = await fetch(`/api/v1/bridge/status/${bridgeTxId}`);
      return await response.json();
    } catch (error) {
      console.error('Failed to get bridge status:', error);
      throw error;
    }
  }

  /**
   * Get supported bridge routes
   */
  static async getSupportedRoutes() {
    try {
      const response = await fetch('/api/v1/bridge/routes');
      return await response.json();
    } catch (error) {
      console.error('Failed to get bridge routes:', error);
      return this.getDefaultRoutes();
    }
  }

  /**
   * Get default routes
   */
  static getDefaultRoutes() {
    return [
      { from: 'ethereum', to: 'polygon', tokens: ['ETH', 'USDT', 'USDC'] },
      { from: 'ethereum', to: 'arbitrum', tokens: ['ETH', 'USDT', 'USDC'] },
      { from: 'ethereum', to: 'optimism', tokens: ['ETH', 'USDT', 'USDC'] },
      { from: 'ethereum', to: 'avalanche', tokens: ['ETH', 'USDT', 'USDC'] },
      { from: 'ethereum', to: 'bsc', tokens: ['ETH', 'BNB', 'USDT'] },
      { from: 'polygon', to: 'ethereum', tokens: ['MATIC', 'USDT', 'USDC'] },
      { from: 'bsc', to: 'ethereum', tokens: ['BNB', 'USDT', 'USDC'] },
    ];
  }
}

// ============================================================================
// NFT Service
// ============================================================================

export class NFTService {
  /**
   * Get NFTs for wallet
   */
  static async getNFTs(walletAddress: string, chain: string) {
    try {
      const response = await fetch(`/api/v1/nfts?address=${walletAddress}&chain=${chain}`);
      return await response.json();
    } catch (error) {
      console.error('Failed to get NFTs:', error);
      return [];
    }
  }

  /**
   * Get NFT collection
   */
  static async getCollection(collectionAddress: string, chain: string) {
    try {
      const response = await fetch(`/api/v1/nfts/collection/${collectionAddress}?chain=${chain}`);
      return await response.json();
    } catch (error) {
      console.error('Failed to get collection:', error);
      throw error;
    }
  }

  /**
   * Transfer NFT
   */
  static async transfer(walletId: string, toAddress: string, tokenId: string, collectionAddress: string) {
    try {
      const response = await fetch('/api/v1/nfts/transfer', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ walletId, toAddress, tokenId, collectionAddress })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to transfer NFT:', error);
      throw error;
    }
  }
}

// ============================================================================
// DApp Browser Service
// ============================================================================

export class DAppService {
  /**
   * Connect to DApp
   */
  static async connect(walletId: string, dappUrl: string) {
    try {
      return await dappApi.connect(walletId, dappUrl);
    } catch (error) {
      console.error('Failed to connect to DApp:', error);
      throw error;
    }
  }

  /**
   * Sign transaction request
   */
  static async signRequest(sessionId: string, request: any) {
    try {
      return await dappApi.signRequest(sessionId, request);
    } catch (error) {
      console.error('Failed to sign request:', error);
      throw error;
    }
  }

  /**
   * Get recommended DApps
   */
  static async getDApps(category?: string) {
    try {
      return await dappApi.getDApps(category);
    } catch (error) {
      console.error('Failed to get DApps:', error);
      return this.getDefaultDApps();
    }
  }

  /**
   * Get default DApps
   */
  static getDefaultDApps() {
    return [
      { id: '1', name: 'Uniswap', url: 'https://app.uniswap.org', category: 'DeFi', description: 'DEX Aggregator' },
      { id: '2', name: 'OpenSea', url: 'https://opensea.io', category: 'NFT', description: 'NFT Marketplace' },
      { id: '3', name: 'Aave', url: 'https://app.aave.com', category: 'DeFi', description: 'Lending Protocol' },
      { id: '4', name: 'Compound', url: 'https://app.compound.finance', category: 'DeFi', description: 'Lending Protocol' },
      { id: '5', name: 'PancakeSwap', url: 'https://pancakeswap.finance', category: 'DeFi', description: 'DEX' },
    ];
  }
}

// Export all services
export default {
  WalletService,
  TransactionService,
  SwapService,
  StakingService,
  BridgeService,
  NFTService,
  DAppService
};
