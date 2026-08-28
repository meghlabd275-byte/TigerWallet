// TigerWallet Admin - Complete API Service
// Connects to Go Admin Service and Rust Admin Fetchers
// Uses PostgreSQL and Redis on the backend

const API_BASE_URL = process.env.NEXT_PUBLIC_ADMIN_API || 'http://localhost:9093';
const WS_BASE_URL = process.env.NEXT_PUBLIC_ADMIN_WS || 'ws://localhost:9093';

class AdminApiService {
  private baseURL: string;
  private wsURL: string;
  private token: string | null = null;

  constructor() {
    this.baseURL = API_BASE_URL;
    this.wsURL = WS_BASE_URL;
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

  async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.baseURL}${endpoint}`;
    
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...(this.token && { 'Authorization': `Bearer ${this.token}` }),
      ...options.headers,
    };

    const response = await fetch(url, { ...options, headers });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ message: 'Unknown error' }));
      throw new Error(error.message || `API Error: ${response.status}`);
    }

    return response.json();
  }

  // ==================== AUTHENTICATION ====================
  
  async login(email: string, password: string, twoFactorCode?: string): Promise<{ token: string; admin: any; two_factor_required?: boolean }> {
    return this.request('/api/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password, two_factor_code: twoFactorCode }),
    });
  }

  async register(data: {
    email: string;
    password: string;
    username: string;
  }): Promise<{ token: string; admin: any }> {
    return this.request('/api/v1/auth/register', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async logout(): Promise<void> {
    await this.request('/api/v1/auth/logout', { method: 'POST' });
    this.clearToken();
  }

  async refreshToken(): Promise<{ token: string }> {
    return this.request('/api/v1/auth/refresh', { method: 'POST' });
  }

  async changePassword(currentPassword: string, newPassword: string): Promise<void> {
    return this.request('/api/v1/auth/password', {
      method: 'POST',
      body: JSON.stringify({ currentPassword, newPassword }),
    });
  }

  // ==================== ADMIN MANAGEMENT ====================

  async getAdmins(): Promise<any[]> {
    return this.request('/api/v1/admins');
  }

  async listAdmins(): Promise<{ data: any[] }> {
    const admins = await this.request<any[]>('/api/v1/admins');
    return { data: admins || [] };
  }

  async getAdmin(id: string): Promise<any> {
    return this.request(`/api/v1/admins/${id}`);
  }

  async createAdmin(data: {
    email: string;
    username: string;
    password: string;
    role: string;
  }): Promise<any> {
    return this.request('/api/v1/admins', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateAdmin(id: string, data: Partial<any>): Promise<any> {
    return this.request(`/api/v1/admins/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteAdmin(id: string): Promise<void> {
    return this.request(`/api/v1/admins/${id}`, { method: 'DELETE' });
  }

  async suspendAdmin(id: string, reason?: string): Promise<void> {
    return this.request(`/api/v1/admins/${id}/suspend`, {
      method: 'POST',
      body: JSON.stringify({ reason: reason || '' }),
    });
  }

  async enableTwoFactor(adminId: string): Promise<{ secret: string; qrCode: string }> {
    return this.request(`/api/v1/admins/${adminId}/two-factor/enable`, {
      method: 'POST',
    });
  }

  async disableTwoFactor(adminId: string, code: string): Promise<void> {
    return this.request(`/api/v1/admins/${adminId}/two-factor/disable`, {
      method: 'POST',
      body: JSON.stringify({ code }),
    });
  }

  // ==================== USER MANAGEMENT ====================

  async getUsers(params?: {
    page?: number;
    pageSize?: number;
    status?: string;
    kycStatus?: string;
    search?: string;
    sortBy?: string;
    sortOrder?: 'ASC' | 'DESC';
  }): Promise<{ data: any[]; total: number; page: number; pageSize: number; totalPages: number }> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    return this.request(`/api/v1/users?${query}`);
  }

  async getUser(id: string): Promise<any> {
    return this.request(`/api/v1/users/${id}`);
  }

  async updateUser(id: string, data: Partial<any>): Promise<any> {
    return this.request(`/api/v1/users/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteUser(id: string): Promise<void> {
    return this.request(`/api/v1/users/${id}`, { method: 'DELETE' });
  }

  async suspendUser(id: string, reason: string): Promise<void> {
    return this.request(`/api/v1/users/${id}/suspend`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
  }

  async unsuspendUser(id: string): Promise<void> {
    return this.request(`/api/v1/users/${id}/unsuspend`, { method: 'POST' });
  }

  async verifyUser(id: string): Promise<void> {
    return this.request(`/api/v1/users/${id}/verify`, { method: 'POST' });
  }

  // ==================== KYC MANAGEMENT ====================

  async getKycRecords(params?: {
    page?: number;
    pageSize?: number;
    status?: string;
    search?: string;
  }): Promise<{ data: any[]; total: number; page: number; pageSize: number; totalPages: number }> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    const res = await this.request<{ data: any[]; total: number; page: number; pageSize: number }>(`/api/v1/kyc?${query}`);
    return { ...res, totalPages: Math.ceil((res.total || 0) / (res.pageSize || 20)) };
  }

  async getKycRecord(id: string): Promise<any> {
    return this.request(`/api/v1/kyc/${id}`);
  }

  async approveKyc(id: string, notes?: string): Promise<any> {
    return this.request(`/api/v1/kyc/${id}/approve`, {
      method: 'POST',
      body: JSON.stringify({ notes }),
    });
  }

  async rejectKyc(id: string, reason: string): Promise<void> {
    return this.request(`/api/v1/kyc/${id}/reject`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
  }

  async requestKycResubmission(id: string, reason: string): Promise<void> {
    return this.request(`/api/v1/kyc/${id}/resubmit`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
  }

  // ==================== TRANSACTION MANAGEMENT ====================

  async getTransactions(params?: {
    page?: number;
    pageSize?: number;
    status?: string;
    type?: string;
    chain?: string;
    token?: string;
    search?: string;
    startDate?: string;
    endDate?: string;
  }): Promise<{ data: any[]; total: number; page: number; pageSize: number; totalPages: number }> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    const res = await this.request<{ data: any[]; total: number; page: number; pageSize: number }>(`/api/v1/transactions?${query}`);
    return { ...res, totalPages: Math.ceil((res.total || 0) / (res.pageSize || 20)) };
  }

  async getTransaction(id: string): Promise<any> {
    return this.request(`/api/v1/transactions/${id}`);
  }

  async flagTransaction(id: string, reason: string): Promise<any> {
    return this.request(`/api/v1/transactions/${id}/flag`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
  }

  async unflagTransaction(id: string): Promise<any> {
    return this.request(`/api/v1/transactions/${id}/unflag`, { method: 'POST' });
  }

  async cancelTransaction(id: string): Promise<any> {
    return this.request(`/api/v1/transactions/${id}/cancel`, { method: 'POST' });
  }

  async getFlaggedTransactions(): Promise<any[]> {
    return this.request('/api/v1/transactions/flagged');
  }

  // ==================== TOKEN MANAGEMENT ====================

  async getTokens(params?: {
    page?: number;
    pageSize?: number;
    chain?: string;
    listed?: boolean;
    search?: string;
  }): Promise<{ data: any[]; total: number; page: number; pageSize: number; totalPages: number }> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    const res = await this.request<{ data: any[]; total: number; page: number; pageSize: number }>(`/api/v1/tokens?${query}`);
    return { ...res, totalPages: Math.ceil((res.total || 0) / (res.pageSize || 20)) };
  }

  async getToken(address: string): Promise<any> {
    return this.request(`/api/v1/tokens/${address}`);
  }

  async createToken(data: {
    address: string;
    name: string;
    symbol: string;
    decimals: number;
    chain: string;
    logoUrl?: string;
    description?: string;
  }): Promise<any> {
    return this.request('/api/v1/tokens', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateToken(address: string, data: Partial<any>): Promise<any> {
    return this.request(`/api/v1/tokens/${address}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteToken(address: string): Promise<void> {
    return this.request(`/api/v1/tokens/${address}`, { method: 'DELETE' });
  }

  async listToken(address: string): Promise<any> {
    return this.request(`/api/v1/tokens/${address}/list`, { method: 'POST' });
  }

  async delistToken(address: string): Promise<any> {
    return this.request(`/api/v1/tokens/${address}/delist`, { method: 'POST' });
  }

  async pauseToken(address: string): Promise<any> {
    return this.request(`/api/v1/tokens/${address}/pause`, { method: 'POST' });
  }

  async unpauseToken(address: string): Promise<any> {
    return this.request(`/api/v1/tokens/${address}/unpause`, { method: 'POST' });
  }

  // ==================== TRADING PAIRS ====================

  async getPairs(params?: {
    page?: number;
    pageSize?: number;
    baseToken?: string;
    quoteToken?: string;
    active?: boolean;
  }): Promise<{ data: any[]; total: number; page: number; pageSize: number }> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    return this.request(`/api/v1/pairs?${query}`);
  }

  async createPair(data: {
    baseToken: string;
    quoteToken: string;
    minTradeAmount: string;
    maxTradeAmount: string;
    makerFee: string;
    takerFee: string;
  }): Promise<any> {
    return this.request('/api/v1/pairs', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updatePair(id: string, data: Partial<any>): Promise<any> {
    return this.request(`/api/v1/pairs/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deletePair(id: string): Promise<void> {
    return this.request(`/api/v1/pairs/${id}`, { method: 'DELETE' });
  }

  async updatePairStatus(id: string, status: boolean): Promise<any> {
    return this.request(`/api/v1/pairs/${id}/status`, {
      method: 'PUT',
      body: JSON.stringify({ active: status }),
    });
  }

  async importPairs(pairs: any[]): Promise<void> {
    return this.request('/api/v1/pairs/import', {
      method: 'POST',
      body: JSON.stringify({ pairs }),
    });
  }

  // ==================== BLOCKCHAIN MANAGEMENT ====================

  async getBlockchains(): Promise<{ data: any[] }> {
    const blockchains = await this.request<any[]>('/api/v1/blockchains');
    return { data: blockchains || [] };
  }

  async getBlockchain(id: string): Promise<any> {
    return this.request(`/api/v1/blockchains/${id}`);
  }

  async createBlockchain(data: {
    name: string;
    symbol: string;
    chainId: number;
    rpcUrl: string;
    explorerUrl: string;
    nativeToken: string;
    isTestnet: boolean;
  }): Promise<any> {
    return this.request('/api/v1/blockchains', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateBlockchain(id: string, data: Partial<any>): Promise<any> {
    return this.request(`/api/v1/blockchains/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteBlockchain(id: string): Promise<void> {
    return this.request(`/api/v1/blockchains/${id}`, { method: 'DELETE' });
  }

  async testBlockchainRpc(id: string): Promise<{ success: boolean; latency: number }> {
    return this.request(`/api/v1/blockchains/${id}/test-rpc`, { method: 'POST' });
  }

  // ==================== FEE CONFIGURATION ====================

  async getFeeConfig(): Promise<any> {
    return this.request('/api/v1/fees');
  }

  async updateFeeConfig(config: any): Promise<any> {
    return this.request('/api/v1/fees', {
      method: 'PUT',
      body: JSON.stringify(config),
    });
  }

  async getFeeHistory(): Promise<any[]> {
    return this.request('/api/v1/fees/history');
  }

  // ==================== WITHDRAWAL MANAGEMENT ====================

  async getWithdrawals(params?: {
    page?: number;
    pageSize?: number;
    status?: string;
    token?: string;
    chain?: string;
  }): Promise<{ data: any[]; total: number; page: number; pageSize: number; totalPages: number }> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    const res = await this.request<{ data: any[]; total: number; page: number; pageSize: number }>(`/api/v1/withdrawals?${query}`);
    return { ...res, totalPages: Math.ceil((res.total || 0) / (res.pageSize || 20)) };
  }

  async getWithdrawal(id: string): Promise<any> {
    return this.request(`/api/v1/withdrawals/${id}`);
  }

  async approveWithdrawal(id: string): Promise<any> {
    return this.request(`/api/v1/withdrawals/${id}/approve`, { method: 'POST' });
  }

  async rejectWithdrawal(id: string, reason: string): Promise<void> {
    return this.request(`/api/v1/withdrawals/${id}/reject`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
  }

  async batchApproveWithdrawals(ids: string[]): Promise<void> {
    return this.request('/api/v1/withdrawals/batch-approve', {
      method: 'POST',
      body: JSON.stringify({ ids }),
    });
  }

  // ==================== WHITE LABEL MANAGEMENT ====================

  async getWhiteLabels(params?: {
    page?: number;
    pageSize?: number;
    status?: string;
  }): Promise<{ data: any[]; total: number; page: number; pageSize: number; totalPages: number }> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    const res = await this.request<{ data: any[]; total: number; page: number; pageSize: number }>(`/api/v1/white-labels?${query}`);
    return { ...res, totalPages: Math.ceil((res.total || 0) / (res.pageSize || 20)) };
  }

  async getWhiteLabel(id: string): Promise<any> {
    return this.request(`/api/v1/white-labels/${id}`);
  }

  async createWhiteLabel(data: {
    name: string;
    domain: string;
    ownerEmail: string;
    ownerName: string;
    logoUrl?: string;
    primaryColor?: string;
    secondaryColor?: string;
  }): Promise<any> {
    return this.request('/api/v1/white-labels', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateWhiteLabel(id: string, data: Partial<any>): Promise<any> {
    return this.request(`/api/v1/white-labels/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteWhiteLabel(id: string): Promise<void> {
    return this.request(`/api/v1/white-labels/${id}`, { method: 'DELETE' });
  }

  async approveWhiteLabel(id: string): Promise<any> {
    return this.request(`/api/v1/white-labels/${id}/approve`, { method: 'POST' });
  }

  // Suspend a white label (backend route is /suspend). rejectWhiteLabel is
  // kept as a backward-compatible alias that suspends (the backend has no
  // separate /reject endpoint).
  async suspendWhiteLabel(id: string): Promise<any> {
    return this.request(`/api/v1/white-labels/${id}/suspend`, { method: 'POST' });
  }

  async rejectWhiteLabel(id: string, reason: string): Promise<void> {
    await this.suspendWhiteLabel(id);
  }

  // SuperAdmin-governed: set which WL products (master_wallet, user_wallet,
  // bots, project_party) a white-label client is permitted to run.
  async setAllowedProducts(id: string, allowedProducts: string[]): Promise<{ id: string; allowed_products: string[] }> {
    return this.request(`/api/v1/white-labels/${id}/allowed-products`, {
      method: 'POST',
      body: JSON.stringify({ allowed_products: allowedProducts }),
    });
  }

  async getWhiteLabelStats(): Promise<{ total: number; active: number; pending: number; suspended: number }> {
    return this.request(`/api/v1/white-labels/stats`);
  }

  // ==================== ANALYTICS ====================

  async getAnalytics(period?: string): Promise<any> {
    const query = period ? `?period=${period}` : '';
    return this.request(`/api/v1/analytics${query}`);
  }

  async getUserAnalytics(period?: string): Promise<any> {
    const query = period ? `?period=${period}` : '';
    return this.request(`/api/v1/analytics/users${query}`);
  }

  async getVolumeAnalytics(period?: string): Promise<any> {
    const query = period ? `?period=${period}` : '';
    return this.request(`/api/v1/analytics/volume${query}`);
  }

  async getRevenueAnalytics(period?: string): Promise<any> {
    const query = period ? `?period=${period}` : '';
    return this.request(`/api/v1/analytics/revenue${query}`);
  }

  async getTransactionAnalytics(period?: string): Promise<any> {
    const query = period ? `?period=${period}` : '';
    return this.request(`/api/v1/analytics/transactions${query}`);
  }

  // ==================== SYSTEM STATUS ====================

  async getSystemStatus(): Promise<any[]> {
    return this.request('/api/v1/system/status');
  }

  async getSystemMetrics(): Promise<any> {
    return this.request('/api/v1/system/metrics');
  }

  async getSystemService(name: string): Promise<any> {
    return this.request(`/api/v1/system/services/${name}`);
  }

  async restartService(name: string): Promise<void> {
    return this.request(`/api/v1/system/services/${name}/restart`, { method: 'POST' });
  }

  // ==================== CONFIGURATION ====================

  async getConfig(): Promise<any> {
    return this.request('/api/v1/config');
  }

  async updateConfig(config: Record<string, any>): Promise<any> {
    return this.request('/api/v1/config', {
      method: 'PUT',
      body: JSON.stringify(config),
    });
  }

  // ==================== AUDIT LOGS ====================

  async getAuditLogs(params?: {
    page?: number;
    pageSize?: number;
    adminId?: string;
    action?: string;
    resource?: string;
    startDate?: string;
    endDate?: string;
  }): Promise<{ data: any[]; total: number; page: number; pageSize: number; totalPages: number }> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    const res = await this.request<{ data: any[]; total: number; page: number; pageSize: number }>(`/api/v1/audit-logs?${query}`);
    return { ...res, totalPages: Math.ceil((res.total || 0) / (res.pageSize || 20)) };
  }

  // ==================== NOTIFICATIONS ====================

  async getNotifications(): Promise<any[]> {
    return this.request('/api/v1/notifications');
  }

  async markNotificationRead(id: string): Promise<void> {
    return this.request(`/api/v1/notifications/${id}/read`, { method: 'PUT' });
  }

  async markAllNotificationsRead(): Promise<void> {
    return this.request('/api/v1/notifications/read-all', { method: 'PUT' });
  }

  // ==================== SESSIONS ====================

  async getSessions(): Promise<any[]> {
    return this.request('/api/v1/sessions');
  }

  async revokeSession(sessionId: string): Promise<void> {
    return this.request(`/api/v1/sessions/${sessionId}`, { method: 'DELETE' });
  }

  async revokeAllSessions(): Promise<void> {
    return this.request('/api/v1/sessions', { method: 'DELETE' });
  }

  // ==================== FEATURE FLAGS ====================

  async getFeatureFlags(): Promise<any[]> {
    return this.request('/api/v1/features');
  }

  async setFeatureFlag(name: string, enabled: boolean): Promise<void> {
    return this.request(`/api/v1/features/${name}`, {
      method: 'PUT',
      body: JSON.stringify({ enabled }),
    });
  }

  // ==================== HEALTH CHECK ====================

  async healthCheck(): Promise<boolean> {
    try {
      await this.request('/api/v1/health');
      return true;
    } catch {
      return false;
    }
  }

  // ---- Liquidity pools ----
  async getLiquidityPools(): Promise<{ pools: any[] }> {
    return this.request('/api/v1/liquidity/pools');
  }
  async getLiquidityPool(id: string): Promise<{ pool: any }> {
    return this.request(`/api/v1/liquidity/pools/${id}`);
  }
  async createLiquidityPool(data: { pair: string; tokenA: string; tokenB: string }): Promise<any> {
    return this.request('/api/v1/liquidity/pools', { method: 'POST', body: JSON.stringify(data) });
  }
  async addLiquidity(id: string, data: { amountA: number; amountB: number }): Promise<any> {
    return this.request(`/api/v1/liquidity/pools/${id}/add`, { method: 'POST', body: JSON.stringify(data) });
  }
  async removeLiquidity(id: string, data: { amount: number }): Promise<any> {
    return this.request(`/api/v1/liquidity/pools/${id}/remove`, { method: 'POST', body: JSON.stringify(data) });
  }
  async getLiquidityStats(): Promise<any> {
    return this.request('/api/v1/liquidity/stats');
  }
}

export const adminApi = new AdminApiService();
export default adminApi;

// Liquidity API facade (used by the Liquidity page)
export const liquidityAPI = {
  getPools: async () => ({ data: (await adminApi.getLiquidityPools()).pools || [] }),
  getStats: async () => ({ data: await adminApi.getLiquidityStats() }),
  createPool: (data: { pair: string; tokenA: string; tokenB: string }) => adminApi.createLiquidityPool(data),
  addLiquidity: (id: string, data: { amountA: number; amountB: number }) => adminApi.addLiquidity(id, data),
  removeLiquidity: (id: string, data: { amount: number }) => adminApi.removeLiquidity(id, data),
};

// Features API facade (real /api/v1/features endpoints)
export const featuresAPI = {
  getAll: async () => ({ data: await adminApi.request<any[]>('/api/v1/features') }),
  toggle: (id: string) => adminApi.request(`/api/v1/features/${id}`, { method: 'PUT', body: JSON.stringify({ enabled: true }) }),
  create: (data: any) => adminApi.request('/api/v1/features', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: any) => adminApi.request(`/api/v1/features/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
};

// Crypto Cards API facade (real /api/v1/crypto-cards endpoints)
export const cryptoCardAPI = {
  getAll: async (status?: string) => ({ data: await adminApi.request<any[]>(`/api/v1/crypto-cards${status ? `?status=${status}` : ''}`) }),
  block: (id: string) => adminApi.request(`/api/v1/crypto-cards/${id}/block`, { method: 'POST', body: JSON.stringify({}) }),
  activate: (id: string) => adminApi.request(`/api/v1/crypto-cards/${id}/activate`, { method: 'POST', body: JSON.stringify({}) }),
};

// Margin Trading API facade (real /api/v1/margin-trading endpoints)
export const marginTradingAPI = {
  getPositions: async () => ({ data: await adminApi.request<any[]>('/api/v1/margin-trading/positions') }),
  getLiquidationStats: async () => ({ data: await adminApi.request<any>('/api/v1/margin-trading/stats') }),
  liquidate: (id: string) => adminApi.request(`/api/v1/margin-trading/positions/${id}/close`, { method: 'POST', body: JSON.stringify({}) }),
};

// P2P Merchant API facade (real /api/v1/p2p-merchants endpoints)
export const p2pMerchantAPI = {
  getMerchants: async (status?: string) => ({ data: await adminApi.request<any[]>(`/api/v1/p2p-merchants${status ? `?status=${status}` : ''}`) }),
  approveMerchant: (id: string) => adminApi.request(`/api/v1/p2p-merchants/${id}/approve`, { method: 'POST', body: JSON.stringify({}) }),
  rejectMerchant: (id: string, reason: string) => adminApi.request(`/api/v1/p2p-merchants/${id}/reject`, { method: 'POST', body: JSON.stringify({ reason }) }),
  getTransactions: async (id: string) => ({ data: await adminApi.request<any[]>(`/api/v1/p2p-merchants/${id}/transactions`) }),
};

// Futures API facade (real /api/v1/futures endpoints)
export const futuresAPI = {
  getAll: async () => ({ data: await adminApi.request<any[]>('/api/v1/futures') }),
  getOne: async (id: string) => ({ data: await adminApi.request<any>(`/api/v1/futures/${id}`) }),
  create: (data: any) => adminApi.request('/api/v1/futures', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: any) => adminApi.request(`/api/v1/futures/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => adminApi.request(`/api/v1/futures/${id}`, { method: 'DELETE' }),
  setStatus: (id: string, status: string) => adminApi.request(`/api/v1/futures/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) }),
};

// Options API facade (real /api/v1/options endpoints)
export const optionsAPI = {
  getAll: async () => ({ data: await adminApi.request<any[]>('/api/v1/options') }),
  getOne: async (id: string) => ({ data: await adminApi.request<any>(`/api/v1/options/${id}`) }),
  create: (data: any) => adminApi.request('/api/v1/options', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: any) => adminApi.request(`/api/v1/options/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => adminApi.request(`/api/v1/options/${id}`, { method: 'DELETE' }),
  setStatus: (id: string, status: string) => adminApi.request(`/api/v1/options/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) }),
};

// Copy Trading API facade (real /api/v1/copy-trading endpoints)
export const copyTradingAPI = {
  getAll: async () => ({ data: await adminApi.request<any[]>('/api/v1/copy-trading') }),
  getOne: async (id: string) => ({ data: await adminApi.request<any>(`/api/v1/copy-trading/${id}`) }),
  create: (data: any) => adminApi.request('/api/v1/copy-trading', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: any) => adminApi.request(`/api/v1/copy-trading/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => adminApi.request(`/api/v1/copy-trading/${id}`, { method: 'DELETE' }),
  setStatus: (id: string, status: string) => adminApi.request(`/api/v1/copy-trading/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) }),
};

// Convert API facade (real /api/v1/convert endpoints)
export const convertAPI = {
  getAll: async () => ({ data: await adminApi.request<any[]>('/api/v1/convert') }),
  getOne: async (id: string) => ({ data: await adminApi.request<any>(`/api/v1/convert/${id}`) }),
  create: (data: any) => adminApi.request('/api/v1/convert', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: any) => adminApi.request(`/api/v1/convert/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => adminApi.request(`/api/v1/convert/${id}`, { method: 'DELETE' }),
  setStatus: (id: string, status: string) => adminApi.request(`/api/v1/convert/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) }),
};

// OnRamp API facade (real /api/v1/onramp endpoints)
export const onRampAPI = {
  getAll: async () => ({ data: await adminApi.request<any[]>('/api/v1/onramp') }),
  getOne: async (id: string) => ({ data: await adminApi.request<any>(`/api/v1/onramp/${id}`) }),
  create: (data: any) => adminApi.request('/api/v1/onramp', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: any) => adminApi.request(`/api/v1/onramp/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => adminApi.request(`/api/v1/onramp/${id}`, { method: 'DELETE' }),
  setStatus: (id: string, status: string) => adminApi.request(`/api/v1/onramp/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) }),
  approve: (id: string) => adminApi.request(`/api/v1/onramp/${id}/approve`, { method: 'POST', body: JSON.stringify({}) }),
  reject: (id: string, reason: string) => adminApi.request(`/api/v1/onramp/${id}/reject`, { method: 'POST', body: JSON.stringify({ reason }) }),
};

// OffRamp API facade (real /api/v1/offramp endpoints)
export const offRampAPI = {
  getAll: async () => ({ data: await adminApi.request<any[]>('/api/v1/offramp') }),
  getOne: async (id: string) => ({ data: await adminApi.request<any>(`/api/v1/offramp/${id}`) }),
  create: (data: any) => adminApi.request('/api/v1/offramp', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: any) => adminApi.request(`/api/v1/offramp/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => adminApi.request(`/api/v1/offramp/${id}`, { method: 'DELETE' }),
  setStatus: (id: string, status: string) => adminApi.request(`/api/v1/offramp/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) }),
  approve: (id: string) => adminApi.request(`/api/v1/offramp/${id}/approve`, { method: 'POST', body: JSON.stringify({}) }),
  reject: (id: string, reason: string) => adminApi.request(`/api/v1/offramp/${id}/reject`, { method: 'POST', body: JSON.stringify({ reason }) }),
};

// P2P Clients API facade (real /api/v1/p2p-clients endpoints)
export const p2pClientsAPI = {
  getAll: async () => ({ data: await adminApi.request<any[]>('/api/v1/p2p-clients') }),
  getOne: async (id: string) => ({ data: await adminApi.request<any>(`/api/v1/p2p-clients/${id}`) }),
  create: (data: any) => adminApi.request('/api/v1/p2p-clients', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: any) => adminApi.request(`/api/v1/p2p-clients/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => adminApi.request(`/api/v1/p2p-clients/${id}`, { method: 'DELETE' }),
  setStatus: (id: string, status: string) => adminApi.request(`/api/v1/p2p-clients/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) }),
};

// Partners API facade (real /api/v1/partners endpoints)
export const partnersAPI = {
  getAll: async () => ({ data: await adminApi.request<any[]>('/api/v1/partners') }),
  getOne: async (id: string) => ({ data: await adminApi.request<any>(`/api/v1/partners/${id}`) }),
  create: (data: any) => adminApi.request('/api/v1/partners', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: any) => adminApi.request(`/api/v1/partners/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => adminApi.request(`/api/v1/partners/${id}`, { method: 'DELETE' }),
  setStatus: (id: string, status: string) => adminApi.request(`/api/v1/partners/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) }),
  approve: (id: string) => adminApi.request(`/api/v1/partners/${id}/approve`, { method: 'POST', body: JSON.stringify({}) }),
  reject: (id: string, reason: string) => adminApi.request(`/api/v1/partners/${id}/reject`, { method: 'POST', body: JSON.stringify({ reason }) }),
};

// Rewards API facade (real /api/v1/rewards endpoints)
export const rewardsAPI = {
  getAll: async () => ({ data: await adminApi.request<any[]>('/api/v1/rewards') }),
  getOne: async (id: string) => ({ data: await adminApi.request<any>(`/api/v1/rewards/${id}`) }),
  create: (data: any) => adminApi.request('/api/v1/rewards', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: any) => adminApi.request(`/api/v1/rewards/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => adminApi.request(`/api/v1/rewards/${id}`, { method: 'DELETE' }),
  setStatus: (id: string, status: string) => adminApi.request(`/api/v1/rewards/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) }),
};

// Marketing API facade (real /api/v1/marketing endpoints)
export const marketingAPI = {
  getAll: async () => ({ data: await adminApi.request<any[]>('/api/v1/marketing') }),
  getOne: async (id: string) => ({ data: await adminApi.request<any>(`/api/v1/marketing/${id}`) }),
  create: (data: any) => adminApi.request('/api/v1/marketing', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: any) => adminApi.request(`/api/v1/marketing/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => adminApi.request(`/api/v1/marketing/${id}`, { method: 'DELETE' }),
  setStatus: (id: string, status: string) => adminApi.request(`/api/v1/marketing/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) }),
};

// RBAC Admin Roles API facade (real /api/v1/roles, /api/v1/permissions, /api/v1/admins endpoints)
export const adminRolesAPI = {
  // Roles
  getRoles: async () => ({ data: await adminApi.request<any[]>('/api/v1/roles') }),
  getRole: async (id: string) => ({ data: await adminApi.request<any>(`/api/v1/roles/${id}`) }),
  createRole: (data: any) => adminApi.request('/api/v1/roles', { method: 'POST', body: JSON.stringify(data) }),
  updateRole: (id: string, data: any) => adminApi.request(`/api/v1/roles/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteRole: (id: string) => adminApi.request(`/api/v1/roles/${id}`, { method: 'DELETE' }),
  // Permissions
  getPermissions: async () => ({ data: await adminApi.request<any[]>('/api/v1/permissions') }),
  getPermission: async (id: string) => ({ data: await adminApi.request<any>(`/api/v1/permissions/${id}`) }),
  createPermission: (data: any) => adminApi.request('/api/v1/permissions', { method: 'POST', body: JSON.stringify(data) }),
  updatePermission: (id: string, data: any) => adminApi.request(`/api/v1/permissions/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deletePermission: (id: string) => adminApi.request(`/api/v1/permissions/${id}`, { method: 'DELETE' }),
  // Admin role/permission assignments
  assignRole: (adminId: string, data: { roleId: string }) => adminApi.request(`/api/v1/admins/${adminId}/roles`, { method: 'POST', body: JSON.stringify(data) }),
  revokeRole: (adminId: string, roleId: string) => adminApi.request(`/api/v1/admins/${adminId}/roles/${roleId}`, { method: 'DELETE' }),
  getEffectivePermissions: async (adminId: string) => ({ data: await adminApi.request<any[]>(`/api/v1/admins/${adminId}/permissions`) }),
};

// Bots API facade (real /api/v1/bots endpoints — governance records only)
export const botsAPI = {
  getAll: async () => ({ data: await adminApi.request<any[]>('/api/v1/bots') }),
  getOne: async (id: string) => ({ data: await adminApi.request<any>(`/api/v1/bots/${id}`) }),
  create: (data: any) => adminApi.request('/api/v1/bots', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: any) => adminApi.request(`/api/v1/bots/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => adminApi.request(`/api/v1/bots/${id}`, { method: 'DELETE' }),
  setStatus: (id: string, status: string) => adminApi.request(`/api/v1/bots/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) }),
  getStats: async () => ({ data: await adminApi.request<any>('/api/v1/bots/stats') }),
  getTiers: async () => ({ data: await adminApi.request<any[]>('/api/v1/bots/tiers') }),
  createTier: (data: any) => adminApi.request('/api/v1/bots/tiers', { method: 'POST', body: JSON.stringify(data) }),
  updateTier: (id: string, data: any) => adminApi.request(`/api/v1/bots/tiers/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteTier: (id: string) => adminApi.request(`/api/v1/bots/tiers/${id}`, { method: 'DELETE' }),
};

// Bots Clients API facade (real /api/v1/bots-clients endpoints — governance records only)
export const botsClientsAPI = {
  getAll: async () => ({ data: await adminApi.request<any[]>('/api/v1/bots-clients') }),
  getOne: async (id: string) => ({ data: await adminApi.request<any>(`/api/v1/bots-clients/${id}`) }),
  create: (data: any) => adminApi.request('/api/v1/bots-clients', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: any) => adminApi.request(`/api/v1/bots-clients/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => adminApi.request(`/api/v1/bots-clients/${id}`, { method: 'DELETE' }),
  setStatus: (id: string, status: string) => adminApi.request(`/api/v1/bots-clients/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) }),
};

// Project Teams API facade (real /api/v1/project-teams endpoints — governance records only)
export const projectTeamsAPI = {
  getAll: async () => ({ data: await adminApi.request<any[]>('/api/v1/project-teams') }),
  getOne: async (id: string) => ({ data: await adminApi.request<any>(`/api/v1/project-teams/${id}`) }),
  create: (data: any) => adminApi.request('/api/v1/project-teams', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: any) => adminApi.request(`/api/v1/project-teams/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => adminApi.request(`/api/v1/project-teams/${id}`, { method: 'DELETE' }),
  setStatus: (id: string, status: string) => adminApi.request(`/api/v1/project-teams/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) }),
  getMembers: async (id: string) => ({ data: await adminApi.request<any[]>(`/api/v1/project-teams/${id}/members`) }),
  addMember: (id: string, data: any) => adminApi.request(`/api/v1/project-teams/${id}/members`, { method: 'POST', body: JSON.stringify(data) }),
  removeMember: (id: string, memberId: string) => adminApi.request(`/api/v1/project-teams/${id}/members/${memberId}`, { method: 'DELETE' }),
};

// Liquidity Sources API facade (real /api/v1/liquidity-sources endpoints — governance records only)
export const liquiditySourcesAPI = {
  getAll: async () => ({ data: await adminApi.request<any[]>('/api/v1/liquidity-sources') }),
  getOne: async (id: string) => ({ data: await adminApi.request<any>(`/api/v1/liquidity-sources/${id}`) }),
  create: (data: any) => adminApi.request('/api/v1/liquidity-sources', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: any) => adminApi.request(`/api/v1/liquidity-sources/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => adminApi.request(`/api/v1/liquidity-sources/${id}`, { method: 'DELETE' }),
  setStatus: (id: string, status: string) => adminApi.request(`/api/v1/liquidity-sources/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) }),
  setPriority: (id: string, priority: number) => adminApi.request(`/api/v1/liquidity-sources/${id}/priority`, { method: 'PUT', body: JSON.stringify({ priority }) }),
  healthCheck: (id: string, data: any) => adminApi.request(`/api/v1/liquidity-sources/${id}/health-check`, { method: 'POST', body: JSON.stringify(data) }),
  getStats: async () => ({ data: await adminApi.request<any>('/api/v1/liquidity-sources/stats') }),
};
