// ============================================================================
// TigerWallet - API Service
// Production-Ready API Client with Real Backend Integration
// ============================================================================

import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse } from 'axios';
import {
  APIResponse,
} from '../types/wallet';

// ============================================================================
// Configuration
// ============================================================================

const API_BASE_URL = process.env.API_BASE_URL || 'https://api.tigerwallet.io';
const API_TIMEOUT = 30000;

// ============================================================================
// API Service Class
// ============================================================================

class APIService {
  private static instance: APIService;
  private client: AxiosInstance;
  private authToken: string | null = null;

  private constructor() {
    this.client = axios.create({
      baseURL: API_BASE_URL,
      timeout: API_TIMEOUT,
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      },
    });

    // Request interceptor
    this.client.interceptors.request.use(
      (config) => {
        // Add auth token if available
        if (this.authToken) {
          config.headers.Authorization = `Bearer ${this.authToken}`;
        }
        
        // Add device info
        config.headers['X-Device-ID'] = this.getDeviceId();
        config.headers['X-App-Version'] = '1.0.0';
        config.headers['X-Platform'] = 'mobile';
        
        return config;
      },
      (error) => Promise.reject(error)
    );

    // Response interceptor
    this.client.interceptors.response.use(
      (response) => response,
      (error) => {
        // Handle specific error codes
        if (error.response?.status === 401) {
          // Token expired, clear auth
          this.clearAuthToken();
        }
        return Promise.reject(error);
      }
    );
  }

  static getInstance(): APIService {
    if (!APIService.instance) {
      APIService.instance = new APIService();
    }
    return APIService.instance;
  }

  // ============================================================================
  // Authentication
  // ============================================================================

  setAuthToken(token: string): void {
    this.authToken = token;
  }

  clearAuthToken(): void {
    this.authToken = null;
  }

  private getDeviceId(): string {
    // Would get from device info
    return 'device-' + Math.random().toString(36).substring(7);
  }

  // ============================================================================
  // HTTP Methods
  // ============================================================================

  async get<T = any>(
    url: string,
    params?: Record<string, any>,
    config?: AxiosRequestConfig
  ): Promise<APIResponse<T>> {
    try {
      const response: AxiosResponse<T> = await this.client.get(url, {
        ...config,
        params,
      });
      return this.formatResponse(response);
    } catch (error: any) {
      return this.formatError(error);
    }
  }

  async post<T = any>(
    url: string,
    data?: any,
    config?: AxiosRequestConfig
  ): Promise<APIResponse<T>> {
    try {
      const response: AxiosResponse<T> = await this.client.post(url, data, config);
      return this.formatResponse(response);
    } catch (error: any) {
      return this.formatError(error);
    }
  }

  async put<T = any>(
    url: string,
    data?: any,
    config?: AxiosRequestConfig
  ): Promise<APIResponse<T>> {
    try {
      const response: AxiosResponse<T> = await this.client.put(url, data, config);
      return this.formatResponse(response);
    } catch (error: any) {
      return this.formatError(error);
    }
  }

  async patch<T = any>(
    url: string,
    data?: any,
    config?: AxiosRequestConfig
  ): Promise<APIResponse<T>> {
    try {
      const response: AxiosResponse<T> = await this.client.patch(url, data, config);
      return this.formatResponse(response);
    } catch (error: any) {
      return this.formatError(error);
    }
  }

  async delete<T = any>(
    url: string,
    config?: AxiosRequestConfig
  ): Promise<APIResponse<T>> {
    try {
      const response: AxiosResponse<T> = await this.client.delete(url, config);
      return this.formatResponse(response);
    } catch (error: any) {
      return this.formatError(error);
    }
  }

  // ============================================================================
  // Response Formatting
  // ============================================================================

  private formatResponse<T>(response: AxiosResponse<T>): APIResponse<T> {
    return {
      success: true,
      data: response.data,
      timestamp: Date.now(),
    };
  }

  private formatError<T>(error: any): APIResponse<T> {
    const errorResponse: APIResponse<T> = {
      success: false,
      timestamp: Date.now(),
    };

    if (error.response) {
      // Server responded with error
      errorResponse.error = {
        code: error.response.status.toString(),
        message: error.response.data?.message || error.response.data?.error || 'Server error',
        details: error.response.data,
      };
    } else if (error.request) {
      // Request made but no response
      errorResponse.error = {
        code: 'NETWORK_ERROR',
        message: 'Network error. Please check your connection.',
      };
    } else {
      // Something else happened
      errorResponse.error = {
        code: 'UNKNOWN_ERROR',
        message: error.message || 'An unexpected error occurred',
      };
    }

    return errorResponse;
  }

  // ============================================================================
  // Wallet API Endpoints
  // ============================================================================

  // Wallet Management
  async createWallet(name: string, encryptedKey: string): Promise<APIResponse<any>> {
    return this.post('/api/v1/wallets', { name, encryptedKey });
  }

  async getWallets(): Promise<APIResponse<any>> {
    return this.get('/api/v1/wallets');
  }

  async getWallet(walletId: string): Promise<APIResponse<any>> {
    return this.get(`/api/v1/wallets/${walletId}`);
  }

  async deleteWallet(walletId: string): Promise<APIResponse<any>> {
    return this.delete(`/api/v1/wallets/${walletId}`);
  }

  // Balance & Transactions
  async getBalance(walletId: string, chainId: number): Promise<APIResponse<any>> {
    return this.get(`/api/v1/wallets/${walletId}/balance`, { chainId });
  }

  async getTokens(walletId: string, chainId: number): Promise<APIResponse<any>> {
    return this.get(`/api/v1/wallets/${walletId}/tokens`, { chainId });
  }

  async getTransactions(walletId: string, chainId: number, page?: number): Promise<APIResponse<any>> {
    return this.get(`/api/v1/wallets/${walletId}/transactions`, { chainId, page });
  }

  // Swap
  async getSwapQuote(params: {
    fromChainId: number;
    toChainId: number;
    fromToken: string;
    toToken: string;
    amount: string;
    fromAddress: string;
  }): Promise<APIResponse<any>> {
    return this.post('/api/v1/swap/quote', params);
  }

  async executeSwap(params: {
    quoteId: string;
    walletId: string;
    slippage: number;
  }): Promise<APIResponse<any>> {
    return this.post('/api/v1/swap/execute', params);
  }

  // Bridge
  async getBridgeQuotes(params: {
    fromChainId: number;
    toChainId: number;
    fromToken: string;
    toToken: string;
    amount: string;
    fromAddress: string;
  }): Promise<APIResponse<any>> {
    return this.post('/api/v1/bridge/quotes', params);
  }

  async executeBridge(params: {
    quoteId: string;
    walletId: string;
    toAddress: string;
  }): Promise<APIResponse<any>> {
    return this.post('/api/v1/bridge/execute', params);
  }

  // Staking
  async getStakingPools(chainId: number): Promise<APIResponse<any>> {
    return this.get('/api/v1/staking/pools', { chainId });
  }

  async stake(params: {
    walletId: string;
    poolId: string;
    amount: string;
  }): Promise<APIResponse<any>> {
    return this.post('/api/v1/staking/stake', params);
  }

  async unstake(params: {
    walletId: string;
    positionId: string;
  }): Promise<APIResponse<any>> {
    return this.post('/api/v1/staking/unstake', params);
  }

  // User Management
  async register(email: string, password: string): Promise<APIResponse<any>> {
    return this.post('/api/v1/auth/register', { email, password });
  }

  async login(email: string, password: string): Promise<APIResponse<any>> {
    return this.post('/api/v1/auth/login', { email, password });
  }

  async logout(): Promise<APIResponse<any>> {
    return this.post('/api/v1/auth/logout');
  }

  async getProfile(): Promise<APIResponse<any>> {
    return this.get('/api/v1/user/profile');
  }

  async updateProfile(data: any): Promise<APIResponse<any>> {
    return this.put('/api/v1/user/profile', data);
  }

  // KYC
  async submitKYC(data: any): Promise<APIResponse<any>> {
    return this.post('/api/v1/kyc/submit', data);
  }

  async getKYCStatus(): Promise<APIResponse<any>> {
    return this.get('/api/v1/kyc/status');
  }

  // Notifications
  async getNotifications(): Promise<APIResponse<any>> {
    return this.get('/api/v1/notifications');
  }

  async markNotificationRead(id: string): Promise<APIResponse<any>> {
    return this.patch(`/api/v1/notifications/${id}`, { read: true });
  }

  // Settings
  async getSettings(): Promise<APIResponse<any>> {
    return this.get('/api/v1/settings');
  }

  async updateSettings(settings: any): Promise<APIResponse<any>> {
    return this.put('/api/v1/settings', settings);
  }

  // Price Data
  async getPrices(tokenIds: string[]): Promise<APIResponse<any>> {
    return this.get('/api/v1/prices', { ids: tokenIds.join(',') });
  }

  async getPriceHistory(tokenId: string, days: number): Promise<APIResponse<any>> {
    return this.get(`/api/v1/prices/${tokenId}/history`, { days });
  }

  // Market Data
  async getMarketData(): Promise<APIResponse<any>> {
    return this.get('/api/v1/market');
  }

  async getTrending(): Promise<APIResponse<any>> {
    return this.get('/api/v1/market/trending');
  }

  // Support
  async contactSupport(subject: string, message: string): Promise<APIResponse<any>> {
    return this.post('/api/v1/support/contact', { subject, message });
  }
}

// ============================================================================
// Export singleton
// ============================================================================

export const API = APIService.getInstance();
export default API;
