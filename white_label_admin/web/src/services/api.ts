/**
 * TigerWallet White Label Admin - Complete API Service
 * API client for White Label Admin operations
 * Connected to Go Backend with PostgreSQL and Redis
 */

const API_BASE_URL = process.env.NEXT_PUBLIC_WHITELABEL_ADMIN_API || 'http://localhost:9092';

class WhiteLabelAdminApiService {
  private baseURL: string;
  private token: string | null = null;

  constructor() {
    this.baseURL = API_BASE_URL;
    if (typeof window !== 'undefined') {
      this.token = localStorage.getItem('whitelabel_admin_token');
    }
  }

  setToken(token: string) {
    this.token = token;
    if (typeof window !== 'undefined') {
      localStorage.setItem('whitelabel_admin_token', token);
    }
  }

  clearToken() {
    this.token = null;
    if (typeof window !== 'undefined') {
      localStorage.removeItem('whitelabel_admin_token');
    }
  }

  private getHeaders(): HeadersInit {
    return {
      'Content-Type': 'application/json',
      ...(this.token && { 'Authorization': `Bearer ${this.token}` }),
    };
  }

  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const url = `${this.baseURL}${endpoint}`;
    const response = await fetch(url, {
      ...options,
      headers: { ...this.getHeaders(), ...options.headers },
    });
    if (!response.ok) {
      const error = await response.json().catch(() => ({ message: 'Unknown error' }));
      throw new Error(error.error || error.message || `API Error: ${response.status}`);
    }
    const text = await response.text();
    if (!text) return undefined as unknown as T;
    return JSON.parse(text) as T;
  }

  // For endpoints that return non-JSON bodies (e.g. CSV export).
  private async requestText(endpoint: string, options: RequestInit = {}): Promise<string> {
    const url = `${this.baseURL}${endpoint}`;
    const response = await fetch(url, {
      ...options,
      headers: { ...this.getHeaders(), ...options.headers },
    });
    if (!response.ok) {
      const error = await response.json().catch(() => ({ message: 'Unknown error' }));
      throw new Error(error.error || error.message || `API Error: ${response.status}`);
    }
    return response.text();
  }

  // ==================== Authentication ====================
  async login(email: string, password: string): Promise<{ token: string; admin: any }> {
    return this.request('/api/v1/auth/login', { method: 'POST', body: JSON.stringify({ email, password }) });
  }

  async logout(): Promise<void> {
    await this.request('/api/v1/auth/logout', { method: 'POST' });
    this.clearToken();
  }

  // ==================== Users ====================
  async getUsers(): Promise<{ users: any[] }> {
    return this.request('/api/v1/admin/users');
  }

  async getUser(id: string): Promise<any> {
    return this.request(`/api/v1/admin/users/${id}`);
  }

  async suspendUser(id: string): Promise<any> {
    return this.request(`/api/v1/admin/users/${id}/suspend`, { method: 'POST' });
  }

  async banUser(id: string): Promise<any> {
    return this.request(`/api/v1/admin/users/${id}/ban`, { method: 'POST' });
  }

  async unbanUser(id: string): Promise<any> {
    return this.request(`/api/v1/admin/users/${id}/unban`, { method: 'POST' });
  }

  async updateUserStatus(id: string, status: string): Promise<any> {
    return this.request(`/api/v1/admin/users/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) });
  }

  // ==================== KYC ====================
  async getKYCRequests(): Promise<{ kyc_requests: any[] }> {
    return this.request('/api/v1/admin/kyc');
  }

  async approveKYC(id: string): Promise<any> {
    return this.request(`/api/v1/admin/kyc/${id}/approve`, { method: 'POST' });
  }

  async rejectKYC(id: string, reason: string): Promise<any> {
    return this.request(`/api/v1/admin/kyc/${id}/reject`, { method: 'POST', body: JSON.stringify({ reason }) });
  }

  // ==================== Transactions ====================
  async getTransactions(): Promise<{ transactions: any[] }> {
    return this.request('/api/v1/admin/transactions');
  }

  async getTransaction(id: string): Promise<any> {
    return this.request(`/api/v1/admin/transactions/${id}`);
  }

  async flagTransaction(id: string): Promise<any> {
    return this.request(`/api/v1/admin/transactions/${id}/flag`, { method: 'POST' });
  }

  async unflagTransaction(id: string): Promise<any> {
    return this.request(`/api/v1/admin/transactions/${id}/unflag`, { method: 'POST' });
  }

  // ==================== Withdrawals ====================
  async getWithdrawals(): Promise<{ withdrawals: any[] }> {
    return this.request('/api/v1/admin/withdrawals');
  }

  async approveWithdrawal(id: string): Promise<any> {
    return this.request(`/api/v1/admin/withdrawals/${id}/approve`, { method: 'POST' });
  }

  async rejectWithdrawal(id: string, reason: string): Promise<any> {
    return this.request(`/api/v1/admin/withdrawals/${id}/reject`, { method: 'POST', body: JSON.stringify({ reason }) });
  }

  async processWithdrawal(id: string, txHash: string): Promise<any> {
    return this.request(`/api/v1/admin/withdrawals/${id}/process`, { method: 'POST', body: JSON.stringify({ tx_hash: txHash }) });
  }

  // ==================== Tokens ====================
  async getTokens(): Promise<{ tokens: any[] }> {
    return this.request('/api/v1/admin/tokens');
  }

  async createToken(data: { symbol: string; name: string; contract_address?: string; decimals?: number; total_supply?: string; chain_id?: number }): Promise<any> {
    return this.request('/api/v1/admin/tokens', { method: 'POST', body: JSON.stringify(data) });
  }

  async updateToken(id: string, data: { name?: string; contract_address?: string; is_active?: boolean; is_verified?: boolean }): Promise<any> {
    return this.request(`/api/v1/admin/tokens/${id}`, { method: 'PUT', body: JSON.stringify(data) });
  }

  async deleteToken(id: string): Promise<void> {
    return this.request(`/api/v1/admin/tokens/${id}`, { method: 'DELETE' });
  }

  // ==================== Trading Pairs ====================
  async getPairs(): Promise<{ pairs: any[] }> {
    return this.request('/api/v1/admin/pairs');
  }

  async createPair(data: { base_token_id: string; quote_token_id: string; pair_name: string; chain_id?: number }): Promise<any> {
    return this.request('/api/v1/admin/pairs', { method: 'POST', body: JSON.stringify(data) });
  }

  async updatePairStatus(id: string, status: string): Promise<any> {
    return this.request(`/api/v1/admin/pairs/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) });
  }

  // ==================== Blockchains ====================
  async getBlockchains(): Promise<{ blockchains: any[] }> {
    return this.request('/api/v1/admin/blockchains');
  }

  async createBlockchain(data: { name: string; symbol: string; chain_id: number; is_evm?: boolean; rpc_url?: string; explorer_url?: string; native_token?: string; decimals?: number }): Promise<any> {
    return this.request('/api/v1/admin/blockchains', { method: 'POST', body: JSON.stringify(data) });
  }

  async updateBlockchain(id: string, data: { name?: string; rpc_url?: string; explorer_url?: string }): Promise<any> {
    return this.request(`/api/v1/admin/blockchains/${id}`, { method: 'PUT', body: JSON.stringify(data) });
  }

  async setBlockchainStatus(id: string, is_active: boolean): Promise<any> {
    return this.request(`/api/v1/admin/blockchains/${id}/status`, { method: 'PUT', body: JSON.stringify({ is_active }) });
  }

  // ==================== Fees ====================
  async getFees(): Promise<{ fees: any[] }> {
    return this.request('/api/v1/admin/fees');
  }

  async createFee(data: { fee_type: string; asset?: string; fee_percent?: string; fee_fixed?: string; min_fee?: string; max_fee?: string; tier?: string; chain_id?: number }): Promise<any> {
    return this.request('/api/v1/admin/fees', { method: 'POST', body: JSON.stringify(data) });
  }

  async updateFees(id: string, config: { fee_percent?: string; fee_fixed?: string; is_active?: boolean }): Promise<any> {
    return this.request(`/api/v1/admin/fees/${id}`, { method: 'PUT', body: JSON.stringify(config) });
  }

  // ==================== Notifications ====================
  async getNotifications(): Promise<{ notifications: any[] }> {
    return this.request('/api/v1/admin/notifications');
  }

  async markNotificationRead(id: string): Promise<any> {
    return this.request(`/api/v1/admin/notifications/${id}/read`, { method: 'PUT' });
  }

  async sendNotification(data: { title: string; message: string; notification_type: string; admin_id: string }): Promise<any> {
    return this.request('/api/v1/admin/notifications/send', { method: 'POST', body: JSON.stringify(data) });
  }

  async broadcastNotification(data: { title: string; message: string; notification_type: string }): Promise<{ broadcast: boolean; recipients: number }> {
    return this.request('/api/v1/admin/notifications/broadcast', { method: 'POST', body: JSON.stringify(data) });
  }

  // ==================== Tickets ====================
  async getTickets(): Promise<{ tickets: any[] }> {
    return this.request('/api/v1/admin/tickets');
  }

  async getTicket(id: string): Promise<{ ticket: any; messages: any[] }> {
    return this.request(`/api/v1/admin/tickets/${id}`);
  }

  async createTicket(data: { title: string; description?: string; ticket_type: string; priority?: string }): Promise<any> {
    return this.request('/api/v1/admin/tickets', { method: 'POST', body: JSON.stringify(data) });
  }

  async updateTicketStatus(id: string, status: string): Promise<any> {
    return this.request(`/api/v1/admin/tickets/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) });
  }

  async addTicketMessage(id: string, message: string, is_internal = false): Promise<any> {
    return this.request(`/api/v1/admin/tickets/${id}/messages`, { method: 'POST', body: JSON.stringify({ message, is_internal }) });
  }

  async assignTicket(id: string, adminId: string): Promise<any> {
    return this.request(`/api/v1/admin/tickets/${id}/assign`, { method: 'PUT', body: JSON.stringify({ admin_id: adminId }) });
  }

  // ==================== Audit logs ====================
  async getAuditLogs(): Promise<{ audit_logs: any[] }> {
    return this.request('/api/v1/admin/audit-logs');
  }

  async exportAuditLogs(): Promise<Blob> {
    const url = `${this.baseURL}/api/v1/admin/audit-logs/export`;
    const response = await fetch(url, { method: 'POST', headers: this.getHeaders() });
    if (!response.ok) {
      const error = await response.json().catch(() => ({ message: 'Unknown error' }));
      throw new Error(error.error || error.message || `API Error: ${response.status}`);
    }
    return response.blob();
  }

  // ==================== Sessions ====================
  async getSessions(): Promise<{ sessions: any[] }> {
    return this.request('/api/v1/admin/sessions');
  }

  async revokeSession(id: string): Promise<any> {
    return this.request(`/api/v1/admin/sessions/${id}`, { method: 'DELETE' });
  }

  async revokeAllSessions(): Promise<any> {
    return this.request('/api/v1/admin/sessions', { method: 'DELETE' });
  }

  // ==================== Stats ====================
  async getDashboardStats(): Promise<{ total_users: number; active_users: number; total_tokens: number; total_pairs: number; open_tickets: number; pending_kyc: number }> {
    return this.request('/api/v1/admin/stats');
  }

  // ==================== White Label Admins (scoped sub-admins) ====================
  async getAdmins(): Promise<any> {
    return this.request('/api/v1/admin/admins');
  }

  async createAdmin(data: { username: string; email: string; password: string }): Promise<any> {
    return this.request('/api/v1/admin/admins', { method: 'POST', body: JSON.stringify(data) });
  }

  async getAdmin(id: string): Promise<any> {
    return this.request(`/api/v1/admin/admins/${id}`);
  }

  async updateAdmin(id: string, data: { username?: string; scopes?: string[]; is_active?: boolean }): Promise<any> {
    return this.request(`/api/v1/admin/admins/${id}`, { method: 'PUT', body: JSON.stringify(data) });
  }

  async deleteAdmin(id: string): Promise<void> {
    return this.request(`/api/v1/admin/admins/${id}`, { method: 'DELETE' });
  }

  async suspendAdmin(id: string): Promise<any> {
    return this.request(`/api/v1/admin/admins/${id}/suspend`, { method: 'POST' });
  }

  async activateAdmin(id: string): Promise<any> {
    return this.request(`/api/v1/admin/admins/${id}/activate`, { method: 'POST' });
  }

  async getScopes(): Promise<any> {
    return this.request('/api/v1/scopes');
  }

  // ==================== Auth (real bcrypt + JWT with scopes) ====================
  async register(data: { username: string; email: string; password: string }): Promise<any> {
    return this.request('/api/v1/admin/auth/register', { method: 'POST', body: JSON.stringify(data) });
  }

  async changePassword(oldPassword: string, newPassword: string): Promise<void> {
    await this.request('/api/v1/admin/auth/change-password', { method: 'POST', body: JSON.stringify({ old_password: oldPassword, new_password: newPassword }) });
  }

  async enable2FA(): Promise<{ secret: string; issuer: string }> {
    return this.request('/api/v1/admin/auth/2fa/enable', { method: 'POST' });
  }

  async disable2FA(code: string): Promise<void> {
    await this.request('/api/v1/admin/auth/2fa/disable', { method: 'POST', body: JSON.stringify({ code }) });
  }

  // ==================== Health ====================
  async healthCheck(): Promise<boolean> {
    try {
      await this.request('/health');
      return true;
    } catch { return false; }
  }
}

export const whiteLabelAdminApi = new WhiteLabelAdminApiService();
export default whiteLabelAdminApi;
