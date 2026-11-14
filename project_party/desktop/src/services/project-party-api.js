// ProjectParty Desktop API client.
//
// Targets the standalone ProjectParty backend (project_party/go/cmd/main.go,
// port 8106, path prefix /api/v1, JWT auth + RBAC). The token is stored in
// localStorage under `projectparty-token` (renderer) and mirrored to in-memory
// module state for main-process use. Base URL is configurable via the
// `PROJECT_PARTY_API_URL` / `VITE_API_URL` env var and defaults to the local
// dev endpoint.
//
// Every method issues a real fetch against the backend — no stubs, fakes, or
// mock data. On any non-2xx response the method throws (fail-closed); it
// never returns fabricated data.
//
// Method set matches project_party/web/src/services/api.ts + the discovery,
// pricing, analytics, compliance routes the task requires.

const BUILD_API_BASE_URL =
  (typeof process !== 'undefined' && process.env && (process.env.PROJECT_PARTY_API_URL || process.env.VITE_API_URL)) ||
  'http://localhost:8106/api/v1';

// A packaged build cannot be pointed at a hosted backend by an env var, so the
// operator can override the endpoint at runtime; the value persists in
// localStorage and is read on every request.
const API_BASE_KEY = 'projectparty-api-base';

function getApiBaseUrl() {
  try {
    if (typeof localStorage !== 'undefined') {
      const stored = localStorage.getItem(API_BASE_KEY);
      if (stored) return stored.replace(/\/+$/, '');
    }
  } catch { /* localStorage may be unavailable outside the renderer */ }
  return BUILD_API_BASE_URL;
}

export function setApiBaseUrl(value) {
  const raw = String(value || '').trim().replace(/\/+$/, '');
  if (!raw) throw new Error('A server URL is required.');
  if (!/^https?:\/\//i.test(raw)) throw new Error('The URL must start with http:// or https://');
  if (typeof localStorage !== 'undefined') localStorage.setItem(API_BASE_KEY, raw);
  return raw;
}

export function getApiBaseUrlForDisplay() {
  return getApiBaseUrl();
}

const TOKEN_KEY = 'projectparty-token';

let authToken = null;

function loadStoredToken() {
  try {
    if (typeof localStorage !== 'undefined') {
      authToken = localStorage.getItem(TOKEN_KEY);
    }
  } catch (_) {
    // localStorage unavailable (main process); token managed in-memory
  }
}
loadStoredToken();

export function setToken(token) {
  authToken = token;
  try {
    if (typeof localStorage !== 'undefined') {
      if (token) localStorage.setItem(TOKEN_KEY, token);
      else localStorage.removeItem(TOKEN_KEY);
    }
  } catch (_) {
    /* ignore */
  }
}

export function getToken() {
  return authToken;
}

export function clearToken() {
  setToken(null);
}

class PartyApiError extends Error {
  constructor(message, status) {
    super(message);
    this.name = 'PartyApiError';
    this.status = status;
  }
}

async function request(path, options = {}, { absolute = false } = {}) {
  const apiBase = getApiBaseUrl();
  const base = absolute ? apiBase.replace(/\/api\/v1$/, '') : apiBase;
  const headers = { 'Content-Type': 'application/json', ...(options.headers || {}) };
  if (authToken) headers.Authorization = `Bearer ${authToken}`;
  const init = { ...options, headers };
  if (init.body === undefined) delete init.body;
  const res = await fetch(`${base}${path}`, init);
  if (!res.ok) {
    let detail = '';
    try {
      const body = await res.json();
      detail = body.error || body.message || JSON.stringify(body);
    } catch {
      try { detail = await res.text(); } catch { detail = res.statusText; }
    }
    throw new PartyApiError(detail || `HTTP ${res.status}`, res.status);
  }
  if (res.status === 204) return null;
  const text = await res.text();
  return text ? JSON.parse(text) : null;
}

const jsonBody = (obj) => JSON.stringify(obj);
const enc = (s) => encodeURIComponent(s);

export const api = {
  // ---- Health (lives at /health, outside /api/v1) ----
  async getHealth() {
    return request('/health', { method: 'GET', headers: {} }, { absolute: true });
  },

  // ---- Auth ----
  async register(username, password) {
    const data = await request('/auth/register', {
      method: 'POST',
      body: jsonBody({ username, password }),
    });
    if (data && data.token) setToken(data.token);
    return data;
  },

  async login(username, password) {
    const data = await request('/auth/login', {
      method: 'POST',
      body: jsonBody({ username, password }),
    });
    if (data && data.token) setToken(data.token);
    return data;
  },

  // ---- Discovery (public) ----
  async getCoins() { return request('/coins', { method: 'GET' }); },
  async searchTokens(q) { return request(`/search?q=${enc(q)}`, { method: 'GET' }); },
  async getFeatured() { return request('/featured', { method: 'GET' }); },
  async getTrending() { return request('/trending', { method: 'GET' }); },
  async getMarket() { return request('/market', { method: 'GET' }); },

  // ---- Tokens ----
  async listTokens(status) {
    const path = status ? `/tokens?status=${enc(status)}` : '/tokens';
    return request(path, { method: 'GET' });
  },
  async getToken(id) { return request(`/tokens/${enc(id)}`, { method: 'GET' }); },
  async createToken(data) { return request('/tokens', { method: 'POST', body: jsonBody(data) }); },
  async updateToken(id, data) { return request(`/tokens/${enc(id)}`, { method: 'PUT', body: jsonBody(data) }); },
  async deleteToken(id) { return request(`/tokens/${enc(id)}`, { method: 'DELETE' }); },
  async submitToken(id) { return request(`/tokens/${enc(id)}/submit`, { method: 'POST' }); },
  async approveToken(id) { return request(`/tokens/${enc(id)}/approve`, { method: 'POST' }); },
  async rejectToken(id) { return request(`/tokens/${enc(id)}/reject`, { method: 'POST' }); },

  // ---- Listings ----
  async listListings(status) {
    const path = status ? `/listings?status=${enc(status)}` : '/listings';
    return request(path, { method: 'GET' });
  },
  async getListing(id) { return request(`/listings/${enc(id)}`, { method: 'GET' }); },
  async createListing(data) { return request('/listings', { method: 'POST', body: jsonBody(data) }); },
  async updateListingStatus(id, status) {
    return request(`/listings/${enc(id)}/status`, { method: 'PUT', body: jsonBody({ status }) });
  },
  async featureListing(id) { return request(`/listings/${enc(id)}/featured`, { method: 'POST' }); },

  // ---- Launchpad ----
  async listLaunchpads(status) {
    const path = status ? `/launchpad?status=${enc(status)}` : '/launchpad';
    return request(path, { method: 'GET' });
  },
  async getLaunchpad(id) { return request(`/launchpad/${enc(id)}`, { method: 'GET' }); },
  async createLaunchpad(data) { return request('/launchpad', { method: 'POST', body: jsonBody(data) }); },
  async contribute(id, amount) {
    return request(`/launchpad/${enc(id)}/contribute`, { method: 'POST', body: jsonBody({ amount }) });
  },
  async listParticipations(id) { return request(`/launchpad/${enc(id)}/participations`, { method: 'GET' }); },
  async claimTokens(id) { return request(`/launchpad/${enc(id)}/claim`, { method: 'POST' }); },
  async cancelLaunchpad(id) { return request(`/launchpad/${enc(id)}/cancel`, { method: 'POST' }); },

  // ---- Market-making ----
  async getMakerOrders(tokenId) {
    const path = tokenId ? `/orders?token_id=${enc(tokenId)}` : '/orders';
    return request(path, { method: 'GET' });
  },
  async getMarketMakerStatus(tokenId) {
    return request(`/marketmaking/status/${enc(tokenId)}`, { method: 'GET' });
  },
  async listMakerTrades(tokenId) {
    const path = tokenId ? `/market-making/trades?token_id=${enc(tokenId)}` : '/market-making/trades';
    return request(path, { method: 'GET' });
  },
  async createMakerOrders(data) {
    return request('/orders', { method: 'POST', body: jsonBody(data) });
  },
  async updateOrderStatus(id, status) {
    return request(`/orders/${enc(id)}/status`, { method: 'PUT', body: jsonBody({ status }) });
  },
  async addLiquidity(data) {
    return request('/liquidity/add', { method: 'POST', body: jsonBody(data) });
  },
  async removeLiquidity(data) {
    return request('/liquidity/remove', { method: 'POST', body: jsonBody(data) });
  },

  // ---- Pricing ----
  async getTokenPrice(tokenId) { return request(`/pricing/${enc(tokenId)}`, { method: 'GET' }); },
  async getPriceHistory(tokenId) { return request(`/pricing/history/${enc(tokenId)}`, { method: 'GET' }); },
  async setTokenPrice(tokenId, price) {
    return request('/pricing/set', { method: 'POST', body: jsonBody({ token_id: tokenId, price }) });
  },
  async updatePrice(tokenId, price) {
    return request('/pricing/update', { method: 'POST', body: jsonBody({ token_id: tokenId, price }) });
  },

  // ---- Analytics (public) ----
  async getTradingVolume() { return request('/analytics/volume', { method: 'GET' }); },
  async getLiquidity() { return request('/analytics/liquidity', { method: 'GET' }); },
  async getHolderCount() { return request('/analytics/holders', { method: 'GET' }); },
  async getTransactionCount() { return request('/analytics/transactions', { method: 'GET' }); },

  // ---- Compliance ----
  async getComplianceStatus(tokenId) { return request(`/compliance/status/${enc(tokenId)}`, { method: 'GET' }); },
  async getAuditStatus(tokenId) { return request(`/compliance/audit/${enc(tokenId)}`, { method: 'GET' }); },
  async getKYCStatus(tokenId) { return request(`/compliance/kyc/${enc(tokenId)}`, { method: 'GET' }); },
  async requestAudit(data) { return request('/compliance/audit', { method: 'POST', body: jsonBody(data) }); },
  async submitKYC(data) { return request('/compliance/kyc/submit', { method: 'POST', body: jsonBody(data) }); },

  // ---- Fees ----
  async getListingFees() { return request('/fees', { method: 'GET' }); },
  async calculateFees(data) { return request('/fees/calculate', { method: 'POST', body: jsonBody(data) }); },
  async payFees(data) { return request('/fees/pay', { method: 'POST', body: jsonBody(data) }); },

  // ---- Admin (JWT + admin/super_admin role; backend enforces) ----
  async verifyTokenContract(id) { return request(`/tokens/${enc(id)}/verify-contract`, { method: 'POST' }); },
  async listFeePayments(status) {
    const path = status ? `/fees/payments?status=${enc(status)}` : '/fees/payments';
    return request(path, { method: 'GET' });
  },
  async verifyFeePayment(paymentId) { return request(`/fees/verify/${enc(paymentId)}`, { method: 'POST' }); },

  // ---- Favorites (auth) ----
  async listFavorites() { return request('/favorites', { method: 'GET' }); },
  async addFavorite(data) { return request('/favorites', { method: 'POST', body: jsonBody(data) }); },
  async removeFavorite(id) { return request(`/favorites/${enc(id)}`, { method: 'DELETE' }); },
};

export default api;
