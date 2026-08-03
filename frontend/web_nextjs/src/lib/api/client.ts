/**
 * TigerWallet API Client Library
 * 
 * Complete TypeScript API client for connecting frontend to all backend services.
 * Provides type-safe access to all TigerWallet services.
 */

import axios, { AxiosInstance } from 'axios';

// ============================================================================
// Types
// ============================================================================

export interface APIResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
}

export interface Wallet {
  id: string;
  address: string;
  chainType: string;
  chainId: number;
  type: 'user' | 'master' | 'white_label';
  name: string;
  createdAt: number;
}

export interface WalletBalance {
  available: string;
  total: string;
  usdValue: number;
}

export interface Transaction {
  id: string;
  hash: string;
  from: string;
  to: string;
  value: string;
  status: 'pending' | 'confirmed' | 'failed';
  timestamp: number;
}

export interface StakingPool {
  id: string;
  name: string;
  chainId: number;
  apy: string;
  totalStaked: string;
  status: 'active' | 'paused';
}

export interface EarnProduct {
  id: string;
  name: string;
  productType: 'fixed' | 'flexible';
  apy: string;
  minDeposit: string;
  status: 'active' | 'paused';
}

export interface NFTCollection {
  id: string;
  name: string;
  contractAddress: string;
  floorPrice: string;
  totalSupply: number;
}

export interface PerpetualMarket {
  id: string;
  pair: string;
  markPrice: string;
  maxLeverage: string;
  status: 'active' | 'paused';
}

// ============================================================================
// API Client
// ============================================================================

class TigerWalletAPI {
  private client: AxiosInstance;
  private static instance: TigerWalletAPI;

  private constructor(baseURL: string = '/api') {
    this.client = axios.create({
      baseURL,
      timeout: 30000,
      headers: { 'Content-Type': 'application/json' },
    });

    this.client.interceptors.request.use((config) => {
      const token = localStorage.getItem('auth_token');
      if (token) config.headers.Authorization = `Bearer ${token}`;
      return config;
    });

    this.client.interceptors.response.use(
      (response) => response,
      (error) => {
        if (error.response?.status === 401) {
          localStorage.removeItem('auth_token');
        }
        return Promise.reject(error);
      }
    );
  }

  public static getInstance(): TigerWalletAPI {
    if (!TigerWalletAPI.instance) {
      TigerWalletAPI.instance = new TigerWalletAPI();
    }
    return TigerWalletAPI.instance;
  }

  setBaseURL(url: string): void {
    this.client.defaults.baseURL = url;
  }

  setAuthToken(token: string): void {
    localStorage.setItem('auth_token', token);
  }

  // Wallet
  async createWallet(name: string, chainIds: number[]): Promise<APIResponse<Wallet>> {
    return (await this.client.post('/wallet/create', { name, chainIds })).data;
  }

  async importWallet(mnemonic: string, name: string): Promise<APIResponse<Wallet>> {
    return (await this.client.post('/wallet/import', { mnemonic, name })).data;
  }

  async getWallets(): Promise<APIResponse<Wallet[]>> {
    return (await this.client.get('/wallet/list')).data;
  }

  async getWalletBalance(walletId: string): Promise<APIResponse<Record<string, WalletBalance>>> {
    return (await this.client.get(`/wallet/${walletId}/balance`)).data;
  }

  async sendTransaction(walletId: string, to: string, value: string, chainId: number): Promise<APIResponse<Transaction>> {
    return (await this.client.post('/wallet/send', { walletId, to, value, chainId })).data;
  }

  // Staking
  async getStakingPools(params?: { chainId?: number }): Promise<APIResponse<StakingPool[]>> {
    return (await this.client.get('/staking/pools', { params })).data;
  }

  async stake(poolId: string, amount: string): Promise<APIResponse<any>> {
    return (await this.client.post('/staking/stake', { poolId, amount })).data;
  }

  async unstake(stakeId: string): Promise<APIResponse<any>> {
    return (await this.client.post('/staking/unstake', { stakeId })).data;
  }

  async claimStakingRewards(stakeId: string): Promise<APIResponse<string>> {
    return (await this.client.post('/staking/claim', { stakeId })).data;
  }

  // Earn
  async getEarnProducts(params?: { chainId?: number }): Promise<APIResponse<EarnProduct[]>> {
    return (await this.client.get('/earn/products', { params })).data;
  }

  async deposit(productId: string, amount: string): Promise<APIResponse<any>> {
    return (await this.client.post('/earn/deposit', { productId, amount })).data;
  }

  async withdraw(depositId: string): Promise<APIResponse<string>> {
    return (await this.client.post('/earn/withdraw', { depositId })).data;
  }

  // NFT
  async getNFTCollections(): Promise<APIResponse<NFTCollection[]>> {
    return (await this.client.get('/nft/collections')).data;
  }

  async getNFTItems(collectionId: string): Promise<APIResponse<any[]>> {
    return (await this.client.get(`/nft/collections/${collectionId}/items`)).data;
  }

  async createListing(itemId: string, price: string): Promise<APIResponse<any>> {
    return (await this.client.post('/nft/list', { itemId, price })).data;
  }

  async buyNFT(listingId: string): Promise<APIResponse<any>> {
    return (await this.client.post('/nft/buy', { listingId })).data;
  }

  // Perpetual
  async getPerpetualMarkets(): Promise<APIResponse<PerpetualMarket[]>> {
    return (await this.client.get('/perpetual/markets')).data;
  }

  async openPosition(marketId: string, side: string, size: string, leverage: string): Promise<APIResponse<any>> {
    return (await this.client.post('/perpetual/open', { marketId, side, size, leverage }));
  }

  async closePosition(positionId: string): Promise<APIResponse<any>> {
    return (await this.client.post('/perpetual/close', { positionId })).data;
  }

  // Copy Trading
  async getTraders(): Promise<APIResponse<any[]>> {
    return (await this.client.get('/copy-trading/traders')).data;
  }

  async startCopying(traderId: string, copyRatio: string): Promise<APIResponse<any>> {
    return (await this.client.post('/copy-trading/start', { traderId, copyRatio })).data;
  }

  // Token Deployer
  async createTokenDeployment(config: any): Promise<APIResponse<any>> {
    return (await this.client.post('/token/create', config)).data;
  }

  // Multisig
  async createMultisigWallet(name: string, owners: string[], requiredSigs: number): Promise<APIResponse<any>> {
    return (await this.client.post('/multisig/create', { name, owners, requiredSigs })).data;
  }

  async signTransaction(txId: string, signature: string): Promise<APIResponse<any>> {
    return (await this.client.post('/multisig/sign', { txId, signature })).data;
  }

  // Airdrop
  async getAirdropCampaigns(): Promise<APIResponse<any[]>> {
    return (await this.client.get('/airdrop/campaigns')).data;
  }

  async claimAirdrop(campaignId: string): Promise<APIResponse<any>> {
    return (await this.client.post('/airdrop/claim', { campaignId })).data;
  }

  // Coupon
  async validateCoupon(code: string): Promise<APIResponse<any>> {
    return (await this.client.get('/coupon/validate', { params: { code } })).data;
  }

  // Red Packets
  async createRedPacket(totalAmount: string, quantity: number, claimType: string): Promise<APIResponse<any>> {
    return (await this.client.post('/red-packets/create', { totalAmount, quantity, claimType })).data;
  }

  async claimRedPacket(packetId: string): Promise<APIResponse<any>> {
    return (await this.client.post('/red-packets/claim', { packetId })).data;
  }

  // Auth
  async login(email: string, password: string): Promise<APIResponse<{ token: string }>> {
    return (await this.client.post('/auth/login', { email, password })).data;
  }

  async register(email: string, password: string, name: string): Promise<APIResponse<any>> {
    return (await this.client.post('/auth/register', { email, password, name })).data;
  }
}

export const api = TigerWalletAPI.getInstance();
export default api;
