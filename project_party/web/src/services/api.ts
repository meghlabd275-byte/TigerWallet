// API Service - ProjectParty
const API_URL = 'http://localhost:8106/api/v1';

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

  // ============== Token Listings ==============
  async getListings() {
    return this.request('/listings');
  }
  async getListing(id: string) {
    return this.request(`/listings/${id}`);
  }
  async createListing(data: any) {
    return this.request('/listings', { method: 'POST', body: JSON.stringify(data) });
  }
  async updateListingStatus(id: string, status: string) {
    return this.request(`/listings/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) });
  }
  async featureListing(id: string) {
    return this.request(`/listings/${id}/featured`, { method: 'POST' });
  }

  // ============== Launchpad (IDO/Presale) ==============
  async getLaunchpads() {
    return this.request('/launchpad');
  }
  async getLaunchpad(id: string) {
    return this.request(`/launchpad/${id}`);
  }
  async createLaunchpad(data: any) {
    return this.request('/launchpad/create', { method: 'POST', body: JSON.stringify(data) });
  }
  async contributeLaunchpad(id: string, amount: string, userId?: string) {
    return this.request(`/launchpad/${id}/contribute`, { method: 'POST', body: JSON.stringify({ amount, user_id: userId }) });
  }
  async claimLaunchpad(id: string, userId: string) {
    return this.request(`/launchpad/${id}/claim`, { method: 'POST', body: JSON.stringify({ user_id: userId }) });
  }
  async cancelLaunchpad(id: string) {
    return this.request(`/launchpad/${id}/cancel`, { method: 'POST' });
  }

  // ============== Market Making ==============
  async getMakerOrders(tokenId?: string) {
    return this.request(`/market-making/orders${tokenId ? `?token_id=${tokenId}` : ''}`);
  }
  async createMakerOrder(data: any) {
    return this.request('/market-making/orders', { method: 'POST', body: JSON.stringify(data) });
  }
  async updateOrderStatus(id: string, status: string) {
    return this.request(`/market-making/orders/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) });
  }
  async getMarketMakerStatus(tokenId: string) {
    return this.request(`/market-making/status/${tokenId}`);
  }
  async addLiquidity(data: any) {
    return this.request('/market-making/liquidity/add', { method: 'POST', body: JSON.stringify(data) });
  }
  async removeLiquidity(data: any) {
    return this.request('/market-making/liquidity/remove', { method: 'POST', body: JSON.stringify(data) });
  }

  // ============== Pricing ==============
  async setTokenPrice(tokenId: string, price: string) {
    return this.request('/pricing/set', { method: 'POST', body: JSON.stringify({ token_id: tokenId, price }) });
  }
  async getTokenPrice(tokenId: string) {
    return this.request(`/pricing/${tokenId}`);
  }
  async getPriceHistory(tokenId: string) {
    return this.request(`/pricing/history/${tokenId}`);
  }
  async updateTokenPrice(tokenId: string, price: string) {
    return this.request('/pricing/update', { method: 'POST', body: JSON.stringify({ token_id: tokenId, price }) });
  }

  // ============== Analytics ==============
  async getTradingVolume() {
    return this.request('/analytics/volume');
  }
  async getLiquidity() {
    return this.request('/analytics/liquidity');
  }
  async getHolderCount(tokenId?: string) {
    return this.request(`/analytics/holders${tokenId ? `?token_id=${tokenId}` : ''}`);
  }
  async getTransactionCount() {
    return this.request('/analytics/transactions');
  }

  // ============== Compliance ==============
  async requestAudit(tokenId: string, auditType: string) {
    return this.request('/compliance/audit', { method: 'POST', body: JSON.stringify({ token_id: tokenId, audit_type: auditType }) });
  }
  async getAuditStatus(tokenId: string) {
    return this.request(`/compliance/audit/${tokenId}`);
  }
  async submitKYC(tokenId: string) {
    return this.request('/compliance/kyc/submit', { method: 'POST', body: JSON.stringify({ token_id: tokenId }) });
  }
  async getKYCStatus(tokenId: string) {
    return this.request(`/compliance/kyc/${tokenId}`);
  }

  // ============== Fees ==============
  async getListingFees() {
    return this.request('/fees');
  }
  async calculateFees(listingType: string, features: string[]) {
    return this.request('/fees/calculate', { method: 'POST', body: JSON.stringify({ listing_type: listingType, features }) });
  }
  async payFees(amount: string, paymentMethod: string) {
    return this.request('/fees/pay', { method: 'POST', body: JSON.stringify({ amount, payment_method: paymentMethod }) });
  }
}

export const api = new ApiService();
