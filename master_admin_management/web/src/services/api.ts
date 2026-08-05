/**
 * TigerWallet Master Admin - Complete API Service
 * High-performance API client for Master Admin operations
 * Connected to Go Backend with PostgreSQL and Redis
 */

const API_BASE_URL = process.env.NEXT_PUBLIC_MASTER_ADMIN_API || 'http://localhost:9091';
const WS_BASE_URL = process.env.NEXT_PUBLIC_MASTER_ADMIN_WS || 'ws://localhost:9091';

class MasterAdminApiService {
  private baseURL: string;
  private wsURL: string;
  private token: string | null = null;

  constructor() {
    this.baseURL = API_BASE_URL;
    this.wsURL = WS_BASE_URL;
    if (typeof window !== 'undefined') {
      this.token = localStorage.getItem('master_admin_token');
    }
  }

  setToken(token: string) {
    this.token = token;
    if (typeof window !== 'undefined') {
      localStorage.setItem('master_admin_token', token);
    }
  }

  clearToken() {
    this.token = null;
    if (typeof window !== 'undefined') {
      localStorage.removeItem('master_admin_token');
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

  async refreshToken(): Promise<{ token: string }> {
    return this.request('/api/v1/auth/refresh', { method: 'POST' });
  }

  // ==================== White Label Management ====================
  async getWhiteLabels(params?: { page?: number; pageSize?: number; status?: string }): Promise<any> {
    const query = new URLSearchParams();
    if (params) Object.entries(params).forEach(([k, v]) => v && query.append(k, String(v)));
    return this.request(`/api/v1/whitelabels?${query}`);
  }

  async getWhiteLabel(id: string): Promise<any> {
    return this.request(`/api/v1/whitelabels/${id}`);
  }

  async createWhiteLabel(data: any): Promise<any> {
    return this.request('/api/v1/whitelabels', { method: 'POST', body: JSON.stringify(data) });
  }

  async updateWhiteLabel(id: string, data: any): Promise<any> {
    return this.request(`/api/v1/whitelabels/${id}`, { method: 'PUT', body: JSON.stringify(data) });
  }

  async deleteWhiteLabel(id: string): Promise<void> {
    return this.request(`/api/v1/whitelabels/${id}`, { method: 'DELETE' });
  }

  async approveWhiteLabel(id: string): Promise<any> {
    return this.request(`/api/v1/whitelabels/${id}/approve`, { method: 'POST' });
  }

  async suspendWhiteLabel(id: string): Promise<any> {
    return this.request(`/api/v1/whitelabels/${id}/suspend`, { method: 'POST' });
  }

  // ==================== Master Admin Management ====================
  async getMasterAdmins(params?: { whiteLabelId?: string }): Promise<any> {
    const query = new URLSearchParams();
    if (params?.whiteLabelId) query.append('whiteLabelId', params.whiteLabelId);
    return this.request(`/api/v1/master-admins?${query}`);
  }

  async getMasterAdmin(id: string): Promise<any> {
    return this.request(`/api/v1/master-admins/${id}`);
  }

  async createMasterAdmin(data: any): Promise<any> {
    return this.request('/api/v1/master-admins', { method: 'POST', body: JSON.stringify(data) });
  }

  async updateMasterAdmin(id: string, data: any): Promise<any> {
    return this.request(`/api/v1/master-admins/${id}`, { method: 'PUT', body: JSON.stringify(data) });
  }

  async deleteMasterAdmin(id: string): Promise<void> {
    return this.request(`/api/v1/master-admins/${id}`, { method: 'DELETE' });
  }

  async suspendMasterAdmin(id: string): Promise<any> {
    return this.request(`/api/v1/master-admins/${id}/suspend`, { method: 'POST' });
  }

  // ==================== User Management ====================
  async getUsers(params?: { whiteLabelId?: string; page?: number; pageSize?: number }): Promise<any> {
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

  async banUser(id: string): Promise<void> {
    return this.request(`/api/v1/users/${id}/ban`, { method: 'POST' });
  }

  // ==================== Transactions ====================
  async getTransactions(params?: { whiteLabelId?: string; page?: number; pageSize?: number }): Promise<any> {
    const query = new URLSearchParams();
    if (params) Object.entries(params).forEach(([k, v]) => v && query.append(k, String(v)));
    return this.request(`/api/v1/transactions?${query}`);
  }

  async getTransaction(id: string): Promise<any> {
    return this.request(`/api/v1/transactions/${id}`);
  }

  // ==================== Analytics ====================
  async getDashboardStats(): Promise<any> {
    return this.request('/api/v1/dashboard/stats');
  }

  async getAnalytics(period?: string): Promise<any> {
    const query = period ? `?period=${period}` : '';
    return this.request(`/api/v1/analytics${query}`);
  }

  // ==================== System ====================
  async getSystemStatus(): Promise<any[]> {
    return this.request('/api/v1/system/status');
  }

  async getSystemMetrics(): Promise<any> {
    return this.request('/api/v1/system/metrics');
  }

  // ==================== Configuration ====================
  async getConfig(): Promise<any> {
    return this.request('/api/v1/config');
  }

  async updateConfig(config: Record<string, any>): Promise<any> {
    return this.request('/api/v1/config', { method: 'PUT', body: JSON.stringify(config) });
  }

  // ==================== Audit Logs ====================
  async getAuditLogs(params?: { page?: number; pageSize?: number }): Promise<any> {
    const query = new URLSearchParams();
    if (params) Object.entries(params).forEach(([k, v]) => v && query.append(k, String(v)));
    return this.request(`/api/v1/audit-logs?${query}`);
  }

  // ==================== Health ====================
  async healthCheck(): Promise<boolean> {
    try {
      await this.request('/api/v1/health');
      return true;
    } catch { return false; }
  }
}

export const masterAdminApi = new MasterAdminApiService();
export default masterAdminApi;
