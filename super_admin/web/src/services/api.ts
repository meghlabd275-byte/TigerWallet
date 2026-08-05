/**
 * TigerWallet Super Admin - Complete API Service
 * High-performance API client with full functionality
 * Connected to Go, Rust, and C++ backends
 */

import type {
  User, UserCreateInput, UserUpdateInput,
  KYCRequest, KYCApproveInput, KYCRejectInput,
  Transaction, TransactionFlagInput, TransactionListParams,
  Withdrawal, WithdrawalApproveInput, WithdrawalRejectInput,
  Token, TokenCreateInput, TokenUpdateInput,
  Blockchain, BlockchainCreateInput,
  TradingPair, TradingPairCreateInput,
  FeeStructure, FeeCreateInput,
  WhiteLabel, WhiteLabelCreateInput, WhiteLabelUpdateInput,
  Admin, AdminCreateInput, AdminUpdateInput,
  Ticket, TicketCreateInput, TicketMessageInput,
  Article, ArticleCreateInput,
  ApprovalWorkflow, ApprovalRequest, ApprovalActionInput,
  DashboardStats, AnalyticsData, ComplianceReport, FinanceReport, SecurityAlert,
  APIKey, Webhook, WebhookCreateInput,
  AuditLog, AuditLogParams,
  SystemStatus, SystemMetrics,
  Notification,
  PaginatedResponse, ApiResponse
} from '../types';

const API_BASE_URL = process.env.NEXT_PUBLIC_SUPER_ADMIN_API || 'http://localhost:9090';
const WS_BASE_URL = process.env.NEXT_PUBLIC_SUPER_ADMIN_WS || 'ws://localhost:9090';

class SuperAdminApiService {
  private baseURL: string;
  private wsURL: string;
  private token: string | null = null;

  constructor() {
    this.baseURL = API_BASE_URL;
    this.wsURL = WS_BASE_URL;
    if (typeof window !== 'undefined') {
      this.token = localStorage.getItem('super_admin_token');
    }
  }

  setToken(token: string) {
    this.token = token;
    if (typeof window !== 'undefined') {
      localStorage.setItem('super_admin_token', token);
    }
  }

  clearToken() {
    this.token = null;
    if (typeof window !== 'undefined') {
      localStorage.removeItem('super_admin_token');
    }
  }

  private getHeaders(): HeadersInit {
    return {
      'Content-Type': 'application/json',
      ...(this.token && { 'Authorization': `Bearer ${this.token}` }),
    };
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.baseURL}${endpoint}`;
    
    const response = await fetch(url, {
      ...options,
      headers: {
        ...this.getHeaders(),
        ...options.headers,
      },
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ message: 'Unknown error' }));
      throw new Error(error.message || `API Error: ${response.status}`);
    }

    return response.json();
  }

  // ==================== Authentication ====================

  async login(email: string, password: string): Promise<{ token: string; admin: Admin }> {
    return this.request('/api/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    });
  }

  async loginWith2FA(email: string, password: string, code: string): Promise<{ token: string; admin: Admin }> {
    return this.request('/api/v1/auth/login/2fa', {
      method: 'POST',
      body: JSON.stringify({ email, password, code }),
    });
  }

  async register(data: { email: string; password: string; username: string }): Promise<{ token: string; admin: Admin }> {
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

  async enable2FA(): Promise<{ secret: string; qrCode: string }> {
    return this.request('/api/v1/auth/2fa/enable', { method: 'POST' });
  }

  async disable2FA(code: string): Promise<void> {
    return this.request('/api/v1/auth/2fa/disable', {
      method: 'POST',
      body: JSON.stringify({ code }),
    });
  }

  async verify2FA(code: string): Promise<{ verified: boolean }> {
    return this.request('/api/v1/auth/2fa/verify', {
      method: 'POST',
      body: JSON.stringify({ code }),
    });
  }

  // ==================== Dashboard & Analytics ====================

  async getDashboardStats(): Promise<DashboardStats> {
    return this.request('/api/v1/dashboard/stats');
  }

  async getAnalytics(period: string = '24h'): Promise<AnalyticsData> {
    return this.request(`/api/v1/analytics?period=${period}`);
  }

  async getUserAnalytics(period: string = '24h'): Promise<AnalyticsData> {
    return this.request(`/api/v1/analytics/users?period=${period}`);
  }

  async getVolumeAnalytics(period: string = '24h'): Promise<AnalyticsData> {
    return this.request(`/api/v1/analytics/volume?period=${period}`);
  }

  async getRevenueAnalytics(period: string = '24h'): Promise<AnalyticsData> {
    return this.request(`/api/v1/analytics/revenue?period=${period}`);
  }

  async getTransactionAnalytics(period: string = '24h'): Promise<AnalyticsData> {
    return this.request(`/api/v1/analytics/transactions?period=${period}`);
  }

  async getChainAnalytics(): Promise<any> {
    return this.request('/api/v1/analytics/chains');
  }

  // ==================== User Management ====================

  async getUsers(params?: {
    page?: number;
    page_size?: number;
    status?: string;
    kyc_status?: string;
    search?: string;
    sort_by?: string;
    sort_order?: 'ASC' | 'DESC';
  }): Promise<PaginatedResponse<User>> {
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

  async getUser(id: string): Promise<User> {
    return this.request(`/api/v1/users/${id}`);
  }

  async createUser(data: UserCreateInput): Promise<User> {
    return this.request('/api/v1/users', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateUser(id: string, data: UserUpdateInput): Promise<User> {
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

  async banUser(id: string, reason: string): Promise<void> {
    return this.request(`/api/v1/users/${id}/ban`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
  }

  async unbanUser(id: string): Promise<void> {
    return this.request(`/api/v1/users/${id}/unban`, { method: 'POST' });
  }

  async verifyUser(id: string): Promise<void> {
    return this.request(`/api/v1/users/${id}/verify`, { method: 'POST' });
  }

  async getUserTransactions(userId: string, params?: TransactionListParams): Promise<PaginatedResponse<Transaction>> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    return this.request(`/api/v1/users/${userId}/transactions?${query}`);
  }

  async getUserWithdrawals(userId: string): Promise<PaginatedResponse<Withdrawal>> {
    return this.request(`/api/v1/users/${userId}/withdrawals`);
  }

  // ==================== KYC Management ====================

  async getKYCRequests(params?: {
    page?: number;
    page_size?: number;
    status?: string;
    risk_level?: string;
    search?: string;
  }): Promise<PaginatedResponse<KYCRequest>> {
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

  async getKYCRequest(id: string): Promise<KYCRequest> {
    return this.request(`/api/v1/kyc/${id}`);
  }

  async approveKYC(id: string, input?: KYCApproveInput): Promise<KYCRequest> {
    return this.request(`/api/v1/kyc/${id}/approve`, {
      method: 'POST',
      body: JSON.stringify(input || {}),
    });
  }

  async rejectKYC(id: string, input: KYCRejectInput): Promise<void> {
    return this.request(`/api/v1/kyc/${id}/reject`, {
      method: 'POST',
      body: JSON.stringify(input),
    });
  }

  async requestKYCResubmission(id: string, reason: string): Promise<void> {
    return this.request(`/api/v1/kyc/${id}/resubmit`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
  }

  async bulkApproveKYC(ids: string[]): Promise<{ approved: number; failed: number }> {
    return this.request('/api/v1/kyc/bulk-approve', {
      method: 'POST',
      body: JSON.stringify({ ids }),
    });
  }

  // ==================== Transaction Management ====================

  async getTransactions(params?: TransactionListParams): Promise<PaginatedResponse<Transaction>> {
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

  async getTransaction(id: string): Promise<Transaction> {
    return this.request(`/api/v1/transactions/${id}`);
  }

  async flagTransaction(id: string, input: TransactionFlagInput): Promise<Transaction> {
    return this.request(`/api/v1/transactions/${id}/flag`, {
      method: 'POST',
      body: JSON.stringify(input),
    });
  }

  async unflagTransaction(id: string): Promise<Transaction> {
    return this.request(`/api/v1/transactions/${id}/unflag`, { method: 'POST' });
  }

  async cancelTransaction(id: string, reason: string): Promise<void> {
    return this.request(`/api/v1/transactions/${id}/cancel`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
  }

  async getFlaggedTransactions(params?: TransactionListParams): Promise<PaginatedResponse<Transaction>> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    return this.request(`/api/v1/transactions/flagged?${query}`);
  }

  async getSuspiciousTransactions(params?: TransactionListParams): Promise<PaginatedResponse<Transaction>> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    return this.request(`/api/v1/transactions/suspicious?${query}`);
  }

  // ==================== Withdrawal Management ====================

  async getWithdrawals(params?: {
    page?: number;
    page_size?: number;
    status?: string;
    chain?: string;
    currency?: string;
    is_urgent?: boolean;
    start_date?: string;
    end_date?: string;
  }): Promise<PaginatedResponse<Withdrawal>> {
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

  async getWithdrawal(id: string): Promise<Withdrawal> {
    return this.request(`/api/v1/withdrawals/${id}`);
  }

  async approveWithdrawal(id: string, input?: WithdrawalApproveInput): Promise<Withdrawal> {
    return this.request(`/api/v1/withdrawals/${id}/approve`, {
      method: 'POST',
      body: JSON.stringify(input || {}),
    });
  }

  async rejectWithdrawal(id: string, input: WithdrawalRejectInput): Promise<void> {
    return this.request(`/api/v1/withdrawals/${id}/reject`, {
      method: 'POST',
      body: JSON.stringify(input),
    });
  }

  async processWithdrawal(id: string): Promise<Withdrawal> {
    return this.request(`/api/v1/withdrawals/${id}/process`, { method: 'POST' });
  }

  async batchApproveWithdrawals(ids: string[]): Promise<{ approved: number; failed: number }> {
    return this.request('/api/v1/withdrawals/batch-approve', {
      method: 'POST',
      body: JSON.stringify({ ids }),
    });
  }

  async getPendingWithdrawalsCount(): Promise<{ count: number; total_amount: number }> {
    return this.request('/api/v1/withdrawals/pending/count');
  }

  // ==================== Token Management ====================

  async getTokens(params?: {
    page?: number;
    page_size?: number;
    status?: string;
    chain?: string;
    search?: string;
  }): Promise<PaginatedResponse<Token>> {
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

  async getToken(address: string): Promise<Token> {
    return this.request(`/api/v1/tokens/${address}`);
  }

  async createToken(data: TokenCreateInput): Promise<Token> {
    return this.request('/api/v1/tokens', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateToken(address: string, data: TokenUpdateInput): Promise<Token> {
    return this.request(`/api/v1/tokens/${address}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteToken(address: string): Promise<void> {
    return this.request(`/api/v1/tokens/${address}`, { method: 'DELETE' });
  }

  async listToken(address: string): Promise<Token> {
    return this.request(`/api/v1/tokens/${address}/list`, { method: 'POST' });
  }

  async delistToken(address: string): Promise<Token> {
    return this.request(`/api/v1/tokens/${address}/delist`, { method: 'POST' });
  }

  async pauseToken(address: string): Promise<Token> {
    return this.request(`/api/v1/tokens/${address}/pause`, { method: 'POST' });
  }

  async unpauseToken(address: string): Promise<Token> {
    return this.request(`/api/v1/tokens/${address}/unpause`, { method: 'POST' });
  }

  async verifyToken(address: string): Promise<Token> {
    return this.request(`/api/v1/tokens/${address}/verify`, { method: 'POST' });
  }

  // ==================== Blockchain Management ====================

  async getBlockchains(params?: {
    page?: number;
    page_size?: number;
    is_active?: boolean;
    is_evm?: boolean;
  }): Promise<PaginatedResponse<Blockchain>> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    return this.request(`/api/v1/blockchains?${query}`);
  }

  async getBlockchain(id: string): Promise<Blockchain> {
    return this.request(`/api/v1/blockchains/${id}`);
  }

  async createBlockchain(data: BlockchainCreateInput): Promise<Blockchain> {
    return this.request('/api/v1/blockchains', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateBlockchain(id: string, data: Partial<BlockchainCreateInput>): Promise<Blockchain> {
    return this.request(`/api/v1/blockchains/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteBlockchain(id: string): Promise<void> {
    return this.request(`/api/v1/blockchains/${id}`, { method: 'DELETE' });
  }

  async testBlockchainRPC(id: string): Promise<{ success: boolean; latency_ms: number; block_number: number }> {
    return this.request(`/api/v1/blockchains/${id}/test-rpc`, { method: 'POST' });
  }

  async activateBlockchain(id: string): Promise<Blockchain> {
    return this.request(`/api/v1/blockchains/${id}/activate`, { method: 'POST' });
  }

  async deactivateBlockchain(id: string): Promise<Blockchain> {
    return this.request(`/api/v1/blockchains/${id}/deactivate`, { method: 'POST' });
  }

  // ==================== Trading Pair Management ====================

  async getTradingPairs(params?: {
    page?: number;
    page_size?: number;
    status?: string;
    chain?: string;
    search?: string;
  }): Promise<PaginatedResponse<TradingPair>> {
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

  async getTradingPair(id: string): Promise<TradingPair> {
    return this.request(`/api/v1/pairs/${id}`);
  }

  async createTradingPair(data: TradingPairCreateInput): Promise<TradingPair> {
    return this.request('/api/v1/pairs', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateTradingPair(id: string, data: Partial<TradingPairCreateInput>): Promise<TradingPair> {
    return this.request(`/api/v1/pairs/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteTradingPair(id: string): Promise<void> {
    return this.request(`/api/v1/pairs/${id}`, { method: 'DELETE' });
  }

  async suspendTradingPair(id: string): Promise<TradingPair> {
    return this.request(`/api/v1/pairs/${id}/suspend`, { method: 'POST' });
  }

  async resumeTradingPair(id: string): Promise<TradingPair> {
    return this.request(`/api/v1/pairs/${id}/resume`, { method: 'POST' });
  }

  async haltTradingPair(id: string): Promise<TradingPair> {
    return this.request(`/api/v1/pairs/${id}/halt`, { method: 'POST' });
  }

  async importTradingPairs(pairs: TradingPairCreateInput[]): Promise<{ imported: number; failed: number }> {
    return this.request('/api/v1/pairs/import', {
      method: 'POST',
      body: JSON.stringify({ pairs }),
    });
  }

  // ==================== Fee Management ====================

  async getFeeStructures(params?: {
    page?: number;
    page_size?: number;
    fee_type?: string;
    asset?: string;
    is_active?: boolean;
  }): Promise<PaginatedResponse<FeeStructure>> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    return this.request(`/api/v1/fees?${query}`);
  }

  async getFeeStructure(id: string): Promise<FeeStructure> {
    return this.request(`/api/v1/fees/${id}`);
  }

  async createFeeStructure(data: FeeCreateInput): Promise<FeeStructure> {
    return this.request('/api/v1/fees', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateFeeStructure(id: string, data: Partial<FeeCreateInput>): Promise<FeeStructure> {
    return this.request(`/api/v1/fees/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteFeeStructure(id: string): Promise<void> {
    return this.request(`/api/v1/fees/${id}`, { method: 'DELETE' });
  }

  async activateFeeStructure(id: string): Promise<FeeStructure> {
    return this.request(`/api/v1/fees/${id}/activate`, { method: 'POST' });
  }

  async deactivateFeeStructure(id: string): Promise<FeeStructure> {
    return this.request(`/api/v1/fees/${id}/deactivate`, { method: 'POST' });
  }

  async getFeeHistory(params?: {
    page?: number;
    page_size?: number;
    fee_type?: string;
    start_date?: string;
    end_date?: string;
  }): Promise<PaginatedResponse<any>> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    return this.request(`/api/v1/fees/history?${query}`);
  }

  // ==================== White Label Management ====================

  async getWhiteLabels(params?: {
    page?: number;
    page_size?: number;
    status?: string;
    plan?: string;
    search?: string;
  }): Promise<PaginatedResponse<WhiteLabel>> {
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

  async getWhiteLabel(id: string): Promise<WhiteLabel> {
    return this.request(`/api/v1/whitelabels/${id}`);
  }

  async createWhiteLabel(data: WhiteLabelCreateInput): Promise<WhiteLabel> {
    return this.request('/api/v1/whitelabels', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateWhiteLabel(id: string, data: WhiteLabelUpdateInput): Promise<WhiteLabel> {
    return this.request(`/api/v1/whitelabels/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteWhiteLabel(id: string): Promise<void> {
    return this.request(`/api/v1/whitelabels/${id}`, { method: 'DELETE' });
  }

  async approveWhiteLabel(id: string): Promise<WhiteLabel> {
    return this.request(`/api/v1/whitelabels/${id}/approve`, { method: 'POST' });
  }

  async rejectWhiteLabel(id: string, reason: string): Promise<void> {
    return this.request(`/api/v1/whitelabels/${id}/reject`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
  }

  async suspendWhiteLabel(id: string): Promise<WhiteLabel> {
    return this.request(`/api/v1/whitelabels/${id}/suspend', { method: 'POST' });
  }

  async activateWhiteLabel(id: string): Promise<WhiteLabel> {
    return this.request(`/api/v1/whitelabels/${id}/activate', { method: 'POST' });
  }

  async updateWhiteLabelFee(id: string, feePercent: number): Promise<WhiteLabel> {
    return this.request(`/api/v1/whitelabels/${id}/fee`, {
      method: 'PUT',
      body: JSON.stringify({ fee_percent: feePercent }),
    });
  }

  async regenerateWhiteLabelAPIKey(id: string): Promise<{ api_key: string }> {
    return this.request(`/api/v1/whitelabels/${id}/regenerate-key', { method: 'POST' });
  }

  async verifyWhiteLabelDomain(id: string, domain: string): Promise<{ verified: boolean }> {
    return this.request(`/api/v1/whitelabels/${id}/verify-domain', {
      method: 'POST',
      body: JSON.stringify({ domain }),
    });
  }

  // ==================== Admin Management ====================

  async getAdmins(params?: {
    page?: number;
    page_size?: number;
    role?: string;
    status?: string;
    search?: string;
  }): Promise<PaginatedResponse<Admin>> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    return this.request(`/api/v1/admins?${query}`);
  }

  async getAdmin(id: string): Promise<Admin> {
    return this.request(`/api/v1/admins/${id}`);
  }

  async createAdmin(data: AdminCreateInput): Promise<Admin> {
    return this.request('/api/v1/admins', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateAdmin(id: string, data: AdminUpdateInput): Promise<Admin> {
    return this.request(`/api/v1/admins/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteAdmin(id: string): Promise<void> {
    return this.request(`/api/v1/admins/${id}', { method: 'DELETE' });
  }

  async suspendAdmin(id: string, reason: string): Promise<void> {
    return this.request(`/api/v1/admins/${id}/suspend', {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
  }

  async activateAdmin(id: string): Promise<void> {
    return this.request(`/api/v1/admins/${id}/activate', { method: 'POST' });
  }

  async updateAdminPermissions(id: string, permissions: string[]): Promise<Admin> {
    return this.request(`/api/v1/admins/${id}/permissions', {
      method: 'PUT',
      body: JSON.stringify({ permissions }),
    });
  }

  async getAdminSessions(adminId: string): Promise<any[]> {
    return this.request(`/api/v1/admins/${adminId}/sessions`);
  }

  async revokeAdminSession(adminId: string, sessionId: string): Promise<void> {
    return this.request(`/api/v1/admins/${adminId}/sessions/${sessionId}`, { method: 'DELETE' });
  }

  async revokeAllAdminSessions(adminId: string): Promise<void> {
    return this.request(`/api/v1/admins/${adminId}/sessions`, { method: 'DELETE' });
  }

  // ==================== Ticket System ====================

  async getTickets(params?: {
    page?: number;
    page_size?: number;
    status?: string;
    priority?: string;
    category?: string;
    assigned_to?: string;
    search?: string;
  }): Promise<PaginatedResponse<Ticket>> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    return this.request(`/api/v1/tickets?${query}`);
  }

  async getTicket(id: string): Promise<Ticket> {
    return this.request(`/api/v1/tickets/${id}`);
  }

  async createTicket(data: TicketCreateInput): Promise<Ticket> {
    return this.request('/api/v1/tickets', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateTicket(id: string, data: { status?: string; priority?: string; assigned_to?: string }): Promise<Ticket> {
    return this.request(`/api/v1/tickets/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async addTicketMessage(ticketId: string, data: TicketMessageInput): Promise<Ticket> {
    return this.request(`/api/v1/tickets/${ticketId}/messages`, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async assignTicket(ticketId: string, adminId: string): Promise<Ticket> {
    return this.request(`/api/v1/tickets/${ticketId}/assign`, {
      method: 'POST',
      body: JSON.stringify({ admin_id: adminId }),
    });
  }

  async closeTicket(ticketId: string): Promise<Ticket> {
    return this.request(`/api/v1/tickets/${ticketId}/close', { method: 'POST' });
  }

  // ==================== Knowledge Base ====================

  async getArticles(params?: {
    page?: number;
    page_size?: number;
    status?: string;
    category?: string;
    search?: string;
  }): Promise<PaginatedResponse<Article>> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    return this.request(`/api/v1/knowledge-base?${query}`);
  }

  async getArticle(id: string): Promise<Article> {
    return this.request(`/api/v1/knowledge-base/${id}`);
  }

  async createArticle(data: ArticleCreateInput): Promise<Article> {
    return this.request('/api/v1/knowledge-base', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateArticle(id: string, data: Partial<ArticleCreateInput>): Promise<Article> {
    return this.request(`/api/v1/knowledge-base/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteArticle(id: string): Promise<void> {
    return this.request(`/api/v1/knowledge-base/${id}', { method: 'DELETE' });
  }

  async publishArticle(id: string): Promise<Article> {
    return this.request(`/api/v1/knowledge-base/${id}/publish', { method: 'POST' });
  }

  async archiveArticle(id: string): Promise<Article> {
    return this.request(`/api/v1/knowledge-base/${id}/archive', { method: 'POST' });
  }

  // ==================== Approval Workflows ====================

  async getApprovalWorkflows(): Promise<ApprovalWorkflow[]> {
    return this.request('/api/v1/workflows');
  }

  async getApprovalWorkflow(id: string): Promise<ApprovalWorkflow> {
    return this.request(`/api/v1/workflows/${id}`);
  }

  async createApprovalWorkflow(data: Partial<ApprovalWorkflow>): Promise<ApprovalWorkflow> {
    return this.request('/api/v1/workflows', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateApprovalWorkflow(id: string, data: Partial<ApprovalWorkflow>): Promise<ApprovalWorkflow> {
    return this.request(`/api/v1/workflows/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteApprovalWorkflow(id: string): Promise<void> {
    return this.request(`/api/v1/workflows/${id}', { method: 'DELETE' });
  }

  async getApprovalRequests(params?: {
    page?: number;
    page_size?: number;
    status?: string;
    workflow_id?: string;
  }): Promise<PaginatedResponse<ApprovalRequest>> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    return this.request(`/api/v1/approvals?${query}`);
  }

  async getApprovalRequest(id: string): Promise<ApprovalRequest> {
    return this.request(`/api/v1/approvals/${id}`);
  }

  async takeApprovalAction(requestId: string, input: ApprovalActionInput): Promise<ApprovalRequest> {
    return this.request(`/api/v1/approvals/${requestId}/action`, {
      method: 'POST',
      body: JSON.stringify(input),
    });
  }

  async cancelApprovalRequest(requestId: string): Promise<ApprovalRequest> {
    return this.request(`/api/v1/approvals/${requestId}/cancel', { method: 'POST' });
  }

  // ==================== Reports ====================

  async getComplianceReports(params?: {
    page?: number;
    page_size?: number;
    type?: string;
    status?: string;
  }): Promise<PaginatedResponse<ComplianceReport>> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    return this.request(`/api/v1/reports/compliance?${query}`);
  }

  async generateComplianceReport(type: string, startDate: string, endDate: string): Promise<ComplianceReport> {
    return this.request('/api/v1/reports/compliance/generate', {
      method: 'POST',
      body: JSON.stringify({ type, start_date: startDate, end_date: endDate }),
    });
  }

  async getFinanceReports(params?: {
    page?: number;
    page_size?: number;
    type?: string;
    period?: string;
  }): Promise<PaginatedResponse<FinanceReport>> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    return this.request(`/api/v1/reports/finance?${query}`);
  }

  async generateFinanceReport(type: string, period: string): Promise<FinanceReport> {
    return this.request('/api/v1/reports/finance/generate', {
      method: 'POST',
      body: JSON.stringify({ type, period }),
    });
  }

  // ==================== Security Alerts ====================

  async getSecurityAlerts(params?: {
    page?: number;
    page_size?: number;
    severity?: string;
    status?: string;
  }): Promise<PaginatedResponse<SecurityAlert>> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    return this.request(`/api/v1/security/alerts?${query}`);
  }

  async resolveSecurityAlert(id: string, resolution: string): Promise<SecurityAlert> {
    return this.request(`/api/v1/security/alerts/${id}/resolve', {
      method: 'POST',
      body: JSON.stringify({ resolution }),
    });
  }

  async markSecurityAlertAsFalsePositive(id: string): Promise<SecurityAlert> {
    return this.request(`/api/v1/security/alerts/${id}/false-positive', { method: 'POST' });
  }

  // ==================== API Keys ====================

  async getAPIKeys(params?: {
    page?: number;
    page_size?: number;
    user_id?: string;
    is_active?: boolean;
  }): Promise<PaginatedResponse<APIKey>> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    return this.request(`/api/v1/api-keys?${query}`);
  }

  async createAPIKey(data: {
    name: string;
    user_id: string;
    permissions: string[];
    rate_limit_per_minute?: number;
    rate_limit_per_day?: number;
    expires_at?: string;
  }): Promise<APIKey> {
    return this.request('/api/v1/api-keys', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async revokeAPIKey(id: string): Promise<void> {
    return this.request(`/api/v1/api-keys/${id}/revoke', { method: 'POST' });
  }

  async regenerateAPIKey(id: string): Promise<{ key: string }> {
    return this.request(`/api/v1/api-keys/${id}/regenerate', { method: 'POST' });
  }

  // ==================== Webhooks ====================

  async getWebhooks(params?: {
    page?: number;
    page_size?: number;
    is_active?: boolean;
  }): Promise<PaginatedResponse<Webhook>> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    return this.request(`/api/v1/webhooks?${query}`);
  }

  async getWebhook(id: string): Promise<Webhook> {
    return this.request(`/api/v1/webhooks/${id}`);
  }

  async createWebhook(data: WebhookCreateInput): Promise<Webhook> {
    return this.request('/api/v1/webhooks', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateWebhook(id: string, data: Partial<WebhookCreateInput>): Promise<Webhook> {
    return this.request(`/api/v1/webhooks/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteWebhook(id: string): Promise<void> {
    return this.request(`/api/v1/webhooks/${id}', { method: 'DELETE' });
  }

  async testWebhook(id: string): Promise<{ success: boolean; response_time_ms: number }> {
    return this.request(`/api/v1/webhooks/${id}/test', { method: 'POST' });
  }

  async activateWebhook(id: string): Promise<Webhook> {
    return this.request(`/api/v1/webhooks/${id}/activate', { method: 'POST' });
  }

  async deactivateWebhook(id: string): Promise<Webhook> {
    return this.request(`/api/v1/webhooks/${id}/deactivate', { method: 'POST' });
  }

  // ==================== Audit Logs ====================

  async getAuditLogs(params?: AuditLogParams): Promise<PaginatedResponse<AuditLog>> {
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

  async getAuditLog(id: string): Promise<AuditLog> {
    return this.request(`/api/v1/audit-logs/${id}`);
  }

  async exportAuditLogs(params: AuditLogParams & { format: 'csv' | 'json' | 'pdf' }): Promise<{ download_url: string }> {
    const query = new URLSearchParams();
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null) {
        query.append(key, String(value));
      }
    });
    return this.request(`/api/v1/audit-logs/export?${query}`);
  }

  // ==================== System ====================

  async getSystemStatus(): Promise<SystemStatus[]> {
    return this.request('/api/v1/system/status');
  }

  async getSystemMetrics(): Promise<SystemMetrics> {
    return this.request('/api/v1/system/metrics');
  }

  async getSystemService(name: string): Promise<SystemStatus> {
    return this.request(`/api/v1/system/services/${name}`);
  }

  async restartService(name: string): Promise<void> {
    return this.request(`/api/v1/system/services/${name}/restart', { method: 'POST' });
  }

  async getSystemLogs(params?: {
    service?: string;
    level?: string;
    start_date?: string;
    end_date?: string;
    page?: number;
    page_size?: number;
  }): Promise<PaginatedResponse<any>> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    return this.request(`/api/v1/system/logs?${query}`);
  }

  // ==================== Configuration ====================

  async getConfig(): Promise<Record<string, any>> {
    return this.request('/api/v1/config');
  }

  async updateConfig(config: Record<string, any>): Promise<Record<string, any>> {
    return this.request('/api/v1/config', {
      method: 'PUT',
      body: JSON.stringify(config),
    });
  }

  async getConfigHistory(params?: {
    page?: number;
    page_size?: number;
  }): Promise<PaginatedResponse<any>> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    return this.request(`/api/v1/config/history?${query}`);
  }

  // ==================== Notifications ====================

  async getNotifications(params?: {
    page?: number;
    page_size?: number;
    is_read?: boolean;
    type?: string;
  }): Promise<PaginatedResponse<Notification>> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          query.append(key, String(value));
        }
      });
    }
    return this.request(`/api/v1/notifications?${query}`);
  }

  async markNotificationRead(id: string): Promise<void> {
    return this.request(`/api/v1/notifications/${id}/read', { method: 'PUT' });
  }

  async markAllNotificationsRead(): Promise<void> {
    return this.request('/api/v1/notifications/read-all', { method: 'PUT' });
  }

  async deleteNotification(id: string): Promise<void> {
    return this.request(`/api/v1/notifications/${id}', { method: 'DELETE' });
  }

  async sendNotification(data: {
    user_id?: string;
    title: string;
    message: string;
    type?: string;
    link?: string;
  }): Promise<Notification> {
    return this.request('/api/v1/notifications/send', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async broadcastNotification(data: {
    title: string;
    message: string;
    type?: string;
    target_roles?: string[];
  }): Promise<{ sent: number }> {
    return this.request('/api/v1/notifications/broadcast', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  // ==================== Sessions ====================

  async getSessions(): Promise<any[]> {
    return this.request('/api/v1/sessions');
  }

  async revokeSession(sessionId: string): Promise<void> {
    return this.request(`/api/v1/sessions/${sessionId}', { method: 'DELETE' });
  }

  async revokeAllSessions(): Promise<void> {
    return this.request('/api/v1/sessions', { method: 'DELETE' });
  }

  // ==================== Feature Flags ====================

  async getFeatureFlags(): Promise<{ name: string; enabled: boolean; description?: string }[]> {
    return this.request('/api/v1/features');
  }

  async setFeatureFlag(name: string, enabled: boolean): Promise<void> {
    return this.request(`/api/v1/features/${name}`, {
      method: 'PUT',
      body: JSON.stringify({ enabled }),
    });
  }

  // ==================== Health Check ====================

  async healthCheck(): Promise<{ status: string; version: string; timestamp: string }> {
    return this.request('/api/v1/health');
  }

  async readinessCheck(): Promise<{ ready: boolean; services: Record<string, boolean> }> {
    return this.request('/api/v1/health/ready');
  }
}

export const superAdminApi = new SuperAdminApiService();
export default superAdminApi;
