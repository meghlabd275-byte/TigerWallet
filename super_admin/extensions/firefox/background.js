// Super Admin Chrome Extension - Background Service Worker
// Drives the real super_admin/go backend on port 8082 (JWT bearer auth).
const API_BASE_URL = 'http://localhost:8082/api/v1/admin';

// 12 governance domains exposed as read-only sections in the popup.
// `resource` is the path segment under /api/v1/admin.
const DOMAINS = [
  { id: 'futures', label: 'Futures', resource: 'futures' },
  { id: 'options', label: 'Options', resource: 'options' },
  { id: 'copy-trading', label: 'Copy Trading', resource: 'copy-trading' },
  { id: 'convert', label: 'Convert', resource: 'convert' },
  { id: 'onramp', label: 'Onramp', resource: 'onramp' },
  { id: 'offramp', label: 'Offramp', resource: 'offramp' },
  { id: 'p2p-clients', label: 'P2P Clients', resource: 'p2p-clients' },
  { id: 'partners', label: 'Partners', resource: 'partners' },
  { id: 'rewards', label: 'Rewards', resource: 'rewards' },
  { id: 'marketing', label: 'Marketing', resource: 'marketing' },
  { id: 'admin-roles', label: 'Admin Roles', resource: 'admin-roles' },
  { id: 'wl-control', label: 'WL Control', resource: 'wl-clients' }
];

// Handle messages from popup and content scripts
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  switch (request.action) {
    case 'getDashboard':
      fetchDashboard().then(sendResponse);
      return true;
    case 'getUsers':
      fetchUsers(request.params).then(sendResponse);
      return true;
    case 'getTransactions':
      fetchTransactions(request.params).then(sendResponse);
      return true;
    case 'getStats':
      fetchStats().then(sendResponse);
      return true;
    case 'getDomain':
      fetchDomain(request.domain).then(sendResponse);
      return true;
    case 'getDomains':
      sendResponse({ domains: DOMAINS });
      return true;
    case 'cryptoCards':
      handleCryptoCards(request.op, request.payload).then(sendResponse);
      return true;
    case 'getTheme':
      chrome.storage.local.get(['theme']).then(result => {
        sendResponse(result.theme || 'system');
      });
      return true;
    case 'setTheme':
      chrome.storage.local.set({ theme: request.theme });
      sendResponse({ success: true });
      return true;
    default:
      sendResponse({ error: 'Unknown action' });
  }
});

// API Functions
async function fetchDashboard() {
  try {
    const token = await getToken();
    const response = await fetch(`${API_BASE_URL}/stats`, {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    return await response.json();
  } catch (error) {
    return { error: error.message };
  }
}

async function fetchUsers(params = {}) {
  try {
    const token = await getToken();
    const query = new URLSearchParams(params).toString();
    const response = await fetch(`${API_BASE_URL}/users?${query}`, {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    return await response.json();
  } catch (error) {
    return { error: error.message };
  }
}

async function fetchTransactions(params = {}) {
  try {
    const token = await getToken();
    const query = new URLSearchParams(params).toString();
    const response = await fetch(`${API_BASE_URL}/transactions?${query}`, {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    return await response.json();
  } catch (error) {
    return { error: error.message };
  }
}

async function fetchStats() {
  try {
    const token = await getToken();
    const response = await fetch(`${API_BASE_URL}/stats`, {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    return await response.json();
  } catch (error) {
    return { error: error.message };
  }
}

// Real GET for any of the 12 governance domains. Returns the parsed JSON or
// an { error } object; never fabricates data.
async function fetchDomain(domain) {
  try {
    const token = await getToken();
    const response = await fetch(`${API_BASE_URL}/${domain}`, {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    const text = await response.text();
    let parsed = null;
    try { parsed = text ? JSON.parse(text) : null; }
    catch (_) { parsed = { raw: text }; }
    if (!response.ok) {
      const msg = (parsed && parsed.error) ? parsed.error : `HTTP ${response.status}`;
      return { error: msg, status: response.status };
    }
    return parsed || { error: 'No data returned by the domain service.' };
  } catch (error) {
    return { error: error.message || 'Failed to reach super-admin backend.' };
  }
}

// Crypto Cards management API: maps to /api/v1/admin/crypto-cards routes on
// the super_admin/go backend (port 8082). Same JWT bearer auth as fetchDomain.
const cryptoCardsAPI = {
  resource: 'crypto-cards',
  getAll: (params = {}) => cryptoCardsRequest('GET', '', null, params),
  getOne: (id) => cryptoCardsRequest('GET', id),
  create: (data) => cryptoCardsRequest('POST', '', data),
  update: (id, data) => cryptoCardsRequest('PUT', id, data),
  delete: (id) => cryptoCardsRequest('DELETE', id),
  block: (id, reason) => cryptoCardsRequest('POST', id + '/block', reason ? { reason } : {}),
  activate: (id) => cryptoCardsRequest('POST', id + '/activate', {}),
  setLimit: (id, data) => cryptoCardsRequest('PUT', id + '/limit', data),
  setStatus: (id, status) => cryptoCardsRequest('PUT', id + '/status', { status })
};

async function cryptoCardsRequest(method, suffix = '', body = null, params = {}) {
  try {
    const token = await getToken();
    const query = Object.keys(params).length ? '?' + new URLSearchParams(params).toString() : '';
    const url = `${API_BASE_URL}/${cryptoCardsAPI.resource}${suffix ? '/' + suffix : ''}${query}`;
    const opts = { method, headers: { 'Authorization': `Bearer ${token}` } };
    if (body !== null) {
      opts.headers['Content-Type'] = 'application/json';
      opts.body = JSON.stringify(body);
    }
    const response = await fetch(url, opts);
    const text = await response.text();
    let parsed = null;
    try { parsed = text ? JSON.parse(text) : null; }
    catch (_) { parsed = { raw: text }; }
    if (!response.ok) {
      const msg = (parsed && parsed.error) ? parsed.error : `HTTP ${response.status}`;
      return { error: msg, status: response.status };
    }
    return parsed || { error: 'No data returned by the crypto-cards service.' };
  } catch (error) {
    return { error: error.message || 'Failed to reach super-admin backend.' };
  }
}

async function handleCryptoCards(op, payload = {}) {
  switch (op) {
    case 'list': return cryptoCardsAPI.getAll(payload.params || {});
    case 'get': return cryptoCardsAPI.getOne(payload.id);
    case 'create': return cryptoCardsAPI.create(payload.body || {});
    case 'update': return cryptoCardsAPI.update(payload.id, payload.body || {});
    case 'delete': return cryptoCardsAPI.delete(payload.id);
    case 'block': return cryptoCardsAPI.block(payload.id, payload.reason);
    case 'activate': return cryptoCardsAPI.activate(payload.id);
    case 'setLimit': return cryptoCardsAPI.setLimit(payload.id, payload.body || {});
    case 'setStatus': return cryptoCardsAPI.setStatus(payload.id, payload.status);
    default: return { error: 'Unknown cryptoCards op: ' + op };
  }
}

async function getToken() {
  const result = await chrome.storage.local.get(['token']);
  return result.token;
}

// Handle extension install
chrome.runtime.onInstalled.addListener((details) => {
  if (details.reason === 'install') {
    console.log('TigerWallet Super Admin extension installed');
  }
});

// Badge update function
function updateBadge(count, color = '#ef4444') {
  chrome.action.setBadgeText({ text: count > 0 ? String(count) : '' });
  chrome.action.setBadgeBackgroundColor({ color });
}
