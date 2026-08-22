// TigerBots Desktop API client.
//
// Targets the standalone Bots backend (mm_bot_platform/bot_api, port 8471,
// path prefix /api/v1). JWT bearer auth; the token is stored in
// localStorage under `tigerbots-token` (renderer) and mirrored to the
// in-memory module state for main-process use. Base URL is configurable via
// the `BOTS_API_URL` / `VITE_API_URL` env var and defaults to the local dev
// endpoint.
//
// Every method issues a real fetch against the backend — no stubs, fakes, or
// mock data. On any non-2xx response the method throws (fail-closed); it
// never returns fabricated data.
//
// Method set mirrors bots/web/src/services/api.ts:
//   auth: register, login, logout
//   bots: listBots, getBot, createBot, deleteBot, startBot, stopBot, pauseBot,
//         listBotExecutions, listBotLogs, listBotInstances, currentBotUser
//   users: listBotUsers, createBotUser, deleteBotUser, listBotTransactions
//   subscriptions: getSubscription, createSubscription
//   fees: getFeeConfigs, updateFeeConfig
//   cex: listCEX, addCEX, removeCEX
//   dex: listDEX, addDEX, removeDEX
//   keys: listAPIKeys, createAPIKey, deleteAPIKey
//   admin: adminListUsers, adminUserStatus, adminStats, adminGetFeeAddresses,
//         adminSetFeeAddress, adminDeleteFeeAddress, adminBotStatus
//   public: publicTiers, health

const API_BASE_URL =
  (typeof process !== 'undefined' && process.env && (process.env.BOTS_API_URL || process.env.VITE_API_URL)) ||
  'http://localhost:8471/api/v1';

const TOKEN_KEY = 'tigerbots-token';

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

class BotApiError extends Error {
  constructor(message, status) {
    super(message);
    this.name = 'BotApiError';
    this.status = status;
  }
}

async function request(path, options = {}) {
  const headers = { 'Content-Type': 'application/json', ...(options.headers || {}) };
  if (authToken) headers.Authorization = `Bearer ${authToken}`;
  const init = { ...options, headers };
  // Bodyless POSTs must not send a body (the Go gin handler rejects empty
  // bodies when a JSON struct is bound).
  if (init.body === undefined) delete init.body;
  const res = await fetch(`${API_BASE_URL}${path}`, init);
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

function jsonBody(obj) {
  return JSON.stringify(obj);
}

export const api = {
  // ---- Auth ----
  async register(username, password, email, walletAddress, role) {
    const body = { username, password };
    if (email) body.email = email;
    if (walletAddress) body.wallet_address = walletAddress;
    if (role) body.role = role;
    const data = await request('/auth/register', {
      method: 'POST',
      body: jsonBody(body),
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

  async logout() {
    try {
      await request('/auth/logout', { method: 'POST' });
    } finally {
      clearToken();
    }
  },

  // ---- Health + public tiers ----
  async health() {
    return request('/health', { method: 'GET', headers: {} });
  },

  async publicTiers() {
    return request('/public/tiers', { method: 'GET' });
  },

  // ---- Bots CRUD + lifecycle ----
  async listBots() {
    return request('/bots', { method: 'GET' });
  },

  async getBot(id) {
    return request(`/bots/${encodeURIComponent(id)}`, { method: 'GET' });
  },

  async createBot({ name, bot_type, config, exchange, pair }) {
    const body = { name, bot_type, config: config || {} };
    if (exchange) body.exchange = exchange;
    if (pair) body.pair = pair;
    return request('/bots', { method: 'POST', body: jsonBody(body) });
  },

  async deleteBot(id) {
    return request(`/bots/${encodeURIComponent(id)}`, { method: 'DELETE' });
  },

  async startBot(id) {
    return request(`/bots/${encodeURIComponent(id)}/start`, { method: 'POST' });
  },

  async stopBot(id) {
    return request(`/bots/${encodeURIComponent(id)}/stop`, { method: 'POST' });
  },

  async pauseBot(id) {
    return request(`/bots/${encodeURIComponent(id)}/pause`, { method: 'POST' });
  },

  async listBotExecutions(id) {
    return request(`/bots/${encodeURIComponent(id)}/executions`, { method: 'GET' });
  },

  async listBotLogs(id) {
    return request(`/bots/${encodeURIComponent(id)}/logs`, { method: 'GET' });
  },

  async listBotInstances() {
    return request('/bots/instances', { method: 'GET' });
  },

  async currentBotUser() {
    return request('/bots/me', { method: 'GET' });
  },

  // ---- Bot users ----
  async listBotUsers() {
    return request('/bots/users', { method: 'GET' });
  },

  async createBotUser({ username, password, email, wallet_address, role }) {
    const body = { username, password };
    if (email) body.email = email;
    if (wallet_address) body.wallet_address = wallet_address;
    if (role) body.role = role;
    return request('/bots/users', { method: 'POST', body: jsonBody(body) });
  },

  async deleteBotUser(id) {
    return request(`/bots/users/${encodeURIComponent(id)}`, { method: 'DELETE' });
  },

  async listBotTransactions() {
    return request('/bots/transactions', { method: 'GET' });
  },

  // ---- Subscriptions ----
  async getSubscription() {
    return request('/subscription', { method: 'GET' });
  },

  async createSubscription({ tier, expires_in }) {
    const body = { tier };
    if (expires_in) body.expires_in = expires_in;
    return request('/subscription', { method: 'POST', body: jsonBody(body) });
  },

  // ---- Fees ----
  async getFeeConfigs() {
    return request('/fees', { method: 'GET' });
  },

  async updateFeeConfig({ id, name, percentage, enabled }) {
    const body = { id };
    if (name !== undefined) body.name = name;
    if (percentage !== undefined) body.percentage = percentage;
    if (enabled !== undefined) body.enabled = enabled;
    return request('/fees', { method: 'PUT', body: jsonBody(body) });
  },

  // ---- CEX connectors ----
  async listCEX() {
    return request('/cex', { method: 'GET' });
  },

  async addCEX({ name, config }) {
    return request('/cex', { method: 'POST', body: jsonBody({ name, config }) });
  },

  async removeCEX(id) {
    return request(`/cex/${encodeURIComponent(id)}`, { method: 'DELETE' });
  },

  // ---- DEX connectors ----
  async listDEX() {
    return request('/dex', { method: 'GET' });
  },

  async addDEX({ name, config }) {
    return request('/dex', { method: 'POST', body: jsonBody({ name, config }) });
  },

  async removeDEX(id) {
    return request(`/dex/${encodeURIComponent(id)}`, { method: 'DELETE' });
  },

  // ---- API keys ----
  async listAPIKeys() {
    return request('/keys', { method: 'GET' });
  },

  async createAPIKey({ exchange, api_key }) {
    return request('/keys', { method: 'POST', body: jsonBody({ exchange, api_key }) });
  },

  async deleteAPIKey(id) {
    return request(`/keys/${encodeURIComponent(id)}`, { method: 'DELETE' });
  },

  // ---- Admin (super-admin / finance-admin) ----
  async adminListUsers() {
    return request('/admin/users', { method: 'GET' });
  },

  async adminUserStatus(id, is_active) {
    return request(`/admin/users/${encodeURIComponent(id)}/status`, {
      method: 'PUT',
      body: jsonBody({ id, is_active }),
    });
  },

  async adminStats() {
    return request('/admin/stats', { method: 'GET' });
  },

  async adminGetFeeAddresses() {
    return request('/admin/fee-addresses', { method: 'GET' });
  },

  async adminSetFeeAddress({ label, chain_id, address }) {
    return request('/admin/fee-addresses', {
      method: 'POST',
      body: jsonBody({ label, chain_id, address }),
    });
  },

  async adminDeleteFeeAddress(id) {
    return request(`/admin/fee-addresses/${encodeURIComponent(id)}`, { method: 'DELETE' });
  },

  async adminBotStatus(id, status) {
    return request(`/admin/bots/${encodeURIComponent(id)}/status`, {
      method: 'POST',
      body: jsonBody({ id, status }),
    });
  },
};

export default api;
