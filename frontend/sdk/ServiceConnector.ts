/**
 * TigerWallet Service Connector
 * Ensures all frontend services are connected to backend
 */

import axios, { AxiosInstance, AxiosError, InternalAxiosRequestConfig } from 'axios';
import { 
  WalletService, 
  SwapService, 
  BridgeService, 
  StakingService, 
  NFTService,
  TokenService,
  UserService,
  AdminService 
} from './services';

// ==================== API Configuration ====================

const API_CONFIG = {
  baseURL: process.env.REACT_APP_API_URL || '/api/v1',
  timeout: 30000,
  retries: 3,
  retryDelay: 1000,
};

// ==================== Service Factory ====================

class ServiceConnector {
  private static instance: ServiceConnector;
  private api: AxiosInstance;
  private services: Map<string, any> = new Map();

  private constructor() {
    this.api = this.createAxiosInstance();
    this.initializeServices();
  }

  public static getInstance(): ServiceConnector {
    if (!ServiceConnector.instance) {
      ServiceConnector.instance = new ServiceConnector();
    }
    return ServiceConnector.instance;
  }

  private createAxiosInstance(): AxiosInstance {
    const api = axios.create({
      baseURL: API_CONFIG.baseURL,
      timeout: API_CONFIG.timeout,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Request interceptor for auth
    api.interceptors.request.use(
      (config: InternalAxiosRequestConfig) => {
        const token = localStorage.getItem('auth_token');
        if (token && config.headers) {
          config.headers.Authorization = `Bearer ${token}`;
        }
        
        // Add request timestamp for debugging
        config.headers['X-Request-Time'] = new Date().toISOString();
        
        return config;
      },
      (error) => Promise.reject(error)
    );

    // Response interceptor for error handling
    api.interceptors.response.use(
      (response) => response,
      async (error: AxiosError) => {
        const originalRequest = error.config as InternalAxiosRequestConfig & { _retryCount?: number };
        
        // Handle token expiration
        if (error.response?.status === 401) {
          localStorage.removeItem('auth_token');
          window.location.href = '/login';
          return Promise.reject(error);
        }

        // Retry logic for network errors
        if (this.shouldRetry(error) && originalRequest) {
          originalRequest._retryCount = (originalRequest._retryCount || 0) + 1;
          
          if (originalRequest._retryCount < API_CONFIG.retries) {
            await this.delay(API_CONFIG.retryDelay * originalRequest._retryCount);
            return api(originalRequest);
          }
        }

        return Promise.reject(error);
      }
    );

    return api;
  }

  private shouldRetry(error: AxiosError): boolean {
    if (!error.response) {
      // Network error
      return true;
    }
    
    const status = error.response.status;
    // Retry on rate limiting or server errors
    return status === 429 || (status >= 500 && status < 600);
  }

  private delay(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  // ==================== Initialize All Services ====================

  private initializeServices() {
    // Core Wallet Services
    this.services.set('wallet', new WalletService(this.api));
    this.services.set('swap', new SwapService(this.api));
    this.services.set('bridge', new BridgeService(this.api));
    this.services.set('staking', new StakingService(this.api));
    this.services.set('nft', new NFTService(this.api));
    this.services.set('token', new TokenService(this.api));
    
    // User Services
    this.services.set('user', new UserService(this.api));
    
    // Admin Services
    this.services.set('admin', new AdminService(this.api));
  }

  // ==================== Service Access ====================

  public getWalletService(): WalletService {
    return this.services.get('wallet');
  }

  public getSwapService(): SwapService {
    return this.services.get('swap');
  }

  public getBridgeService(): BridgeService {
    return this.services.get('bridge');
  }

  public getStakingService(): StakingService {
    return this.services.get('staking');
  }

  public getNFTService(): NFTService {
    return this.services.get('nft');
  }

  public getTokenService(): TokenService {
    return this.services.get('token');
  }

  public getUserService(): UserService {
    return this.services.get('user');
  }

  public getAdminService(): AdminService {
    return this.services.get('admin');
  }

  // ==================== Health Check ====================

  public async checkConnection(): Promise<boolean> {
    try {
      const response = await this.api.get('/health');
      return response.status === 200;
    } catch {
      return false;
    }
  }

  public async checkAllServices(): Promise<Record<string, boolean>> {
    const results: Record<string, boolean> = {};
    
    const endpoints = [
      { name: 'wallet', path: '/wallet/status' },
      { name: 'swap', path: '/swap/health' },
      { name: 'bridge', path: '/bridge/health' },
      { name: 'staking', path: '/staking/health' },
      { name: 'nft', path: '/nft/health' },
      { name: 'token', path: '/token/health' },
    ];

    for (const endpoint of endpoints) {
      try {
        await this.api.get(endpoint.path);
        results[endpoint.name] = true;
      } catch {
        results[endpoint.name] = false;
      }
    }

    return results;
  }
}

// ==================== Service Interfaces ====================

export class WalletService {
  constructor(private api: AxiosInstance) {}

  async getBalance(address: string, chainId: number) {
    const response = await this.api.get(`/wallet/${address}/balance`, {
      params: { chainId },
    });
    return response.data;
  }

  async getTransactions(address: string, options?: any) {
    const response = await this.api.get(`/wallet/${address}/transactions`, {
      params: options,
    });
    return response.data;
  }

  async sendTransaction(tx: any) {
    const response = await this.api.post('/wallet/send', tx);
    return response.data;
  }

  async getWalletAddresses(seedPhrase: string, chains: number[]) {
    const response = await this.api.post('/wallet/generate-addresses', {
      seedPhrase,
      chains,
    });
    return response.data;
  }
}

export class SwapService {
  constructor(private api: AxiosInstance) {}

  async getQuote(params: any) {
    const response = await this.api.post('/swap/quote', params);
    return response.data;
  }

  async executeSwap(params: any) {
    const response = await this.api.post('/swap/execute', params);
    return response.data;
  }

  async getRoutes(fromToken: string, toToken: string, amount: string) {
    const response = await this.api.get('/swap/routes', {
      params: { fromToken, toToken, amount },
    });
    return response.data;
  }
}

export class BridgeService {
  constructor(private api: AxiosInstance) {}

  async getQuote(params: any) {
    const response = await this.api.post('/bridge/quote', params);
    return response.data;
  }

  async executeBridge(params: any) {
    const response = await this.api.post('/bridge/execute', params);
    return response.data;
  }

  async getSupportedChains() {
    const response = await this.api.get('/bridge/chains');
    return response.data;
  }

  async getProviders() {
    const response = await this.api.get('/bridge/providers');
    return response.data;
  }
}

export class StakingService {
  constructor(private api: AxiosInstance) {}

  async getPools(chain: string) {
    const response = await this.api.get(`/staking/pools`, {
      params: { chain },
    });
    return response.data;
  }

  async stake(params: any) {
    const response = await this.api.post('/staking/stake', params);
    return response.data;
  }

  async unstake(params: any) {
    const response = await this.api.post('/staking/unstake', params);
    return response.data;
  }

  async getRewards(address: string) {
    const response = await this.api.get(`/staking/rewards/${address}`);
    return response.data;
  }
}

export class NFTService {
  constructor(private api: AxiosInstance) {}

  async getCollections(address: string) {
    const response = await this.api.get(`/nft/collections`, {
      params: { address },
    });
    return response.data;
  }

  async getNFTs(collection: string, address: string) {
    const response = await this.api.get(`/nft/${collection}`, {
      params: { address },
    });
    return response.data;
  }

  async transferNFT(params: any) {
    const response = await this.api.post('/nft/transfer', params);
    return response.data;
  }
}

export class TokenService {
  constructor(private api: AxiosInstance) {}

  async getTokens(chainId: number) {
    const response = await this.api.get(`/tokens`, {
      params: { chainId },
    });
    return response.data;
  }

  async getTokenInfo(address: string, chainId: number) {
    const response = await this.api.get(`/tokens/${address}`, {
      params: { chainId },
    });
    return response.data;
  }

  async searchTokens(query: string) {
    const response = await this.api.get('/tokens/search', {
      params: { q: query },
    });
    return response.data;
  }
}

export class UserService {
  constructor(private api: AxiosInstance) {}

  async register(data: any) {
    const response = await this.api.post('/auth/register', data);
    return response.data;
  }

  async login(data: any) {
    const response = await this.api.post('/auth/login', data);
    if (response.data.token) {
      localStorage.setItem('auth_token', response.data.token);
    }
    return response.data;
  }

  async logout() {
    localStorage.removeItem('auth_token');
    await this.api.post('/auth/logout');
  }

  async getProfile() {
    const response = await this.api.get('/user/profile');
    return response.data;
  }

  async updateProfile(data: any) {
    const response = await this.api.patch('/user/profile', data);
    return response.data;
  }

  async submitKYC(data: any) {
    const response = await this.api.post('/user/kyc', data);
    return response.data;
  }
}

export class AdminService {
  constructor(private api: AxiosInstance) {}

  // Dashboard
  async getStats() {
    const response = await this.api.get('/admin/dashboard/stats');
    return response.data;
  }

  // Users
  async getUsers(params?: any) {
    const response = await this.api.get('/admin/users', { params });
    return response.data;
  }

  async updateUserKYC(userId: string, action: string) {
    const response = await this.api.post(`/admin/users/${userId}/kyc`, { action });
    return response.data;
  }

  // White Labels
  async getWhiteLabels() {
    const response = await this.api.get('/admin/whitelabels');
    return response.data;
  }

  async createWhiteLabel(data: any) {
    const response = await this.api.post('/admin/whitelabels', data);
    return response.data;
  }

  async updateWhiteLabelStatus(id: string, status: string) {
    const response = await this.api.patch(`/admin/whitelabels/${id}/status`, { status });
    return response.data;
  }

  // Blockchains
  async getBlockchains() {
    const response = await this.api.get('/admin/blockchains');
    return response.data;
  }

  async createBlockchain(data: any) {
    const response = await this.api.post('/admin/blockchains', data);
    return response.data;
  }

  // Tokens
  async getTokens() {
    const response = await this.api.get('/admin/tokens');
    return response.data;
  }

  async createToken(data: any) {
    const response = await this.api.post('/admin/tokens', data);
    return response.data;
  }

  // Trading Pairs
  async getPairs() {
    const response = await this.api.get('/admin/pairs');
    return response.data;
  }

  async createPair(data: any) {
    const response = await this.api.post('/admin/pairs', data);
    return response.data;
  }

  // Transactions
  async getTransactions(params?: any) {
    const response = await this.api.get('/admin/transactions', { params });
    return response.data;
  }

  // Fees
  async getFeeConfigs() {
    const response = await this.api.get('/admin/fees');
    return response.data;
  }

  async updateFeeConfig(id: string, data: any) {
    const response = await this.api.patch(`/admin/fees/${id}`, data);
    return response.data;
  }

  // Audit Logs
  async getAuditLogs(params?: any) {
    const response = await this.api.get('/admin/audit-logs', { params });
    return response.data;
  }
}

// ==================== Export Singleton ====================

export const services = ServiceConnector.getInstance();
export default services;
