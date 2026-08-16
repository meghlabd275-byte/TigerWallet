/**
 * TigerWallet White Label Admin - Complete API Service
 * API client for White Label Admin operations
 * Connected to Go Backend with PostgreSQL and Redis
 */

const API_BASE_URL = process.env.NEXT_PUBLIC_WHITELABEL_ADMIN_API || 'http://localhost:8082';

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

  // ==================== Domain backends (governance records; no fund movement) ====================
  // Futures
  async getFuturesPositions(): Promise<any> { return this.request('/api/v1/admin/futures'); }
  async getFuturesPosition(id: string): Promise<any> { return this.request(`/api/v1/admin/futures/${id}`); }
  async createFuturesPosition(data: any): Promise<any> { return this.request('/api/v1/admin/futures', { method: 'POST', body: JSON.stringify(data) }); }
  async updateFuturesPosition(id: string, data: any): Promise<any> { return this.request(`/api/v1/admin/futures/${id}`, { method: 'PUT', body: JSON.stringify(data) }); }
  async deleteFuturesPosition(id: string): Promise<void> { return this.request(`/api/v1/admin/futures/${id}`, { method: 'DELETE' }); }
  async updateFuturesPositionStatus(id: string, status: string): Promise<any> { return this.request(`/api/v1/admin/futures/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) }); }

  // Options
  async getOptionsContracts(): Promise<any> { return this.request('/api/v1/admin/options'); }
  async getOptionsContract(id: string): Promise<any> { return this.request(`/api/v1/admin/options/${id}`); }
  async createOptionsContract(data: any): Promise<any> { return this.request('/api/v1/admin/options', { method: 'POST', body: JSON.stringify(data) }); }
  async updateOptionsContract(id: string, data: any): Promise<any> { return this.request(`/api/v1/admin/options/${id}`, { method: 'PUT', body: JSON.stringify(data) }); }
  async deleteOptionsContract(id: string): Promise<void> { return this.request(`/api/v1/admin/options/${id}`, { method: 'DELETE' }); }
  async updateOptionsContractStatus(id: string, status: string): Promise<any> { return this.request(`/api/v1/admin/options/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) }); }

  // Copy-trading
  async getCopyTradingConfigs(): Promise<any> { return this.request('/api/v1/admin/copy-trading'); }
  async getCopyTradingConfig(id: string): Promise<any> { return this.request(`/api/v1/admin/copy-trading/${id}`); }
  async createCopyTradingConfig(data: any): Promise<any> { return this.request('/api/v1/admin/copy-trading', { method: 'POST', body: JSON.stringify(data) }); }
  async updateCopyTradingConfig(id: string, data: any): Promise<any> { return this.request(`/api/v1/admin/copy-trading/${id}`, { method: 'PUT', body: JSON.stringify(data) }); }
  async deleteCopyTradingConfig(id: string): Promise<void> { return this.request(`/api/v1/admin/copy-trading/${id}`, { method: 'DELETE' }); }
  async updateCopyTradingConfigStatus(id: string, status: string): Promise<any> { return this.request(`/api/v1/admin/copy-trading/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) }); }

  // Convert
  async getConvertOrders(): Promise<any> { return this.request('/api/v1/admin/convert'); }
  async getConvertOrder(id: string): Promise<any> { return this.request(`/api/v1/admin/convert/${id}`); }
  async createConvertOrder(data: any): Promise<any> { return this.request('/api/v1/admin/convert', { method: 'POST', body: JSON.stringify(data) }); }
  async updateConvertOrder(id: string, data: any): Promise<any> { return this.request(`/api/v1/admin/convert/${id}`, { method: 'PUT', body: JSON.stringify(data) }); }
  async deleteConvertOrder(id: string): Promise<void> { return this.request(`/api/v1/admin/convert/${id}`, { method: 'DELETE' }); }
  async updateConvertOrderStatus(id: string, status: string): Promise<any> { return this.request(`/api/v1/admin/convert/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) }); }

  // Onramp (governance approve/reject; no fund movement)
  async getOnrampOrders(): Promise<any> { return this.request('/api/v1/admin/onramp'); }
  async getOnrampOrder(id: string): Promise<any> { return this.request(`/api/v1/admin/onramp/${id}`); }
  async createOnrampOrder(data: any): Promise<any> { return this.request('/api/v1/admin/onramp', { method: 'POST', body: JSON.stringify(data) }); }
  async updateOnrampOrder(id: string, data: any): Promise<any> { return this.request(`/api/v1/admin/onramp/${id}`, { method: 'PUT', body: JSON.stringify(data) }); }
  async deleteOnrampOrder(id: string): Promise<void> { return this.request(`/api/v1/admin/onramp/${id}`, { method: 'DELETE' }); }
  async approveOnrampOrder(id: string): Promise<any> { return this.request(`/api/v1/admin/onramp/${id}/approve`, { method: 'POST' }); }
  async rejectOnrampOrder(id: string, reason: string): Promise<any> { return this.request(`/api/v1/admin/onramp/${id}/reject`, { method: 'POST', body: JSON.stringify({ reason }) }); }

  // Offramp (governance approve/reject; no fund movement)
  async getOfframpOrders(): Promise<any> { return this.request('/api/v1/admin/offramp'); }
  async getOfframpOrder(id: string): Promise<any> { return this.request(`/api/v1/admin/offramp/${id}`); }
  async createOfframpOrder(data: any): Promise<any> { return this.request('/api/v1/admin/offramp', { method: 'POST', body: JSON.stringify(data) }); }
  async updateOfframpOrder(id: string, data: any): Promise<any> { return this.request(`/api/v1/admin/offramp/${id}`, { method: 'PUT', body: JSON.stringify(data) }); }
  async deleteOfframpOrder(id: string): Promise<void> { return this.request(`/api/v1/admin/offramp/${id}`, { method: 'DELETE' }); }
  async approveOfframpOrder(id: string): Promise<any> { return this.request(`/api/v1/admin/offramp/${id}/approve`, { method: 'POST' }); }
  async rejectOfframpOrder(id: string, reason: string): Promise<any> { return this.request(`/api/v1/admin/offramp/${id}/reject`, { method: 'POST', body: JSON.stringify({ reason }) }); }

  // P2P clients
  async getP2PClients(): Promise<any> { return this.request('/api/v1/admin/p2p-clients'); }
  async getP2PClient(id: string): Promise<any> { return this.request(`/api/v1/admin/p2p-clients/${id}`); }
  async createP2PClient(data: any): Promise<any> { return this.request('/api/v1/admin/p2p-clients', { method: 'POST', body: JSON.stringify(data) }); }
  async updateP2PClient(id: string, data: any): Promise<any> { return this.request(`/api/v1/admin/p2p-clients/${id}`, { method: 'PUT', body: JSON.stringify(data) }); }
  async deleteP2PClient(id: string): Promise<void> { return this.request(`/api/v1/admin/p2p-clients/${id}`, { method: 'DELETE' }); }
  async updateP2PClientStatus(id: string, status: string): Promise<any> { return this.request(`/api/v1/admin/p2p-clients/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) }); }

  // Partners (status + approve/reject governance)
  async getPartners(): Promise<any> { return this.request('/api/v1/admin/partners'); }
  async getPartner(id: string): Promise<any> { return this.request(`/api/v1/admin/partners/${id}`); }
  async createPartner(data: any): Promise<any> { return this.request('/api/v1/admin/partners', { method: 'POST', body: JSON.stringify(data) }); }
  async updatePartner(id: string, data: any): Promise<any> { return this.request(`/api/v1/admin/partners/${id}`, { method: 'PUT', body: JSON.stringify(data) }); }
  async deletePartner(id: string): Promise<void> { return this.request(`/api/v1/admin/partners/${id}`, { method: 'DELETE' }); }
  async updatePartnerStatus(id: string, status: string): Promise<any> { return this.request(`/api/v1/admin/partners/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) }); }
  async approvePartner(id: string): Promise<any> { return this.request(`/api/v1/admin/partners/${id}/approve`, { method: 'POST' }); }
  async rejectPartner(id: string, reason: string): Promise<any> { return this.request(`/api/v1/admin/partners/${id}/reject`, { method: 'POST', body: JSON.stringify({ reason }) }); }

  // Rewards
  async getRewardCampaigns(): Promise<any> { return this.request('/api/v1/admin/rewards'); }
  async getRewardCampaign(id: string): Promise<any> { return this.request(`/api/v1/admin/rewards/${id}`); }
  async createRewardCampaign(data: any): Promise<any> { return this.request('/api/v1/admin/rewards', { method: 'POST', body: JSON.stringify(data) }); }
  async updateRewardCampaign(id: string, data: any): Promise<any> { return this.request(`/api/v1/admin/rewards/${id}`, { method: 'PUT', body: JSON.stringify(data) }); }
  async deleteRewardCampaign(id: string): Promise<void> { return this.request(`/api/v1/admin/rewards/${id}`, { method: 'DELETE' }); }
  async updateRewardCampaignStatus(id: string, status: string): Promise<any> { return this.request(`/api/v1/admin/rewards/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) }); }

  // Marketing
  async getMarketingCampaigns(): Promise<any> { return this.request('/api/v1/admin/marketing'); }
  async getMarketingCampaign(id: string): Promise<any> { return this.request(`/api/v1/admin/marketing/${id}`); }
  async createMarketingCampaign(data: any): Promise<any> { return this.request('/api/v1/admin/marketing', { method: 'POST', body: JSON.stringify(data) }); }
  async updateMarketingCampaign(id: string, data: any): Promise<any> { return this.request(`/api/v1/admin/marketing/${id}`, { method: 'PUT', body: JSON.stringify(data) }); }
  async deleteMarketingCampaign(id: string): Promise<void> { return this.request(`/api/v1/admin/marketing/${id}`, { method: 'DELETE' }); }
  async updateMarketingCampaignStatus(id: string, status: string): Promise<any> { return this.request(`/api/v1/admin/marketing/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) }); }

  // ==================== Structured RBAC (integrates with scopes) ====================
  async getAdminRoles(): Promise<any> { return this.request('/api/v1/admin/admin-roles'); }
  async createAdminRole(data: { name: string; description?: string; permissions: string[] }): Promise<any> { return this.request('/api/v1/admin/admin-roles', { method: 'POST', body: JSON.stringify(data) }); }
  async getAdminRole(id: string): Promise<any> { return this.request(`/api/v1/admin/admin-roles/${id}`); }
  async updateAdminRole(id: string, data: any): Promise<any> { return this.request(`/api/v1/admin/admin-roles/${id}`, { method: 'PUT', body: JSON.stringify(data) }); }
  async deleteAdminRole(id: string): Promise<void> { return this.request(`/api/v1/admin/admin-roles/${id}`, { method: 'DELETE' }); }
  async getAdminPermissions(): Promise<any> { return this.request('/api/v1/admin/admin-permissions'); }
  async createAdminPermission(data: { name: string; description?: string; category?: string }): Promise<any> { return this.request('/api/v1/admin/admin-permissions', { method: 'POST', body: JSON.stringify(data) }); }
  async assignAdminRole(adminId: string, roleId: string): Promise<any> { return this.request(`/api/v1/admin/admins/${adminId}/role`, { method: 'POST', body: JSON.stringify({ role_id: roleId }) }); }
  async revokeAdminRole(adminId: string, roleId: string): Promise<any> { return this.request(`/api/v1/admin/admins/${adminId}/role/${roleId}`, { method: 'DELETE' }); }
  async getAdminEffectivePermissions(adminId: string): Promise<any> { return this.request(`/api/v1/admin/admins/${adminId}/permissions`); }

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
