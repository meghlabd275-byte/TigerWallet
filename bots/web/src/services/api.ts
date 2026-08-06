// API Service - Connects to Bots Backend
const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8107/api/v1';

class ApiService {
  private token: string | null = null;

  setToken(token: string) {
    this.token = token;
  }

  private async request(endpoint: string, options: RequestInit = {}) {
    const headers: HeadersInit = { 'Content-Type': 'application/json' };
    if (this.token) {
      (headers as any)['Authorization'] = `Bearer ${this.token}`;
    }

    const response = await fetch(`${API_BASE_URL}${endpoint}`, {
      ...options,
      headers: { ...headers, ...options.headers }
    });

    if (!response.ok) throw new Error(await response.text());
    return response.json();
  }

  async getBots() {
    return this.request('/bots');
  }

  async getBot(id: string) {
    return this.request(`/bots/${id}`);
  }

  async createBot(data: any) {
    return this.request('/bots', { method: 'POST', body: JSON.stringify(data) });
  }

  async startBot(id: string) {
    return this.request(`/bots/${id}/start`, { method: 'POST' });
  }

  async stopBot(id: string) {
    return this.request(`/bots/${id}/stop`, { method: 'POST' });
  }

  async deleteBot(id: string) {
    return this.request(`/bots/${id}`, { method: 'DELETE' });
  }

  async getBotTrades(id: string) {
    return this.request(`/bots/${id}/trades`);
  }

  async getBotPerformance(id: string, period?: string) {
    return this.request(`/bots/${id}/performance${period ? `?period=${period}` : ''}`);
  }

  async getStrategies() {
    return this.request('/strategies');
  }

  async getTrades() {
    return this.request('/trades');
  }

  async getMarketPrices(pairs: string[]) {
    return this.request(`/market/prices?pairs=${pairs.join(',')}`);
  }
}

export const api = new ApiService();
export default api;
