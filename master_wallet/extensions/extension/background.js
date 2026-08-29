/**
 * MasterWallet MV3 service worker.
 *
 * This is a REAL service worker: it relays messages from the popup and content
 * scripts to the canonical backend (http://localhost:8450 / prod master-api)
 * via the service modules loaded with importScripts. There is NO simulation,
 * NO fake data, NO chrome.storage-based wallet backend. chrome.storage is used
 * ONLY for: the JWT auth context (token/user id/role/current wallet id) and the
 * persisted UI theme preference.
 *
 * Message protocol (chrome.runtime.sendMessage):
 *   { type: "<action>", payload: {...} }
 * Response (async callback or Promise):
 *   { ok: true, data: <backend response> } | { ok: false, error: "<message>" }
 */

'use strict';

// MV3 classic service worker: load the service modules into the shared global
// scope. Order matters: config -> keccak -> apiClient -> services that depend
// on them.
importScripts(
  'services/config.js',
  'services/keccak256.js',
  'services/apiClient.js',
  'services/masterWalletService.js',
  'services/AccountAbstractionService.js',
  'services/PaymasterService.js',
  'services/PrivacyService.js',
  'services/SuperAdminService.js',
  'services/TaxAnalyticsService.js',
  'services/PasskeyService.js',
  'services/BiometricService.js'
);

// Fail-closed accessor for the service namespaces; if a module failed to attach
// (importScripts error) we never silently fall back to fake behavior.
function ns(name) {
  const obj = (typeof globalThis !== 'undefined') ? globalThis[name] : undefined;
  if (!obj) throw new Error('Service module not loaded: ' + name);
  return obj;
}

const API = () => ns('MW_API');
const WALLET = () => ns('MW_SERVICE').masterWalletService;

// ---------------------------------------------------------------------------
// Theme persistence (light/dark). Stored in chrome.storage.local; the popup
// and content scripts apply `data-theme` on <html>. No theme is fabricated.
// ---------------------------------------------------------------------------
async function handleThemeGet() {
  return new Promise((resolve) => {
    chrome.storage.local.get('mw_theme', (res) => resolve(res.mw_theme || 'light'));
  });
}

async function handleThemeSet(theme) {
  if (theme !== 'light' && theme !== 'dark') {
    throw new Error('Invalid theme: ' + theme);
  }
  await new Promise((resolve) => chrome.storage.local.set({ mw_theme: theme }, resolve));
  // Notify all tabs + popup to apply the theme to their document.
  try {
    const tabs = await chrome.tabs.query({});
    for (const t of tabs) {
      if (t.id != null) {
        chrome.tabs
          .sendMessage(t.id, { type: 'MW_THEME_CHANGED', theme })
          .catch(() => { /* tab may not have a content script */ });
      }
    }
  } catch (_) { /* ignore */ }
  return { theme };
}

// ---------------------------------------------------------------------------
// Auth message handlers. These call the real /api/v1/auth/* routes.
// ---------------------------------------------------------------------------
async function handleAuthRegister(payload) {
  const { email, password, name } = payload || {};
  if (!email || !password || !name) throw new Error('email, password, name required');
  const data = await API().authedFetch('/auth/register', {
    method: 'POST',
    auth: false,
    body: { email, password, name },
  });
  if (data && data.token) {
    await API().setAuthContext({
      token: data.token,
      userId: data.user_id,
      email: data.email,
      role: data.role,
    });
  }
  return data;
}

async function handleAuthLogin(payload) {
  const { email, password } = payload || {};
  if (!email || !password) throw new Error('email and password required');
  const data = await API().authedFetch('/auth/login', {
    method: 'POST',
    auth: false,
    body: { email, password },
  });
  if (data && data.token) {
    await API().setAuthContext({
      token: data.token,
      userId: data.user_id,
      email: data.email,
      role: data.role,
    });
  }
  return data;
}

async function handleAuthLogout() {
  await API().clearAuthContext();
  return { ok: true };
}

async function handleAuthContext() {
  return API().getAuthContext();
}

// ---------------------------------------------------------------------------
// Generic authenticated relay. The popup asks for a contract route and the
// service worker executes it against the real backend with the stored JWT.
// Supported relay "route" groups map to masterWalletService methods.
// ---------------------------------------------------------------------------
async function handleRelay(payload) {
  const { action, args } = payload || {};
  if (!action) throw new Error('relay: action required');
  const svc = WALLET();
  const ctx = await API().getAuthContext();
  if (!ctx.token) throw new Error('Not authenticated');

  // Master wallet endpoints require a current wallet id for :id routes.
  const requiresWalletId = ![
    'listMasterWallets',
    'createMasterWallet',
    'register',
    'login',
    'logout',
    'getAuthContext',
    'listChains',
    'getGas',
    'getPrice',
    'txHistory',
    'getTransactionHistory',
    'health',
    'apiHealth',
  ].includes(action);

  const walletId = (args && args[0]) || ctx.currentWalletId;
  if (requiresWalletId && !walletId) {
    throw new Error('No current master wallet id selected');
  }

  switch (action) {
    case 'listMasterWallets':
      return svc.listMasterWallets();
    case 'createMasterWallet':
      return svc.createMasterWallet(args[0]);
    case 'getMasterWallet':
      return svc.getMasterWallet(walletId);
    case 'deleteMasterWallet':
      return svc.deleteMasterWallet(walletId);
    case 'getBalance':
      return svc.getBalance(walletId);
    case 'signTransaction':
      return svc.signTransaction(walletId, args[1]);
    case 'listSubWallets':
      return svc.listSubWallets(walletId);
    case 'createSubWallet':
      return svc.createSubWallet(walletId, args[1]);
    case 'getSubWalletBalance':
      return svc.getSubWalletBalance(walletId, args[1]);
    case 'transferSubWallet':
      return svc.transferFromSubWallet(walletId, args[1], args[2]);
    case 'listTransactions':
      return svc.listTransactions(walletId);
    case 'createTransaction':
      return svc.createTransaction(walletId, args[1]);
    case 'approveTransaction':
      return svc.approveTransaction(walletId, args[1]);
    case 'rejectTransaction':
      return svc.rejectTransaction(walletId, args[1]);
    case 'listPolicies':
      return svc.listPolicies(walletId);
    case 'createPolicy':
      return svc.createPolicy(walletId, args[1]);
    case 'updatePolicy':
      return svc.updatePolicy(walletId, args[1], args[2]);
    case 'deletePolicy':
      return svc.deletePolicy(walletId, args[1]);
    case 'listFees':
      return svc.listFees(walletId);
    case 'createFee':
      return svc.createFee(walletId, args[1]);
    case 'deleteFee':
      return svc.deleteFee(walletId, args[1]);
    case 'listAutoSign':
      return svc.listAutoSignRules(walletId);
    case 'createAutoSign':
      return svc.createAutoSignRule(walletId, args[1]);
    case 'deleteAutoSign':
      return svc.deleteAutoSignRule(walletId, args[1]);
    case 'listUsers':
      return svc.listUsers(walletId);
    case 'createUser':
      return svc.createUser(walletId, args[1]);
    case 'deleteUser':
      return svc.deleteUser(walletId, args[1]);
    case 'getAudit':
      return svc.getAudit(walletId);
    case 'getAnalyticsVolume':
      return svc.getAnalyticsVolume(walletId);
    case 'getAnalyticsTransactions':
      return svc.getAnalyticsTransactions(walletId);
    case 'getAnalyticsWallets':
      return svc.getAnalyticsWallets(walletId);
    case 'listNotifications':
      return svc.listNotifications(walletId);
    case 'createNotification':
      return svc.createNotification(walletId, args[1]);
    case 'listWebhooks':
      return svc.listWebhooks(walletId);
    case 'createWebhook':
      return svc.createWebhook(walletId, args[1]);
    case 'deleteWebhook':
      return svc.deleteWebhook(walletId, args[1]);
    case 'getTreasury':
      return svc.getTreasuryOverview(walletId);
    case 'getTreasuryTransactions':
      return svc.getTreasuryTransactions(walletId);
    case 'treasuryTransfer':
      return svc.treasuryTransfer(walletId, args[1]);
    case 'treasurySweep':
      return svc.treasurySweep(walletId, args[1]);
    case 'listMultisigWallets':
      return svc.listMultisigWallets(walletId);
    case 'createMultisigWallet':
      return svc.createMultisigWallet(walletId, args[1]);
    case 'listMultisigTransactions':
      return svc.listMultisigTransactions(walletId, args[1]);
    case 'createMultisigTransaction':
      return svc.createMultisigTransaction(walletId, args[1], args[2]);
    case 'signMultisigTransaction':
      return svc.signMultisigTransaction(walletId, args[1]);
    case 'executeMultisigTransaction':
      return svc.executeMultisigTransaction(walletId, args[1]);
    case 'getTransaction':
      return svc.getTransaction(walletId, args[1]);
    case 'getMultisigWalletDetail':
      return svc.getMultisigWalletDetail(walletId, args[1]);
    case 'updateWallet':
      return svc.updateWallet(walletId, args[1]);
    // ---------- Passkeys (backend is the relying party) ----------
    case 'registerPasskey':
      return svc.registerPasskey(walletId, args[1]);
    case 'listPasskeys':
      return svc.listPasskeys(walletId);
    case 'deletePasskey':
      return svc.deletePasskey(walletId, args[1]);
    case 'verifyPasskeyAssertion':
      return svc.verifyPasskeyAssertion(walletId, args[1]);
    case 'requestWithdrawal':
      return svc.requestWithdrawal(walletId, args[1]);
    case 'revenuePayout':
      return svc.revenuePayout(walletId, args[1]);
    case 'setCurrentWallet':
      await API().setAuthContext({ currentWalletId: args[0] });
      return { currentWalletId: args[0] };
    case 'listChains':
      return svc.listChains();
    case 'getGas':
      return svc.getGas(args[0]);
    case 'getPrice':
      return svc.getPrice(args[0]);
    case 'txHistory':
      return svc.getTransactionHistory(args[0], args[1]);
    case 'health':
      return svc.health();
    case 'apiHealth':
      return svc.apiHealth();
    // ---------- Auto-sign bridge ----------
    case 'userWalletAutoSign':
      return svc.userWalletAutoSign(walletId, args[1]);
    case 'checkAutoSignPolicy':
      return svc.checkAutoSignPolicy(walletId, args[1]);
    // ---------- Auth (no wallet id) ----------
    case 'register':
      return svc.register(args[0]);
    case 'login':
      return svc.login(args[0]);
    case 'logout':
      return svc.logout();
    case 'getAuthContext':
      return svc.getAuthContext();
    // ---------- Update master wallet ----------
    case 'updateMasterWallet':
      return svc.updateMasterWallet(walletId, args[1]);
    // ---------- UserWallet governance: EVM chains ----------
    case 'listUserEVMChains':
      return svc.listUserEVMChains(walletId);
    case 'addUserEVMChain':
      return svc.addUserEVMChain(walletId, args[1]);
    case 'updateUserEVMChain':
      return svc.updateUserEVMChain(walletId, args[1], args[2]);
    case 'removeUserEVMChain':
      return svc.removeUserEVMChain(walletId, args[1]);
    // ---------- UserWallet governance: non-EVM chains ----------
    case 'listUserNonEVMChains':
      return svc.listUserNonEVMChains(walletId);
    case 'addUserNonEVMChain':
      return svc.addUserNonEVMChain(walletId, args[1]);
    case 'updateUserNonEVMChain':
      return svc.updateUserNonEVMChain(walletId, args[1], args[2]);
    case 'removeUserNonEVMChain':
      return svc.removeUserNonEVMChain(walletId, args[1]);
    // ---------- UserWallet governance: tokens ----------
    case 'listUserTokens':
      return svc.listUserTokens(walletId, args[1]);
    case 'addUserToken':
      return svc.addUserToken(walletId, args[1]);
    case 'updateUserToken':
      return svc.updateUserToken(walletId, args[1], args[2]);
    case 'removeUserToken':
      return svc.removeUserToken(walletId, args[1]);
    // ---------- UserWallet governance: addresses + auto-sign ----------
    case 'deriveUserAddress':
      return svc.deriveUserAddress(walletId, args[1]);
    case 'listUserWalletAddresses':
      return svc.listUserWalletAddresses(walletId);
    case 'autoSignTransaction':
      return svc.autoSignTransaction(walletId, args[1]);
    case 'listAutoSignLogs':
      return svc.listAutoSignLogs(walletId);
    // ---------- Feature flags ----------
    case 'listFeatureFlags':
      return svc.listFeatureFlags(walletId);
    case 'addFeatureFlag':
      return svc.addFeatureFlag(walletId, args[1]);
    case 'updateFeatureFlag':
      return svc.updateFeatureFlag(walletId, args[1], args[2]);
    case 'removeFeatureFlag':
      return svc.removeFeatureFlag(walletId, args[1]);
    // ---------- Treasury alias ----------
    case 'getTreasuryOverview':
      return svc.getTreasuryOverview(walletId);
    // ---------- Update routes (full CRUD parity with backend) ----------
    case 'updateFee':
      return svc.updateFee(walletId, args[1], args[2]);
    case 'updateAutoSignRule':
      return svc.updateAutoSignRule(walletId, args[1], args[2]);
    case 'updateUser':
      return svc.updateUser(walletId, args[1], args[2]);
    case 'updateNotification':
      return svc.updateNotification(walletId, args[1], args[2]);
    case 'updateWebhook':
      return svc.updateWebhook(walletId, args[1], args[2]);
    // ---------- Auto-sign daemon policy ----------
    case 'getAutoSignPolicy':
      return svc.getAutoSignPolicy(walletId);
    case 'updateAutoSignPolicy':
      return svc.updateAutoSignPolicy(walletId, args[1]);
    default:
      throw new Error('Unknown relay action: ' + action);
  }
}

// ---------------------------------------------------------------------------
// Message router.
// ---------------------------------------------------------------------------
const HANDLERS = {
  MW_THEME_GET: handleThemeGet,
  MW_THEME_SET: (p) => handleThemeSet(p.theme),
  MW_AUTH_REGISTER: handleAuthRegister,
  MW_AUTH_LOGIN: handleAuthLogin,
  MW_AUTH_LOGOUT: handleAuthLogout,
  MW_AUTH_CONTEXT: handleAuthContext,
  MW_RELAY: handleRelay,
};

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (!message || typeof message !== 'object' || !message.type) {
    sendResponse({ ok: false, error: 'Invalid message: missing type' });
    return false;
  }
  const handler = HANDLERS[message.type];
  if (!handler) {
    sendResponse({ ok: false, error: 'Unknown message type: ' + message.type });
    return false;
  }

  Promise.resolve()
    .then(() => handler(message.payload))
    .then(
      (data) => sendResponse({ ok: true, data }),
      (err) => sendResponse({ ok: false, error: (err && err.message) ? err.message : String(err) })
    );
  return true; // keep the message channel open for the async response
});

// Apply the persisted theme to a freshly-opened popup window if it queries.
chrome.runtime.onStartup.addListener(() => {
  // Nothing to fabricate; only ensure config is cached on worker start.
  API().refreshConfig && API().refreshConfig().catch(() => { /* backend may be down; UI reports */ });
});

chrome.runtime.onInstalled.addListener(() => {
  API().refreshConfig && API().refreshConfig().catch(() => { /* backend may be down */ });
});
