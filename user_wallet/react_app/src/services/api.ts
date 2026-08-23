/**
 * TigerWallet User API Service - Production Ready
 * Complete backend connectivity for all wallet operations
 * All endpoints connected to Go/Rust/C++ backend microservices
 */

import axios, { AxiosInstance, AxiosError } from 'axios';

// ============================================================================
// Types - Complete Wallet System
// ============================================================================

export interface Wallet {
  id: string;
  name: string;
  address: string;
  chain: string;
  balance: string;
  balanceUSD: number;
  tokens: TokenBalance[];
  type: 'master' | 'user' | 'watch';
  createdAt: number;
  lastSynced: number;
}

export interface TokenBalance {
  symbol: string;
  name: string;
  address: string;
  balance: string;
  balanceUSD: number;
  logoUrl: string;
  decimals: number;
  chainId: number;
  isNative: boolean;
  priceChange24h: number;
}

export interface Transaction {
  id: string;
  hash: string;
  from: string;
  to: string;
  amount: string;
  symbol: string;
  status: 'pending' | 'confirmed' | 'failed';
  timestamp: number;
  type: 'send' | 'receive' | 'swap' | 'approve' | 'stake' | 'unstake' | 'bridge' | 'mint' | 'burn';
  fee: string;
  gasUsed?: string;
  gasPrice?: string;
  blockNumber?: number;
  confirmations?: number;
}

export interface SwapQuote {
  fromToken: string;
  toToken: string;
  fromAmount: string;
  toAmount: string;
  priceImpact: number;
  route: string[];
  estimatedGas: string;
  slippage: number;
  priceRoute: PriceRoute[];
}

export interface PriceRoute {
  pool: string;
  fromToken: string;
  toToken: string;
  swapFee: number;
}

export interface Chain {
  id: string;
  name: string;
  symbol: string;
  rpcUrl: string;
  explorerUrl: string;
  chainId: number;
  isEVM: boolean;
  logoUrl: string;
  color: string;
  type: 'evm' | 'solana' | 'ton' | 'aptos' | 'sui' | 'tron' | 'cosmos' | 'bitcoin';
  nativeCurrency: {
    name: string;
    symbol: string;
    decimals: number;
  };
  rpcEndpoints: string[];
  explorerApiUrl: string;
}

export interface Token {
  address: string;
  symbol: string;
  name: string;
  decimals: number;
  logoUrl: string;
  chain: string;
  priceUSD: number;
  volume24h: number;
  marketCap: number;
  priceChange24h: number;
  totalSupply: string;
  isVerified: boolean;
}

export interface DApp {
  id: string;
  name: string;
  url: string;
  logoUrl: string;
  category: string;
  description: string;
  chains: string[];
  lastUsed?: number;
}

export interface User {
  id: string;
  email: string;
  username: string;
  kycStatus: 'none' | 'pending' | 'verified' | 'rejected';
  createdAt: number;
  twoFactorEnabled: boolean;
  referralCode: string;
}

export interface StakingPosition {
  id: string;
  validator: string;
  amount: string;
  reward: string;
  unlockTime: number;
  status: 'active' | 'unlocking' | 'unlocked';
  chain: string;
}

export interface NFTCollection {
  id: string;
  name: string;
  symbol: string;
  address: string;
  chain: string;
  totalSupply: number;
  floorPrice: string;
  imageUrl: string;
}

export interface NFT {
  id: string;
  tokenId: string;
  collectionAddress: string;
  name: string;
  description: string;
  imageUrl: string;
  attributes: NFTAttribute[];
  owner: string;
  price?: string;
  chain: string;
}

export interface NFTAttribute {
  trait_type: string;
  value: string;
}

export interface BridgeTransaction {
  id: string;
  fromChain: string;
  toChain: string;
  fromToken: string;
  toToken: string;
  amount: string;
  status: 'pending' | 'processing' | 'completed' | 'failed';
  depositTxHash?: string;
  receiveTxHash?: string;
  estimatedTime: number;
}

export interface MasterWallet extends Wallet {
  withdrawFee: number;
  swapFee: number;
  transactionFee: number;
  supportedChains: string[];
  autoSettlement: boolean;
}

export interface WhiteLabelClient {
  id: string;
  name: string;
  domain: string;
  branding: {
    logo: string;
    primaryColor: string;
    secondaryColor: string;
  };
  features: string[];
  status: 'active' | 'suspended' | 'pending';
  createdAt: number;
}

// ============================================================================
// API Configuration - Production Backend
// ============================================================================

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8443/api/v1';
const WS_URL = import.meta.env.VITE_WS_URL || '';

// C++ High-Performance Endpoints (Ultra-Low Latency)
const TRADING_ENGINE_URL = import.meta.env.VITE_TRADING_ENGINE_URL || '';
const PRICE_FEED_URL = import.meta.env.VITE_PRICE_FEED_URL || '';

// Rust Security Endpoints
const SECURITY_URL = import.meta.env.VITE_SECURITY_URL || '';

// ============================================================================
// API Client
// ============================================================================

const createApiClient = (): AxiosInstance => {
  const client = axios.create({
    baseURL: API_BASE_URL,
    timeout: 30000,
    headers: {
      'Content-Type': 'application/json',
    },
  });

  // Request interceptor
  client.interceptors.request.use(
    (config) => {
      const token = localStorage.getItem('user_token');
      if (token) {
        config.headers.Authorization = `Bearer ${token}`;
      }
      return config;
    },
    (error) => Promise.reject(error)
  );

  // Response interceptor
  client.interceptors.response.use(
    (response) => response,
    (error: AxiosError) => {
      if (error.response?.status === 401) {
        localStorage.removeItem('user_token');
        window.location.href = '/login';
      }
      return Promise.reject(error);
    }
  );

  return client;
};

const api = createApiClient();

// ============================================================================
// Wallet API
// ============================================================================

export const walletApi = {
  // Get all user wallets
  getWallets: async (): Promise<Wallet[]> => {
    const response = await api.get('/wallets');
    return response.data.wallets;
  },

  // Get wallet by ID
  getWallet: async (walletId: string): Promise<Wallet> => {
    const response = await api.get(`/wallets/${walletId}`);
    return response.data.wallet;
  },

  // Create new wallet
  createWallet: async (chain: string, name: string, password?: string): Promise<Wallet> => {
    // Canonical wallet_api backend: POST /wallets with {label, password, chain_id, entropy_bits}
    // The backend generates a REAL BIP-39 mnemonic (CSPRNG entropy + checksum)
    // and returns it once for backup display. The client never fabricates a mnemonic.
    const chainIdMap: Record<string, number> = {
      ethereum: 1, mainnet: 1,
      bsc: 56, 'binance-smart-chain': 56,
      polygon: 137, matic: 137,
      arbitrum: 42161, arbitrum-one: 42161,
      optimism: 10, base: 8453, avalanche: 43114,
    };
    const chainId = chainIdMap[chain.toLowerCase()] ?? parseInt(String(chain)) || 1;
    const body: Record<string, unknown> = {
      label: name,
      password: password && password.length >= 8 ? password : undefined,
      chain_id: chainId,
      entropy_bits: 256,
    };
    // password is required by the backend (min 8). If not provided, the caller
    // must supply one; otherwise the backend rejects the request.
    if (!body.password) {
      throw new Error('A password (min 8 chars) is required to create a wallet');
    }
    const response = await api.post('/wallets', body);
    const data = response.data;
    return {
      id: data.id,
      name: data.label,
      address: data.address,
      chain,
      balance: '0',
      balanceUSD: 0,
      tokens: [],
      type: 'user',
      createdAt: Date.now(),
      lastSynced: Date.now(),
      seedPhrase: data.mnemonic, // backend-generated, returned once
    } as Wallet & { seedPhrase?: string };
  },

  // Import wallet with seed phrase
  importWallet: async (seedPhrase: string, chain: string, name: string): Promise<Wallet> => {
    const response = await api.post('/wallets/import', { seedPhrase, chain, name });
    return response.data.wallet;
  },

  // Get wallet balance
  getBalance: async (walletId: string): Promise<{ balance: string; tokens: TokenBalance[] }> => {
    const response = await api.get(`/wallets/${walletId}/balance`);
    return response.data;
  },

  // Get transaction history
  getTransactions: async (walletId: string, page = 1, limit = 20): Promise<Transaction[]> => {
    const response = await api.get(`/wallets/${walletId}/transactions`, {
      params: { page, limit },
    });
    return response.data.transactions;
  },
};

// ============================================================================
// Send/Receive API
// ============================================================================

export const transactionApi = {
  // Send transaction
  send: async (
    walletId: string,
    to: string,
    amount: string,
    tokenAddress: string,
    gasPrice?: string
  ): Promise<{ hash: string; nonce: number }> => {
    const response = await api.post(`/wallets/${walletId}/send`, {
      to,
      amount,
      tokenAddress,
      gasPrice,
    });
    return response.data;
  },

  // Get gas estimate
  estimateGas: async (
    from: string,
    to: string,
    amount: string,
    tokenAddress?: string
  ): Promise<{ gasLimit: string; gasPrice: string; totalFee: string }> => {
    const response = await api.post('/transactions/estimate-gas', {
      from,
      to,
      amount,
      tokenAddress,
    });
    return response.data;
  },

  // Get transaction status
  getStatus: async (txHash: string): Promise<Transaction> => {
    const response = await api.get(`/transactions/${txHash}`);
    return response.data.transaction;
  },

  // Cancel pending transaction
  cancel: async (walletId: string, nonce: number): Promise<{ hash: string }> => {
    const response = await api.post(`/wallets/${walletId}/cancel`, { nonce });
    return response.data;
  },
};

// ============================================================================
// Swap API
// ============================================================================

export const swapApi = {
  // Get swap quote
  getQuote: async (
    fromToken: string,
    toToken: string,
    amount: string,
    slippage?: number
  ): Promise<SwapQuote> => {
    const response = await api.get('/swap/quote', {
      params: { fromToken, toToken, amount, slippage: slippage || 0.5 },
    });
    return response.data.quote;
  },

  // Execute swap
  execute: async (
    walletId: string,
    fromToken: string,
    toToken: string,
    fromAmount: string,
    minReceived: string,
    route: string[]
  ): Promise<{ hash: string }> => {
    const response = await api.post('/swap/execute', {
      walletId,
      fromToken,
      toToken,
      fromAmount,
      minReceived,
      route,
    });
    return response.data;
  },

  // Get supported tokens
  getTokens: async (chain: string): Promise<Token[]> => {
    const response = await api.get('/swap/tokens', { params: { chain } });
    return response.data.tokens;
  },
};

// ============================================================================
// Chain API
// ============================================================================

export const chainApi = {
  // Get all supported chains
  getChains: async (): Promise<Chain[]> => {
    const response = await api.get('/chains');
    return response.data.chains;
  },

  // Add custom chain
  addChain: async (chain: Partial<Chain>): Promise<Chain> => {
    const response = await api.post('/chains', chain);
    return response.data.chain;
  },

  // Update chain
  updateChain: async (chainId: string, updates: Partial<Chain>): Promise<Chain> => {
    const response = await api.put(`/chains/${chainId}`, updates);
    return response.data.chain;
  },

  // Delete chain
  deleteChain: async (chainId: string): Promise<void> => {
    await api.delete(`/chains/${chainId}`);
  },

  // Get chain RPC status
  getRpcStatus: async (chainId: string): Promise<{ status: string; latency: number }> => {
    const response = await api.get(`/chains/${chainId}/rpc-status`);
    return response.data;
  },
};

// ============================================================================
// DApp Browser API
// ============================================================================

export const dappApi = {
  // Get recommended DApps
  getDApps: async (category?: string): Promise<DApp[]> => {
    const response = await api.get('/dapps', { params: { category } });
    return response.data.dapps;
  },

  // Connect to DApp
  connect: async (walletId: string, dappUrl: string): Promise<{ sessionId: string }> => {
    const response = await api.post('/dapps/connect', { walletId, dappUrl });
    return response.data;
  },

  // Sign transaction request from DApp
  signRequest: async (sessionId: string, request: any): Promise<{ signature: string }> => {
    const response = await api.post(`/dapps/${sessionId}/sign`, request);
    return response.data;
  },
};

// ============================================================================
// User API
// ============================================================================

export const userApi = {
  // Get current user
  getMe: async (): Promise<User> => {
    const response = await api.get('/users/me');
    return response.data.user;
  },

  // Update profile
  updateProfile: async (updates: Partial<User>): Promise<User> => {
    const response = await api.put('/users/me', updates);
    return response.data.user;
  },

  // Get KYC status
  getKycStatus: async (): Promise<{ status: string; level: number }> => {
    const response = await api.get('/users/me/kyc');
    return response.data;
  },

  // Submit KYC
  submitKyc: async (documents: any): Promise<{ submissionId: string }> => {
    const response = await api.post('/users/me/kyc', documents);
    return response.data;
  },
};

// ============================================================================
// WebSocket Service
// ============================================================================

class WebSocketService {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private listeners: Map<string, Set<(data: any) => void>> = new Map();

  connect(token: string): void {
    if (this.ws?.readyState === WebSocket.OPEN) return;

    this.ws = new WebSocket(`${WS_URL}?token=${token}`);

    this.ws.onopen = () => {
      console.log('WebSocket connected');
      this.reconnectAttempts = 0;
    };

    this.ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        const { type, payload } = data;
        this.listeners.get(type)?.forEach((callback) => callback(payload));
      } catch (error) {
        console.error('WebSocket message parse error:', error);
      }
    };

    this.ws.onclose = () => {
      console.log('WebSocket disconnected');
      this.reconnect();
    };

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };
  }

  private reconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max reconnection attempts reached');
      return;
    }
    this.reconnectAttempts++;
    const token = localStorage.getItem('user_token');
    if (token) {
      setTimeout(() => this.connect(token), 1000 * this.reconnectAttempts);
    }
  }

  subscribe(event: string, callback: (data: any) => void): void {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set());
    }
    this.listeners.get(event)!.add(callback);
  }

  unsubscribe(event: string, callback: (data: any) => void): void {
    this.listeners.get(event)?.delete(callback);
  }

  send(type: string, payload: any): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type, payload }));
    }
  }

  disconnect(): void {
    this.ws?.close();
    this.ws = null;
  }
}

export const wsService = new WebSocketService();

// ============================================================================
// Export
// ============================================================================

export default api;
