// API Service - ProjectParty
const API_URL = 'http://localhost:8106/api/v1';

class ApiService {
  private token: string | null = null;

  setToken(token: string) { this.token = token; }

  private async request(endpoint: string, options: RequestInit = {}) {
    const headers: any = { 'Content-Type': 'application/json' };
    if (this.token) headers['Authorization'] = `Bearer ${this.token}`;
    const res = await fetch(`${API_URL}${endpoint}`, { ...options, headers: { ...headers, ...options.headers } });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  }

  async getCoins(network?: string) {
    return this.request(`/coins${network ? `?network=${network}` : ''}`);
  }

  async getTokens(search?: string) {
    return this.request(`/search${search ? `?q=${search}` : ''}`);
  }

  async getToken(id: string) {
    return this.request(`/tokens/${id}`);
  }

  async getFeatured() {
    return this.request('/featured');
  }

  async getTrending() {
    return this.request('/trending');
  }

  async getMarket() {
    return this.request('/market');
  }

  async getFavorites() {
    return this.request('/favorites');
  }

  async addFavorite(tokenId: string) {
    return this.request('/favorites', { method: 'POST', body: JSON.stringify({ token_id: tokenId }) });
  }

  async removeFavorite(tokenId: string) {
    return this.request(`/favorites/${tokenId}`, { method: 'DELETE' });
  }

  async submitToken(data: any) {
    return this.request('/tokens', { method: 'POST', body: JSON.stringify(data) });
  }
}

export const api = new ApiService();
