/**
 * SuperAdminService - admin/feature-flag client for the extension.
 *
 * The canonical contract exposes: users CRUD, audit log, and analytics under
 * /master-wallet/:id. There is NO /api/super-admin/* contract. The previous
 * implementation invented admin endpoints (authenticate, change-password,
 * enable-2fa, create-admin, ...). Those have been removed.
 *
 * Admin authentication is the standard /auth/login route; admin-only actions
 * are enforced by the backend role (ctx.role). Feature flags are persisted
 * locally only as a UI cache and always reflect backend state on read.
 */

'use strict';

// UMD: CommonJS require under node/tests, globalThis under MV3 service worker.
const { authedFetch, getAuthContext } = (typeof require === 'function')
  ? require('./apiClient.js')
  : ((globalThis.MW_API) || {});

class MasterSuperAdminService {
  constructor(masterWalletId) {
    this.masterWalletId = masterWalletId;
    this.isInitialized = false;
  }

  async initialize() {
    if (!this.masterWalletId) throw new Error('masterWalletId is required');
    this.isInitialized = true;
    return true;
  }

  _assertAuthed(role) {
    return (async () => {
      if (!this.isInitialized) throw new Error('SuperAdmin service not initialized');
      const ctx = await getAuthContext();
      if (!ctx.token) throw new Error('Not authenticated');
      if (role && ctx.role !== role) {
        throw new Error('Not authorized: requires role ' + role);
      }
      return ctx;
    })();
  }

  async isAdmin() {
    const ctx = await getAuthContext();
    return ctx.role === 'ADMIN' || ctx.role === 'SUPER_ADMIN';
  }

  // ---- Users (canonical /master-wallet/:id/users) ----

  async listUsers() {
    const ctx = await this._assertAuthed();
    const res = await authedFetch('/master-wallet/' + this.masterWalletId + '/users', { method: 'GET' });
    return res.users || res || [];
  }

  async createUser(user) {
    await this._assertAuthed('ADMIN');
    return authedFetch('/master-wallet/' + this.masterWalletId + '/users', { method: 'POST', body: user });
  }

  async deleteUser(userId) {
    await this._assertAuthed('ADMIN');
    return authedFetch('/master-wallet/' + this.masterWalletId + '/users/' + userId, { method: 'DELETE' });
  }

  // ---- Audit (canonical /master-wallet/:id/audit) ----

  async getAuditLogs() {
    const ctx = await this._assertAuthed();
    const res = await authedFetch('/master-wallet/' + this.masterWalletId + '/audit', { method: 'GET' });
    return res.logs || res.entries || res || [];
  }

  // ---- Analytics (canonical /master-wallet/:id/analytics/*) ----

  async getAnalyticsVolume() {
    await this._assertAuthed();
    return authedFetch('/master-wallet/' + this.masterWalletId + '/analytics/volume', { method: 'GET' });
  }

  async getAnalyticsTransactions() {
    await this._assertAuthed();
    const res = await authedFetch('/master-wallet/' + this.masterWalletId + '/analytics/transactions', { method: 'GET' });
    return res.transactions || res || [];
  }

  async getAnalyticsWallets() {
    await this._assertAuthed();
    const res = await authedFetch('/master-wallet/' + this.masterWalletId + '/analytics/wallets', { method: 'GET' });
    return res.wallets || res || [];
  }

  // ---- Feature flags (local UI cache only) ----
  // These reflect client-side UI preferences, NOT server-enforced security
  // flags. Any security-relevant decision must consult the backend role.

  async getFeatureFlags() {
    return new Promise((resolve) => {
      try {
        chrome.storage.local.get('mw_feature_flags', (res) => {
          resolve(res && res.mw_feature_flags ? res.mw_feature_flags : {});
        });
      } catch (e) {
        resolve({});
      }
    });
  }

  async setFeatureFlag(name, enabled) {
    const flags = await this.getFeatureFlags();
    flags[name] = { enabled, updatedAt: Date.now() };
    return new Promise((resolve) => {
      try {
        chrome.storage.local.set({ mw_feature_flags: flags }, () => resolve(true));
      } catch (e) {
        resolve(false);
      }
    });
  }
}

if (typeof module !== 'undefined' && module.exports) {
  module.exports = { MasterSuperAdminService };
}
if (typeof globalThis !== 'undefined') {
  globalThis.MW_SUPERADMIN = { MasterSuperAdminService };
}
