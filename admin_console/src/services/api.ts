/**
 * TigerWallet Admin Console - API Service
 * Complete API client for admin console operations
 */

const API_BASE_URL = process.env.NEXT_PUBLIC_ADMIN_CONSOLE_API || 'http://localhost:8082';

class AdminConsoleApiService {
  private token: string | null = null;

  constructor() {
    if (typeof window !== 'undefined') {
      this.token = localStorage.getItem('admin_console_token');
    }
  }

  setToken(token: string) {
    this.token = token;
    if (typeof window !== 'undefined') localStorage.setItem('admin_console_token', token);
  }

  clearToken() {
    this.token = null;
    if (typeof window !== 'undefined') localStorage.removeItem('admin_console_token');
  }

  private getHeaders(): HeadersInit {
    return { 'Content-Type': 'application/json', ...(this.token && { 'Authorization': `Bearer ${this.token}` }) };
  }

  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const res = await fetch(`${API_BASE_URL}${endpoint}`, { ...options, headers: { ...this.getHeaders(), ...options.headers } });
    if (!res.ok) throw new Error(`API Error: ${res.status}`);
    return res.json();
  }

  // Auth
  async login(email: string, password: string) { return this.request('/api/v1/auth/login', { method: 'POST', body: JSON.stringify({ email, password }) }); }
  async logout() { await this.request('/api/v1/auth/logout', { method: 'POST' }); this.clearToken(); }

  // Dashboard
  async getDashboardStats() { return this.request('/api/v1/dashboard/stats'); }
  async getAnalytics(period?: string) { return this.request(`/api/v1/analytics${period ? `?period=${period}` : ''}`); }

  // Users
  async getUsers(params?: any) { 
    const q = new URLSearchParams(); if (params) Object.entries(params).forEach(([k, v]) => v && q.append(k, String(v)));
    return this.request(`/api/v1/users?${q}`); 
  }
  async suspendUser(id: string) { return this.request(`/api/v1/users/${id}/suspend`, { method: 'POST' }); }

  // KYC
  async getKYC() { return this.request('/api/v1/kyc'); }
  async approveKYC(id: string) { return this.request(`/api/v1/kyc/${id}/approve`, { method: 'POST' }); }
  async rejectKYC(id: string, reason: string) { return this.request(`/api/v1/kyc/${id}/reject`, { method: 'POST', body: JSON.stringify({ reason }) }); }

  // Transactions
  async getTransactions(params?: any) { 
    const q = new URLSearchParams(); if (params) Object.entries(params).forEach(([k, v]) => v && q.append(k, String(v)));
    return this.request(`/api/v1/transactions?${q}`); 
  }

  // Tokens
  async getTokens() { return this.request('/api/v1/tokens'); }
  async listToken(address: string) { return this.request(`/api/v1/tokens/${address}/list`, { method: 'POST' }); }

  // Fees
  async getFees() { return this.request('/api/v1/fees'); }
  async updateFees(config: any) { return this.request('/api/v1/fees', { method: 'PUT', body: JSON.stringify(config) }); }

  // System
  async getSystemStatus() { return this.request('/api/v1/system/status'); }

  // Theme
  async getConfig() { return this.request('/api/v1/config'); }
  async updateConfig(config: any) { return this.request('/api/v1/config', { method: 'PUT', body: JSON.stringify(config) }); }

  async healthCheck() { try { await this.request('/api/v1/health'); return true; } catch { return false; } }
}

export const adminConsoleApi = new AdminConsoleApiService();
export default adminConsoleApi;
