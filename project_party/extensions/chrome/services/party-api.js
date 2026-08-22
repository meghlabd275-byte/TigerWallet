// ProjectParty Chrome Extension API client.
//
// Targets the standalone ProjectParty backend (project_party/go/cmd/main.go,
// port 8106, path prefix /api/v1, JWT auth + RBAC). The token is persisted in
// chrome.storage.local under `projectparty-token`. Base URL is configurable
// via the `partyApiUrl` storage entry and defaults to the local dev endpoint.
//
// Every method issues a real fetch against the backend — no stubs, fakes, or
// mock data. On any non-2xx response the method throws (fail-closed); it
// never returns fabricated data.
//
// Method set matches project_party/web/src/services/api.ts + the discovery,
// pricing, analytics, compliance routes the task requires.

const DEFAULT_API_URL = 'http://localhost:8106/api/v1';
const TOKEN_KEY = 'projectparty-token';
const API_URL_KEY = 'partyApiUrl';

let cachedToken = null;
let cachedBaseUrl = null;

async function readStorage(key) {
  const res = await chrome.storage.local.get(key);
  return res[key];
}
async function writeStorage(key, value) {
  await chrome.storage.local.set({ [key]: value });
}
async function removeStorage(key) {
  await chrome.storage.local.remove(key);
}

export async function getBaseUrl() {
  if (cachedBaseUrl) return cachedBaseUrl;
  cachedBaseUrl = (await readStorage(API_URL_KEY)) || DEFAULT_API_URL;
  return cachedBaseUrl;
}

export async function setBaseUrl(url) {
  const cleaned = url && url.replace(/\/$/, '');
  cachedBaseUrl = cleaned ? `${cleaned}/api/v1` : DEFAULT_API_URL;
  await writeStorage(API_URL_KEY, cachedBaseUrl);
  return cachedBaseUrl;
}

export async function setToken(token) {
  cachedToken = token;
  if (token) await writeStorage(TOKEN_KEY, token);
  else await removeStorage(TOKEN_KEY);
}

export async function getToken() {
  if (cachedToken) return cachedToken;
  cachedToken = (await readStorage(TOKEN_KEY)) || null;
  return cachedToken;
}

export async function clearToken() {
  cachedToken = null;
  await removeStorage(TOKEN_KEY);
}

class PartyApiError extends Error {
  constructor(message, status) {
    super(message);
    this.name = 'PartyApiError';
    this.status = status;
  }
}

async function request(path, options = {}, { absolute = false } = {}) {
  const base = await getBaseUrl();
  const urlBase = absolute ? base.replace(/\/api\/v1$/, '') : base;
  const headers = { 'Content-Type': 'application/json', ...(options.headers || {}) };
  const token = await getToken();
  if (token) headers.Authorization = `Bearer ${token}`;
  const init = { ...options, headers };
  if (init.body === undefined) delete init.body;
  const res = await fetch(`${urlBase}${path}`, init);
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
  // ---- Health ----
  async getHealth() { return request('/health', { method: 'GET', headers: {} }, { absolute: true }); },

  // ---- Auth ----
  async register(username, password) {
    const data = await request('/auth/register', { method: 'POST', body: jsonBody({ username, password }) });
    if (data && data.token) await setToken(data.token);
    return data;
  },
  async login(username, password) {
    const data = await request('/auth/login', { method: 'POST', body: jsonBody({ username, password }) });
    if (data && data.token) await setToken(data.token);
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
  async createLaunchpad(data) { return request('/launchpad/create', { method: 'POST', body: jsonBody(data) }); },
  async contribute(id, amount) {
    return request(`/launchpad/${enc(id)}/contribute`, { method: 'POST', body: jsonBody({ amount }) });
  },
  async claimTokens(id) { return request(`/launchpad/${enc(id)}/claim`, { method: 'POST' }); },
  async cancelLaunchpad(id) { return request(`/launchpad/${enc(id)}/cancel`, { method: 'POST' }); },

  // ---- Market-making ----
  async getMakerOrders(tokenId) {
    const path = tokenId ? `/market-making/orders?token_id=${enc(tokenId)}` : '/market-making/orders';
    return request(path, { method: 'GET' });
  },
  async getMarketMakerStatus(tokenId) { return request(`/market-making/status/${enc(tokenId)}`, { method: 'GET' }); },
  async createMakerOrders(data) { return request('/market-making/orders', { method: 'POST', body: jsonBody(data) }); },
  async updateOrderStatus(id, status) {
    return request(`/market-making/orders/${enc(id)}/status`, { method: 'PUT', body: jsonBody({ status }) });
  },
  async addLiquidity(data) { return request('/market-making/liquidity/add', { method: 'POST', body: jsonBody(data) }); },
  async removeLiquidity(data) { return request('/market-making/liquidity/remove', { method: 'POST', body: jsonBody(data) }); },

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
  async getAuditStatus(tokenId) { return request(`/compliance/audit/${enc(tokenId)}`, { method: 'GET' }); },
  async getKYCStatus(tokenId) { return request(`/compliance/kyc/${enc(tokenId)}`, { method: 'GET' }); },
  async requestAudit(data) { return request('/compliance/audit', { method: 'POST', body: jsonBody(data) }); },
  async submitKYC(data) { return request('/compliance/kyc/submit', { method: 'POST', body: jsonBody(data) }); },

  // ---- Fees ----
  async getListingFees() { return request('/fees', { method: 'GET' }); },
  async calculateFees(data) { return request('/fees/calculate', { method: 'POST', body: jsonBody(data) }); },
  async payFees(data) { return request('/fees/pay', { method: 'POST', body: jsonBody(data) }); },

  // ---- Favorites (auth) ----
  async listFavorites() { return request('/favorites', { method: 'GET' }); },
  async addFavorite(data) { return request('/favorites', { method: 'POST', body: jsonBody(data) }); },
  async removeFavorite(id) { return request(`/favorites/${enc(id)}`, { method: 'DELETE' }); },
};

export default api;
