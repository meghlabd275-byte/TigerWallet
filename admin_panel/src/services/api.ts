/**
 * TigerWallet RBAC Admin Panel - API Service
 * Production-ready API client that connects to Go backend
 * No stubs - Real API calls only
 */

const API_BASE_URL = 'http://localhost:8081/api/v1/admin';

// ==================== TYPES ====================

export interface User {
  id: string;
  email: string;
  username: string;
  wallet_address: string;
  kyc_status: 'none' | 'pending' | 'verified' | 'rejected';
  status: 'active' | 'suspended' | 'banned';
  created_at: string;
  last_login: string;
  balance: Record<string, number>;
  two_factor_enabled: boolean;
  ip_address: string;
  country: string;
}

export interface KYCRequest {
  id: string;
  user_id: string;
  doc_type: 'identity' | 'address' | 'selfie';
  status: 'none' | 'pending' | 'verified' | 'rejected';
  document_url: string;
  submitted_at: string;
  reviewed_at?: string;
  reviewed_by?: string;
  reject_reason?: string;
}

export interface Transaction {
  id: string;
  user_id: string;
  type: 'deposit' | 'withdrawal' | 'transfer' | 'swap';
  amount: number;
  currency: string;
  status: 'pending' | 'completed' | 'failed';
  from_address: string;
  to_address: string;
  tx_hash: string;
  timestamp: string;
  fee: number;
  chain_id: number;
}

export interface TradingPair {
  id: string;
  base: string;
  quote: string;
  pair_name: string;
  price: number;
  volume_24h: number;
  liquidity: number;
  status: 'active' | 'suspended' | 'halted';
  chain_id: number;
  created_at: string;
  updated_at: string;
}

export interface LiquidityPool {
  id: string;
  pair_id: string;
  user_id: string;
  base_amount: number;
  quote_amount: number;
  liquidity: number;
  apr: number;
  created_at: string;
}

export interface FeeStructure {
  id: string;
  fee_type: 'withdrawal' | 'deposit' | 'trading' | 'swap';
  asset: string;
  fee_percent: number;
  fee_fixed: number;
  min_fee: number;
  max_fee?: number;
  tier: string;
  is_active: boolean;
  chain_id: number;
}

export interface Blockchain {
  id: string;
  name: string;
  symbol: string;
  chain_id: number;
  is_evm: boolean;
  rpc_url: string;
  explorer_url: string;
  native_token: string;
  decimals: number;
  is_active: boolean;
  avg_gas_price_gwei: number;
}

export interface BotInstance {
  id: string;
  user_id: string;
  bot_type: string;
  name: string;
  status: 'running' | 'stopped' | 'error' | 'paused';
  connected_dexs: number;
  connected_cexs: number;
  total_pnl: number;
  total_volume: number;
  total_orders: number;
  avg_latency_us: number;
  created_at: string;
  last_trade_at: string;
}

export interface BotTier {
  id: string;
  name: string;
  display_name: string;
  monthly_fee_usd: number;
  per_dex_fee_usd: number;
  per_cex_fee_usd: number;
  max_bots: number;
  max_dexs: number;
  max_cexs: number;
  max_position_usd: number;
  max_daily_volume: number;
  latency_target_ms: number;
  is_active: boolean;
}

export interface APIKey {
  id: string;
  user_id: string;
  name: string;
  key: string;
  tier: 'free' | 'basic' | 'pro' | 'enterprise';
  permissions: {
    trading: boolean;
    reading: boolean;
    withdrawal: boolean;
  };
  rate_limit_per_min: number;
  rate_limit_per_day: number;
  is_active: boolean;
  last_used_at?: string;
  expires_at: string;
  created_at: string;
}

export interface PlatformStats {
  total_users: number;
  active_users: number;
  total_volume: number;
  total_transactions: number;
  total_fees: number;
  active_bots: number;
  total_bots: number;
  active_cex_connections: number;
  active_dex_connections: number;
}

// ==================== API CLIENT ====================

class APIClient {
  private baseUrl: string;
  private token: string | null = null;

  constructor(baseUrl: string = API_BASE_URL) {
    this.baseUrl = baseUrl;
    // Load token from localStorage if available
    if (typeof window !== 'undefined') {
      this.token = localStorage.getItem('admin_token');
    }
  }

  setToken(token: string) {
    this.token = token;
    if (typeof window !== 'undefined') {
      localStorage.setItem('admin_token', token);
    }
  }

  clearToken() {
    this.token = null;
    if (typeof window !== 'undefined') {
      localStorage.removeItem('admin_token');
    }
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;
    
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...(options.headers || {}),
    };

    if (this.token) {
      (headers as Record<string, string>)['Authorization'] = `Bearer ${this.token}`;
    }

    try {
      const response = await fetch(url, {
        ...options,
        headers,
      });

      if (!response.ok) {
        const error = await response.json().catch(() => ({ error: 'Request failed' }));
        throw new Error(error.error || `HTTP ${response.status}`);
      }

      return response.json();
    } catch (error) {
      console.error(`API Error [${endpoint}]:`, error);
      throw error;
    }
  }

  // ==================== USER MANAGEMENT ====================

  async getUsers(): Promise<{ users: User[] }> {
    return this.request<{ users: User[] }>('/users');
  }

  async getUser(id: string): Promise<{ user: User }> {
    return this.request<{ user: User }>(`/users/${id}`);
  }

  async searchUsers(query: string): Promise<{ users: User[] }> {
    return this.request<{ users: User[] }>(`/users/search?q=${encodeURIComponent(query)}`);
  }

  async updateUserStatus(id: string, status: string): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>(`/users/${id}/status`, {
      method: 'PUT',
      body: JSON.stringify({ status }),
    });
  }

  async banUser(id: string): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>(`/users/${id}/ban`, {
      method: 'POST',
    });
  }

  async unbanUser(id: string): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>(`/users/${id}/unban`, {
      method: 'POST',
    });
  }

  async suspendUser(id: string): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>(`/users/${id}/suspend`, {
      method: 'POST',
    });
  }

  // ==================== KYC MANAGEMENT ====================

  async getKYCRequests(): Promise<{ kyc_requests: KYCRequest[] }> {
    return this.request<{ kyc_requests: KYCRequest[] }>('/kyc');
  }

  async approveKYC(id: string): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>(`/kyc/${id}/approve`, {
      method: 'POST',
    });
  }

  async rejectKYC(id: string, reason: string): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>(`/kyc/${id}/reject`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
  }

  // ==================== TRANSACTION MANAGEMENT ====================

  async getTransactions(): Promise<{ transactions: Transaction[] }> {
    return this.request<{ transactions: Transaction[] }>('/transactions');
  }

  async getTransaction(id: string): Promise<{ transaction: Transaction }> {
    return this.request<{ transaction: Transaction }>(`/transactions/${id}`);
  }

  // ==================== TRADING PAIR MANAGEMENT ====================

  async getTradingPairs(): Promise<{ trading_pairs: TradingPair[] }> {
    return this.request<{ trading_pairs: TradingPair[] }>('/pairs');
  }

  async createTradingPair(data: Partial<TradingPair>): Promise<{ trading_pair: TradingPair }> {
    return this.request<{ trading_pair: TradingPair }>('/pairs', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updatePairStatus(id: string, status: string): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>(`/pairs/${id}/status`, {
      method: 'PUT',
      body: JSON.stringify({ status }),
    });
  }

  async suspendPair(id: string): Promise<{ success: boolean }> {
    return this.updatePairStatus(id, 'suspended');
  }

  async resumePair(id: string): Promise<{ success: boolean }> {
    return this.updatePairStatus(id, 'active');
  }

  async haltPair(id: string): Promise<{ success: boolean }> {
    return this.updatePairStatus(id, 'halted');
  }

  // ==================== FEE MANAGEMENT ====================

  async getFeeStructures(): Promise<{ fees: FeeStructure[] }> {
    return this.request<{ fees: FeeStructure[] }>('/fees');
  }

  async createFeeStructure(data: Partial<FeeStructure>): Promise<{ fee: FeeStructure }> {
    return this.request<{ fee: FeeStructure }>('/fees', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateFee(id: string, data: Partial<FeeStructure>): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>(`/fees/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  // ==================== BLOCKCHAIN MANAGEMENT ====================

  async getBlockchains(): Promise<{ blockchains: Blockchain[] }> {
    return this.request<{ blockchains: Blockchain[] }>('/blockchains');
  }

  async getBlockchain(id: string): Promise<{ blockchain: Blockchain }> {
    return this.request<{ blockchain: Blockchain }>(`/blockchains/${id}`);
  }

  async createBlockchain(data: Partial<Blockchain>): Promise<{ blockchain: Blockchain }> {
    return this.request<{ blockchain: Blockchain }>('/blockchains', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateBlockchain(id: string, data: Partial<Blockchain>): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>(`/blockchains/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async setBlockchainStatus(id: string, isActive: boolean): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>(`/blockchains/${id}/status`, {
      method: 'PUT',
      body: JSON.stringify({ is_active: isActive }),
    });
  }

  // ==================== BOT MANAGEMENT ====================

  async getBotInstances(): Promise<{ bots: BotInstance[] }> {
    return this.request<{ bots: BotInstance[] }>('/bots');
  }

  async getBotTiers(): Promise<{ bot_tiers: BotTier[] }> {
    return this.request<{ bot_tiers: BotTier[] }>('/bot-tiers');
  }

  async updateBotStatus(id: string, status: string): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>(`/bots/${id}/status`, {
      method: 'PUT',
      body: JSON.stringify({ status }),
    });
  }

  async pauseBot(id: string): Promise<{ success: boolean }> {
    return this.updateBotStatus(id, 'paused');
  }

  async resumeBot(id: string): Promise<{ success: boolean }> {
    return this.updateBotStatus(id, 'running');
  }

  async stopBot(id: string): Promise<{ success: boolean }> {
    return this.updateBotStatus(id, 'stopped');
  }

  // ==================== API KEY MANAGEMENT ====================

  async getAPIKeys(): Promise<{ api_keys: APIKey[] }> {
    return this.request<{ api_keys: APIKey[] }>('/api-keys');
  }

  async createAPIKey(data: Partial<APIKey>): Promise<{ api_key: APIKey }> {
    return this.request<{ api_key: APIKey }>('/api-keys', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async revokeAPIKey(id: string): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>(`/api-keys/${id}/revoke`, {
      method: 'POST',
    });
  }

  // ==================== PLATFORM STATS ====================

  async getPlatformStats(): Promise<{ stats: PlatformStats }> {
    return this.request<{ stats: PlatformStats }>('/stats');
  }
}

// ==================== EXPORT SINGLETON ====================

export const apiClient = new APIClient();

// Export types for external use
export type {
  User,
  KYCRequest,
  Transaction,
  TradingPair,
  LiquidityPool,
  FeeStructure,
  Blockchain,
  BotInstance,
  BotTier,
  APIKey,
  PlatformStats,
};

export default apiClient;
