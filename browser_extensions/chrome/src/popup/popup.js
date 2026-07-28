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
  const tokens = [
    { symbol: 'ETH', name: 'Ethereum', balance: '0', value: 0 },
    { symbol: 'USDC', name: 'USD Coin', balance: '0', value: 0 },
    { symbol: 'USDT', name: 'Tether USD', balance: '0', value: 0 },
  ];
  
  elements.tokensList.innerHTML = tokens.map(token => `
    <div class="token-item">
      <div class="token-icon">${token.symbol[0]}</div>
      <div class="token-info">
        <div class="token-symbol">${token.symbol}</div>
        <div class="token-name">${token.name}</div>
      </div>
      <div class="token-balance">
        <div class="token-amount">${token.balance}</div>
        <div class="token-value">$${token.value.toFixed(2)}</div>
      </div>
    </div>
  `).join('');
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
