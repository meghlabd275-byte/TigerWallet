/**
 * TigerWallet Popup Script
 */

// ========================================
// State
// ========================================

let wallet = null;
let isLocked = true;
let isConnected = false;
let theme = 'dark';
let currentChain = 1;

// ========================================
// DOM Elements
// ========================================

const views = {
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
};

// ========================================
// Initialize
// ========================================

document.addEventListener('DOMContentLoaded', async () => {
  await loadSettings();
  await checkWallet();
  setupEventListeners();
  applyTheme();
});

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

async function checkWallet() {
  try {
    wallet = await sendMessage({ method: 'tiger_getWallet' });
    
    if (wallet) {
      const accounts = await sendMessage({ method: 'eth_accounts' });
      isConnected = accounts && accounts.length > 0;
      
      if (isConnected) {
        showView('connected');
        await loadWalletData();
      } else {
        showView('disconnected');
      }
    } else {
      showView('disconnected');
    }
  } catch (e) {
    console.error('Failed to check wallet:', e);
    showView('disconnected');
  }
}

async function loadWalletData() {
  if (!wallet) return;
  
  // Update UI
  elements.accountAddress.textContent = formatAddress(wallet.address);
  
  // Load balance
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
  
  // Load tokens
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
      tokens = Array.isArray(data.tokens) ? data.tokens :
        (Array.isArray(data.result) ? data.result : []);
    }
  } catch (e) {
    // Leave tokens empty; the list will show an empty state.
  }

  // Build the list with DOM APIs (createElement + textContent) so backend
  // values cannot inject markup (XSS-safe).
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
// View Management
// ========================================

function showView(viewName) {
  Object.values(views).forEach(view => view.classList.add('hidden'));
  views[viewName]?.classList.remove('hidden');
}

// ========================================
// Theme
// ========================================

function applyTheme() {
  document.documentElement.setAttribute('data-theme', theme);
  elements.themeBtn.textContent = theme === 'dark' ? '🌙' : '☀️';
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
  elements.themeBtn.addEventListener('click', toggleTheme);
  
  // Action buttons
  document.getElementById('send-btn')?.addEventListener('click', () => openTab('send'));
  document.getElementById('receive-btn')?.addEventListener('click', () => openTab('receive'));
  document.getElementById('swap-btn')?.addEventListener('click', () => openTab('swap'));
  document.getElementById('bridge-btn')?.addEventListener('click', () => openTab('bridge'));
  document.getElementById('settings-btn')?.addEventListener('click', () => openTab('settings'));
  document.getElementById('settings-footer-btn')?.addEventListener('click', () => openTab('settings'));
  
  // Account buttons
  document.getElementById('copy-btn')?.addEventListener('click', copyAddress);
  document.getElementById('qr-btn')?.addEventListener('click', showQR);
  
  // Wallet buttons
  document.getElementById('create-wallet-btn')?.addEventListener('click', createWallet);
  document.getElementById('import-wallet-btn')?.addEventListener('click', importWallet);
  document.getElementById('unlock-btn')?.addEventListener('click', unlock);
  document.getElementById('password-input')?.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') unlock();
  });
  
  // Network selector
  document.getElementById('network-btn')?.addEventListener('click', showNetworkSelector);
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
  // Would show QR modal
  showNotification('QR Code');
}

async function createWallet() {
  try {
    const name = 'My Wallet';
    const password = 'password123'; // Would get from input
    
    wallet = await sendMessage({
      method: 'tiger_createWallet',
      params: [name, password],
    });
    
    isConnected = true;
    showView('connected');
    await loadWalletData();
    showNotification('Wallet created!');
  } catch (e) {
    showNotification('Failed to create wallet');
  }
}

async function importWallet() {
  // Would show import form
  showNotification('Import wallet');
}

async function unlock() {
  const password = document.getElementById('password-input').value;
  
  try {
    await sendMessage({
      method: 'tiger_unlock',
      params: [password],
    });
    
    isLocked = false;
    showView('connected');
    await loadWalletData();
  } catch (e) {
    showNotification('Invalid password');
  }
}

function openTab(tab) {
  // Would open tab or navigate
  console.log('Open tab:', tab);
}

function showNetworkSelector() {
  // Would show network selector modal
  console.log('Show network selector');
}

// ========================================
// Utilities
// ========================================

function formatAddress(address) {
  if (!address) return '';
  return `${address.slice(0, 6)}...${address.slice(-4)}`;
}

function showNotification(message) {
  // Simple notification - would use toast in production
  alert(message);
}

function sendMessage(message) {
  return new Promise((resolve, reject) => {
    chrome.runtime.sendMessage(message, response => {
      if (chrome.runtime.lastError) {
        reject(chrome.runtime.lastError);
      } else if (response?.error) {
        reject(new Error(response.error));
      } else {
        resolve(response?.result);
      }
    });
  });
}
