/**
 * MasterWallet extension - shared configuration.
 *
 * All HTTP calls target the canonical Go backend (see
 * CANONICAL_API_CONTRACT.md). The base URL is configurable via the
 * MASTER_WALLET_API_URL storage key; it defaults to the local dev backend.
 * host_permissions in manifest.json MUST allow this origin.
 */

'use strict';

// Default to the canonical dev backend. Production deployments override this
// by setting MASTER_WALLET_API_URL in chrome.storage.local (or by setting
// window.__MASTER_WALLET_API_URL__ before the service worker loads).
const DEFAULT_API_BASE = 'http://localhost:8450';

function resolveApiBase() {
  try {
    if (typeof window !== 'undefined' && window.__MASTER_WALLET_API_URL__) {
      return String(window.__MASTER_WALLET_API_URL__).replace(/\/+$/, '');
    }
  } catch (_) { /* window may be undefined in service worker top scope */ }

  // Synchronously resolve from storage is not possible; callers must use
  // getConfig() (async). getDefaultApiBase() is the fallback for sync code.
  return DEFAULT_API_BASE;
}

const CONFIG = {
  apiBase: DEFAULT_API_BASE,
  prodApiBase: 'https://master-api.tigerwallet.com',
  apiPrefix: '/api/v1',
  wsPath: '/ws',
  storageKeys: {
    authToken: 'mw_auth_token',
    userId: 'mw_user_id',
    email: 'mw_email',
    role: 'mw_role',
    currentWalletId: 'mw_current_wallet_id',
    theme: 'mw_theme',
    autoApprove: 'mw_auto_approve',
  },
};

/**
 * Async config loader. Reads the overridden base URL from storage and returns
 * the effective config object. Falls back to the default base if storage is
 * unavailable (e.g. outside an extension context).
 */
async function getConfig() {
  let storedBase = null;
  try {
    if (typeof chrome !== 'undefined' && chrome.storage && chrome.storage.local) {
      const res = await chrome.storage.local.get('MASTER_WALLET_API_URL');
      storedBase = res && res.MASTER_WALLET_API_URL ? res.MASTER_WALLET_API_URL : null;
    }
  } catch (_) { /* ignore */ }
  return {
    ...CONFIG,
    apiBase: (storedBase || DEFAULT_API_BASE).replace(/\/+$/, ''),
  };
}

function getDefaultApiBase() {
  return resolveApiBase();
}

// UMD: CommonJS for node/tests, globalThis for MV3 service worker (importScripts).
if (typeof module !== 'undefined' && module.exports) {
  module.exports = { CONFIG, getConfig, getDefaultApiBase, DEFAULT_API_BASE };
}
if (typeof globalThis !== 'undefined') {
  globalThis.MW_CONFIG = { CONFIG, getConfig, getDefaultApiBase, DEFAULT_API_BASE };
}
