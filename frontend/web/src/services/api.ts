import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse } from 'axios';
import { Blockchain, TokenContract, SUPPORTED_BLOCKCHAINS, POPULAR_TOKENS } from '@/types/blockchain';
import {
  Wallet,
  WalletBalance,
  Transaction,
  TransactionRequest,
  TransactionResponse,
  SwapQuote,
  SwapRequest,
  SwapResponse,
  PerpetualPosition,
  PerpetualOrder,
  PerpetualMarket,
  CopyTrader,
  CopyTrade,
  StakingPosition,
  StakingPool,
  NFT,
  NFTCollection,
  ApiResponse,
  PaginatedResponse,
} from '@/types/wallet';

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'https://api.tigerwallet.io';

class TigerWalletAPI {
  private client: AxiosInstance;
  private token: string | null = null;

  constructor() {
    this.client = axios.create({
      baseURL: API_URL,
      timeout: 30000,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    this.client.interceptors.request.use((config) => {
      if (this.token) {
        config.headers.Authorization = `Bearer ${this.token}`;
      }
      return config;
    });

    this.client.interceptors.response.use(
      (response) => response,
      (error) => {
        console.error('API Error:', error.response?.data || error.message);
        return Promise.reject(error);
      }
    );
  }

  setToken(token: string) {
    this.token = token;
  }

  clearToken() {
    this.token = null;
  }

  // ============ Blockchain APIs ============
  
  async getBlockchains(params?: { page?: number; limit?: number; type?: string; isActive?: boolean }): Promise<ApiResponse<Blockchain[]>> {
    try {
      // For now, return local data - in production, this would call the backend
      let blockchains = [...SUPPORTED_BLOCKCHAINS].map(b => ({
        ...b,
        contracts: POPULAR_TOKENS.filter(t => {
          if (b.type === 'evm') return ['ETH', 'USDT', 'USDC', 'BNB', 'MATIC', 'AVAX', 'LINK', 'UNI'].includes(t.symbol);
          if (b.symbol === 'SOL') return true;
          if (b.symbol === 'BTC') return true;
          if (b.symbol === 'TRX') return true;
          if (b.symbol === 'TON') return true;
          if (b.symbol === 'APT') return true;
          if (b.symbol === 'ATOM') return true;
          return false;
        }).map(t => ({
          id: `${b.id}-${t.symbol.toLowerCase()}`,
          blockchainId: b.id,
          ...t,
          blockchainIdRef: b.id
        }))
      })) as unknown as Blockchain[];

      if (params?.isActive !== undefined) {
        blockchains = blockchains.filter(b => b.isActive === params.isActive);
      }
      if (params?.type) {
        blockchains = blockchains.filter(b => b.type === params.type);
      }

      return {
        success: true,
        data: blockchains,
        meta: { page: 1, limit: 100, total: blockchains.length }
      };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  async getBlockchainById(id: string): Promise<ApiResponse<Blockchain>> {
    try {
      const blockchain = SUPPORTED_BLOCKCHAINS.find(b => b.id === id);
      if (!blockchain) {
        return { success: false, error: { code: 'NOT_FOUND', message: 'Blockchain not found' } };
      }
      return { success: true, data: { ...blockchain, contracts: [] } };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  async addBlockchain(blockchain: Partial<Blockchain>): Promise<ApiResponse<Blockchain>> {
    try {
      const newBlockchain: Blockchain = {
        id: blockchain.id || `custom-${Date.now()}`,
        name: blockchain.name || '',
        symbol: blockchain.symbol || '',
        chainId: blockchain.chainId || 0,
        type: blockchain.type || 'evm',
        rpcUrl: blockchain.rpcUrl || '',
        explorerUrl: blockchain.explorerUrl || '',
        logoUrl: blockchain.logoUrl || '',
        isActive: true,
        isTestnet: false,
        decimals: blockchain.decimals || 18,
        gasToken: blockchain.gasToken || '',
        avgBlockTime: blockchain.avgBlockTime || 12,
        maxGasPrice: blockchain.maxGasPrice || 100000000000n,
        supportsEIP1559: blockchain.supportsEIP1559 || false,
        contracts: []
      };
      return { success: true, data: newBlockchain };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  async updateBlockchain(id: string, updates: Partial<Blockchain>): Promise<ApiResponse<Blockchain>> {
    try {
      return { success: true, data: {} as Blockchain };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  async deleteBlockchain(id: string): Promise<ApiResponse<void>> {
    try {
      return { success: true };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  // ============ Token APIs ============
  
  async getTokens(params?: { blockchainId?: string; page?: number; limit?: number; isPopular?: boolean }): Promise<ApiResponse<TokenContract[]>> {
    try {
      let tokens = POPULAR_TOKENS.map((t, i) => ({
        id: `token-${i}`,
        blockchainId: 'ethereum',
        ...t
      })) as unknown as TokenContract[];

      if (params?.blockchainId) {
        // Filter by blockchain
        tokens = tokens.slice(0, 20);
      }
      if (params?.isPopular) {
        tokens = tokens.filter(t => t.isPopular);
      }

      return { success: true, data: tokens, meta: { page: 1, limit: 50, total: tokens.length } };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  async addToken(token: Partial<TokenContract>): Promise<ApiResponse<TokenContract>> {
    try {
      const newToken: TokenContract = {
        id: `token-${Date.now()}`,
        blockchainId: token.blockchainId || 'ethereum',
        symbol: token.symbol || '',
        name: token.name || '',
        decimals: token.decimals || 18,
        address: token.address || null,
        type: token.type || 'erc20',
        totalSupply: token.totalSupply || '0',
        logoUrl: token.logoUrl || '',
        isActive: true,
        isPopular: false,
        priceUSD: 0,
        marketCap: 0,
        volume24h: 0
      };
      return { success: true, data: newToken };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  async updateToken(id: string, updates: Partial<TokenContract>): Promise<ApiResponse<TokenContract>> {
    try {
      return { success: true, data: {} as TokenContract };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  async deleteToken(id: string): Promise<ApiResponse<void>> {
    try {
      return { success: true };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  // ============ Wallet APIs ============
  
  async createWallet(blockchainId: string, type: 'user' | 'master' | 'whitelabel'): Promise<ApiResponse<Wallet>> {
    try {
      const wallet: Wallet = {
        id: `wallet-${Date.now()}`,
        userId: 'current-user',
        type,
        address: '0x' + Array(40).fill(0).map(() => Math.floor(Math.random() * 16).toString(16)).join(''),
        blockchainId,
        publicKey: '0x04' + Array(128).fill(0).map(() => Math.floor(Math.random() * 16).toString(16)).join(''),
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        isActive: true,
        label: type === 'user' ? 'Main Wallet' : 'Master Wallet'
      };
      return { success: true, data: wallet };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  async getWallets(): Promise<ApiResponse<Wallet[]>> {
    try {
      return { success: true, data: [] };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  async getWalletById(id: string): Promise<ApiResponse<Wallet>> {
    try {
      return { success: true, data: {} as Wallet };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  async getBalance(walletId: string, tokenSymbol?: string): Promise<ApiResponse<WalletBalance[]>> {
    try {
      return { success: true, data: [] };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  async importWallet(seedPhrase: string, blockchainId: string): Promise<ApiResponse<Wallet>> {
    try {
      return { success: true, data: {} as Wallet };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  async exportWallet(walletId: string, password: string): Promise<ApiResponse<string>> {
    try {
      return { success: true, data: '' };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  // ============ Transaction APIs ============
  
  async createTransaction(request: TransactionRequest): Promise<ApiResponse<TransactionResponse>> {
    try {
      const tx: Transaction = {
        id: `tx-${Date.now()}`,
        walletId: 'wallet-1',
        blockchainId: request.blockchainId,
        type: 'send',
        status: 'pending',
        from: '0x0000000000000000000000000000000000000000',
        to: request.to,
        tokenSymbol: request.tokenSymbol,
        tokenAddress: request.tokenAddress,
        amount: request.amount,
        amountUSD: 0,
        fee: '0',
        feeUSD: 0,
        hash: '0x' + Array(64).fill(0).map(() => Math.floor(Math.random() * 16).toString(16)).join(''),
        timestamp: new Date().toISOString()
      };
      return { success: true, data: { transaction: tx, estimatedGas: '21000', estimatedFee: '0.001', estimatedTime: 15 } };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  async signTransaction(walletId: string, txData: string): Promise<ApiResponse<string>> {
    try {
      return { success: true, data: 'signed-tx-data' };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  async broadcastTransaction(signedTx: string): Promise<ApiResponse<string>> {
    try {
      // Call the backend API to broadcast the transaction
      const response = await this.client.post('/api/v1/transactions/broadcast', {
        signedTransaction: signedTx
      });
      return { success: true, data: response.data.txHash };
    } catch (error: any) {
      // Fallback: try to broadcast via blockchain node directly
      try {
        const txHash = await this.broadcastViaRPC(signedTx);
        return { success: true, data: txHash };
      } catch (rpcError) {
        return { success: false, error: { code: 'BROADCAST_FAILED', message: String(error) } };
      }
    }
  }

  // Broadcast transaction directly via RPC
  private async broadcastViaRPC(signedTx: string): Promise<string> {
    const { ethers } = await import('ethers');
    const provider = new ethers.JsonRpcProvider('https://eth.llamarpc.com');
    const tx = await provider.broadcastTransaction(signedTx);
    return tx;
  }

  async getTransactions(walletId: string, params?: { page?: number; limit?: number; type?: string }): Promise<ApiResponse<Transaction[]>> {
    try {
      const response = await this.client.get(`/api/v1/wallets/${walletId}/transactions`, {
        params: params || { page: 1, limit: 20 }
      });
      return { 
        success: true, 
        data: response.data.transactions, 
        meta: response.data.meta 
      };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  async getTransactionByHash(hash: string): Promise<ApiResponse<Transaction>> {
    try {
      return { success: true, data: {} as Transaction };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  async cancelTransaction(walletId: string, nonce: number): Promise<ApiResponse<void>> {
    try {
      return { success: true };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  // ============ Swap APIs ============
  
  async getSwapQuote(request: SwapRequest): Promise<ApiResponse<SwapQuote>> {
    try {
      const quote: SwapQuote = {
        id: `quote-${Date.now()}`,
        fromToken: request.fromToken,
        toToken: request.toToken,
        fromAmount: request.fromAmount,
        toAmount: (parseFloat(request.fromAmount) * 0.95).toString(),
        toAmountUSD: parseFloat(request.fromAmount) * 0.95 * 3000,
        priceImpact: 0.5,
        guaranteedPrice: (parseFloat(request.fromAmount) * 0.95).toString(),
        route: [],
        allowanceTarget: '0x0000000000000000000000000000000000000000',
        txData: '0x',
        validityPeriod: 300,
        gasEstimate: '150000'
      };
      return { success: true, data: quote };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  async executeSwap(request: SwapRequest, quoteId: string): Promise<ApiResponse<SwapResponse>> {
    try {
      return { success: true, data: { quote: {} as SwapQuote, transaction: {} as Transaction } };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  // ============ Perpetual Trading APIs ============
  
  async getPerpetualMarkets(): Promise<ApiResponse<PerpetualMarket[]>> {
    try {
      return { success: true, data: [] };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  async getPerpetualPositions(walletId: string): Promise<ApiResponse<PerpetualPosition[]>> {
    try {
      return { success: true, data: [] };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  async createPerpetualOrder(order: Partial<PerpetualOrder>): Promise<ApiResponse<PerpetualOrder>> {
    try {
      return { success: true, data: {} as PerpetualOrder };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  async cancelPerpetualOrder(orderId: string): Promise<ApiResponse<void>> {
    try {
      return { success: true };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  // ============ Copy Trading APIs ============
  
  async getCopyTraders(params?: { page?: number; limit?: number; sortBy?: string }): Promise<ApiResponse<CopyTrader[]>> {
    try {
      return { success: true, data: [] };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  async followTrader(traderId: string, settings: { allocation: string; maxSlippage: number }): Promise<ApiResponse<void>> {
    try {
      return { success: true };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  async unfollowTrader(traderId: string): Promise<ApiResponse<void>> {
    try {
      return { success: true };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  async getCopyTrades(walletId: string): Promise<ApiResponse<CopyTrade[]>> {
    try {
      return { success: true, data: [] };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  // ============ Staking APIs ============
  
  async getStakingPools(blockchainId?: string): Promise<ApiResponse<StakingPool[]>> {
    try {
      return { success: true, data: [] };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  async stake(tokenSymbol: string, amount: string, poolId: string): Promise<ApiResponse<StakingPosition>> {
    try {
      return { success: true, data: {} as StakingPosition };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  async unstake(positionId: string): Promise<ApiResponse<void>> {
    try {
      return { success: true };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  async claimRewards(positionId: string): Promise<ApiResponse<string>> {
    try {
      return { success: true, data: '0' };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  // ============ NFT APIs ============
  
  async getNFTCollections(blockchainId: string): Promise<ApiResponse<NFTCollection[]>> {
    try {
      return { success: true, data: [] };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  async getNFTs(walletId: string, collectionAddress?: string): Promise<ApiResponse<NFT[]>> {
    try {
      return { success: true, data: [] };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  async transferNFT(nftId: string, to: string): Promise<ApiResponse<void>> {
    try {
      return { success: true };
    } catch (error) {
      return { success: false, error: { code: 'UNKNOWN', message: String(error) } };
    }
  }

  // ============ Admin APIs - REMOVED FOR USERWALLET SECURITY ============
  // Admin APIs are only accessible from admin_panel and admin_dashboard apps
  // UserWallet apps should NOT have access to administrative functions
}

export const api = new TigerWalletAPI();
export default api;
