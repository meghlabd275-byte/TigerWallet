/**
 * MasterWallet extension - authenticated HTTP client for the canonical backend.
 *
 * Every request:
 *   - uses an absolute URL built from the canonical BASE_URL (never a bare
 *     "/api/..." relative path),
 *   - carries `Authorization: Bearer <JWT>` for protected routes,
 *   - fails closed on non-2xx / network error (throws) so callers cannot
 *     silently fall back to fake data.
 */

'use strict';

// UMD: CommonJS require under node/tests, globalThis under MV3 service worker.
const { getConfig } = (typeof require === 'function')
  ? require('./config.js')
  : (globalThis.MW_CONFIG || {});

function storageGet(keys) {
  return new Promise((resolve) => {
    try {
      chrome.storage.local.get(keys, (res) => resolve(res || {}));
    } catch (e) {
      resolve({});
    }
  });
}

function storageSet(obj) {
  return new Promise((resolve) => {
    try {
      chrome.storage.local.set(obj, () => resolve(true));
    } catch (e) {
      resolve(false);
    }
  });
}

function storageRemove(keys) {
  return new Promise((resolve) => {
    try {
      chrome.storage.local.remove(keys, () => resolve(true));
    } catch (e) {
      resolve(false);
    }
  });
}

async function getAuthToken() {
  const res = await storageGet('mw_auth_token');
  return res.mw_auth_token || null;
}

async function getAuthContext() {
  const res = await storageGet([
    'mw_auth_token',
    'mw_user_id',
    'mw_email',
    'mw_role',
    'mw_current_wallet_id',
  ]);
  return {
    token: res.mw_auth_token || null,
    userId: res.mw_user_id || null,
    email: res.mw_email || null,
    role: res.mw_role || null,
    currentWalletId: res.mw_current_wallet_id || null,
  };
}

async function setAuthContext(ctx) {
  const obj = {};
  if (ctx.token !== undefined) obj.mw_auth_token = ctx.token;
  if (ctx.userId !== undefined) obj.mw_user_id = ctx.userId;
  if (ctx.email !== undefined) obj.mw_email = ctx.email;
  if (ctx.role !== undefined) obj.mw_role = ctx.role;
  if (ctx.currentWalletId !== undefined) obj.mw_current_wallet_id = ctx.currentWalletId;
  await storageSet(obj);
}

async function clearAuthContext() {
  await storageRemove([
    'mw_auth_token',
    'mw_user_id',
    'mw_email',
    'mw_role',
    'mw_current_wallet_id',
  ]);
}

function buildUrl(path, query) {
  const cfg = CONFIG_SYNC();
  let p = path.startsWith('/') ? path : '/' + path;
  // The contract routes already include /api/v1, but tolerate callers that
  // pass the contract path verbatim.
  if (!p.startsWith(cfg.apiPrefix) && !p.startsWith('/health') && !p.startsWith('/ws')) {
    p = cfg.apiPrefix + p;
  }
  let url = cfg.apiBase + p;
  if (query && Object.keys(query).length > 0) {
    const qs = new URLSearchParams();
    for (const [k, v] of Object.entries(query)) {
      if (v !== undefined && v !== null) qs.append(k, v);
    }
    const s = qs.toString();
    if (s) url += (url.includes('?') ? '&' : '?') + s;
  }
  return url;
}

// Synchronous base resolver used by buildUrl before a request is dispatched.
// getConfig() is async (reads storage); authedFetch resolves config first, so
// buildUrl is only called after config is known.
let _cachedCfg = null;
async function refreshConfig() {
  _cachedCfg = await getConfig();
  return _cachedCfg;
}
function CONFIG_SYNC() {
  return _cachedCfg || { apiBase: 'http://localhost:8450', apiPrefix: '/api/v1' };
}

/**
 * Authed fetch. Throws on any non-2xx response or network failure.
 * @param {string} path - contract path (e.g. "/auth/login", "/master-wallet")
 * @param {object} opts - { method, body, query, auth (default true), headers }
 */
async function authedFetch(path, opts = {}) {
  await refreshConfig();
  const cfg = CONFIG_SYNC();
  const method = (opts.method || 'GET').toUpperCase();
  const url = buildUrl(path, opts.query);
  const headers = { 'Content-Type': 'application/json', ...(opts.headers || {}) };

  if (opts.auth !== false) {
    const token = await getAuthToken();
    if (!token) {
      throw new Error('Not authenticated: no JWT available for ' + method + ' ' + url);
    }
    headers['Authorization'] = 'Bearer ' + token;
  }

  const fetchOpts = { method, headers };
  if (opts.body !== undefined && method !== 'GET') {
    fetchOpts.body = typeof opts.body === 'string' ? opts.body : JSON.stringify(opts.body);
  }

  let response;
  try {
    response = await fetch(url, fetchOpts);
  } catch (e) {
    throw new Error('Network error contacting backend (' + method + ' ' + url + '): ' + e.message);
  }

  if (!response.ok) {
    let detail = '';
    try {
      const ct = response.headers.get('content-type') || '';
      if (ct.includes('application/json')) {
        detail = JSON.stringify(await response.json());
      } else {
        detail = (await response.text()).slice(0, 200);
      }
    } catch (_) { /* ignore parse failure */ }
    throw new Error('Backend ' + response.status + ' for ' + method + ' ' + url + (detail ? ': ' + detail : ''));
  }

  const ct = response.headers.get('content-type') || '';
  if (ct.includes('application/json')) {
    return await response.json();
  }
  return response.text();
}

// UMD: CommonJS for node/tests, globalThis for MV3 service worker (importScripts).
if (typeof module !== 'undefined' && module.exports) {
  module.exports = {
    authedFetch,
    getAuthToken,
    getAuthContext,
    setAuthContext,
    clearAuthContext,
    buildUrl,
    refreshConfig,
    getConfig,
  };
}
if (typeof globalThis !== 'undefined') {
  globalThis.MW_API = {
    authedFetch,
    getAuthToken,
    getAuthContext,
    setAuthContext,
    clearAuthContext,
    buildUrl,
    refreshConfig,
    getConfig,
  };
}
