// TigerWallet Admin - Complete API Service
// Connects to Go Admin Service and Rust Admin Fetchers
// Uses PostgreSQL and Redis on the backend

const API_BASE_URL = process.env.NEXT_PUBLIC_ADMIN_API || 'http://localhost:8080';
const WS_BASE_URL = process.env.NEXT_PUBLIC_ADMIN_WS || 'ws://localhost:8080';

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

  private async request<T>(
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
  
  async login(email: string, password: string): Promise<{ token: string; admin: any }> {
    return this.request('/api/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
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

  async suspendAdmin(id: string, reason: string): Promise<void> {
    return this.request(`/api/v1/admins/${id}/suspend`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
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
  }): Promise<{ data: any[]; total: number; page: number; pageSize: number }> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    return this.request(`/api/v1/kyc?${query}`);
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
  }): Promise<{ data: any[]; total: number; page: number; pageSize: number }> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    return this.request(`/api/v1/transactions?${query}`);
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
  }): Promise<{ data: any[]; total: number; page: number; pageSize: number }> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    return this.request(`/api/v1/tokens?${query}`);
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

  async getBlockchains(): Promise<any[]> {
    return this.request('/api/v1/blockchains');
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
  }): Promise<{ data: any[]; total: number; page: number; pageSize: number }> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    return this.request(`/api/v1/withdrawals?${query}`);
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
  }): Promise<{ data: any[]; total: number; page: number; pageSize: number }> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    return this.request(`/api/v1/whitelabels?${query}`);
  }

  async getWhiteLabel(id: string): Promise<any> {
    return this.request(`/api/v1/whitelabels/${id}`);
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
    return this.request('/api/v1/whitelabels', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateWhiteLabel(id: string, data: Partial<any>): Promise<any> {
    return this.request(`/api/v1/whitelabels/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteWhiteLabel(id: string): Promise<void> {
    return this.request(`/api/v1/whitelabels/${id}`, { method: 'DELETE' });
  }

  async approveWhiteLabel(id: string): Promise<any> {
    return this.request(`/api/v1/whitelabels/${id}/approve`, { method: 'POST' });
  }

  async rejectWhiteLabel(id: string, reason: string): Promise<void> {
    return this.request(`/api/v1/whitelabels/${id}/reject`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
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
  }): Promise<{ data: any[]; total: number; page: number; pageSize: number }> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    return this.request(`/api/v1/audit-logs?${query}`);
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
}

export const adminApi = new AdminApiService();
export default adminApi;
