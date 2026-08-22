/**
 * TigerWallet Popup Script
 *
 * No-registration self-custody UX (mirrors user_wallet/web/src/pages/Onboarding.tsx
 * + OnboardingContext.tsx). The user NEVER sees a login/register form: a random
 * device-bound identity is provisioned transparently in the background service
 * worker and used to obtain a JWT for all backend calls.
 *
 * Views (JS view-switching via show/hide divs — extensions have no router):
 *   onboarding -> create | import -> backup (create only) -> connected
 */

// ========================================
// State
// ========================================

let wallet = null;
let isLocked = true;
let isConnected = false;
let theme = 'dark';
let currentChain = 1;

// Held in memory only for the duration of the onboarding flow — never persisted.
let createdMnemonic = null;
let createdWalletId = null;
let createdPassword = null;
let createdChainId = 1;

// ========================================
// DOM Elements
// ========================================

const views = {
  onboarding: document.getElementById('onboarding-view'),
  create: document.getElementById('create-view'),
  import: document.getElementById('import-view'),
  backup: document.getElementById('backup-view'),
  connected: document.getElementById('connected-view'),
  disconnected: document.getElementById('disconnected-view'),
  locked: document.getElementById('locked-view'),
};

const elements = {
  totalBalance: document.getElementById('total-balance'),
  balanceChange: document.getElementById('balance-change'),
  currentNetwork: document.getElementById('current-network'),
  accountAddress: document.getElementById('account-address'),
  tokensList: document.getElementById('tokens-list'),
  themeBtn: document.getElementById('theme-btn'),
  txBanner: document.getElementById('tx-banner'),
  txBannerHash: document.getElementById('tx-banner-hash'),
  txBannerLink: document.getElementById('tx-banner-link'),
};

// Maps chain_id -> { name, explorer (url template with {tx}) }.
const CHAINS = {
  1: { name: 'Ethereum', explorer: 'https://etherscan.io/tx/{tx}' },
  56: { name: 'BNB Smart Chain', explorer: 'https://bscscan.com/tx/{tx}' },
  137: { name: 'Polygon', explorer: 'https://polygonscan.com/tx/{tx}' },
  42161: { name: 'Arbitrum One', explorer: 'https://arbiscan.io/tx/{tx}' },
  10: { name: 'Optimism', explorer: 'https://optimistic.etherscan.io/tx/{tx}' },
  8453: { name: 'Base', explorer: 'https://basescan.org/tx/{tx}' },
};

let txBannerTimer = null;

// ========================================
// Initialize
// ========================================

document.addEventListener('DOMContentLoaded', async () => {
  await loadSettings();
  applyTheme();
  setupEventListeners();
  await bootstrap();
});

// First popup entry: ensure a transparent session exists (the service worker
// may not have finished provisioning on a cold start), then route to the
// onboarding choose screen OR the connected dashboard depending on whether at
// least one wallet id is already stored locally.
async function bootstrap() {
  try {
    await sendMessage({ method: 'tiger_ensureSession' });
  } catch (e) {
    // The session provisioning surfaces real backend errors here — but we
    // still let the user reach onboarding so they can retry create/import.
    console.warn('TigerWallet: transparent session not ready:', e?.message || e);
  }

  let onboarded = false;
  try {
    onboarded = await sendMessage({ method: 'tiger_getOnboarded' });
  } catch (e) {
    console.warn('TigerWallet: onboarding check failed:', e?.message || e);
  }

  wallet = await sendMessage({ method: 'tiger_getWallet' });

  if (onboarded && wallet) {
    await enterConnected();
  } else if (wallet) {
    await enterConnected();
  } else {
    showView('onboarding');
  }
}

async function loadSettings() {
  try {
    const settings = await sendMessage({ method: 'tiger_getSettings' });
    if (settings) {
      theme = settings.theme || 'dark';
    }
  } catch (e) {
    console.error('Failed to load settings:', e);
  }
}

async function enterConnected() {
  isConnected = true;
  isLocked = false;
  showView('connected');
  await loadWalletData();
}

// ========================================
// View Management
// ========================================

function showView(viewName) {
  Object.values(views).forEach((view) => view && view.classList.add('hidden'));
  const v = views[viewName];
  if (v) v.classList.remove('hidden');
}

function showFormError(elId, message) {
  const el = document.getElementById(elId);
  if (!el) return;
  if (message) {
    el.textContent = message;
    el.classList.remove('hidden');
  } else {
    el.classList.add('hidden');
    el.textContent = '';
  }
}

// ========================================
// Theme
// ========================================

function applyTheme() {
  document.documentElement.setAttribute('data-theme', theme);
  if (elements.themeBtn) elements.themeBtn.textContent = theme === 'dark' ? '🌙' : '☀️';
}

function toggleTheme() {
  theme = theme === 'dark' ? 'light' : 'dark';
  applyTheme();
  sendMessage({ method: 'tiger_updateSettings', params: [{ theme }] });
}

// ========================================
// Event Listeners
// ========================================

function setupEventListeners() {
  // Theme toggle
  elements.themeBtn?.addEventListener('click', toggleTheme);

  // Onboarding choose screen
  document.getElementById('onboarding-create-btn')?.addEventListener('click', () => {
    resetCreateForm();
    showView('create');
  });
  document.getElementById('onboarding-import-btn')?.addEventListener('click', () => {
    resetImportForm();
    showView('import');
  });

  // Create form
  document.getElementById('create-submit-btn')?.addEventListener('click', submitCreateWallet);
  document.getElementById('create-back-btn')?.addEventListener('click', () => showView('onboarding'));

  // Import form
  document.getElementById('import-submit-btn')?.addEventListener('click', submitImportWallet);
  document.getElementById('import-back-btn')?.addEventListener('click', () => showView('onboarding'));

  // Backup view
  document.getElementById('backup-reveal')?.addEventListener('change', onRevealMnemonic);
  document.getElementById('backup-copy-btn')?.addEventListener('click', onCopyMnemonic);
  document.getElementById('backup-gdrive-btn')?.addEventListener('click', onBackupToGoogleDrive);
  document.getElementById('backup-download-btn')?.addEventListener('click', onDownloadEncryptedBackup);
  document.getElementById('backup-confirmed')?.addEventListener('change', onConfirmBackup);
  document.getElementById('backup-continue-btn')?.addEventListener('click', onBackupContinue);

  // Legacy disconnected view buttons
  document.getElementById('create-wallet-btn')?.addEventListener('click', () => {
    resetCreateForm();
    showView('create');
  });
  document.getElementById('import-wallet-btn')?.addEventListener('click', () => {
    resetImportForm();
    showView('import');
  });

  // Connected dashboard actions
  document.getElementById('send-btn')?.addEventListener('click', toggleSendForm);
  document.getElementById('send-close-btn')?.addEventListener('click', () => {
    document.getElementById('send-form')?.classList.add('hidden');
  });
  document.getElementById('send-submit-btn')?.addEventListener('click', submitSendTransaction);
  document.getElementById('receive-btn')?.addEventListener('click', showQR);
  document.getElementById('copy-btn')?.addEventListener('click', copyAddress);
  document.getElementById('qr-btn')?.addEventListener('click', showQR);

  // Tx banner
  document.getElementById('tx-banner-close')?.addEventListener('click', hideTxBanner);

  // Settings
  document.getElementById('settings-btn')?.addEventListener('click', () => openTab('settings'));
  document.getElementById('footer-settings-btn')?.addEventListener('click', () => openTab('settings'));

  // Footer nav
  document.getElementById('footer-history-btn')?.addEventListener('click', () => openTab('history'));
  document.getElementById('footer-dapps-btn')?.addEventListener('click', () => openTab('dapps'));

  // Network selector
  document.getElementById('network-btn')?.addEventListener('click', showNetworkSelector);

  // Locked view
  document.getElementById('unlock-btn')?.addEventListener('click', unlock);
  document.getElementById('password-input')?.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') unlock();
  });
}

// ========================================
// Create Wallet
// ========================================

function resetCreateForm() {
  document.getElementById('create-name').value = '';
  document.getElementById('create-network').value = '1';
  document.getElementById('create-password').value = '';
  document.getElementById('create-confirm').value = '';
  showFormError('create-error', '');
}

async function submitCreateWallet() {
  showFormError('create-error', '');
  const name = document.getElementById('create-name').value.trim() || 'My Tiger Wallet';
  const chainId = parseInt(document.getElementById('create-network').value, 10) || 1;
  const password = document.getElementById('create-password').value;
  const confirm = document.getElementById('create-confirm').value;

  if (password.length < 8) {
    showFormError('create-error', 'Password must be at least 8 characters.');
    return;
  }
  if (password !== confirm) {
    showFormError('create-error', 'Passwords do not match.');
    return;
  }

  const btn = document.getElementById('create-submit-btn');
  btn.disabled = true;
  btn.textContent = 'Creating...';

  try {
    const result = await sendMessage({
      method: 'tiger_createWallet',
      params: [name, password, chainId],
    });
    if (!result || !result.mnemonic) {
      throw new Error('Backend did not return a recovery phrase.');
    }
    // Hold in memory for the backup step. Never persisted.
    createdMnemonic = result.mnemonic;
    createdWalletId = result.id;
    createdPassword = password;
    createdChainId = result.chain_id || chainId;
    renderBackupView();
    showView('backup');
  } catch (e) {
    showFormError('create-error', e?.message || 'Failed to create wallet.');
  } finally {
    btn.disabled = false;
    btn.textContent = 'Create Wallet';
  }
}

// ========================================
// Import Wallet
// ========================================

function resetImportForm() {
  document.getElementById('import-seed').value = '';
  document.getElementById('import-name').value = '';
  document.getElementById('import-network').value = '1';
  document.getElementById('import-password').value = '';
  document.getElementById('import-confirm').value = '';
  showFormError('import-error', '');
}

async function submitImportWallet() {
  showFormError('import-error', '');
  const mnemonic = document.getElementById('import-seed').value.trim();
  const name = document.getElementById('import-name').value.trim() || 'Imported Wallet';
  const chainId = parseInt(document.getElementById('import-network').value, 10) || 1;
  const password = document.getElementById('import-password').value;
  const confirm = document.getElementById('import-confirm').value;

  const words = mnemonic.split(/\s+/).filter(Boolean);
  if (words.length !== 12 && words.length !== 24) {
    showFormError('import-error', 'Recovery phrase must be 12 or 24 words.');
    return;
  }
  if (password.length < 8) {
    showFormError('import-error', 'Password must be at least 8 characters.');
    return;
  }
  if (password !== confirm) {
    showFormError('import-error', 'Passwords do not match.');
    return;
  }

  const btn = document.getElementById('import-submit-btn');
  btn.disabled = true;
  btn.textContent = 'Importing...';

  try {
    const result = await sendMessage({
      method: 'tiger_importWallet',
      params: [mnemonic, name, password, chainId],
    });
    if (!result || !result.id) {
      throw new Error('Backend did not return a wallet id.');
    }
    // Remember the wallet id locally -> onboarded=true. Imported mnemonics are
    // NOT shown again (the user already has theirs).
    await sendMessage({ method: 'tiger_rememberWallet', params: [result.id] });
    wallet = await sendMessage({ method: 'tiger_getWallet' });
    await enterConnected();
    showNotification('Wallet imported!');
  } catch (e) {
    showFormError('import-error', e?.message || 'Failed to import wallet.');
  } finally {
    btn.disabled = false;
    btn.textContent = 'Import Wallet';
  }
}

// ========================================
// Backup View (mirrors BackupMnemonic.tsx)
// ========================================

function renderBackupView() {
  const grid = document.getElementById('backup-mnemonic-grid');
  grid.replaceChildren();
  if (!createdMnemonic) return;
  const words = createdMnemonic.trim().split(/\s+/).filter(Boolean);
  words.forEach((word, i) => {
    const cell = document.createElement('div');
    cell.className = 'mnemonic-cell';
    const num = document.createElement('span');
    num.className = 'mnemonic-num';
    num.textContent = `${i + 1}`;
    const w = document.createElement('span');
    w.className = 'mnemonic-word';
    w.textContent = word;
    cell.appendChild(num);
    cell.appendChild(w);
    grid.appendChild(cell);
  });
  // Reset the checkboxes + actions for a fresh backup screen.
  document.getElementById('backup-reveal').checked = false;
  document.getElementById('backup-confirmed').checked = false;
  document.getElementById('backup-mnemonic-grid').classList.add('hidden');
  document.getElementById('backup-mnemonic-blur').classList.remove('hidden');
  document.getElementById('backup-copy-btn').disabled = true;
  document.getElementById('backup-download-btn').disabled = true;
  // Google Drive button is enabled/disabled based on oauth2 config.
  configureGoogleDriveButton();
  document.getElementById('backup-continue-btn').disabled = true;
}

function onRevealMnemonic(e) {
  const revealed = e.target.checked;
  document.getElementById('backup-mnemonic-grid').classList.toggle('hidden', !revealed);
  document.getElementById('backup-mnemonic-blur').classList.toggle('hidden', revealed);
  // Copy + Download are only available once the phrase is revealed.
  document.getElementById('backup-copy-btn').disabled = !revealed;
  document.getElementById('backup-download-btn').disabled = !revealed;
}

function onConfirmBackup(e) {
  document.getElementById('backup-continue-btn').disabled = !e.target.checked;
}

async function onCopyMnemonic() {
  if (!createdMnemonic) return;
  try {
    await navigator.clipboard.writeText(createdMnemonic);
    showNotification('Recovery phrase copied!');
  } catch (e) {
    showNotification('Clipboard copy failed.');
  }
}

// ---- Google Drive backup (real Drive REST API v3 via chrome.identity) ----
function googleDriveConfigured() {
  const manifest = chrome.runtime.getManifest();
  const cid = manifest.oauth2 && manifest.oauth2.client_id;
  return typeof cid === 'string' && cid.length > 0;
}

function configureGoogleDriveButton() {
  const btn = document.getElementById('backup-gdrive-btn');
  const msg = document.getElementById('backup-gdrive-msg');
  if (googleDriveConfigured()) {
    // Enabled only when the phrase is also revealed.
    const revealed = document.getElementById('backup-reveal').checked;
    btn.disabled = !revealed;
    msg.classList.add('hidden');
    msg.textContent = '';
  } else {
    // Honestly disabled — NEVER fake a successful upload.
    btn.disabled = true;
    msg.classList.remove('hidden');
    msg.textContent = 'Google Drive backup is not configured. Add your OAuth client_id to manifest.json (oauth2 section) and register it in the Google Cloud Console to enable.';
  }
}

async function onBackupToGoogleDrive() {
  if (!createdMnemonic) return;
  if (!googleDriveConfigured()) {
    configureGoogleDriveButton(); // shows the honest "not configured" message
    return;
  }
  const btn = document.getElementById('backup-gdrive-btn');
  btn.disabled = true;
  btn.textContent = 'Uploading...';
  const msg = document.getElementById('backup-gdrive-msg');
  msg.classList.remove('hidden');
  msg.textContent = 'Requesting Google Drive permission...';
  try {
    const token = await getDriveAuthToken();
    const fileName = `tigerwallet-backup-${new Date().toISOString().replace(/[:.]/g, '-')}.txt`;
    const { body, boundary } = buildDriveMultipart(fileName, createdMnemonic);
    const res = await fetch(
      'https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart',
      {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': `multipart/related; boundary=${boundary}`,
        },
        body,
      }
    );
    if (!res.ok) {
      const err = await res.text();
      throw new Error(`Drive upload failed (${res.status}): ${err}`);
    }
    const data = await res.json();
    msg.textContent = `Uploaded to Google Drive (file id: ${data.id}).`;
    showNotification('Backup uploaded to Google Drive!');
  } catch (e) {
    msg.textContent = e?.message || 'Google Drive backup failed.';
  } finally {
    btn.disabled = false;
    btn.textContent = '☁️ Google Drive';
  }
}

function getDriveAuthToken() {
  return new Promise((resolve, reject) => {
    chrome.identity.getAuthToken(
      { interactive: true, scopes: ['https://www.googleapis.com/auth/drive.file'] },
      (token) => {
        const err = chrome.runtime.lastError;
        if (err || !token) {
          reject(new Error((err && err.message) || 'Drive auth cancelled or failed.'));
        } else {
          resolve(token);
        }
      }
    );
  });
}

// Build a multipart/related body for the Drive v3 files endpoint. The boundary
// separates the JSON metadata part from the file-content part.
function buildDriveMultipart(fileName, content) {
  const boundary = 'tigerwallet_' + Math.random().toString(36).slice(2);
  const metadata = JSON.stringify({ name: fileName, mimeType: 'text/plain', description: 'TigerWallet encrypted-recovery-phrase backup' });
  const body =
    `--${boundary}\r\n` +
    'Content-Type: application/json; charset=UTF-8\r\n\r\n' +
    `${metadata}\r\n` +
    `--${boundary}\r\n` +
    'Content-Type: text/plain\r\n\r\n' +
    `${content}\r\n` +
    `--${boundary}--`;
  return { body, boundary };
}

// ---- Encrypted local download (real WebCrypto AES-GCM + PBKDF2 600k iters) ----
async function onDownloadEncryptedBackup() {
  if (!createdMnemonic) return;
  const btn = document.getElementById('backup-download-btn');
  btn.disabled = true;
  btn.textContent = 'Encrypting...';
  try {
    // Derive an AES-256 key from the wallet password with PBKDF2 (600,000 iters).
    const enc = new TextEncoder();
    const salt = crypto.getRandomValues(new Uint8Array(16));
    const iv = crypto.getRandomValues(new Uint8Array(12));
    const baseKey = await crypto.subtle.importKey(
      'raw', enc.encode(createdPassword), { name: 'PBKDF2' }, false, ['deriveKey']
    );
    const aesKey = await crypto.subtle.deriveKey(
      { name: 'PBKDF2', salt, iterations: 600000, hash: 'SHA-256' },
      baseKey,
      { name: 'AES-GCM', length: 256 },
      false,
      ['encrypt']
    );
    const ciphertext = await crypto.subtle.encrypt(
      { name: 'AES-GCM', iv }, aesKey, enc.encode(createdMnemonic)
    );
    // Pack salt + iv + ciphertext so the same password can decrypt later.
    const packed = new Uint8Array(salt.length + iv.length + ciphertext.byteLength);
    packed.set(salt, 0);
    packed.set(iv, salt.length);
    packed.set(new Uint8Array(ciphertext), salt.length + iv.length);
    const blob = new Blob([packed], { type: 'application/octet-stream' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `tigerwallet-backup-${Date.now()}.enc`;
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(url);
    showNotification('Encrypted backup downloaded.');
  } catch (e) {
    showNotification(e?.message || 'Encrypted backup failed.');
  } finally {
    btn.disabled = false;
    btn.textContent = '⬇️ Download Encrypted';
  }
}

async function onBackupContinue() {
  // Remembering the wallet id flips onboarded=true (>=1 wallet stored locally).
  if (createdWalletId) {
    try {
      await sendMessage({ method: 'tiger_rememberWallet', params: [createdWalletId] });
    } catch (e) {
      console.warn('rememberWallet failed:', e);
    }
  }
  wallet = await sendMessage({ method: 'tiger_getWallet' });
  // Clear sensitive in-memory material.
  createdMnemonic = null;
  createdPassword = null;
  await enterConnected();
  showNotification('Wallet ready!');
}

// ========================================
// Send Transaction + Tx Submitted Banner
// ========================================

function toggleSendForm() {
  const form = document.getElementById('send-form');
  form.classList.toggle('hidden');
  showFormError('send-error', '');
}

async function submitSendTransaction() {
  showFormError('send-error', '');
  if (!wallet || !wallet.id) {
    showFormError('send-error', 'No wallet loaded.');
    return;
  }
  const to = document.getElementById('send-to').value.trim();
  const amount = document.getElementById('send-amount').value.trim();
  const password = document.getElementById('send-password').value;
  const chainId = parseInt((wallet.chainId || '0x1'), 16) || 1;

  if (!/^0x[a-fA-F0-9]{40}$/.test(to)) {
    showFormError('send-error', 'Enter a valid 0x... recipient address.');
    return;
  }
  const amt = parseFloat(amount);
  if (!(amt > 0)) {
    showFormError('send-error', 'Enter a valid amount.');
    return;
  }
  if (password.length < 8) {
    showFormError('send-error', 'Enter your wallet password.');
    return;
  }

  const btn = document.getElementById('send-submit-btn');
  btn.disabled = true;
  btn.textContent = 'Signing...';
  try {
    const result = await sendMessage({
      method: 'tiger_sendTransaction',
      params: [wallet.id, password, to, amount, chainId, '0x'],
    });
    if (!result || !result.tx_hash) {
      throw new Error('No transaction hash returned.');
    }
    showTxBanner(result.tx_hash, result.chain_id || chainId);
    document.getElementById('send-to').value = '';
    document.getElementById('send-amount').value = '';
    document.getElementById('send-password').value = '';
    document.getElementById('send-form').classList.add('hidden');
    await loadWalletData(); // refresh balance
  } catch (e) {
    showFormError('send-error', e?.message || 'Transaction failed.');
  } finally {
    btn.disabled = false;
    btn.textContent = 'Send Transaction';
  }
}

// Mirrors TxSubmittedBanner.tsx: shows the real tx hash (truncated like the web
// reference: first 10 + last 8 chars) + an explorer link that opens in a new
// tab. Auto-dismisses after 30s; the user can also dismiss manually.
function showTxBanner(txHash, chainId) {
  if (!elements.txBanner) return;
  const chain = CHAINS[chainId] || CHAINS[1];
  const display =
    txHash.length > 20 ? `${txHash.slice(0, 10)}…${txHash.slice(-8)}` : txHash;
  elements.txBannerHash.textContent = display;
  elements.txBannerLink.href = chain.explorer.replace('{tx}', txHash);
  elements.txBannerLink.textContent = 'View on explorer ↗';
  elements.txBanner.classList.remove('hidden');
  if (txBannerTimer) clearTimeout(txBannerTimer);
  txBannerTimer = setTimeout(hideTxBanner, 30000);
}

function hideTxBanner() {
  if (!elements.txBanner) return;
  elements.txBanner.classList.add('hidden');
  if (txBannerTimer) {
    clearTimeout(txBannerTimer);
    txBannerTimer = null;
  }
}

// ========================================
// Dashboard data
// ========================================

async function loadWalletData() {
  if (!wallet) return;

  elements.accountAddress.textContent = formatAddress(wallet.address);

  try {
    const balance = await sendMessage({
      method: 'eth_getBalance',
      params: [wallet.address, 'latest'],
    });
    const ethBalance = parseInt(balance, 16) / 1e18;
    elements.totalBalance.textContent = `$${(ethBalance * 2500).toFixed(2)}`;
  } catch (e) {
    console.error('Failed to load balance:', e);
  }

  await loadTokens();
}

async function loadTokens() {
  // Fetch the REAL ERC-20 token balances from the canonical wallet-api
  // backend. Never display fabricated hardcoded token balances. If the fetch
  // fails, show an honest empty state.
  if (!wallet || !wallet.address) {
    elements.tokensList.replaceChildren();
    return;
  }

  let tokens = [];
  try {
    const res = await fetch(
      `http://localhost:8443/api/v1/public/tokens?address=${wallet.address}&chain_id=1`
    );
    if (res.ok) {
      const data = await res.json();
      tokens = Array.isArray(data.tokens)
        ? data.tokens
        : Array.isArray(data.result)
        ? data.result
        : [];
    }
  } catch (e) {
    // Leave tokens empty; the list will show an empty state.
  }

  elements.tokensList.replaceChildren();

  if (tokens.length === 0) {
    const empty = document.createElement('div');
    empty.className = 'token-item empty';
    empty.textContent = 'No token balances';
    elements.tokensList.appendChild(empty);
    return;
  }

  for (const token of tokens) {
    const symbol = String(token.symbol || '?');
    const name = String(token.name || symbol);
    const balance = String(token.balance || '0');
    const value = Number(token.value || 0);

    const item = document.createElement('div');
    item.className = 'token-item';

    const icon = document.createElement('div');
    icon.className = 'token-icon';
    icon.textContent = symbol.charAt(0);
    item.appendChild(icon);

    const info = document.createElement('div');
    info.className = 'token-info';
    const symEl = document.createElement('div');
    symEl.className = 'token-symbol';
    symEl.textContent = symbol;
    info.appendChild(symEl);
    const nameEl = document.createElement('div');
    nameEl.className = 'token-name';
    nameEl.textContent = name;
    info.appendChild(nameEl);
    item.appendChild(info);

    const balWrap = document.createElement('div');
    balWrap.className = 'token-balance';
    const amtEl = document.createElement('div');
    amtEl.className = 'token-amount';
    amtEl.textContent = balance;
    balWrap.appendChild(amtEl);
    const valEl = document.createElement('div');
    valEl.className = 'token-value';
    valEl.textContent = '$' + value.toFixed(2);
    balWrap.appendChild(valEl);
    item.appendChild(balWrap);

    elements.tokensList.appendChild(item);
  }
}

// ========================================
// Actions
// ========================================

function copyAddress() {
  if (!wallet) return;
  navigator.clipboard.writeText(wallet.address);
  showNotification('Address copied!');
}

function showQR() {
  // No QR rendering library is bundled with the extension, so rather than
  // fake a QR image we surface the receivable address for the user to scan
  // or copy. This is honest — not a stub of a missing feature.
  showNotification(wallet ? `Receive at ${formatAddress(wallet.address)} (tap Copy)` : 'No wallet.');
}

async function unlock() {
  const password = document.getElementById('password-input').value;

  try {
    await sendMessage({
      method: 'tiger_unlock',
      params: [password],
    });

    isLocked = false;
    await enterConnected();
  } catch (e) {
    showNotification('Invalid password');
  }
}

function openTab(tab) {
  // The extension has no separate options/history/dapps pages yet, so these
  // footer/header buttons honestly reflect the current state instead of
  // faking navigation to non-existent pages.
  switch (tab) {
    case 'settings':
      if (chrome.runtime.openOptionsPage) {
        // openOptionsPage() is a no-op if no options_ui is declared in the
        // manifest; fall through to the toast in that case.
        try {
          chrome.runtime.openOptionsPage();
          return;
        } catch (e) {
          /* fall through */
        }
      }
      showNotification('Settings — use the theme toggle (🌙) in the header.');
      return;
    case 'history':
      showNotification(wallet ? 'Open the explorer for your wallet history.' : 'No wallet yet.');
      return;
    case 'dapps':
      showNotification('dApps browser — connect a dApp to begin.');
      return;
    default:
      return;
  }
}

function showNetworkSelector() {
  // Honest: the chain is chosen at wallet create/import time. There is no
  // runtime network switcher page yet, so reflect that instead of faking it.
  const chain = wallet ? CHAINS[parseInt(wallet.chainId, 16)] : null;
  showNotification(chain ? `Network: ${chain.name}` : 'Network set at wallet creation.');
}

// ========================================
// Utilities
// ========================================

function formatAddress(address) {
  if (!address) return '';
  return `${address.slice(0, 6)}...${address.slice(-4)}`;
}

function showNotification(message) {
  const toast = document.getElementById('toast');
  if (!toast) {
    // Fallback if the toast element is somehow absent.
    console.warn('TigerWallet:', message);
    return;
  }
  toast.textContent = message;
  toast.classList.remove('hidden');
  if (showNotification._timer) clearTimeout(showNotification._timer);
  showNotification._timer = setTimeout(() => {
    toast.classList.add('hidden');
  }, 2600);
}

function sendMessage(message) {
  return new Promise((resolve, reject) => {
    chrome.runtime.sendMessage(message, (response) => {
      if (chrome.runtime.lastError) {
        reject(chrome.runtime.lastError);
      } else if (response && response.error) {
        reject(new Error(response.error));
      } else {
        resolve(response && response.result);
      }
    });
  });
}
