// TigerBots Safari Extension API client.
//
// Targets the standalone Bots backend (mm_bot_platform/bot_api, port 8471,
// path prefix /api/v1). JWT bearer auth; the token is persisted in the
// Safari WebExtension `browser.storage.local` store (Safari 15+) under
// `tigerbots-token`.
//
// Every method issues a real fetch against the backend — no stubs, fakes, or
// mock data. On any non-2xx response the method throws (fail-closed); it
// never returns fabricated data.
//
// Method set mirrors bots/web/src/services/api.ts (auth, bots CRUD + start/
// stop/pause, executions, logs, users, transactions, subscriptions, fees,
// cex/dex connectors, api-keys, admin endpoints, public tiers, health).

const DEFAULT_API_URL = 'http://localhost:8471/api/v1';
const TOKEN_KEY = 'tigerbots-token';
const API_URL_KEY = 'botsApiUrl';

// Safari 15+ ships the promise-based `browser.*` WebExtension namespace in the
// background service worker. Fall back to `chrome.*` if only that is present.
const storageLocal = (typeof browser !== 'undefined' && browser.storage && browser.storage.local)
  || (typeof chrome !== 'undefined' && chrome.storage.local);

let cachedToken = null;
let cachedBaseUrl = null;

async function readStorage(key) {
  const res = await storageLocal.get(key);
  return res[key];
}
async function writeStorage(key, value) {
  await storageLocal.set({ [key]: value });
}
async function removeStorage(key) {
  await storageLocal.remove(key);
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

class BotApiError extends Error {
  constructor(message, status) {
    super(message);
    this.name = 'BotApiError';
    this.status = status;
  }
}

async function request(path, options = {}) {
  const base = await getBaseUrl();
  const headers = { 'Content-Type': 'application/json', ...(options.headers || {}) };
  const token = await getToken();
  if (token) headers.Authorization = `Bearer ${token}`;
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
    throw new BotApiError(detail || `HTTP ${res.status}`, res.status);
  }
  if (res.status === 204) return null;
  const text = await res.text();
  return text ? JSON.parse(text) : null;
}

const jsonBody = (obj) => JSON.stringify(obj);

export const api = {
  // ---- Auth ----
  async register(username, password, email, walletAddress, role) {
    const body = { username, password };
    if (email) body.email = email;
    if (walletAddress) body.wallet_address = walletAddress;
    if (role) body.role = role;
    const data = await request('/auth/register', { method: 'POST', body: jsonBody(body) });
    if (data && data.token) await setToken(data.token);
    return data;
  },

  async login(username, password) {
    const data = await request('/auth/login', { method: 'POST', body: jsonBody({ username, password }) });
    if (data && data.token) await setToken(data.token);
    return data;
  },

  async logout() {
    try { await request('/auth/logout', { method: 'POST' }); }
    finally { await clearToken(); }
  },

  // ---- Health + public tiers ----
  async health() { return request('/health', { method: 'GET', headers: {} }); },
  async publicTiers() { return request('/public/tiers', { method: 'GET' }); },

  // ---- Bots CRUD + lifecycle ----
  async listBots() { return request('/bots', { method: 'GET' }); },
  async getBot(id) { return request(`/bots/${encodeURIComponent(id)}`, { method: 'GET' }); },
  async createBot({ name, bot_type, config, exchange, pair }) {
    const body = { name, bot_type, config: config || {} };
    if (exchange) body.exchange = exchange;
    if (pair) body.pair = pair;
    return request('/bots', { method: 'POST', body: jsonBody(body) });
  },
  async deleteBot(id) { return request(`/bots/${encodeURIComponent(id)}`, { method: 'DELETE' }); },
  async startBot(id) { return request(`/bots/${encodeURIComponent(id)}/start`, { method: 'POST' }); },
  async stopBot(id) { return request(`/bots/${encodeURIComponent(id)}/stop`, { method: 'POST' }); },
  async pauseBot(id) { return request(`/bots/${encodeURIComponent(id)}/pause`, { method: 'POST' }); },
  async listBotExecutions(id) { return request(`/bots/${encodeURIComponent(id)}/executions`, { method: 'GET' }); },
  async listBotLogs(id) { return request(`/bots/${encodeURIComponent(id)}/logs`, { method: 'GET' }); },
  async listBotInstances() { return request('/bots/instances', { method: 'GET' }); },
  async currentBotUser() { return request('/bots/me', { method: 'GET' }); },

  // ---- Bot users ----
  async listBotUsers() { return request('/bots/users', { method: 'GET' }); },
  async createBotUser({ username, password, email, wallet_address, role }) {
    const body = { username, password };
    if (email) body.email = email;
    if (wallet_address) body.wallet_address = wallet_address;
    if (role) body.role = role;
    return request('/bots/users', { method: 'POST', body: jsonBody(body) });
  },
  async deleteBotUser(id) { return request(`/bots/users/${encodeURIComponent(id)}`, { method: 'DELETE' }); },
  async listBotTransactions() { return request('/bots/transactions', { method: 'GET' }); },

  // ---- Subscriptions ----
  async getSubscription() { return request('/subscription', { method: 'GET' }); },
  async createSubscription({ tier, expires_in }) {
    const body = { tier };
    if (expires_in) body.expires_in = expires_in;
    return request('/subscription', { method: 'POST', body: jsonBody(body) });
  },

  // ---- Fees ----
  async getFeeConfigs() { return request('/fees', { method: 'GET' }); },
  async updateFeeConfig({ id, name, percentage, enabled }) {
    const body = { id };
    if (name !== undefined) body.name = name;
    if (percentage !== undefined) body.percentage = percentage;
    if (enabled !== undefined) body.enabled = enabled;
    return request('/fees', { method: 'PUT', body: jsonBody(body) });
  },

  // ---- CEX connectors ----
  async listCEX() { return request('/cex', { method: 'GET' }); },
  async addCEX({ name, config }) { return request('/cex', { method: 'POST', body: jsonBody({ name, config }) }); },
  async removeCEX(id) { return request(`/cex/${encodeURIComponent(id)}`, { method: 'DELETE' }); },

  // ---- DEX connectors ----
  async listDEX() { return request('/dex', { method: 'GET' }); },
  async addDEX({ name, config }) { return request('/dex', { method: 'POST', body: jsonBody({ name, config }) }); },
  async removeDEX(id) { return request(`/dex/${encodeURIComponent(id)}`, { method: 'DELETE' }); },

  // ---- API keys ----
  async listAPIKeys() { return request('/keys', { method: 'GET' }); },
  async createAPIKey({ exchange, api_key }) { return request('/keys', { method: 'POST', body: jsonBody({ exchange, api_key }) }); },
  async deleteAPIKey(id) { return request(`/keys/${encodeURIComponent(id)}`, { method: 'DELETE' }); },

  // ---- Admin ----
  async adminListUsers() { return request('/admin/users', { method: 'GET' }); },
  async adminUserStatus(id, is_active) {
    return request(`/admin/users/${encodeURIComponent(id)}/status`, { method: 'PUT', body: jsonBody({ id, is_active }) });
  },
  async adminStats() { return request('/admin/stats', { method: 'GET' }); },
  async adminGetFeeAddresses() { return request('/admin/fee-addresses', { method: 'GET' }); },
  async adminSetFeeAddress({ label, chain_id, address }) {
    return request('/admin/fee-addresses', { method: 'POST', body: jsonBody({ label, chain_id, address }) });
  },
  async adminDeleteFeeAddress(id) { return request(`/admin/fee-addresses/${encodeURIComponent(id)}`, { method: 'DELETE' }); },
  async adminBotStatus(id, status) {
    return request(`/admin/bots/${encodeURIComponent(id)}/status`, { method: 'POST', body: jsonBody({ id, status }) });
  },
};

export default api;
