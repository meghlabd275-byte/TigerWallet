/// <reference types="vite/client" />

// ============================================================================
// TYPES
// ============================================================================

export interface WhiteLabelClient {
  id: string;
  name: string;
  domain: string;
  subdomain?: string;
  status: 'pending' | 'active' | 'suspended' | 'halted' | 'expired' | 'revoked';
  plan: 'starter' | 'professional' | 'enterprise' | 'custom';
  features: Record<string, boolean>;
  branding: Record<string, string>;
  maxUsers: number;
  currentUsers: number;
  feePercent: number;
  createdAt: string;
  updatedAt: string;
  approvedAt?: string;
}

export interface WhiteLabelAdmin {
  id: string;
  clientId?: string;
  email: string;
  name: string;
  role: 'super_admin' | 'admin' | 'manager' | 'support';
  permissions: string[];
  status: string;
  twoFactorEnabled: boolean;
  lastLogin?: string;
  createdAt: string;
}

export interface Product {
  id: string;
  clientId?: string;
  name: string;
  type: 'trading' | 'perpetual' | 'staking' | 'nft' | 'wallet' | 'bridge' | 'launchpad';
  description?: string;
  status: 'enabled' | 'disabled' | 'maintenance';
  fee: number;
  minDeposit: number;
  maxDeposit: number;
  minWithdrawal: number;
  maxWithdrawal: number;
  features: string[];
  settings: Record<string, any>;
  sortOrder: number;
  createdAt: string;
}

export interface TradingPair {
  id: string;
  clientId?: string;
  baseToken: string;
  quoteToken: string;
  chainId: number;
  pairAddress?: string;
  status: 'active' | 'suspended' | 'halted';
  fee: number;
  minTrade: number;
  maxTrade: number;
  liquidity: number;
  pricePrecision: number;
  quantityPrecision: number;
  createdAt: string;
}

export interface Blockchain {
  id: number;
  name: string;
  symbol: string;
  category: 'evm' | 'solana' | 'aptos' | 'sui' | 'ton' | 'bitcoin' | 'cosmos' | 'polkadot';
  rpcUrls: string[];
  explorerUrls: string[];
  status: 'enabled' | 'disabled';
  isDefault: boolean;
  iconUrl?: string;
}

export interface AuditLog {
  id: string;
  clientId?: string;
  adminId?: string;
  action: string;
  resourceType: string;
  resourceId?: string;
  details: Record<string, any>;
  ipAddress?: string;
  userAgent?: string;
  status: string;
  createdAt: string;
}

export interface Notification {
  id: string;
  clientId?: string;
  adminId?: string;
  type: string;
  title: string;
  message?: string;
  data: Record<string, any>;
  read: boolean;
  readAt?: string;
  createdAt: string;
}

export interface DashboardStats {
  totalClients: number;
  activeClients: number;
  pendingClients: number;
  totalAdmins: number;
  totalProducts: number;
  totalPairs: number;
  totalPools: number;
  totalTokens: number;
  totalBots: number;
  totalUsers: number;
  volume24h: number;
  volume7d: number;
  volume30d: number;
  revenue24h: number;
  revenue7d: number;
  revenue30d: number;
}

export interface PaginatedResponse<T> {
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
  data: T[];
}

export interface LoginRequest {
  email: string;
  password: string;
  twoFactorCode?: string;
}

export interface LoginResponse {
  token: string;
  admin: WhiteLabelAdmin;
  expiresAt: string;
}

// ============================================================================
// API CONFIG
// ============================================================================

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8095/api/v1';

class ApiService {
  private token: string | null = null;

  constructor() {
    this.token = localStorage.getItem('admin_token');
  }

  setToken(token: string) {
    this.token = token;
    localStorage.setItem('admin_token', token);
  }

  clearToken() {
    this.token = null;
    localStorage.removeItem('admin_token');
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

    const response = await fetch(`${API_BASE_URL}${endpoint}`, {
      ...options,
      headers,
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Request failed' }));
      throw new Error(error.error || 'Request failed');
    }

    return response.json();
  }

  // Auth
  async login(data: LoginRequest): Promise<LoginResponse> {
    const response = await this.request<LoginResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify(data),
    });
    this.setToken(response.token);
    return response;
  }

  async logout(): Promise<void> {
    await this.request('/auth/logout', { method: 'POST' }).catch(() => {});
    this.clearToken();
  }

  // Dashboard
  async getDashboard(): Promise<DashboardStats> {
    return this.request('/dashboard');
  }

  // Clients
  async getClients(params?: { page?: number; pageSize?: number; query?: string; status?: string }): Promise<PaginatedResponse<WhiteLabelClient>> {
    const queryParams = new URLSearchParams();
    if (params?.page) queryParams.set('page', params.page.toString());
    if (params?.pageSize) queryParams.set('pageSize', params.pageSize.toString());
    if (params?.query) queryParams.set('query', params.query);
    if (params?.status) queryParams.set('status', params.status);
    
    return this.request(`/clients?${queryParams.toString()}`);
  }

  async getClient(id: string): Promise<WhiteLabelClient> {
    return this.request(`/clients/${id}`);
  }

  async createClient(data: Partial<WhiteLabelClient>): Promise<WhiteLabelClient> {
    return this.request('/clients', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateClient(id: string, data: Partial<WhiteLabelClient>): Promise<WhiteLabelClient> {
    return this.request(`/clients/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteClient(id: string): Promise<void> {
    await this.request(`/clients/${id}`, { method: 'DELETE' });
  }

  async approveClient(id: string): Promise<WhiteLabelClient> {
    return this.request(`/clients/${id}/approve`, { method: 'POST' });
  }

  async suspendClient(id: string): Promise<WhiteLabelClient> {
    return this.request(`/clients/${id}/suspend`, { method: 'POST' });
  }

  async resumeClient(id: string): Promise<WhiteLabelClient> {
    return this.request(`/clients/${id}/resume`, { method: 'POST' });
  }

  async haltClient(id: string): Promise<WhiteLabelClient> {
    return this.request(`/clients/${id}/halt`, { method: 'POST' });
  }

  // Admins
  async getAdmins(params?: { page?: number; pageSize?: number; query?: string; clientId?: string }): Promise<PaginatedResponse<WhiteLabelAdmin>> {
    const queryParams = new URLSearchParams();
    if (params?.page) queryParams.set('page', params.page.toString());
    if (params?.pageSize) queryParams.set('pageSize', params.pageSize.toString());
    if (params?.query) queryParams.set('query', params.query);
    if (params?.clientId) queryParams.set('clientId', params.clientId);
    
    return this.request(`/admins?${queryParams.toString()}`);
  }

  async getAdmin(id: string): Promise<WhiteLabelAdmin> {
    return this.request(`/admins/${id}`);
  }

  async createAdmin(data: Partial<WhiteLabelAdmin> & { password: string }): Promise<WhiteLabelAdmin> {
    return this.request('/admins', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateAdmin(id: string, data: Partial<WhiteLabelAdmin>): Promise<WhiteLabelAdmin> {
    return this.request(`/admins/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteAdmin(id: string): Promise<void> {
    await this.request(`/admins/${id}`, { method: 'DELETE' });
  }

  // Products
  async getProducts(params?: { page?: number; pageSize?: number; query?: string; status?: string; clientId?: string }): Promise<PaginatedResponse<Product>> {
    const queryParams = new URLSearchParams();
    if (params?.page) queryParams.set('page', params.page.toString());
    if (params?.pageSize) queryParams.set('pageSize', params.pageSize.toString());
    if (params?.query) queryParams.set('query', params.query);
    if (params?.status) queryParams.set('status', params.status);
    if (params?.clientId) queryParams.set('clientId', params.clientId);
    
    return this.request(`/products?${queryParams.toString()}`);
  }

  async getProduct(id: string): Promise<Product> {
    return this.request(`/products/${id}`);
  }

  async createProduct(data: Partial<Product>): Promise<Product> {
    return this.request('/products', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateProduct(id: string, data: Partial<Product>): Promise<Product> {
    return this.request(`/products/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteProduct(id: string): Promise<void> {
    await this.request(`/products/${id}`, { method: 'DELETE' });
  }

  async toggleProduct(id: string): Promise<Product> {
    return this.request(`/products/${id}/toggle`, { method: 'POST' });
  }

  // Trading Pairs
  async getTradingPairs(params?: { page?: number; pageSize?: number; query?: string; status?: string; clientId?: string }): Promise<PaginatedResponse<TradingPair>> {
    const queryParams = new URLSearchParams();
    if (params?.page) queryParams.set('page', params.page.toString());
    if (params?.pageSize) queryParams.set('pageSize', params.pageSize.toString());
    if (params?.query) queryParams.set('query', params.query);
    if (params?.status) queryParams.set('status', params.status);
    if (params?.clientId) queryParams.set('clientId', params.clientId);
    
    return this.request(`/pairs?${queryParams.toString()}`);
  }

  async getTradingPair(id: string): Promise<TradingPair> {
    return this.request(`/pairs/${id}`);
  }

  async createTradingPair(data: Partial<TradingPair>): Promise<TradingPair> {
    return this.request('/pairs', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateTradingPair(id: string, data: Partial<TradingPair>): Promise<TradingPair> {
    return this.request(`/pairs/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteTradingPair(id: string): Promise<void> {
    await this.request(`/pairs/${id}`, { method: 'DELETE' });
  }

  async suspendTradingPair(id: string): Promise<TradingPair> {
    return this.request(`/pairs/${id}/suspend`, { method: 'POST' });
  }

  async resumeTradingPair(id: string): Promise<TradingPair> {
    return this.request(`/pairs/${id}/resume`, { method: 'POST' });
  }

  async haltTradingPair(id: string): Promise<TradingPair> {
    return this.request(`/pairs/${id}/halt`, { method: 'POST' });
  }

  // Blockchains
  async getBlockchains(): Promise<Blockchain[]> {
    return this.request('/blockchains');
  }

  async updateBlockchain(id: number, data: Partial<Blockchain>): Promise<Blockchain> {
    return this.request(`/blockchains/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async enableBlockchain(id: number): Promise<Blockchain> {
    return this.request(`/blockchains/${id}/enable`, { method: 'POST' });
  }

  async disableBlockchain(id: number): Promise<Blockchain> {
    return this.request(`/blockchains/${id}/disable`, { method: 'POST' });
  }

  // Audit Logs
  async getAuditLogs(params?: { page?: number; pageSize?: number; query?: string; clientId?: string }): Promise<PaginatedResponse<AuditLog>> {
    const queryParams = new URLSearchParams();
    if (params?.page) queryParams.set('page', params.page.toString());
    if (params?.pageSize) queryParams.set('pageSize', params.pageSize.toString());
    if (params?.query) queryParams.set('query', params.query);
    if (params?.clientId) queryParams.set('clientId', params.clientId);
    
    return this.request(`/audit-logs?${queryParams.toString()}`);
  }

  // Notifications
  async getNotifications(params?: { page?: number; pageSize?: number; clientId?: string; adminId?: string }): Promise<PaginatedResponse<Notification>> {
    const queryParams = new URLSearchParams();
    if (params?.page) queryParams.set('page', params.page.toString());
    if (params?.pageSize) queryParams.set('pageSize', params.pageSize.toString());
    if (params?.clientId) queryParams.set('clientId', params.clientId);
    if (params?.adminId) queryParams.set('adminId', params.adminId);
    
    return this.request(`/notifications?${queryParams.toString()}`);
  }

  async markNotificationRead(id: string): Promise<void> {
    await this.request(`/notifications/${id}/read`, { method: 'POST' });
  }
}

export const api = new ApiService();
export default api;
