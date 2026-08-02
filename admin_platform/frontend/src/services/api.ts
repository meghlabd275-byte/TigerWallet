/**
 * TigerWallet Admin Platform - API Service
 * Connects to C++ Super Admin Backend, Go RBAC Admin Backend
 * Production-ready with real API calls
 */

// Base URLs for different backends
const CPLUS_BACKEND_URL = process.env.REACT_APP_CPLUS_BACKEND_URL || 'http://localhost:8080';
const GO_BACKEND_URL = process.env.REACT_APP_GO_BACKEND_URL || 'http://localhost:8081';

// ============================================================================
// TYPES
// ============================================================================

export interface User {
  id: string;
  email: string;
  wallet_address?: string;
  username?: string;
  kyc_status: string;
  kyc_level: number;
  status: string;
  risk_score: number;
  tags: string[];
  created_at: string;
  updated_at: string;
  last_login?: string;
}

export interface UserKYC {
  id: string;
  user_id: string;
  kyc_type: string;
  document_type?: string;
  document_id?: string;
  first_name?: string;
  last_name?: string;
  date_of_birth?: string;
  nationality?: string;
  address: Record<string, any>;
  documents: any[];
  status: string;
  rejection_reason?: string;
  reviewed_by?: string;
  reviewed_at?: string;
  created_at: string;
  updated_at: string;
}

export interface Transaction {
  id: string;
  user_id: string;
  type: string;
  status: string;
  chain_id: string;
  token_address?: string;
  amount: string;
  fee: string;
  from_address?: string;
  to_address?: string;
  tx_hash?: string;
  metadata: Record<string, any>;
  created_at: string;
  completed_at?: string;
  updated_at: string;
}

export interface TradingPair {
  id: string;
  name: string;
  base_token: string;
  quote_token: string;
  chain_id: string;
  dex_id?: string;
  pair_address?: string;
  status: string;
  maker_fee: string;
  taker_fee: string;
  min_trade_amount: string;
  max_trade_amount?: string;
  created_at: string;
  updated_at: string;
}

export interface Blockchain {
  id: string;
  name: string;
  symbol: string;
  chain_type: string;
  chain_id?: number;
  rpc_urls: string[];
  explorer_urls: string[];
  is_active: boolean;
  is_maintenance: boolean;
  native_token: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export interface FeeStructure {
  id: string;
  name: string;
  fee_type: string;
  chain_id?: string;
  token_address?: string;
  maker_fee: string;
  taker_fee: string;
  fixed_fee: string;
  min_fee: string;
  max_fee?: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface LiquidityPool {
  id: string;
  pair_id: string;
  provider_address: string;
  liquidity_tokens: string;
  reserve_a: string;
  reserve_b: string;
  apr: string;
  created_at: string;
  updated_at: string;
}

export interface TradingBot {
  id: string;
  user_id: string;
  name: string;
  bot_type: string;
  tier: string;
  status: string;
  config: Record<string, any>;
  pnl: string;
  volume_24h: string;
  created_at: string;
  updated_at: string;
}

export interface CEXConnection {
  id: string;
  name: string;
  exchange: string;
  status: string;
  can_trade: boolean;
  can_withdraw: boolean;
  sync_status: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export interface DEXConnection {
  id: string;
  name: string;
  dex_name: string;
  chain_id: string;
  router_address?: string;
  factory_address?: string;
  status: string;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export interface TokenListingRequest {
  id: string;
  token_name: string;
  token_symbol: string;
  token_address: string;
  chain_id: string;
  requester_id: string;
  tier: string;
  status: string;
  one_time_fee: string;
  monthly_fee: string;
  rejection_reason?: string;
  reviewed_by?: string;
  reviewed_at?: string;
  created_at: string;
  updated_at: string;
}

export interface UserAPIKey {
  id: string;
  user_id: string;
  name: string;
  permissions: string[];
  rate_limit_minute: number;
  rate_limit_day: number;
  tier: string;
  is_active: boolean;
  last_used?: string;
  expires_at?: string;
  created_at: string;
}

// Super Admin types
export interface Admin {
  id: string;
  username: string;
  email: string;
  role: string;
  security_level: number;
  permissions: string[];
  two_factor_enabled: boolean;
  status: string;
  created_at: string;
  last_login?: string;
}

export interface WhiteLabel {
  id: string;
  name: string;
  domain: string;
  api_key?: string;
  fee_percent: number;
  status: string;
  custom_branding: boolean;
  features: string[];
  approved_by?: string;
  approved_at?: string;
  created_at: string;
}

export interface AuditLog {
  id: string;
  admin_id?: string;
  action: string;
  entity_type?: string;
  entity_id?: string;
  details: Record<string, any>;
  ip_address?: string;
  user_agent?: string;
  created_at: string;
}

export interface DashboardStats {
  total_users: number;
  active_users: number;
  suspended_users: number;
  kyc_pending: number;
  total_transactions: number;
  volume_24h: number;
}

export interface PaginatedResponse<T> {
  data: T[];
  meta: {
    page: number;
    limit: number;
    total: number;
  };
}

// ============================================================================
// AUTH SERVICE (Go Backend)
// ============================================================================

class AuthService {
  private baseUrl = GO_BACKEND_URL;

  async login(email: string, password: string): Promise<{ token: string; user: any }> {
    const response = await fetch(`${this.baseUrl}/api/v1/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Login failed');
    }

    const data = await response.json();
    localStorage.setItem('token', data.token);
    localStorage.setItem('user', JSON.stringify(data.user));
    return data;
  }

  async logout(): Promise<void> {
    const token = localStorage.getItem('token');
    if (token) {
      try {
        await fetch(`${this.baseUrl}/api/v1/auth/logout`, {
          method: 'POST',
          headers: { 
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json'
          },
        });
      } catch (e) {}
    }
    localStorage.removeItem('token');
    localStorage.removeItem('user');
  }

  getToken(): string | null {
    return localStorage.getItem('token');
  }

  getUser(): any {
    const user = localStorage.getItem('user');
    return user ? JSON.parse(user) : null;
  }

  isAuthenticated(): boolean {
    return !!this.getToken();
  }
}

// ============================================================================
// API CLIENT
// ============================================================================

class APIClient {
  private baseUrl = GO_BACKEND_URL;
  private token: string | null = null;

  constructor() {
    this.token = localStorage.getItem('token');
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...options.headers,
    };

    if (this.token) {
      (headers as Record<string, string>)['Authorization'] = `Bearer ${this.token}`;
    }

    const response = await fetch(`${this.baseUrl}${endpoint}`, {
      ...options,
      headers,
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Request failed' }));
      throw new Error(error.error || 'Request failed');
    }

    return response.json();
  }

  // Dashboard
  async getDashboardStats(): Promise<DashboardStats> {
    return this.request<DashboardStats>('/api/v1/dashboard');
  }

  // Users
  async listUsers(params?: { status?: string; kyc_status?: string; page?: number; limit?: number }): Promise<PaginatedResponse<User>> {
    const queryParams = new URLSearchParams();
    if (params?.status) queryParams.set('status', params.status);
    if (params?.kyc_status) queryParams.set('kyc_status', params.kyc_status);
    if (params?.page) queryParams.set('page', String(params.page));
    if (params?.limit) queryParams.set('limit', String(params.limit));
    
    return this.request<PaginatedResponse<User>>(`/api/v1/users?${queryParams}`);
  }

  async getUser(id: string): Promise<User> {
    return this.request<User>(`/api/v1/users/${id}`);
  }

  async updateUser(id: string, data: Partial<User>): Promise<void> {
    await this.request(`/api/v1/users/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async banUser(id: string): Promise<void> {
    await this.request(`/api/v1/users/${id}/ban`, { method: 'POST' });
  }

  async unbanUser(id: string): Promise<void> {
    await this.request(`/api/v1/users/${id}/unban`, { method: 'POST' });
  }

  async suspendUser(id: string): Promise<void> {
    await this.request(`/api/v1/users/${id}/suspend`, { method: 'POST' });
  }

  async getUserBalance(userId: string): Promise<{ balances: any[] }> {
    return this.request(`/api/v1/users/${userId}/balance`);
  }

  // KYC
  async listKYC(params?: { status?: string; page?: number; limit?: number }): Promise<PaginatedResponse<UserKYC>> {
    const queryParams = new URLSearchParams();
    if (params?.status) queryParams.set('status', params.status);
    if (params?.page) queryParams.set('page', String(params.page));
    if (params?.limit) queryParams.set('limit', String(params.limit));
    
    return this.request<PaginatedResponse<UserKYC>>(`/api/v1/kyc?${queryParams}`);
  }

  async getKYC(id: string): Promise<UserKYC> {
    return this.request<UserKYC>(`/api/v1/kyc/${id}`);
  }

  async approveKYC(id: string): Promise<void> {
    await this.request(`/api/v1/kyc/${id}/approve`, { method: 'POST' });
  }

  async rejectKYC(id: string, reason: string): Promise<void> {
    await this.request(`/api/v1/kyc/${id}/reject`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
  }

  // Transactions
  async listTransactions(params?: { type?: string; status?: string; page?: number; limit?: number }): Promise<PaginatedResponse<Transaction>> {
    const queryParams = new URLSearchParams();
    if (params?.type) queryParams.set('type', params.type);
    if (params?.status) queryParams.set('status', params.status);
    if (params?.page) queryParams.set('page', String(params.page));
    if (params?.limit) queryParams.set('limit', String(params.limit));
    
    return this.request<PaginatedResponse<Transaction>>(`/api/v1/transactions?${queryParams}`);
  }

  async getTransaction(id: string): Promise<Transaction> {
    return this.request<Transaction>(`/api/v1/transactions/${id}`);
  }

  // Trading Pairs
  async listPairs(params?: { status?: string; page?: number; limit?: number }): Promise<PaginatedResponse<TradingPair>> {
    const queryParams = new URLSearchParams();
    if (params?.status) queryParams.set('status', params.status);
    if (params?.page) queryParams.set('page', String(params.page));
    if (params?.limit) queryParams.set('limit', String(params.limit));
    
    return this.request<PaginatedResponse<TradingPair>>(`/api/v1/pairs?${queryParams}`);
  }

  async createPair(data: Partial<TradingPair>): Promise<TradingPair> {
    return this.request<TradingPair>('/api/v1/pairs', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updatePair(id: string, data: Partial<TradingPair>): Promise<void> {
    await this.request(`/api/v1/pairs/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async suspendPair(id: string): Promise<void> {
    await this.request(`/api/v1/pairs/${id}/suspend`, { method: 'POST' });
  }

  async resumePair(id: string): Promise<void> {
    await this.request(`/api/v1/pairs/${id}/resume`, { method: 'POST' });
  }

  async haltPair(id: string): Promise<void> {
    await this.request(`/api/v1/pairs/${id}/halt`, { method: 'POST' });
  }

  // Blockchains
  async listBlockchains(): Promise<{ data: Blockchain[] }> {
    return this.request<{ data: Blockchain[] }>('/api/v1/blockchains');
  }

  async getBlockchain(id: string): Promise<Blockchain> {
    return this.request<Blockchain>(`/api/v1/blockchains/${id}`);
  }

  async createBlockchain(data: Partial<Blockchain>): Promise<Blockchain> {
    return this.request<Blockchain>('/api/v1/blockchains', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateBlockchain(id: string, data: Partial<Blockchain>): Promise<void> {
    await this.request(`/api/v1/blockchains/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async setMaintenance(id: string, maintenance: boolean): Promise<void> {
    await this.request(`/api/v1/blockchains/${id}/maintenance`, {
      method: 'POST',
      body: JSON.stringify({ maintenance }),
    });
  }

  async activateBlockchain(id: string, active: boolean): Promise<void> {
    await this.request(`/api/v1/blockchains/${id}/activate`, {
      method: 'POST',
      body: JSON.stringify({ active }),
    });
  }

  // Fees
  async listFees(feeType?: string): Promise<{ data: FeeStructure[] }> {
    const query = feeType ? `?type=${feeType}` : '';
    return this.request<{ data: FeeStructure[] }>(`/api/v1/fees${query}`);
  }

  async createFee(data: Partial<FeeStructure>): Promise<FeeStructure> {
    return this.request<FeeStructure>('/api/v1/fees', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateFee(id: string, data: Partial<FeeStructure>): Promise<void> {
    await this.request(`/api/v1/fees/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  // Liquidity
  async listLiquidity(): Promise<{ data: LiquidityPool[] }> {
    return this.request<{ data: LiquidityPool[] }>('/api/v1/liquidity');
  }

  async addLiquidity(data: Partial<LiquidityPool>): Promise<LiquidityPool> {
    return this.request<LiquidityPool>('/api/v1/liquidity', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async removeLiquidity(id: string): Promise<void> {
    await this.request(`/api/v1/liquidity/${id}`, { method: 'DELETE' });
  }

  // Bots
  async listBots(): Promise<{ data: TradingBot[] }> {
    return this.request<{ data: TradingBot[] }>('/api/v1/bots');
  }

  async pauseBot(id: string): Promise<void> {
    await this.request(`/api/v1/bots/${id}/pause`, { method: 'PUT' });
  }

  async resumeBot(id: string): Promise<void> {
    await this.request(`/api/v1/bots/${id}/resume`, { method: 'PUT' });
  }

  // CEX
  async listCEX(): Promise<{ data: CEXConnection[] }> {
    return this.request<{ data: CEXConnection[] }>('/api/v1/cex');
  }

  async createCEX(data: Partial<CEXConnection>): Promise<CEXConnection> {
    return this.request<CEXConnection>('/api/v1/cex', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  // DEX
  async listDEX(): Promise<{ data: DEXConnection[] }> {
    return this.request<{ data: DEXConnection[] }>('/api/v1/dex');
  }

  async createDEX(data: Partial<DEXConnection>): Promise<DEXConnection> {
    return this.request<DEXConnection>('/api/v1/dex', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  // Token Listings
  async listTokenListings(params?: { status?: string; page?: number; limit?: number }): Promise<PaginatedResponse<TokenListingRequest>> {
    const queryParams = new URLSearchParams();
    if (params?.status) queryParams.set('status', params.status);
    if (params?.page) queryParams.set('page', String(params.page));
    if (params?.limit) queryParams.set('limit', String(params.limit));
    
    return this.request<PaginatedResponse<TokenListingRequest>>(`/api/v1/token-listings?${queryParams}`);
  }

  async approveTokenListing(id: string): Promise<void> {
    await this.request(`/api/v1/token-listings/${id}/approve`, { method: 'POST' });
  }

  async rejectTokenListing(id: string): Promise<void> {
    await this.request(`/api/v1/token-listings/${id}/reject`, { method: 'POST' });
  }

  // API Keys
  async listAPIKeys(userId?: string): Promise<{ data: UserAPIKey[] }> {
    const query = userId ? `?user_id=${userId}` : '';
    return this.request<{ data: UserAPIKey[] }>(`/api/v1/api-keys${query}`);
  }

  async createAPIKey(data: { user_id: string; name: string; permissions?: string[] }): Promise<{ api_key: string; id: string; name: string }> {
    return this.request<{ api_key: string; id: string; name: string }>('/api/v1/api-keys', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async revokeAPIKey(id: string): Promise<void> {
    await this.request(`/api/v1/api-keys/${id}/revoke`, { method: 'POST' });
  }
}

// ============================================================================
// SUPER ADMIN API (C++ Backend)
// ============================================================================

class SuperAdminAPI {
  private baseUrl = CPLUS_BACKEND_URL;

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const token = localStorage.getItem('super_admin_token');
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...options.headers,
    };

    if (token) {
      (headers as Record<string, string>)['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(`${this.baseUrl}${endpoint}`, {
      ...options,
      headers,
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Request failed' }));
      throw new Error(error.error || 'Request failed');
    }

    return response.json();
  }

  async login(username: string, password: string, twoFactorCode?: string): Promise<{ token: string }> {
    const body: any = { username, password };
    if (twoFactorCode) body.two_factor_code = twoFactorCode;
    
    const response = await this.request<{ token: string }>('/api/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify(body),
    });
    
    localStorage.setItem('super_admin_token', response.token);
    return response;
  }

  async logout(): Promise<void> {
    const token = localStorage.getItem('super_admin_token');
    if (token) {
      try {
        await this.request('/api/v1/auth/logout', { method: 'POST' });
      } catch (e) {}
    }
    localStorage.removeItem('super_admin_token');
  }

  async getDashboardStats(): Promise<any> {
    return this.request('/api/v1/dashboard');
  }

  async listAdmins(params?: { filter?: string; page?: number; limit?: number }): Promise<{ data: Admin[] }> {
    const queryParams = new URLSearchParams();
    if (params?.filter) queryParams.set('filter', params.filter);
    if (params?.page) queryParams.set('page', String(params.page));
    if (params?.limit) queryParams.set('limit', String(params.limit));
    
    return this.request<{ data: Admin[] }>(`/api/v1/admins?${queryParams}`);
  }

  async createAdmin(data: { username: string; password: string; email: string; role: string }): Promise<Admin> {
    return this.request<Admin>('/api/v1/admins', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateAdmin(id: string, data: Partial<Admin>): Promise<void> {
    await this.request(`/api/v1/admins/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async suspendAdmin(id: string): Promise<void> {
    await this.request(`/api/v1/admins/${id}/suspend`, { method: 'POST' });
  }

  async activateAdmin(id: string): Promise<void> {
    await this.request(`/api/v1/admins/${id}/activate`, { method: 'POST' });
  }

  async deleteAdmin(id: string): Promise<void> {
    await this.request(`/api/v1/admins/${id}`, { method: 'DELETE' });
  }

  async listWhiteLabels(params?: { status?: string; page?: number; limit?: number }): Promise<{ data: WhiteLabel[] }> {
    const queryParams = new URLSearchParams();
    if (params?.status) queryParams.set('status', params.status);
    if (params?.page) queryParams.set('page', String(params.page));
    if (params?.limit) queryParams.set('limit', String(params.limit));
    
    return this.request<{ data: WhiteLabel[] }>(`/api/v1/whitelabels?${queryParams}`);
  }

  async createWhiteLabel(data: { name: string; domain: string }): Promise<WhiteLabel> {
    return this.request<WhiteLabel>('/api/v1/whitelabels', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async approveWhiteLabel(id: string): Promise<void> {
    await this.request(`/api/v1/whitelabels/${id}/approve`, { method: 'POST' });
  }

  async revokeWhiteLabel(id: string): Promise<void> {
    await this.request(`/api/v1/whitelabels/${id}/revoke`, { method: 'POST' });
  }

  async suspendWhiteLabel(id: string): Promise<void> {
    await this.request(`/api/v1/whitelabels/${id}/suspend`, { method: 'POST' });
  }

  async updateWhiteLabelFee(id: string, feePercent: number): Promise<void> {
    await this.request(`/api/v1/whitelabels/${id}/fee`, {
      method: 'PUT',
      body: JSON.stringify({ fee_percent: feePercent }),
    });
  }

  async getAuditLogs(params?: { admin_id?: string; action?: string; page?: number; limit?: number }): Promise<{ data: AuditLog[] }> {
    const queryParams = new URLSearchParams();
    if (params?.admin_id) queryParams.set('admin_id', params.admin_id);
    if (params?.action) queryParams.set('action', params.action);
    if (params?.page) queryParams.set('page', String(params.page));
    if (params?.limit) queryParams.set('limit', String(params.limit));
    
    return this.request<{ data: AuditLog[] }>(`/api/v1/audit?${queryParams}`);
  }

  async enable2FA(): Promise<{ secret: string; backup_codes: string[] }> {
    return this.request('/api/v1/2fa/enable', { method: 'POST' });
  }

  async disable2FA(code: string): Promise<void> {
    await this.request('/api/v1/2fa/disable', {
      method: 'POST',
      body: JSON.stringify({ code }),
    });
  }

  async addIPWhitelist(ipCidr: string, description: string): Promise<void> {
    await this.request('/api/v1/ip-whitelist', {
      method: 'POST',
      body: JSON.stringify({ ip_cidr: ipCidr, description }),
    });
  }

  async listIPWhitelist(): Promise<{ data: any[] }> {
    return this.request('/api/v1/ip-whitelist');
  }

  async removeIPWhitelist(id: string): Promise<void> {
    await this.request(`/api/v1/ip-whitelist/${id}`, { method: 'DELETE' });
  }
}

// ============================================================================
// EXPORTS
// ============================================================================

export const authService = new AuthService();
export const apiClient = new APIClient();
export const superAdminAPI = new SuperAdminAPI();

export type {
  User,
  UserKYC,
  Transaction,
  TradingPair,
  Blockchain,
  FeeStructure,
  LiquidityPool,
  TradingBot,
  CEXConnection,
  DEXConnection,
  TokenListingRequest,
  UserAPIKey,
  Admin,
  WhiteLabel,
  AuditLog,
  DashboardStats,
};
