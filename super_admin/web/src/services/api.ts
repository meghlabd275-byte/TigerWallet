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

const API_BASE_URL = process.env.NEXT_PUBLIC_SUPER_ADMIN_API || 'http://localhost:8082';
const WS_BASE_URL = process.env.NEXT_PUBLIC_SUPER_ADMIN_WS || 'ws://localhost:8082';

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
    return this.request(`/api/v1/whitelabels/${id}/suspend`, { method: 'POST' });
  }

  async activateWhiteLabel(id: string): Promise<WhiteLabel> {
    return this.request(`/api/v1/whitelabels/${id}/activate`, { method: 'POST' });
  }

  async updateWhiteLabelFee(id: string, feePercent: number): Promise<WhiteLabel> {
    return this.request(`/api/v1/whitelabels/${id}/fee`, {
      method: 'PUT',
      body: JSON.stringify({ fee_percent: feePercent }),
    });
  }

  async regenerateWhiteLabelAPIKey(id: string): Promise<{ api_key: string }> {
    return this.request(`/api/v1/whitelabels/${id}/regenerate-key`, { method: 'POST' });
  }

  async verifyWhiteLabelDomain(id: string, domain: string): Promise<{ verified: boolean }> {
    return this.request(`/api/v1/whitelabels/${id}/verify-domain`, {
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
    return this.request(`/api/v1/admin/admins?${query}`);
  }

  async getAdmin(id: string): Promise<Admin> {
    return this.request(`/api/v1/admin/admins/${id}`);
  }

  async createAdmin(data: AdminCreateInput): Promise<Admin> {
    return this.request('/api/v1/admin/admins', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateAdmin(id: string, data: AdminUpdateInput): Promise<Admin> {
    return this.request(`/api/v1/admin/admins/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteAdmin(id: string): Promise<void> {
    return this.request(`/api/v1/admin/admins/${id}`, { method: 'DELETE' });
  }

  async suspendAdmin(id: string, reason: string): Promise<void> {
    return this.request(`/api/v1/admin/admins/${id}/suspend`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
  }

  async activateAdmin(id: string): Promise<void> {
    return this.request(`/api/v1/admin/admins/${id}/activate`, { method: 'POST' });
  }

  async updateAdminPermissions(id: string, permissions: string[]): Promise<Admin> {
    return this.request(`/api/v1/admin/admins/${id}/permissions`, {
      method: 'PUT',
      body: JSON.stringify({ permissions }),
    });
  }

  async getAdminSessions(adminId: string): Promise<any[]> {
    return this.request(`/api/v1/admin/admins/${adminId}/sessions`);
  }

  async revokeAdminSession(adminId: string, sessionId: string): Promise<void> {
    return this.request(`/api/v1/admin/admins/${adminId}/sessions/${sessionId}`, { method: 'DELETE' });
  }

  async revokeAllAdminSessions(adminId: string): Promise<void> {
    return this.request(`/api/v1/admin/admins/${adminId}/sessions`, { method: 'DELETE' });
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
    return this.request(`/api/v1/tickets/${ticketId}/close`, { method: 'POST' });
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
    return this.request(`/api/v1/knowledge-base/${id}`, { method: 'DELETE' });
  }

  async publishArticle(id: string): Promise<Article> {
    return this.request(`/api/v1/knowledge-base/${id}/publish`, { method: 'POST' });
  }

  async archiveArticle(id: string): Promise<Article> {
    return this.request(`/api/v1/knowledge-base/${id}/archive`, { method: 'POST' });
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
    return this.request(`/api/v1/workflows/${id}`, { method: 'DELETE' });
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
    return this.request(`/api/v1/approvals/${requestId}/cancel`, { method: 'POST' });
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
    return this.request(`/api/v1/security/alerts/${id}/resolve`, {
      method: 'POST',
      body: JSON.stringify({ resolution }),
    });
  }

  async markSecurityAlertAsFalsePositive(id: string): Promise<SecurityAlert> {
    return this.request(`/api/v1/security/alerts/${id}/false-positive`, { method: 'POST' });
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
    return this.request(`/api/v1/api-keys/${id}/revoke`, { method: 'POST' });
  }

  async regenerateAPIKey(id: string): Promise<{ key: string }> {
    return this.request(`/api/v1/api-keys/${id}/regenerate`, { method: 'POST' });
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
    return this.request(`/api/v1/webhooks/${id}`, { method: 'DELETE' });
  }

  async testWebhook(id: string): Promise<{ success: boolean; response_time_ms: number }> {
    return this.request(`/api/v1/webhooks/${id}/test`, { method: 'POST' });
  }

  async activateWebhook(id: string): Promise<Webhook> {
    return this.request(`/api/v1/webhooks/${id}/activate`, { method: 'POST' });
  }

  async deactivateWebhook(id: string): Promise<Webhook> {
    return this.request(`/api/v1/webhooks/${id}/deactivate`, { method: 'POST' });
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
    return this.request(`/api/v1/system/services/${name}/restart`, { method: 'POST' });
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
    return this.request(`/api/v1/notifications/${id}/read`, { method: 'PUT' });
  }

  async markAllNotificationsRead(): Promise<void> {
    return this.request('/api/v1/notifications/read-all', { method: 'PUT' });
  }

  async deleteNotification(id: string): Promise<void> {
    return this.request(`/api/v1/notifications/${id}`, { method: 'DELETE' });
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
    return this.request(`/api/v1/sessions/${sessionId}`, { method: 'DELETE' });
  }

  async revokeAllSessions(): Promise<void> {
    return this.request('/api/v1/sessions', { method: 'DELETE' });
  }

  // ==================== Feature Flags ====================

  async getFeatureFlags(): Promise<{ name: string; enabled: boolean; description?: string }[]> {
    return this.request('/api/v1/admin/features');
  }

  async setFeatureFlag(name: string, enabled: boolean): Promise<void> {
    return this.request(`/api/v1/admin/features/${name}`, {
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

  // ==================== Email Notifications ====================

  async sendEmail(data: {
    to_email: string;
    to_name?: string;
    subject: string;
    body_html?: string;
    body_text?: string;
    reply_to?: string;
    cc?: string;
    bcc?: string;
  }): Promise<{ status: string }> {
    return this.request('/emails', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async sendTemplateEmail(data: {
    to_email: string;
    to_name?: string;
    template_id: string;
    variables: Record<string, any>;
  }): Promise<{ status: string; template_id: string }> {
    return this.request('/emails/template', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async queueEmail(data: {
    to_email: string;
    to_name?: string;
    subject: string;
    body_html?: string;
    body_text?: string;
    priority?: number;
    scheduled_at?: string;
  }): Promise<{ status: string; message_id: string }> {
    return this.request('/emails/queue', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async unsubscribeEmail(email: string, reason?: string, categories?: string): Promise<{ status: string }> {
    return this.request('/unsubscribe', {
      method: 'POST',
      body: JSON.stringify({ email, reason, categories }),
    });
  }

  async addEmailRecipient(email: string, name?: string): Promise<{ status: string }> {
    return this.request('/recipients', {
      method: 'POST',
      body: JSON.stringify({ email, name }),
    });
  }

  async getEmailStats(): Promise<{ total: number; sent: number; failed: number; queued: number; rate: number }> {
    return this.request('/stats');
  }

  // ==================== SMS Notifications ====================

  async sendSMS(data: {
    to: string;
    message: string;
  }): Promise<{ status: string; message_id: string }> {
    return this.request('/sms', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async sendTemplateSMS(data: {
    to: string;
    template_id: string;
    variables: Record<string, any>;
  }): Promise<{ status: string; message_id: string }> {
    return this.request('/sms/template', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async verifyPhone(phone: string): Promise<{ status: string; verification_code: string }> {
    return this.request('/verify/send', {
      method: 'POST',
      body: JSON.stringify({ phone }),
    });
  }

  async confirmPhoneVerification(phone: string, code: string): Promise<{ status: string }> {
    return this.request('/verify/confirm', {
      method: 'POST',
      body: JSON.stringify({ phone, code }),
    });
  }

  async getSMSStats(): Promise<{ total: number; sent: number; failed: number; queued: number; balance: number }> {
    return this.request('/stats');
  }

  // ==================== Push Notifications ====================

  async sendPush(data: {
    user_id: string;
    token: string;
    platform: string;
    title: string;
    body?: string;
    data?: string;
    sound?: string;
    badge?: number;
  }): Promise<{ status: string; message_id: string }> {
    return this.request('/push', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async sendTemplatePush(data: {
    user_id: string;
    token: string;
    platform: string;
    template_id: string;
    variables: Record<string, any>;
  }): Promise<{ status: string; message_id: string }> {
    return this.request('/push/template', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async registerDevice(data: {
    user_id: string;
    token: string;
    platform: string;
    app_version?: string;
    language?: string;
    timezone?: string;
  }): Promise<{ status: string }> {
    return this.request('/devices', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async unregisterDevice(token: string): Promise<{ status: string }> {
    return this.request(`/devices/${token}`, { method: 'DELETE' });
  }

  async getPushStats(): Promise<{ total: number; sent: number; failed: number; queued: number }> {
    return this.request('/stats');
  }

  // ==================== Two-Factor Authentication ====================

  async setup2FA(user_id: string, method?: string, phone?: string, email?: string): Promise<{
    secret: string;
    qr_code_url: string;
  }> {
    return this.request('/2fa/setup', {
      method: 'POST',
      body: JSON.stringify({ user_id, method, phone, email }),
    });
  }

  async verify2FASetup(user_id: string, secret: string, code: string): Promise<{
    status: string;
    backup_codes: string[];
  }> {
    return this.request('/2fa/verify-setup', {
      method: 'POST',
      body: JSON.stringify({ user_id, secret, code }),
    });
  }

  async adminVerify2FA(user_id: string, code?: string, backup_code?: string): Promise<{ status: string; method: string }> {
    return this.request('/2fa/verify', {
      method: 'POST',
      body: JSON.stringify({ user_id, code, backup_code }),
    });
  }

  async adminDisable2FA(user_id: string, code: string): Promise<{ status: string }> {
    return this.request('/2fa/disable', {
      method: 'POST',
      body: JSON.stringify({ user_id, code }),
    });
  }

  async get2FAStatus(user_id: string): Promise<{ enabled: boolean }> {
    return this.request(`/2fa/status/${user_id}`);
  }

  async get2FAConfig(user_id: string): Promise<any> {
    return this.request(`/2fa/config/${user_id}`);
  }

  async regenerateBackupCodes(user_id: string, code: string): Promise<{ backup_codes: string[] }> {
    return this.request('/2fa/regenerate-codes', {
      method: 'POST',
      body: JSON.stringify({ user_id, code }),
    });
  }

  // ==================== Support Tickets ====================

  async adminCreateTicket(data: {
    user_id: string;
    user_email: string;
    user_name?: string;
    subject: string;
    description: string;
    category?: string;
    priority?: string;
    channel?: string;
  }): Promise<any> {
    return this.request('/tickets', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async adminGetTicket(ticket_id: string): Promise<{ ticket: any; messages: any[] }> {
    return this.request(`/tickets/${ticket_id}`);
  }

  async getUserTickets(user_id: string): Promise<{ tickets: any[] }> {
    return this.request(`/tickets/user/${user_id}`);
  }

  async getOpenTickets(assigned_to?: string, status?: string): Promise<{ tickets: any[] }> {
    const params = new URLSearchParams();
    if (assigned_to) params.append('assigned_to', assigned_to);
    if (status) params.append('status', status);
    return this.request(`/tickets/open?${params}`);
  }

  async adminAddTicketMessage(data: {
    ticket_id: string;
    sender_id: string;
    sender_name?: string;
    sender_email: string;
    content: string;
    sender_type?: string;
    is_internal?: boolean;
  }): Promise<any> {
    return this.request(`/tickets/${data.ticket_id}/messages`, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateTicketStatus(ticket_id: string, status: string, assigned_to?: string): Promise<{ status: string }> {
    return this.request(`/tickets/${ticket_id}/status`, {
      method: 'PUT',
      body: JSON.stringify({ ticket_id, status, assigned_to }),
    });
  }

  async getTicketSLA(ticket_id: string): Promise<any> {
    return this.request(`/tickets/${ticket_id}/sla`);
  }

  async getCannedResponses(category?: string): Promise<{ responses: any[] }> {
    const params = category ? `?category=${category}` : '';
    return this.request(`/canned-responses${params}`);
  }

  // ==================== Knowledge Base ====================

  async getKnowledgeBaseCategories(): Promise<{ categories: any[] }> {
    return this.request('/knowledgebase');
  }

  async searchKnowledgeBase(category_id?: number, search?: string): Promise<{ articles: any[] }> {
    const params = new URLSearchParams();
    if (category_id) params.append('category_id', String(category_id));
    if (search) params.append('search', search);
    return this.request(`/knowledgebase/search?${params}`);
  }

  async getKnowledgeBaseArticle(slug: string): Promise<any> {
    return this.request(`/knowledgebase/articles/${slug}`);
  }

  // ==================== Reports ====================

  async createReport(data: {
    report_type: string;
    title: string;
    user_id: string;
    format?: string;
    parameters?: Record<string, any>;
  }): Promise<any> {
    return this.request('/reports', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async getReport(report_id: string): Promise<any> {
    return this.request(`/reports/${report_id}`);
  }

  async getUserReports(user_id: string): Promise<{ reports: any[] }> {
    return this.request(`/reports/user/${user_id}`);
  }

  async downloadReport(report_id: string): Promise<Blob> {
    const response = await fetch(`${this.baseURL}/reports/${report_id}/download`, {
      headers: this.getHeaders(),
    });
    if (!response.ok) {
      throw new Error('Failed to download report');
    }
    return response.blob();
  }

  async getReportTemplates(): Promise<{ templates: any[] }> {
    return this.request('/templates');
  }

  // ==================== Integrations ====================

  // Slack
  async sendSlackMessage(data: {
    channel: string;
    text: string;
    blocks?: any[];
  }): Promise<{ status: string; message_ts: string }> {
    return this.request('/slack/message', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async sendSlackAlert(data: {
    channel: string;
    alert_type: string;
    title: string;
    message: string;
    severity?: string;
  }): Promise<{ status: string }> {
    return this.request('/slack/alert', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  // Datadog
  async sendDatadogMetric(data: {
    name: string;
    value: number;
    tags?: Record<string, string>;
  }): Promise<{ status: string }> {
    return this.request('/datadog/metric', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async sendDatadogEvent(data: {
    title: string;
    text: string;
    alert_type?: string;
    priority?: string;
    tags?: Record<string, string>;
  }): Promise<{ status: string; event_id: string }> {
    return this.request('/datadog/event', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async sendTransactionMetric(data: {
    type: string;
    status: string;
    chain: string;
    value: number;
  }): Promise<{ status: string }> {
    return this.request('/datadog/transaction', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async queryDatadogMetrics(query: string, from?: number, to?: number): Promise<{ results: any[] }> {
    return this.request('/datadog/query', {
      method: 'POST',
      body: JSON.stringify({ query, from, to }),
    });
  }

  // ==================== IP Whitelist ====================

  async getIPWhitelist(): Promise<{ ips: any[] }> {
    return this.request('/security/whitelist');
  }

  async addToIPWhitelist(ip: string, description?: string): Promise<{ status: string }> {
    return this.request('/security/whitelist', {
      method: 'POST',
      body: JSON.stringify({ ip, description }),
    });
  }

  async removeFromIPWhitelist(ip: string): Promise<{ status: string }> {
    return this.request(`/security/whitelist/${ip}`, { method: 'DELETE' });
  }

  async checkIPWhitelist(ip: string): Promise<{ allowed: boolean }> {
    return this.request(`/security/whitelist/check?ip=${ip}`);
  }

  // ==================== Rate Limiting ====================

  async getRateLimits(): Promise<{ limits: any[] }> {
    return this.request('/security/rate-limits');
  }

  async setRateLimit(data: {
    endpoint: string;
    requests_per_minute: number;
    burst?: number;
  }): Promise<{ status: string }> {
    return this.request('/security/rate-limits', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async getRateLimitStats(): Promise<{ stats: any }> {
    return this.request('/security/rate-limits/stats');
  }

  // ==================== Cloudflare Integration ====================

  async createWAFRule(data: {
    name: string;
    action: string;
    expression: string;
    priority?: number;
  }): Promise<any> {
    return this.request('/waf/rules', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async getWAFRules(): Promise<{ rules: any[] }> {
    return this.request('/waf/rules');
  }

  async deleteWAFRule(rule_id: string): Promise<{ status: string }> {
    return this.request(`/waf/rules/${rule_id}`, { method: 'DELETE' });
  }

  async addIPRule(data: {
    ip: string;
    rule_type: string;
    reason?: string;
    expires_at?: string;
  }): Promise<any> {
    return this.request('/ip-rules', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async getIPRules(type?: string): Promise<{ rules: any[] }> {
    const params = type ? `?type=${type}` : '';
    return this.request(`/ip-rules${params}`);
  }

  async removeIPRule(ip: string): Promise<{ status: string }> {
    return this.request(`/ip-rules/${ip}`, { method: 'DELETE' });
  }

  async createRateLimit(data: {
    name: string;
    path: string;
    requests_per_minute: number;
    burst?: number;
    action?: string;
  }): Promise<any> {
    return this.request('/rate-limits', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async getRateLimitRules(): Promise<{ rules: any[] }> {
    return this.request('/rate-limits');
  }

  async createDNSRecord(data: {
    name: string;
    type: string;
    content: string;
    proxied?: boolean;
    ttl?: number;
  }): Promise<any> {
    return this.request('/dns-records', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async getDNSRecords(type?: string): Promise<{ records: any[] }> {
    const params = type ? `?type=${type}` : '';
    return this.request(`/dns-records${params}`);
  }

  async getFirewallEvents(ip?: string, limit?: number): Promise<{ events: any[] }> {
    const params = new URLSearchParams();
    if (ip) params.append('ip', ip);
    if (limit) params.append('limit', String(limit));
    return this.request(`/firewall/events?${params}`);
  }

  async getSecurityStats(): Promise<any> {
    return this.request('/security/stats');
  }

  // ==================== PagerDuty Integration ====================

  async triggerPagerDutyEvent(data: {
    title: string;
    body?: string;
    severity?: string;
    source?: string;
    custom_details?: Record<string, any>;
  }): Promise<{ status: string; dedup_key: string }> {
    return this.request('/events/trigger', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async resolvePagerDutyEvent(data: {
    dedup_key: string;
    title?: string;
  }): Promise<{ status: string }> {
    return this.request('/events/resolve', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async sendPagerDutyAlert(data: {
    alert_type: string;
    data: Record<string, any>;
  }): Promise<{ status: string }> {
    return this.request('/alerts', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async getPagerDutyIncidents(status?: string): Promise<{ incidents: any[] }> {
    const params = status ? `?status=${status}` : '';
    return this.request(`/incidents${params}`);
  }

  async getPagerDutyServices(): Promise<{ services: any[] }> {
    return this.request('/services');
  }

  async getPagerDutyOnCall(): Promise<{ oncalls: any[] }> {
    return this.request('/oncall');
  }

  // ==================== Excel Reports ====================

  async createExcelReport(data: {
    report_type: string;
    title?: string;
    user_id: string;
    parameters?: Record<string, any>;
  }): Promise<any> {
    return this.request('/reports', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async getExcelReport(report_id: string): Promise<any> {
    return this.request(`/reports/${report_id}`);
  }

  async downloadExcelReport(report_id: string): Promise<Blob> {
    const response = await fetch(`${this.baseURL}/reports/${report_id}/download`, {
      headers: this.getHeaders(),
    });
    if (!response.ok) {
      throw new Error('Failed to download report');
    }
    return response.blob();
  }

  // ==================== Fraud Detection ====================

  async analyzeTransaction(data: {
    user_id: string;
    tx_type: string;
    ip_address?: string;
    country?: string;
    device?: string;
    amount: number;
    currency?: string;
  }): Promise<any> {
    return this.request('/analyze', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async getFraudAlerts(user_id?: string, status?: string): Promise<{ alerts: any[] }> {
    const params = new URLSearchParams();
    if (user_id) params.append('user_id', user_id);
    if (status) params.append('status', status);
    return this.request(`/alerts?${params}`);
  }

  async resolveFraudAlert(data: {
    alert_id: string;
    resolution: string;
    resolved_by?: string;
  }): Promise<{ status: string }> {
    return this.request('/alerts/resolve', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async blockUser(data: {
    user_id: string;
    reason: string;
    severity?: string;
    expires_at?: string;
  }): Promise<{ status: string }> {
    return this.request('/block', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async unblockUser(user_id: string): Promise<{ status: string }> {
    return this.request(`/block/${user_id}`, { method: 'DELETE' });
  }

  // ---- Bots Management ----
  async getBots(): Promise<{ bots: any[] }> {
    return this.request('/api/v1/admin/bots');
  }
  async getBot(id: string): Promise<{ bot: any }> {
    return this.request(`/api/v1/admin/bots/${id}`);
  }
  async createBot(data: { name: string; bot_type: string }): Promise<{ message: string }> {
    return this.request('/api/v1/admin/bots', { method: 'POST', body: JSON.stringify(data) });
  }
  async updateBot(id: string, data: { name?: string }): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/bots/${id}`, { method: 'PUT', body: JSON.stringify(data) });
  }
  async deleteBot(id: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/bots/${id}`, { method: 'DELETE' });
  }
  async updateBotStatus(id: string, status: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/bots/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) });
  }
  async getBotTiers(): Promise<{ tiers: any[] }> {
    return this.request('/api/v1/admin/bots/tiers');
  }
  async createBotTier(data: any): Promise<{ message: string }> {
    return this.request('/api/v1/admin/bots/tiers', { method: 'POST', body: JSON.stringify(data) });
  }
  async updateBotTier(id: string, data: any): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/bots/tiers/${id}`, { method: 'PUT', body: JSON.stringify(data) });
  }
  async deleteBotTier(id: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/bots/tiers/${id}`, { method: 'DELETE' });
  }

  // ---- BotsClients Management ----
  async getBotsClients(): Promise<{ clients: any[] }> {
    return this.request('/api/v1/admin/bots-clients');
  }
  async getBotsClient(id: string): Promise<{ client: any }> {
    return this.request(`/api/v1/admin/bots-clients/${id}`);
  }
  async createBotsClient(data: { name: string; company?: string; email?: string; permission_level?: string }): Promise<any> {
    return this.request('/api/v1/admin/bots-clients', { method: 'POST', body: JSON.stringify(data) });
  }
  async updateBotsClient(id: string, data: { name?: string }): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/bots-clients/${id}`, { method: 'PUT', body: JSON.stringify(data) });
  }
  async deleteBotsClient(id: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/bots-clients/${id}`, { method: 'DELETE' });
  }
  async updateBotsClientStatus(id: string, status: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/bots-clients/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) });
  }

  // ---- Project Teams ----
  async getProjectTeams(): Promise<{ teams: any[] }> {
    return this.request('/api/v1/admin/project-teams');
  }
  async getProjectTeam(id: string): Promise<{ team: any }> {
    return this.request(`/api/v1/admin/project-teams/${id}`);
  }
  async createProjectTeam(data: { name: string; description?: string }): Promise<{ message: string }> {
    return this.request('/api/v1/admin/project-teams', { method: 'POST', body: JSON.stringify(data) });
  }
  async updateProjectTeam(id: string, data: { name?: string }): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/project-teams/${id}`, { method: 'PUT', body: JSON.stringify(data) });
  }
  async deleteProjectTeam(id: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/project-teams/${id}`, { method: 'DELETE' });
  }
  async getProjectTeamMembers(id: string): Promise<{ members: any[] }> {
    return this.request(`/api/v1/admin/project-teams/${id}/members`);
  }
  async addProjectTeamMember(teamId: string, data: { user_id?: string; role?: string }): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/project-teams/${teamId}/members`, { method: 'POST', body: JSON.stringify(data) });
  }
  async removeProjectTeamMember(teamId: string, memberId: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/project-teams/${teamId}/members/${memberId}`, { method: 'DELETE' });
  }

  // ---- MasterWallets ----
  async getMasterWallets(): Promise<{ wallets: any[] }> {
    return this.request('/api/v1/admin/master-wallets');
  }
  async getMasterWallet(id: string): Promise<{ wallet: any }> {
    return this.request(`/api/v1/admin/master-wallets/${id}`);
  }
  async createMasterWallet(data: { name: string; address?: string; chain_id?: number }): Promise<{ message: string }> {
    return this.request('/api/v1/admin/master-wallets', { method: 'POST', body: JSON.stringify(data) });
  }
  async updateMasterWallet(id: string, data: { name?: string }): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/master-wallets/${id}`, { method: 'PUT', body: JSON.stringify(data) });
  }
  async deleteMasterWallet(id: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/master-wallets/${id}`, { method: 'DELETE' });
  }
  async getMasterWalletBalance(id: string): Promise<{ balance: number }> {
    return this.request(`/api/v1/admin/master-wallets/${id}/balance`);
  }

  // ---- UserWallets ----
  async getUserWallets(): Promise<{ wallets: any[] }> {
    return this.request('/api/v1/admin/user-wallets');
  }
  async getUserWallet(id: string): Promise<{ wallet: any }> {
    return this.request(`/api/v1/admin/user-wallets/${id}`);
  }
  async createUserWallet(data: { name: string; master_wallet_id?: string; address?: string; chain_id?: number }): Promise<{ message: string }> {
    return this.request('/api/v1/admin/user-wallets', { method: 'POST', body: JSON.stringify(data) });
  }
  async updateUserWallet(id: string, data: { name?: string }): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/user-wallets/${id}`, { method: 'PUT', body: JSON.stringify(data) });
  }
  async deleteUserWallet(id: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/user-wallets/${id}`, { method: 'DELETE' });
  }
  async getUserWalletBalance(id: string): Promise<{ balance: number }> {
    return this.request(`/api/v1/admin/user-wallets/${id}/balance`);
  }

  // ---- WL Clients ----
  async getWLClients(): Promise<{ clients: any[] }> {
    return this.request('/api/v1/admin/wl-clients');
  }
  async createWLClient(data: { name: string; domain?: string }): Promise<{ message: string }> {
    return this.request('/api/v1/admin/wl-clients', { method: 'POST', body: JSON.stringify(data) });
  }
  async updateWLClient(id: string, data: { name?: string }): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/wl-clients/${id}`, { method: 'PUT', body: JSON.stringify(data) });
  }
  async deleteWLClient(id: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/wl-clients/${id}`, { method: 'DELETE' });
  }
  async updateWLClientStatus(id: string, status: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/wl-clients/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) });
  }

  // ---- WL MasterWallets / UserWallets / Bots / BotsClients / ProjectTeams ----
  async getWLMasterWallets(): Promise<{ wallets: any[] }> {
    return this.request('/api/v1/admin/wl-master-wallets');
  }
  async createWLMasterWallet(data: { name: string; address?: string; chain_id?: number }): Promise<{ message: string }> {
    return this.request('/api/v1/admin/wl-master-wallets', { method: 'POST', body: JSON.stringify(data) });
  }
  async deleteWLMasterWallet(id: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/wl-master-wallets/${id}`, { method: 'DELETE' });
  }
  async getWLUserWallets(): Promise<{ wallets: any[] }> {
    return this.request('/api/v1/admin/wl-user-wallets');
  }
  async createWLUserWallet(data: { name: string; master_wallet_id?: string; chain_id?: number }): Promise<{ message: string }> {
    return this.request('/api/v1/admin/wl-user-wallets', { method: 'POST', body: JSON.stringify(data) });
  }
  async deleteWLUserWallet(id: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/wl-user-wallets/${id}`, { method: 'DELETE' });
  }
  async getWLBots(): Promise<{ bots: any[] }> {
    return this.request('/api/v1/admin/wl-bots');
  }
  async createWLBot(data: { name: string; bot_type: string }): Promise<{ message: string }> {
    return this.request('/api/v1/admin/wl-bots', { method: 'POST', body: JSON.stringify(data) });
  }
  async deleteWLBot(id: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/wl-bots/${id}`, { method: 'DELETE' });
  }
  async getWLBotsClients(): Promise<{ clients: any[] }> {
    return this.request('/api/v1/admin/wl-bots-clients');
  }
  async createWLBotsClient(data: { name: string; company?: string }): Promise<{ message: string }> {
    return this.request('/api/v1/admin/wl-bots-clients', { method: 'POST', body: JSON.stringify(data) });
  }
  async deleteWLBotsClient(id: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/wl-bots-clients/${id}`, { method: 'DELETE' });
  }
  async getWLProjectTeams(): Promise<{ teams: any[] }> {
    return this.request('/api/v1/admin/wl-project-teams');
  }
  async createWLProjectTeam(data: { name: string; description?: string }): Promise<{ message: string }> {
    return this.request('/api/v1/admin/wl-project-teams', { method: 'POST', body: JSON.stringify(data) });
  }
  async deleteWLProjectTeam(id: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/wl-project-teams/${id}`, { method: 'DELETE' });
  }

  // ==================== Domain admin governance (PostgreSQL-backed) ====================
  // Governance records only — these never move crypto assets. Fund movement is
  // performed exclusively by the wallet owner via the canonical wallet backend.

  // ---- Futures positions ----
  async getFutures(): Promise<{ positions: any[] }> {
    return this.request('/api/v1/admin/futures');
  }
  async getFuture(id: string): Promise<{ position: any }> {
    return this.request(`/api/v1/admin/futures/${id}`);
  }
  async createFuture(data: {
    pair: string; side: string; size?: number; leverage?: number;
    entry_price?: number; liquidation_price?: number; margin?: number; chain_id?: number;
  }): Promise<{ message: string }> {
    return this.request('/api/v1/admin/futures', { method: 'POST', body: JSON.stringify(data) });
  }
  async updateFuture(id: string, data: {
    pair?: string; side?: string; size?: number; leverage?: number;
    entry_price?: number; liquidation_price?: number; margin?: number; chain_id?: number;
  }): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/futures/${id}`, { method: 'PUT', body: JSON.stringify(data) });
  }
  async deleteFuture(id: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/futures/${id}`, { method: 'DELETE' });
  }
  async updateFutureStatus(id: string, status: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/futures/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) });
  }

  // ---- Options contracts ----
  async getOptions(): Promise<{ contracts: any[] }> {
    return this.request('/api/v1/admin/options');
  }
  async getOption(id: string): Promise<{ contract: any }> {
    return this.request(`/api/v1/admin/options/${id}`);
  }
  async createOption(data: {
    underlying: string; option_type: string; strike?: number; expiry?: string;
    premium?: number; size?: number; chain_id?: number;
  }): Promise<{ message: string }> {
    return this.request('/api/v1/admin/options', { method: 'POST', body: JSON.stringify(data) });
  }
  async updateOption(id: string, data: {
    underlying?: string; option_type?: string; strike?: number; expiry?: string;
    premium?: number; size?: number; chain_id?: number;
  }): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/options/${id}`, { method: 'PUT', body: JSON.stringify(data) });
  }
  async deleteOption(id: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/options/${id}`, { method: 'DELETE' });
  }
  async updateOptionStatus(id: string, status: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/options/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) });
  }

  // ---- Copy trading configs ----
  async getCopyTrading(): Promise<{ configs: any[] }> {
    return this.request('/api/v1/admin/copy-trading');
  }
  async getCopyTradingItem(id: string): Promise<{ config: any }> {
    return this.request(`/api/v1/admin/copy-trading/${id}`);
  }
  async createCopyTrading(data: {
    follower_id?: string; leader_id?: string; allocation?: number; max_leverage?: number;
  }): Promise<{ message: string }> {
    return this.request('/api/v1/admin/copy-trading', { method: 'POST', body: JSON.stringify(data) });
  }
  async updateCopyTrading(id: string, data: {
    follower_id?: string; leader_id?: string; allocation?: number; max_leverage?: number;
  }): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/copy-trading/${id}`, { method: 'PUT', body: JSON.stringify(data) });
  }
  async deleteCopyTrading(id: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/copy-trading/${id}`, { method: 'DELETE' });
  }
  async updateCopyTradingStatus(id: string, status: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/copy-trading/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) });
  }

  // ---- Convert orders ----
  async getConvert(): Promise<{ orders: any[] }> {
    return this.request('/api/v1/admin/convert');
  }
  async getConvertItem(id: string): Promise<{ order: any }> {
    return this.request(`/api/v1/admin/convert/${id}`);
  }
  async createConvert(data: {
    user_id?: string; from_token: string; to_token: string;
    from_amount?: number; to_amount?: number; rate?: number; chain_id?: number;
  }): Promise<{ message: string }> {
    return this.request('/api/v1/admin/convert', { method: 'POST', body: JSON.stringify(data) });
  }
  async updateConvert(id: string, data: {
    from_token?: string; to_token?: string;
    from_amount?: number; to_amount?: number; rate?: number; chain_id?: number;
  }): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/convert/${id}`, { method: 'PUT', body: JSON.stringify(data) });
  }
  async deleteConvert(id: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/convert/${id}`, { method: 'DELETE' });
  }
  async updateConvertStatus(id: string, status: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/convert/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) });
  }

  // ---- Onramp orders ----
  async getOnRampOrders(): Promise<{ orders: any[] }> {
    return this.request('/api/v1/admin/onramp');
  }
  async getOnRampOrder(id: string): Promise<{ order: any }> {
    return this.request(`/api/v1/admin/onramp/${id}`);
  }
  async createOnRampOrder(data: {
    user_id?: string; provider: string; fiat_currency: string; crypto_token: string;
    fiat_amount?: number; crypto_amount?: number;
  }): Promise<{ message: string }> {
    return this.request('/api/v1/admin/onramp', { method: 'POST', body: JSON.stringify(data) });
  }
  async updateOnRampOrder(id: string, data: {
    provider?: string; fiat_currency?: string; crypto_token?: string;
    fiat_amount?: number; crypto_amount?: number;
  }): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/onramp/${id}`, { method: 'PUT', body: JSON.stringify(data) });
  }
  async deleteOnRampOrder(id: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/onramp/${id}`, { method: 'DELETE' });
  }
  async approveOnRampOrder(id: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/onramp/${id}/approve`, { method: 'POST' });
  }
  async rejectOnRampOrder(id: string, reason: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/onramp/${id}/reject`, { method: 'POST', body: JSON.stringify({ reason }) });
  }

  // ---- Offramp orders ----
  async getOffRampOrders(): Promise<{ orders: any[] }> {
    return this.request('/api/v1/admin/offramp');
  }
  async getOffRampOrder(id: string): Promise<{ order: any }> {
    return this.request(`/api/v1/admin/offramp/${id}`);
  }
  async createOffRampOrder(data: {
    user_id?: string; provider: string; crypto_token: string; fiat_currency: string;
    crypto_amount?: number; fiat_amount?: number;
  }): Promise<{ message: string }> {
    return this.request('/api/v1/admin/offramp', { method: 'POST', body: JSON.stringify(data) });
  }
  async updateOffRampOrder(id: string, data: {
    provider?: string; crypto_token?: string; fiat_currency?: string;
    crypto_amount?: number; fiat_amount?: number;
  }): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/offramp/${id}`, { method: 'PUT', body: JSON.stringify(data) });
  }
  async deleteOffRampOrder(id: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/offramp/${id}`, { method: 'DELETE' });
  }
  async approveOffRampOrder(id: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/offramp/${id}/approve`, { method: 'POST' });
  }
  async rejectOffRampOrder(id: string, reason: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/offramp/${id}/reject`, { method: 'POST', body: JSON.stringify({ reason }) });
  }

  // ---- P2P clients ----
  async getP2PClients(): Promise<{ clients: any[] }> {
    return this.request('/api/v1/admin/p2p-clients');
  }
  async getP2PClient(id: string): Promise<{ client: any }> {
    return this.request(`/api/v1/admin/p2p-clients/${id}`);
  }
  async createP2PClient(data: { user_id?: string; username: string }): Promise<{ message: string }> {
    return this.request('/api/v1/admin/p2p-clients', { method: 'POST', body: JSON.stringify(data) });
  }
  async updateP2PClient(id: string, data: { username?: string }): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/p2p-clients/${id}`, { method: 'PUT', body: JSON.stringify(data) });
  }
  async deleteP2PClient(id: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/p2p-clients/${id}`, { method: 'DELETE' });
  }
  async updateP2PClientStatus(id: string, status: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/p2p-clients/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) });
  }

  // ---- P2P merchants ----
  async getP2PMerchants(): Promise<{ merchants: any[] }> {
    return this.request('/api/v1/admin/p2p-merchants');
  }
  async getP2PMerchant(id: string): Promise<{ merchant: any }> {
    return this.request(`/api/v1/admin/p2p-merchants/${id}`);
  }
  async createP2PMerchant(data: { name: string; email?: string }): Promise<{ message: string }> {
    return this.request('/api/v1/admin/p2p-merchants', { method: 'POST', body: JSON.stringify(data) });
  }
  async updateP2PMerchant(id: string, data: { name?: string; email?: string }): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/p2p-merchants/${id}`, { method: 'PUT', body: JSON.stringify(data) });
  }
  async deleteP2PMerchant(id: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/p2p-merchants/${id}`, { method: 'DELETE' });
  }
  async updateP2PMerchantStatus(id: string, status: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/p2p-merchants/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) });
  }
  async approveP2PMerchant(id: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/p2p-merchants/${id}/approve`, { method: 'POST' });
  }
  async rejectP2PMerchant(id: string, reason: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/p2p-merchants/${id}/reject`, { method: 'POST', body: JSON.stringify({ reason }) });
  }

  // ---- Partners ----
  async getPartners(): Promise<{ partners: any[] }> {
    return this.request('/api/v1/admin/partners');
  }
  async getPartner(id: string): Promise<{ partner: any }> {
    return this.request(`/api/v1/admin/partners/${id}`);
  }
  async createPartner(data: { name: string; contact_email?: string; revenue_share?: number }): Promise<{ message: string }> {
    return this.request('/api/v1/admin/partners', { method: 'POST', body: JSON.stringify(data) });
  }
  async updatePartner(id: string, data: { name?: string; contact_email?: string; revenue_share?: number }): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/partners/${id}`, { method: 'PUT', body: JSON.stringify(data) });
  }
  async deletePartner(id: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/partners/${id}`, { method: 'DELETE' });
  }
  async updatePartnerStatus(id: string, status: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/partners/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) });
  }
  async approvePartner(id: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/partners/${id}/approve`, { method: 'POST' });
  }
  async rejectPartner(id: string, reason: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/partners/${id}/reject`, { method: 'POST', body: JSON.stringify({ reason }) });
  }

  // ---- Rewards campaigns ----
  async getRewards(): Promise<{ campaigns: any[] }> {
    return this.request('/api/v1/admin/rewards');
  }
  async getReward(id: string): Promise<{ campaign: any }> {
    return this.request(`/api/v1/admin/rewards/${id}`);
  }
  async createReward(data: {
    name: string; reward_type: string; amount?: number; token?: string;
    start_at?: string; end_at?: string;
  }): Promise<{ message: string }> {
    return this.request('/api/v1/admin/rewards', { method: 'POST', body: JSON.stringify(data) });
  }
  async updateReward(id: string, data: {
    name?: string; reward_type?: string; amount?: number; token?: string;
    start_at?: string; end_at?: string;
  }): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/rewards/${id}`, { method: 'PUT', body: JSON.stringify(data) });
  }
  async deleteReward(id: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/rewards/${id}`, { method: 'DELETE' });
  }
  async updateRewardStatus(id: string, status: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/rewards/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) });
  }

  // ---- Marketing campaigns ----
  async getMarketing(): Promise<{ campaigns: any[] }> {
    return this.request('/api/v1/admin/marketing');
  }
  async getMarketingItem(id: string): Promise<{ campaign: any }> {
    return this.request(`/api/v1/admin/marketing/${id}`);
  }
  async createMarketing(data: {
    name: string; channel: string; budget?: number; start_at?: string; end_at?: string;
  }): Promise<{ message: string }> {
    return this.request('/api/v1/admin/marketing', { method: 'POST', body: JSON.stringify(data) });
  }
  async updateMarketing(id: string, data: {
    name?: string; channel?: string; budget?: number; start_at?: string; end_at?: string;
  }): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/marketing/${id}`, { method: 'PUT', body: JSON.stringify(data) });
  }
  async deleteMarketing(id: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/marketing/${id}`, { method: 'DELETE' });
  }
  async updateMarketingStatus(id: string, status: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/marketing/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) });
  }

  // ---- Structured RBAC: custom admin roles + granular permissions ----
  async getAdminRoles(): Promise<{ roles: any[] }> {
    return this.request('/api/v1/admin/roles');
  }
  async getAdminRole(id: string): Promise<{ role: any }> {
    return this.request(`/api/v1/admin/roles/${id}`);
  }
  async createAdminRole(data: { name: string; description?: string; permissions?: string[] }): Promise<{ message: string }> {
    return this.request('/api/v1/admin/roles', { method: 'POST', body: JSON.stringify(data) });
  }
  async updateAdminRole(id: string, data: { name?: string; description?: string; permissions?: string[]; is_active?: boolean }): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/roles/${id}`, { method: 'PUT', body: JSON.stringify(data) });
  }
  async deleteAdminRole(id: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/roles/${id}`, { method: 'DELETE' });
  }
  async getAdminPermissions(): Promise<{ permissions: any[] }> {
    return this.request('/api/v1/admin/permissions');
  }
  async createAdminPermission(data: { name: string; description?: string; category?: string }): Promise<{ message: string }> {
    return this.request('/api/v1/admin/permissions', { method: 'POST', body: JSON.stringify(data) });
  }
  async assignAdminRole(adminId: string, roleId: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/admins/${adminId}/roles`, { method: 'POST', body: JSON.stringify({ role_id: roleId }) });
  }
  async revokeAdminRole(adminId: string, roleId: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/admins/${adminId}/roles/${roleId}`, { method: 'DELETE' });
  }
  async getAdminEffectivePermissions(adminId: string): Promise<{ admin_id: string; permissions: string[] }> {
    return this.request(`/api/v1/admin/admins/${adminId}/permissions`);
  }

  // ==================== Crypto Cards ====================

  async getCryptoCards(params?: { page?: number; page_size?: number; status?: string; search?: string }): Promise<PaginatedResponse<any>> {
    const query = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) query.append(key, String(value));
      });
    }
    return this.request(`/api/v1/admin/crypto-cards?${query}`);
  }

  async getCryptoCard(id: string): Promise<{ card: any }> {
    return this.request(`/api/v1/admin/crypto-cards/${id}`);
  }

  async createCryptoCard(data: {
    user_id?: string; card_number?: string; network?: string; currency?: string;
    balance?: number; daily_limit?: number; monthly_limit?: number; status?: string; metadata?: Record<string, any>;
  }): Promise<{ message: string; card?: any }> {
    return this.request('/api/v1/admin/crypto-cards', { method: 'POST', body: JSON.stringify(data) });
  }

  async updateCryptoCard(id: string, data: {
    card_number?: string; network?: string; currency?: string; balance?: number;
    daily_limit?: number; monthly_limit?: number; metadata?: Record<string, any>;
  }): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/crypto-cards/${id}`, { method: 'PUT', body: JSON.stringify(data) });
  }

  async deleteCryptoCard(id: string): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/crypto-cards/${id}`, { method: 'DELETE' });
  }

  async blockCryptoCard(id: string, reason?: string): Promise<{ message: string; status?: string }> {
    return this.request(`/api/v1/admin/crypto-cards/${id}/block`, {
      method: 'POST', body: JSON.stringify(reason ? { reason } : {}),
    });
  }

  async activateCryptoCard(id: string): Promise<{ message: string; status?: string }> {
    return this.request(`/api/v1/admin/crypto-cards/${id}/activate`, { method: 'POST' });
  }

  async setCryptoCardLimit(id: string, data: { daily_limit?: number; monthly_limit?: number }): Promise<{ message: string }> {
    return this.request(`/api/v1/admin/crypto-cards/${id}/limit`, { method: 'PUT', body: JSON.stringify(data) });
  }

  async updateCryptoCardStatus(id: string, status: string): Promise<{ message: string; status?: string }> {
    return this.request(`/api/v1/admin/crypto-cards/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) });
  }

  // ---- Trading control-plane (/api/v1/admin/trading/*) ----
  async getTradingOverview(): Promise<any> {
    return this.request('/api/v1/admin/trading/overview');
  }
  async getTradingAudit(): Promise<{ audit: any[] }> {
    return this.request('/api/v1/admin/trading/audit');
  }
  async haltTradingVertical(vertical: string): Promise<any> {
    return this.request(`/api/v1/admin/trading/halt/${vertical}`, { method: 'POST' });
  }
  async resumeTradingVertical(vertical: string): Promise<any> {
    return this.request(`/api/v1/admin/trading/resume/${vertical}`, { method: 'POST' });
  }
  async getTradingContracts(): Promise<{ contracts: any[] }> {
    return this.request('/api/v1/admin/trading/contracts');
  }
  async createTradingContract(data: {
    kind: string; symbol: string; base_asset: string; quote_asset: string;
    chain_id?: number; max_leverage?: number; min_size?: string; tick_size?: string;
  }): Promise<any> {
    return this.request('/api/v1/admin/trading/contracts', { method: 'POST', body: JSON.stringify(data) });
  }
  async stopTradingContract(id: string): Promise<any> {
    return this.request(`/api/v1/admin/trading/contracts/${id}/stop`, { method: 'POST', body: JSON.stringify({ status: 'stopped' }) });
  }
  async resumeTradingContract(id: string): Promise<any> {
    return this.request(`/api/v1/admin/trading/contracts/${id}/resume`, { method: 'POST', body: JSON.stringify({ status: 'active' }) });
  }
  async deleteTradingContract(id: string): Promise<any> {
    return this.request(`/api/v1/admin/trading/contracts/${id}`, { method: 'DELETE' });
  }
  async getTradingPools(): Promise<{ pools: any[] }> {
    return this.request('/api/v1/admin/trading/pools');
  }
  async createTradingPool(data: {
    chain_id: number; dex: string; pool_address?: string; token0: string; token1: string; fee_bps?: number;
  }): Promise<any> {
    return this.request('/api/v1/admin/trading/pools', { method: 'POST', body: JSON.stringify(data) });
  }
  async stopTradingPool(id: string): Promise<any> {
    return this.request(`/api/v1/admin/trading/pools/${id}/stop`, { method: 'POST', body: JSON.stringify({ status: 'stopped' }) });
  }
  async resumeTradingPool(id: string): Promise<any> {
    return this.request(`/api/v1/admin/trading/pools/${id}/resume`, { method: 'POST', body: JSON.stringify({ status: 'active' }) });
  }
  async deleteTradingPool(id: string): Promise<any> {
    return this.request(`/api/v1/admin/trading/pools/${id}`, { method: 'DELETE' });
  }
  async stopTradingPairLifecycle(id: string): Promise<any> {
    return this.request(`/api/v1/admin/trading/pairs/${id}/stop`, { method: 'POST', body: JSON.stringify({ status: 'stopped' }) });
  }
  async resumeTradingPairLifecycle(id: string): Promise<any> {
    return this.request(`/api/v1/admin/trading/pairs/${id}/resume`, { method: 'POST', body: JSON.stringify({ status: 'active' }) });
  }
  async getTradingMarginMarkets(): Promise<{ margin_markets: any[] }> {
    return this.request('/api/v1/admin/trading/margin-markets');
  }
  async createTradingMarginMarket(data: {
    symbol: string; base_asset: string; quote_asset: string; max_leverage?: number; borrow_cap?: string;
  }): Promise<any> {
    return this.request('/api/v1/admin/trading/margin-markets', { method: 'POST', body: JSON.stringify(data) });
  }
  async stopTradingMarginMarket(id: string): Promise<any> {
    return this.request(`/api/v1/admin/trading/margin-markets/${id}/stop`, { method: 'POST', body: JSON.stringify({ status: 'stopped' }) });
  }
  async resumeTradingMarginMarket(id: string): Promise<any> {
    return this.request(`/api/v1/admin/trading/margin-markets/${id}/resume`, { method: 'POST', body: JSON.stringify({ status: 'active' }) });
  }
  async deleteTradingMarginMarket(id: string): Promise<any> {
    return this.request(`/api/v1/admin/trading/margin-markets/${id}`, { method: 'DELETE' });
  }
}

export const superAdminApi = new SuperAdminApiService();
export default superAdminApi;

/**
 * Crypto-cards management facade mirroring the `/api/v1/admin/crypto-cards`
 * routes on the super_admin/go backend (port 8082). Same JWT auth and base URL
 * as `superAdminApi`, exposed as a grouped object for the CryptoCards page.
 */
export const cryptoCardsAPI = {
  getAll: (params?: { page?: number; page_size?: number; status?: string; search?: string }) =>
    superAdminApi.getCryptoCards(params),
  getOne: (id: string) => superAdminApi.getCryptoCard(id),
  create: (data: Parameters<SuperAdminApiService['createCryptoCard']>[0]) =>
    superAdminApi.createCryptoCard(data),
  update: (id: string, data: Parameters<SuperAdminApiService['updateCryptoCard']>[1]) =>
    superAdminApi.updateCryptoCard(id, data),
  delete: (id: string) => superAdminApi.deleteCryptoCard(id),
  block: (id: string, reason?: string) => superAdminApi.blockCryptoCard(id, reason),
  activate: (id: string) => superAdminApi.activateCryptoCard(id),
  setLimit: (id: string, data: { daily_limit?: number; monthly_limit?: number }) =>
    superAdminApi.setCryptoCardLimit(id, data),
  setStatus: (id: string, status: string) => superAdminApi.updateCryptoCardStatus(id, status),
};
