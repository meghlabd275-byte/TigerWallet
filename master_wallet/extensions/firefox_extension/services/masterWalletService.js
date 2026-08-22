/**
 * MasterWalletService - browser extension client for the canonical backend
 * (see CANONICAL_API_CONTRACT.md). All HTTP goes through apiClient.authedFetch
 * which targets the absolute BASE_URL with Bearer JWT auth and fails closed.
 */

'use strict';

// UMD: CommonJS require under node/tests, globalThis under MV3 service worker.
const {
  authedFetch,
  getAuthContext,
  setAuthContext,
  clearAuthContext,
} = (typeof require === 'function')
  ? require('./apiClient.js')
  : (globalThis.MW_API || {});

const CHAIN_CONFIGS = {
  1: { name: 'Ethereum', symbol: 'ETH', decimals: 18 },
  56: { name: 'BNB Smart Chain', symbol: 'BNB', decimals: 18 },
  137: { name: 'Polygon', symbol: 'MATIC', decimals: 18 },
  42161: { name: 'Arbitrum One', symbol: 'ETH', decimals: 18 },
};

class MasterWalletService {
  constructor() {
    this.currentWalletId = null;
  }

  _wid(id) {
    const wid = id || this.currentWalletId;
    if (!wid) throw new Error('No master wallet id provided or selected');
    return wid;
  }

  // ---------- Auth ----------

  async register({ email, password, name }) {
    const data = await authedFetch('/auth/register', {
      method: 'POST',
      auth: false,
      body: { email, password, name },
    });
    await setAuthContext({
      token: data.token,
      userId: data.user_id,
      email: data.email,
      role: data.role,
    });
    return data;
  }

  async login({ email, password }) {
    const data = await authedFetch('/auth/login', {
      method: 'POST',
      auth: false,
      body: { email, password },
    });
    await setAuthContext({
      token: data.token,
      userId: data.user_id,
      email: data.email,
      role: data.role,
    });
    return data;
  }

  async logout() {
    await clearAuthContext();
    this.currentWalletId = null;
    return true;
  }

  async getAuthContext() {
    return getAuthContext();
  }

  async setCurrentWallet(walletId) {
    this.currentWalletId = walletId;
    await setAuthContext({ currentWalletId: walletId });
    return true;
  }

  // ---------- Master wallets ----------

  async listMasterWallets() {
    const res = await authedFetch('/master-wallet', { method: 'GET' });
    return res.wallets || [];
  }

  async createMasterWallet({ name, password, chain_id = 1 }) {
    const wallet = await authedFetch('/master-wallet', {
      method: 'POST',
      body: { name, password, chain_id },
    });
    await this.setCurrentWallet(wallet.id || wallet.wallet_id);
    return wallet;
  }

  async getMasterWallet(id) {
    return authedFetch('/master-wallet/' + this._wid(id), { method: 'GET' });
  }

  async updateMasterWallet(id, body) {
    return authedFetch('/master-wallet/' + this._wid(id), { method: 'PUT', body });
  }

  // Alias for the canonical update method (PUT /master-wallet/:id). Accepts the
  // same partial body {name?, is_active?, daily_limit?, per_transaction_limit?,
  // metadata?} and returns {id, updated}.
  async updateWallet(id, body) {
    return authedFetch('/master-wallet/' + this._wid(id), { method: 'PUT', body });
  }

  async deleteMasterWallet(id) {
    return authedFetch('/master-wallet/' + this._wid(id), { method: 'DELETE' });
  }

  async getBalance(id, chainId) {
    const query = chainId ? { chain_id: chainId } : undefined;
    return authedFetch('/master-wallet/' + this._wid(id) + '/balance', { method: 'GET', query });
  }

  async signTransaction(id, { to, amount, password, token }) {
    const body = { to, amount, password };
    if (token !== undefined) body.token = token;
    return authedFetch('/master-wallet/' + this._wid(id) + '/sign', { method: 'POST', body });
  }

  // ---------- Sub wallets ----------

  async listSubWallets(id) {
    const res = await authedFetch('/master-wallet/' + this._wid(id) + '/sub-wallets', { method: 'GET' });
    return res.sub_wallets || res.wallets || res || [];
  }

  async createSubWallet(id, { name, password, chain_id = 1 }) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/sub-wallets', {
      method: 'POST',
      body: { name, password, chain_id },
    });
  }

  async getSubWalletBalance(id, sid, chainId) {
    const query = chainId ? { chain_id: chainId } : undefined;
    return authedFetch('/master-wallet/' + this._wid(id) + '/sub-wallets/' + sid + '/balance', {
      method: 'GET',
      query,
    });
  }

  async transferFromSubWallet(id, sid, { to, amount, password, token }) {
    const body = { to, amount, password };
    if (token !== undefined) body.token = token;
    return authedFetch('/master-wallet/' + this._wid(id) + '/sub-wallets/' + sid + '/transfer', {
      method: 'POST',
      body,
    });
  }

  // ---------- Transactions ----------

  async listTransactions(id) {
    const res = await authedFetch('/master-wallet/' + this._wid(id) + '/transactions', { method: 'GET' });
    return res.transactions || [];
  }

  async getTransaction(masterId, txId) {
    const data = await authedFetch('/master-wallet/' + this._wid(masterId) + '/transactions/' + txId, { method: 'GET' });
    return data.transaction;
  }

  async createTransaction(id, { to, amount, password, token }) {
    const body = { to, amount, password };
    if (token !== undefined) body.token = token;
    return authedFetch('/master-wallet/' + this._wid(id) + '/transactions', { method: 'POST', body });
  }

  async approveTransaction(id, tid) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/transactions/' + tid + '/approve', { method: 'POST' });
  }

  async rejectTransaction(id, tid) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/transactions/' + tid + '/reject', { method: 'POST' });
  }

  // ---------- Policies ----------

  async listPolicies(id) {
    const res = await authedFetch('/master-wallet/' + this._wid(id) + '/policies', { method: 'GET' });
    return res.policies || res || [];
  }

  async createPolicy(id, rule) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/policies', { method: 'POST', body: rule });
  }

  async updatePolicy(id, pid, updates) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/policies/' + pid, { method: 'PUT', body: updates });
  }

  async deletePolicy(id, pid) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/policies/' + pid, { method: 'DELETE' });
  }

  // ---------- Fees ----------

  async listFees(id) {
    const res = await authedFetch('/master-wallet/' + this._wid(id) + '/fees', { method: 'GET' });
    return res.fees || res || [];
  }

  async createFee(id, fee) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/fees', { method: 'POST', body: fee });
  }

  async updateFee(id, fid, updates) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/fees/' + fid, { method: 'PUT', body: updates });
  }

  async deleteFee(id, fid) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/fees/' + fid, { method: 'DELETE' });
  }

  // ---------- Auto-sign rules ----------

  async listAutoSignRules(id) {
    const res = await authedFetch('/master-wallet/' + this._wid(id) + '/auto-sign', { method: 'GET' });
    return res.rules || res || [];
  }

  async createAutoSignRule(id, rule) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/auto-sign', { method: 'POST', body: rule });
  }

  async updateAutoSignRule(id, rid, updates) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/auto-sign/' + rid, { method: 'PUT', body: updates });
  }

  async deleteAutoSignRule(id, rid) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/auto-sign/' + rid, { method: 'DELETE' });
  }

  // ---------- Users ----------

  async listUsers(id) {
    const res = await authedFetch('/master-wallet/' + this._wid(id) + '/users', { method: 'GET' });
    return res.users || res || [];
  }

  async createUser(id, user) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/users', { method: 'POST', body: user });
  }

  async updateUser(id, uid, updates) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/users/' + uid, { method: 'PUT', body: updates });
  }

  async deleteUser(id, uid) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/users/' + uid, { method: 'DELETE' });
  }

  // ---------- Audit + Analytics ----------

  async getAudit(id) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/audit', { method: 'GET' });
  }

  async getAnalyticsVolume(id) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/analytics/volume', { method: 'GET' });
  }

  async getAnalyticsTransactions(id) {
    const res = await authedFetch('/master-wallet/' + this._wid(id) + '/analytics/transactions', { method: 'GET' });
    return res.transactions || res || [];
  }

  async getAnalyticsWallets(id) {
    const res = await authedFetch('/master-wallet/' + this._wid(id) + '/analytics/wallets', { method: 'GET' });
    return res.wallets || res || [];
  }

  // ---------- Notifications + Webhooks ----------

  async listNotifications(id) {
    const res = await authedFetch('/master-wallet/' + this._wid(id) + '/notifications', { method: 'GET' });
    return res.notifications || res || [];
  }

  async createNotification(id, n) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/notifications', { method: 'POST', body: n });
  }

  async updateNotification(id, nid, updates) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/notifications/' + nid, { method: 'PUT', body: updates });
  }

  async listWebhooks(id) {
    const res = await authedFetch('/master-wallet/' + this._wid(id) + '/webhooks', { method: 'GET' });
    return res.webhooks || res || [];
  }

  async createWebhook(id, w) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/webhooks', { method: 'POST', body: w });
  }

  async updateWebhook(id, wid, updates) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/webhooks/' + wid, { method: 'PUT', body: updates });
  }

  async deleteWebhook(id, wid) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/webhooks/' + wid, { method: 'DELETE' });
  }

  // ---------- Treasury ----------

  async getTreasuryOverview(id) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/treasury', { method: 'GET' });
  }

  async getTreasuryTransactions(id) {
    const res = await authedFetch('/master-wallet/' + this._wid(id) + '/treasury/transactions', { method: 'GET' });
    return res.transactions || res || [];
  }

  async treasuryTransfer(id, { to, amount, password }) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/treasury/transfer', {
      method: 'POST',
      body: { to, amount, password },
    });
  }

  async treasurySweep(id, { to, password }) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/treasury/sweep', {
      method: 'POST',
      body: { to, password },
    });
  }

  // ---------- Multisig ----------

  async listMultisigWallets(id) {
    const res = await authedFetch('/master-wallet/' + this._wid(id) + '/multisig/wallets', { method: 'GET' });
    return res.wallets || res || [];
  }

  async createMultisigWallet(id, { name, owners, threshold }) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/multisig/wallets', {
      method: 'POST',
      body: { name, owners, threshold },
    });
  }

  async getMultisigWalletDetail(masterId, walletId) {
    const data = await authedFetch('/master-wallet/' + this._wid(masterId) + '/multisig/wallets/' + walletId, { method: 'GET' });
    return data.multisig_wallet;
  }

  async listMultisigTransactions(id, mwid) {
    const res = await authedFetch('/master-wallet/' + this._wid(id) + '/multisig/wallets/' + mwid + '/transactions', { method: 'GET' });
    return res.transactions || res || [];
  }

  async createMultisigTransaction(id, mwid, body) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/multisig/wallets/' + mwid + '/transactions', {
      method: 'POST',
      body,
    });
  }

  async signMultisigTransaction(id, tid) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/multisig/transactions/' + tid + '/sign', { method: 'POST' });
  }

  async executeMultisigTransaction(id, tid) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/multisig/transactions/' + tid + '/execute', { method: 'POST' });
  }

  // ---------- User EVM chains ----------

  async listUserEVMChains(id) {
    const res = await authedFetch('/master-wallet/' + this._wid(id) + '/user-chains/evm', { method: 'GET' });
    return res.chains || res || [];
  }

  async addUserEVMChain(id, chain) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/user-chains/evm', { method: 'POST', body: chain });
  }

  async updateUserEVMChain(id, chainId, updates) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/user-chains/evm/' + chainId, { method: 'PUT', body: updates });
  }

  async removeUserEVMChain(id, chainId) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/user-chains/evm/' + chainId, { method: 'DELETE' });
  }

  // ---------- User non-EVM chains ----------

  async listUserNonEVMChains(id) {
    const res = await authedFetch('/master-wallet/' + this._wid(id) + '/user-chains/nonevm', { method: 'GET' });
    return res.chains || res || [];
  }

  async addUserNonEVMChain(id, chain) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/user-chains/nonevm', { method: 'POST', body: chain });
  }

  async updateUserNonEVMChain(id, chainId, updates) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/user-chains/nonevm/' + chainId, { method: 'PUT', body: updates });
  }

  async removeUserNonEVMChain(id, chainId) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/user-chains/nonevm/' + chainId, { method: 'DELETE' });
  }

  // ---------- User tokens ----------

  async listUserTokens(id, chainId) {
    const query = chainId !== undefined ? { chain_id: chainId } : undefined;
    const res = await authedFetch('/master-wallet/' + this._wid(id) + '/user-tokens', { method: 'GET', query });
    return res.tokens || res || [];
  }

  async addUserToken(id, token) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/user-tokens', { method: 'POST', body: token });
  }

  async updateUserToken(id, tokenId, updates) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/user-tokens/' + tokenId, { method: 'PUT', body: updates });
  }

  async removeUserToken(id, tokenId) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/user-tokens/' + tokenId, { method: 'DELETE' });
  }

  // ---------- Address derivation ----------

  async deriveUserAddress(id, body) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/derive-user-address', { method: 'POST', body });
  }

  async listUserWalletAddresses(id) {
    const res = await authedFetch('/master-wallet/' + this._wid(id) + '/user-wallet-addresses', { method: 'GET' });
    return res.addresses || res || [];
  }

  // ---------- Auto-sign ----------

  async autoSignTransaction(id, body) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/auto-sign-transaction', { method: 'POST', body });
  }

  async listAutoSignLogs(id) {
    const res = await authedFetch('/master-wallet/' + this._wid(id) + '/auto-sign-logs', { method: 'GET' });
    return res.logs || res || [];
  }

  // ---------- Auto-sign bridge (MasterWallet-owner policy auto-approval of UserWallet txs) ----------

  async userWalletAutoSign(id, body) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/user-wallet-auto-sign', { method: 'POST', body });
  }

  async checkAutoSignPolicy(id, body) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/check-auto-sign-policy', { method: 'POST', body });
  }

  // ---------- Feature flags ----------

  async listFeatureFlags(id) {
    const res = await authedFetch('/master-wallet/' + this._wid(id) + '/feature-flags', { method: 'GET' });
    return res.flags || res || [];
  }

  async addFeatureFlag(id, flag) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/feature-flags', { method: 'POST', body: flag });
  }

  async updateFeatureFlag(id, flagId, updates) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/feature-flags/' + flagId, { method: 'PUT', body: updates });
  }

  async removeFeatureFlag(id, flagId) {
    return authedFetch('/master-wallet/' + this._wid(id) + '/feature-flags/' + flagId, { method: 'DELETE' });
  }

  // ---------- Passkeys (backend is the relying party) ----------

  async registerPasskey(masterId, body) {
    return authedFetch('/master-wallet/' + this._wid(masterId) + '/passkey/register', { method: 'POST', body });
  }

  async listPasskeys(masterId) {
    const data = await authedFetch('/master-wallet/' + this._wid(masterId) + '/passkey/credentials', { method: 'GET' });
    return data.passkeys || [];
  }

  async deletePasskey(masterId, credId) {
    return authedFetch('/master-wallet/' + this._wid(masterId) + '/passkey/credentials/' + credId, { method: 'DELETE' });
  }

  async verifyPasskeyAssertion(masterId, body) {
    return authedFetch('/master-wallet/' + this._wid(masterId) + '/passkey/verify-assertion', { method: 'POST', body });
  }

  // ---------- Two-party gate (withdrawal request + revenue payout) ----------

  async requestWithdrawal(masterId, { to_address, amount_wei, currency, chain_id }) {
    const body = { to_address, amount_wei };
    if (currency !== undefined) body.currency = currency;
    if (chain_id !== undefined) body.chain_id = chain_id;
    return authedFetch('/master-wallet/' + this._wid(masterId) + '/withdrawal-request', { method: 'POST', body });
  }

  async revenuePayout(masterId, { to, amount, password, gas_limit, withdrawal_id }) {
    const body = { to, amount, password };
    if (gas_limit !== undefined) body.gas_limit = gas_limit;
    if (withdrawal_id !== undefined) body.withdrawal_id = withdrawal_id;
    return authedFetch('/master-wallet/' + this._wid(masterId) + '/revenue-payout', { method: 'POST', body });
  }

  // ---------- Public (no auth) ----------

  async listChains() {
    const res = await authedFetch('/chains', { method: 'GET', auth: false });
    return res.chains || res || [];
  }

  async getGas(chainId) {
    return authedFetch('/gas', { method: 'GET', auth: false, query: { chain_id: chainId } });
  }

  async getPrice(coinId = 'ethereum') {
    return authedFetch('/price', { method: 'GET', auth: false, query: { coin_id: coinId } });
  }

  async getTransactionHistory(address, chainId) {
    const res = await authedFetch('/transactions/history', {
      method: 'GET',
      auth: false,
      query: { address, chain_id: chainId },
    });
    return res.transactions || res || [];
  }

  async health() {
    return authedFetch('/health', { method: 'GET', auth: false });
  }

  async apiHealth() {
    return authedFetch('/api/v1/health', { method: 'GET', auth: false });
  }

  // ---------- Local chain metadata ----------

  getSupportedChains() {
    return Object.entries(CHAIN_CONFIGS).map(([id, config]) => ({ id: Number(id), ...config }));
  }
}

const masterWalletService = new MasterWalletService();

// UMD: CommonJS for node/tests, globalThis for MV3 service worker (importScripts).
if (typeof module !== 'undefined' && module.exports) {
  module.exports = { masterWalletService, MasterWalletService };
}
if (typeof globalThis !== 'undefined') {
  globalThis.MW_SERVICE = { masterWalletService, MasterWalletService };
}
