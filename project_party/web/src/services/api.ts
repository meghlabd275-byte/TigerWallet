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
  async register(email: string, password: string) {
    // Public self-registration always creates a plain user; roles/scopes are
    // assigned by the WL client owner, never from the client.
    return this.request('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ email, password })
    });
  }

  async login(email: string, password: string) {
    return this.request<{ token: string; user_id: string; email: string; wl_client_id: string; role?: string; scopes?: string[] }>('/auth/login', {
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
    return this.request('/market-making/configs', { method: 'POST', body: JSON.stringify(data) });
  }

  async listMarketMakingConfigs() {
    return this.request<{ market_making_configs: any[] }>('/market-making/configs');
  }

  async deleteMarketMakingConfig(id: string) {
    return this.request(`/market-making/configs/${id}`, { method: 'DELETE' });
  }

  // ==================== Market-making orders + settled trades ====================
  async listMakerOrders(tokenId?: string) {
    const q = tokenId ? `?token_id=${encodeURIComponent(tokenId)}` : '';
    return this.request<{ orders: any[] }>(`/market-making/orders${q}`);
  }

  async createMakerOrder(data: { token_id: string; side: 'buy' | 'sell'; price: string; quantity: string }) {
    return this.request<{ order: any }>('/market-making/orders', { method: 'POST', body: JSON.stringify(data) });
  }

  async listMakerTrades(tokenId?: string) {
    const q = tokenId ? `?token_id=${encodeURIComponent(tokenId)}` : '';
    return this.request<{ trades: any[] }>(`/market-making/trades${q}`);
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

  // Admin-only: verify a fee payment's on-chain tx receipt.
  async verifyFeePayment(paymentId: string) {
    return this.request<{ message: string; status: string; tx_hash: string; block_number: string; gas_used: number }>(
      `/fees/verify/${paymentId}`, { method: 'POST' }
    );
  }

  // Admin-only: verify a token's on-chain contract (ERC-20 interface check).
  async verifyTokenContract(tokenId: string) {
    return this.request<{ message: string; contract_verified: boolean; on_chain_name: string; on_chain_symbol: string; on_chain_decimals: number; on_chain_supply: string; match: boolean }>(
      `/tokens/${tokenId}/verify-contract`, { method: 'POST' }
    );
  }

  // ==================== Admin ====================
  async approveToken(id: string) {
    return this.request(`/tokens/${id}/approve`, { method: 'POST' });
  }

  async rejectToken(id: string, reason?: string) {
    return this.request(`/tokens/${id}/reject`, { method: 'POST', body: JSON.stringify({ reason }) });
  }

  async toggleFeatured(id: string) {
    return this.request(`/tokens/${id}/featured`, { method: 'POST' });
  }

  async listFeePayments(status?: string) {
    const q = status ? `?status=${encodeURIComponent(status)}` : '';
    return this.request<{ fee_payments: any[]; total: number }>(`/fees/payments${q}`);
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

  // ==================== Pricing ====================
  async getTokenPrice(tokenId: string) {
    return this.request<{ token_id: string; price: string; change_24h: string; market_cap: string; volume_24h: string }>(`/pricing?token_id=${tokenId}`);
  }

  async getTokenHistory(tokenId: string) {
    return this.request<{ history: any[] }>(`/pricing/history/${tokenId}`);
  }

  async getMarketData() {
    return this.request<{ tokens: any[]; total_market_cap: string; total_volume: string; active_tokens: number }>(`/market`);
  }

  // ==================== Analytics ====================
  async getHolders(tokenId?: string) {
    const query = tokenId ? `?token_id=${tokenId}` : '';
    return this.request<{ holders: any[]; total: number }>(`/analytics/holders${query}`);
  }

  async getTransactions(tokenId?: string, limit = 50) {
    const params = new URLSearchParams({ limit: String(limit) });
    if (tokenId) params.set('token_id', tokenId);
    return this.request<{ transactions: any[]; total: number }>(`/analytics/transactions?${params}`);
  }

  async getTokenVolume(tokenId?: string) {
    const query = tokenId ? `?token_id=${tokenId}` : '';
    return this.request<{ volume_24h: string; volume_7d: string; total: string }>(`/analytics/volume${query}`);
  }

  async getTokenLiquidity(tokenId?: string) {
    const query = tokenId ? `?token_id=${tokenId}` : '';
    return this.request<{ liquidity: any[]; total: string }>(`/analytics/liquidity${query}`);
  }

  // ==================== Compliance ====================
  async getKYCStatus(tokenId: string) {
    return this.request<{ kyc_status: string; audit_status: string; verified: boolean }>(`/compliance/kyc/${tokenId}`);
  }

  async submitKYC(data: { token_id: string; documents: string[] }) {
    return this.request<{ message: string; submission_id: string }>(`/compliance/kyc/submit`, { method: 'POST', body: JSON.stringify(data) });
  }

  async submitAudit(data: { token_id: string; audit_report: string; auditor: string }) {
    return this.request<{ message: string; audit_id: string }>(`/compliance/audit`, { method: 'POST', body: JSON.stringify(data) });
  }

  async getAuditReport(tokenId: string) {
    return this.request<{ audit: any }>(`/compliance/audit/${tokenId}`);
  }

  async getTokenStatus(tokenId: string) {
    return this.request<{ status: string; kyc_verified: boolean; audit_passed: boolean; listing_approved: boolean }>(`/compliance/status/${tokenId}`);
  }
}

export const api = new ApiService();
