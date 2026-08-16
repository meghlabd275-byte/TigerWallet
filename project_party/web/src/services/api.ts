// API Service — WL-ProjectParty standalone backend.
// Targets the WL backend (:8464 external / :8106 internal) via the Vite dev
// proxy (see vite.config.ts) so the SPA uses same-origin /api paths and never
// hardcodes a host. Production nginx rewrites the same paths. One real fetch
// per backend route — no stubs, no fakes, no mocks.
const API_BASE = '/api/v1';

export interface ApiError extends Error {
  status?: number;
}

class ApiService {
  private token: string | null = null;

  constructor() {
    if (typeof window !== 'undefined') {
      this.token = localStorage.getItem('projectparty-token');
    }
  }

  setToken(token: string) {
    this.token = token;
    if (typeof window !== 'undefined') {
      localStorage.setItem('projectparty-token', token);
    }
  }

  clearToken() {
    this.token = null;
    if (typeof window !== 'undefined') {
      localStorage.removeItem('projectparty-token');
    }
  }

  getStoredToken() {
    return this.token;
  }

  private async request<T = any>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const headers: Record<string, string> = { 'Content-Type': 'application/json' };
    if (this.token) headers['Authorization'] = `Bearer ${this.token}`;
    const res = await fetch(`${API_BASE}${endpoint}`, {
      ...options,
      headers: { ...headers, ...(options.headers as Record<string, string>) }
    });
    if (!res.ok) {
      let detail = '';
      try {
        const body = await res.json();
        detail = body.error || body.message || JSON.stringify(body);
      } catch {
        try { detail = await res.text(); } catch { detail = res.statusText; }
      }
      const err = new Error(detail || `Request failed (${res.status})`) as ApiError;
      err.status = res.status;
      throw err;
    }
    if (res.status === 204) return undefined as T;
    const text = await res.text();
    return (text ? JSON.parse(text) : undefined) as T;
  }

  // ==================== Health (proxied at /health, outside /api/v1) ====================
  async getHealth() {
    const headers: Record<string, string> = {};
    if (this.token) headers['Authorization'] = `Bearer ${this.token}`;
    const res = await fetch('/health', { headers });
    if (!res.ok) throw new Error(`Health check failed (${res.status})`);
    return res.json();
  }

  // ==================== Auth ====================
  async register(email: string, password: string, role?: string) {
    return this.request('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ email, password, role: role || 'user' })
    });
  }

  async login(email: string, password: string) {
    return this.request<{ token: string; user_id: string; email: string; wl_client_id: string }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password })
    });
  }

  // ==================== Tokens ====================
  async createToken(data: {
    name: string; symbol: string; contract_address?: string; chain_id?: number;
    decimals?: number; logo_url?: string; description?: string; website?: string;
    status?: string; listing_type?: string;
  }) {
    return this.request('/tokens', { method: 'POST', body: JSON.stringify(data) });
  }

  async listTokens(status?: string) {
    return this.request<{ tokens: any[] }>(`/tokens${status ? `?status=${encodeURIComponent(status)}` : ''}`);
  }

  async getToken(id: string) {
    return this.request(`/tokens/${id}`);
  }

  async updateToken(id: string, data: {
    name: string; symbol: string; contract_address?: string; chain_id?: number;
    decimals?: number; logo_url?: string; description?: string; website?: string;
    status?: string; listing_type?: string;
  }) {
    return this.request(`/tokens/${id}`, { method: 'PUT', body: JSON.stringify(data) });
  }

  async deleteToken(id: string) {
    return this.request(`/tokens/${id}`, { method: 'DELETE' });
  }

  // ==================== Listings ====================
  async createListing(data: {
    token_id: string; pair?: string; pair_token?: string; base_token?: string;
    quote_token?: string; launch_type?: string; initial_price?: string;
    start_time?: string; end_time?: string; status?: string;
  }) {
    return this.request('/listings', { method: 'POST', body: JSON.stringify(data) });
  }

  async listListings(status?: string) {
    return this.request<{ listings: any[] }>(`/listings${status ? `?status=${encodeURIComponent(status)}` : ''}`);
  }

  // ==================== Launchpad ====================
  async createLaunchpadProject(data: {
    token_id: string; name: string; description?: string;
    start_time?: string; end_time?: string; total_supply?: string;
    price_per_token?: string; status?: string;
  }) {
    return this.request('/launchpad', { method: 'POST', body: JSON.stringify(data) });
  }

  async listLaunchpadProjects(status?: string) {
    return this.request<{ launchpad_projects: any[] }>(`/launchpad${status ? `?status=${encodeURIComponent(status)}` : ''}`);
  }

  async getLaunchpadProject(id: string) {
    return this.request(`/launchpad/${id}`);
  }

  async participateInLaunchpad(id: string, amount: string) {
    return this.request(`/launchpad/${id}/participate`, {
      method: 'POST',
      body: JSON.stringify({ amount })
    });
  }

  async listParticipations(id: string) {
    return this.request<{ participations: any[] }>(`/launchpad/${id}/participations`);
  }

  // ==================== Market-making configs ====================
  async createMarketMakingConfig(data: {
    token_id: string; pair: string; spread?: string; enabled?: boolean;
  }) {
    return this.request('/market-making', { method: 'POST', body: JSON.stringify(data) });
  }

  async listMarketMakingConfigs() {
    return this.request<{ market_making_configs: any[] }>('/market-making');
  }

  // ==================== Fee configs ====================
  async createFeeConfig(data: {
    token_id: string; fee_type?: string; fee_percentage?: string;
    min_fee?: string; max_fee?: string; name?: string; percentage?: number;
    enabled?: boolean;
  }) {
    return this.request('/fees', { method: 'POST', body: JSON.stringify(data) });
  }

  async listFeeConfigs() {
    return this.request<{ fee_configs: any[] }>('/fees');
  }

  // ==================== Favorites ====================
  async addFavorite(data: { token_id: string; notes?: string } | string) {
    const body = typeof data === 'string' ? { token_id: data } : data;
    return this.request('/favorites', { method: 'POST', body: JSON.stringify(body) });
  }

  async listFavorites() {
    return this.request<{ favorites: any[] }>('/favorites');
  }

  async removeFavorite(id: string) {
    return this.request(`/favorites/${id}`, { method: 'DELETE' });
  }
}

export const api = new ApiService();
