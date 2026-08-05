/**
 * Admin Services API
 */
const API_BASE_URL = process.env.NEXT_PUBLIC_ADMIN_SERVICES_API || 'http://localhost:8091';

class AdminServicesApi {
  private token: string | null = null;
  constructor() { if (typeof window !== 'undefined') this.token = localStorage.getItem('admin_services_token'); }
  setToken(t: string) { this.token = t; if (typeof window !== 'undefined') localStorage.setItem('admin_services_token', t); }
  clearToken() { this.token = null; if (typeof window !== 'undefined') localStorage.removeItem('admin_services_token'); }
  private getHeaders() { return { 'Content-Type': 'application/json', ...(this.token && { 'Authorization': `Bearer ${this.token}` }) }; }
  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const res = await fetch(`${API_BASE_URL}${endpoint}`, { ...options, headers: { ...this.getHeaders(), ...options.headers } });
    if (!res.ok) throw new Error(`API Error: ${res.status}`);
    return res.json();
  }
  async login(email: string, password: string) { const r = await this.request<{ token: string }>('/api/v1/auth/login', { method: 'POST', body: JSON.stringify({ email, password }) }); this.setToken(r.token); return r; }
  async logout() { await this.request('/api/v1/auth/logout', { method: 'POST' }); this.clearToken(); }
  async getDashboard() { return this.request<any>('/api/v1/dashboard'); }
  async getDashboardStats() { return this.request<any>('/api/v1/dashboard/stats'); }
  async getServices() { return this.request<any>('/api/v1/services'); }
  async createService(data: any) { return this.request<any>('/api/v1/services', { method: 'POST', body: JSON.stringify(data) }); }
  async getService(id: string) { return this.request<any>(`/api/v1/services/${id}`); }
  async updateService(id: string, data: any) { return this.request<any>(`/api/v1/services/${id}`, { method: 'PUT', body: JSON.stringify(data) }); }
  async deleteService(id: string) { return this.request<any>(`/api/v1/services/${id}`, { method: 'DELETE' }); }
  async startService(id: string) { return this.request<any>(`/api/v1/services/${id}/start`, { method: 'POST' }); }
  async stopService(id: string) { return this.request<any>(`/api/v1/services/${id}/stop`, { method: 'POST' }); }
  async getServiceLogs(id: string) { return this.request<any>(`/api/v1/services/${id}/logs`); }
  async getServiceMetrics(id: string) { return this.request<any>(`/api/v1/services/${id}/metrics`); }
  async getWebhooks() { return this.request<any>('/api/v1/webhooks'); }
  async createWebhook(data: any) { return this.request<any>('/api/v1/webhooks', { method: 'POST', body: JSON.stringify(data) }); }
  async updateWebhook(id: string, data: any) { return this.request<any>(`/api/v1/webhooks/${id}`, { method: 'PUT', body: JSON.stringify(data) }); }
  async deleteWebhook(id: string) { return this.request<any>(`/api/v1/webhooks/${id}`, { method: 'DELETE' }); }
  async testWebhook(id: string) { return this.request<any>(`/api/v1/webhooks/${id}/test`, { method: 'POST' }); }
  async getJobs() { return this.request<any>('/api/v1/jobs'); }
  async createJob(data: any) { return this.request<any>('/api/v1/jobs', { method: 'POST', body: JSON.stringify(data) }); }
  async updateJob(id: string, data: any) { return this.request<any>(`/api/v1/jobs/${id}`, { method: 'PUT', body: JSON.stringify(data) }); }
  async deleteJob(id: string) { return this.request<any>(`/api/v1/jobs/${id}`, { method: 'DELETE' }); }
  async runJob(id: string) { return this.request<any>(`/api/v1/jobs/${id}/run`, { method: 'POST' }); }
  async getLogs() { return this.request<any>('/api/v1/logs'); }
  async getMetrics() { return this.request<any>('/api/v1/metrics'); }
}

export const adminServicesApi = new AdminServicesApi();
export default adminServicesApi;
