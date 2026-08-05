/**
 * TigerWallet Admin System - Complete API Service
 * Connected to Go Backend with PostgreSQL and Redis
 */

const API_BASE_URL = process.env.NEXT_PUBLIC_ADMIN_SYSTEM_API || 'http://localhost:8090';

class AdminSystemApiService {
  private token: string | null = null;

  constructor() {
    if (typeof window !== 'undefined') {
      this.token = localStorage.getItem('admin_system_token');
    }
  }

  setToken(token: string) {
    this.token = token;
    if (typeof window !== 'undefined') localStorage.setItem('admin_system_token', token);
  }

  clearToken() {
    this.token = null;
    if (typeof window !== 'undefined') localStorage.removeItem('admin_system_token');
  }

  private getHeaders(): HeadersInit {
    return { 'Content-Type': 'application/json', ...(this.token && { 'Authorization': `Bearer ${this.token}` }) };
  }

  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const res = await fetch(`${API_BASE_URL}${endpoint}`, { ...options, headers: { ...this.getHeaders(), ...options.headers } });
    if (!res.ok) {
      const error = await res.json().catch(() => ({ message: 'Unknown error' }));
      throw new Error(error.message || `API Error: ${res.status}`);
    }
    return res.json();
  }

  // Auth
  async login(email: string, password: string) {
    const response = await this.request<{ token: string; user: any }>('/api/v1/auth/login', { method: 'POST', body: JSON.stringify({ email, password }) });
    this.setToken(response.token);
    return response;
  }

  async logout() {
    await this.request('/api/v1/auth/logout', { method: 'POST' });
    this.clearToken();
  }

  async refreshToken() {
    return this.request<{ token: string }>('/api/v1/auth/refresh', { method: 'POST' });
  }

  async forgotPassword(email: string) {
    return this.request<{ message: string }>('/api/v1/auth/forgot-password', { method: 'POST', body: JSON.stringify({ email }) });
  }

  async resetPassword(token: string, newPassword: string) {
    return this.request<{ message: string }>('/api/v1/auth/reset-password', { method: 'POST', body: JSON.stringify({ token, new_password: newPassword }) });
  }

  // Dashboard
  async getDashboard() {
    return this.request<any>('/api/v1/dashboard');
  }

  async getDashboardStats() {
    return this.request<any>('/api/v1/dashboard/stats');
  }

  // Users
  async getUsers(params?: { page?: number; limit?: number; role?: string }) {
    const q = new URLSearchParams();
    if (params) Object.entries(params).forEach(([k, v]) => v && q.append(k, String(v)));
    return this.request<any>(`/api/v1/users?${q}`);
  }

  async getUser(id: string) {
    return this.request<any>(`/api/v1/users/${id}`);
  }

  async createUser(data: { email: string; username: string; password: string; role: string; permissions?: string[]; white_label_id?: string }) {
    return this.request<any>('/api/v1/users', { method: 'POST', body: JSON.stringify(data) });
  }

  async updateUser(id: string, data: any) {
    return this.request<any>(`/api/v1/users/${id}`, { method: 'PUT', body: JSON.stringify(data) });
  }

  async deleteUser(id: string) {
    return this.request<any>(`/api/v1/users/${id}`, { method: 'DELETE' });
  }

  async activateUser(id: string) {
    return this.request<any>(`/api/v1/users/${id}/activate`, { method: 'POST' });
  }

  async deactivateUser(id: string) {
    return this.request<any>(`/api/v1/users/${id}/deactivate`, { method: 'POST' });
  }

  async resetUserPassword(id: string, newPassword: string) {
    return this.request<any>(`/api/v1/users/${id}/reset-password`, { method: 'POST', body: JSON.stringify({ new_password: newPassword }) });
  }

  async updateUserPermissions(id: string, permissions: string[]) {
    return this.request<any>(`/api/v1/users/${id}/permissions`, { method: 'PUT', body: JSON.stringify({ permissions }) });
  }

  // Config
  async getConfig() {
    return this.request<any>('/api/v1/config');
  }

  async getConfigItem(key: string) {
    return this.request<any>(`/api/v1/config/${key}`);
  }

  async updateConfig(data: { key: string; value: string; description?: string; is_encrypted?: boolean; category?: string }) {
    return this.request<any>('/api/v1/config', { method: 'PUT', body: JSON.stringify(data) });
  }

  async deleteConfig(key: string) {
    return this.request<any>(`/api/v1/config/${key}`, { method: 'DELETE' });
  }

  // Audit Logs
  async getAuditLogs(params?: { page?: number; limit?: number; action?: string; user_id?: string }) {
    const q = new URLSearchParams();
    if (params) Object.entries(params).forEach(([k, v]) => v && q.append(k, String(v)));
    return this.request<any>(`/api/v1/audit-logs?${q}`);
  }

  async exportAuditLogs(params?: { start_date?: string; end_date?: string }) {
    const q = new URLSearchParams();
    if (params) Object.entries(params).forEach(([k, v]) => v && q.append(k, String(v)));
    return this.request<any>(`/api/v1/audit-logs/export?${q}`);
  }

  // Notifications
  async getNotifications() {
    return this.request<any>('/api/v1/notifications');
  }

  async markNotificationRead(id: string) {
    return this.request<any>(`/api/v1/notifications/${id}/read`, { method: 'PUT' });
  }

  async markAllNotificationsRead() {
    return this.request<any>('/api/v1/notifications/read-all', { method: 'PUT' });
  }

  async deleteNotification(id: string) {
    return this.request<any>(`/api/v1/notifications/${id}`, { method: 'DELETE' });
  }

  // Feature Flags
  async getFeatureFlags() {
    return this.request<any>('/api/v1/features');
  }

  async createFeatureFlag(data: { name: string; description?: string; is_enabled?: boolean; rollout_percent?: number; target_roles?: string[] }) {
    return this.request<any>('/api/v1/features', { method: 'POST', body: JSON.stringify(data) });
  }

  async updateFeatureFlag(id: string, data: any) {
    return this.request<any>(`/api/v1/features/${id}`, { method: 'PUT', body: JSON.stringify(data) });
  }

  async deleteFeatureFlag(id: string) {
    return this.request<any>(`/api/v1/features/${id}`, { method: 'DELETE' });
  }

  // API Keys
  async getAPIKeys() {
    return this.request<any>('/api/v1/api-keys');
  }

  async createAPIKey(data: { name: string; permissions?: string[]; expires_in?: number }) {
    return this.request<any>('/api/v1/api-keys', { method: 'POST', body: JSON.stringify(data) });
  }

  async deleteAPIKey(id: string) {
    return this.request<any>(`/api/v1/api-keys/${id}`, { method: 'DELETE' });
  }

  async rotateAPIKey(id: string) {
    return this.request<any>(`/api/v1/api-keys/${id}/rotate`, { method: 'POST' });
  }

  // System
  async getSystemStatus() {
    return this.request<any>('/api/v1/system/status');
  }

  async getSystemMetrics() {
    return this.request<any>('/api/v1/system/metrics');
  }

  async getSystemHealth() {
    return this.request<any>('/api/v1/system/health');
  }

  // Health
  async healthCheck() {
    try {
      await this.request('/health');
      return true;
    } catch { return false; }
  }
}

export const adminSystemApi = new AdminSystemApiService();
export default adminSystemApi;
