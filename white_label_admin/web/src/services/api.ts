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
      throw new Error(error.message || `API Error: ${response.status}`);
    }
    return response.json();
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
  async getUsers(params?: { page?: number; pageSize?: number }): Promise<any> {
    const query = new URLSearchParams();
    if (params) Object.entries(params).forEach(([k, v]) => v && query.append(k, String(v)));
    return this.request(`/api/v1/users?${query}`);
  }

  async getUser(id: string): Promise<any> {
    return this.request(`/api/v1/users/${id}`);
  }

  async suspendUser(id: string): Promise<void> {
    return this.request(`/api/v1/users/${id}/suspend`, { method: 'POST' });
  }

  // ==================== KYC ====================
  async getKYCRequests(): Promise<any> {
    return this.request('/api/v1/kyc');
  }

  async approveKYC(id: string): Promise<any> {
    return this.request(`/api/v1/kyc/${id}/approve`, { method: 'POST' });
  }

  async rejectKYC(id: string, reason: string): Promise<any> {
    return this.request(`/api/v1/kyc/${id}/reject`, { method: 'POST', body: JSON.stringify({ reason }) });
  }

  // ==================== Transactions ====================
  async getTransactions(params?: { page?: number; pageSize?: number }): Promise<any> {
    const query = new URLSearchParams();
    if (params) Object.entries(params).forEach(([k, v]) => v && query.append(k, String(v)));
    return this.request(`/api/v1/transactions?${query}`);
  }

  async getTransaction(id: string): Promise<any> {
    return this.request(`/api/v1/transactions/${id}`);
  }

  // ==================== Withdrawals ====================
  async getWithdrawals(): Promise<any> {
    return this.request('/api/v1/withdrawals');
  }

  async approveWithdrawal(id: string): Promise<any> {
    return this.request(`/api/v1/withdrawals/${id}/approve`, { method: 'POST' });
  }

  async rejectWithdrawal(id: string, reason: string): Promise<any> {
    return this.request(`/api/v1/withdrawals/${id}/reject`, { method: 'POST', body: JSON.stringify({ reason }) });
  }

  // ==================== Tokens ====================
  async getTokens(): Promise<any> {
    return this.request('/api/v1/tokens');
  }

  async listToken(address: string): Promise<any> {
    return this.request(`/api/v1/tokens/${address}/list`, { method: 'POST' });
  }

  // ==================== Fees ====================
  async getFees(): Promise<any> {
    return this.request('/api/v1/fees');
  }

  async updateFees(config: any): Promise<any> {
    return this.request('/api/v1/fees', { method: 'PUT', body: JSON.stringify(config) });
  }

  // ==================== Analytics ====================
  async getDashboardStats(): Promise<any> {
    return this.request('/api/v1/dashboard/stats');
  }

  async getAnalytics(period?: string): Promise<any> {
    const query = period ? `?period=${period}` : '';
    return this.request(`/api/v1/analytics${query}`);
  }

  // ==================== Settings ====================
  async getConfig(): Promise<any> {
    return this.request('/api/v1/config');
  }

  async updateConfig(config: Record<string, any>): Promise<any> {
    return this.request('/api/v1/config', { method: 'PUT', body: JSON.stringify(config) });
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
      await this.request('/api/v1/health');
      return true;
    } catch { return false; }
  }
}

export const whiteLabelAdminApi = new WhiteLabelAdminApiService();
export default whiteLabelAdminApi;
