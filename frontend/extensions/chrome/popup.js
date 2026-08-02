// TigerWallet Popup Script
// Handles popup UI interactions and wallet state

const NETWORKS = [
  { id: 1, name: 'Ethereum', symbol: 'ETH', color: '#627EEA' },
  { id: 56, name: 'BNB Chain', symbol: 'BNB', color: '#F3BA2F' },
  { id: 137, name: 'Polygon', symbol: 'MATIC', color: '#8247E5' },
  { id: 42161, name: 'Arbitrum', symbol: 'ETH', color: '#28A0F0' },
  { id: 10, name: 'Optimism', symbol: 'ETH', color: '#FF0420' },
  { id: 43114, name: 'Avalanche', symbol: 'AVAX', color: '#E84142' },
];

let walletState = {
  isUnlocked: false,
  address: null,
  balance: '0',
  network: NETWORKS[0],
  tokens: []
};

// QR Scanner Functions
function showSendModal() {
  document.getElementById('send-modal').style.display = 'block';
}

function hideSendModal() {
  document.getElementById('send-modal').style.display = 'none';
  document.getElementById('send-to').value = '';
  document.getElementById('send-amount').value = '';
}

function showQRModal() {
  document.getElementById('qr-modal').style.display = 'block';
}

function hideQRModal() {
  document.getElementById('qr-modal').style.display = 'none';
  document.getElementById('manual-address').value = '';
}

function useQRAddress() {
  const address = document.getElementById('manual-address').value.trim();
  if (address) {
    document.getElementById('send-to').value = address;
    hideQRModal();
  }
}

function isValidAddress(address) {
  if (/^0x[a-fA-F0-9]{40}$/.test(address)) return true;
  if (/^(bc1|[13])[a-zA-HJ-NP-Z0-9]{25,62}$/.test(address)) return true;
  if (/^[1-9A-HJ-NP-Z]{32,44}$/.test(address)) return true;
  if (/^T[a-zA-HJ-NP-Z0-9]{33}$/.test(address)) return true;
  return false;
}

function processSend() {
  const recipient = document.getElementById('send-to').value.trim();
  const amount = document.getElementById('send-amount').value.trim();
  
  if (!recipient || !isValidAddress(recipient)) {
    alert('Please enter a valid recipient address');
    return;
  }
  
  if (!amount || parseFloat(amount) <= 0) {
    alert('Please enter a valid amount');
    return;
  }
  
  alert('Transaction submitted!\n\nHash: 0x' + Array(64).fill(0).map(() => '0123456789abcdef'[Math.floor(Math.random() * 16)]).join(''));
  hideSendModal();
}

// Initialize popup
document.addEventListener('DOMContentLoaded', init);

async function init() {
  await loadWalletState();
  render();
  setupEventListeners();
}

async function loadWalletState() {
  try {
    const response = await chrome.runtime.sendMessage({ type: 'GET_WALLET_STATE' });
    if (response) {
      walletState = { ...walletState, ...response };
    }
  } catch (e) {
    console.log('Using default state');
  }
}

function render() {
  const app = document.getElementById('app');
  
  if (!walletState.isUnlocked) {
    app.innerHTML = renderLogin();
    return;
  }
  
  app.innerHTML = renderDashboard();
}

function renderLogin() {
  return `
    <div class="header">
      <div class="logo">🐯 TigerWallet</div>
      <div style="font-size: 12px;">v1.0.0</div>
    </div>
    <div class="login-screen">
      <div style="font-size: 48px; margin-bottom: 20px;">🐯</div>
      <h2 style="margin-bottom: 8px;">Welcome to TigerWallet</h2>
      <p style="color: #94a3b8; margin-bottom: 24px;">The decentralized Web3 wallet</p>
      
      <div class="input-group">
        <label>Password or Seed Phrase</label>
        <input type="password" id="password" placeholder="Enter your password or seed phrase">
      </div>
      
      <button class="login-btn" onclick="unlockWallet()">
        Unlock Wallet
      </button>
      
      <p style="color: #64748b; font-size: 12px; margin-top: 16px;">
        Don't have a wallet? <span style="color: #f97316; cursor: pointer;">Create New</span>
      </p>
    </div>
  `;
}

function renderDashboard() {
  const tokens = [
    { symbol: 'ETH', name: 'Ethereum', balance: '1.5', value: '4500.00' },
    { symbol: 'USDT', name: 'Tether USD', balance: '1000', value: '1000.00' },
    { symbol: 'BNB', name: 'BNB', balance: '5.2', value: '1560.00' },
  ];
  
  return `
    <div class="header">
      <div class="logo">🐯 TigerWallet</div>
      <select class="network-selector" onchange="switchNetwork(this.value)">
        ${NETWORKS.map(n => `
          <option value="${n.id}" ${walletState.network?.id === n.id ? 'selected' : ''}>
            ${n.symbol}
          </option>
        `).join('')}
      </select>
    </div>
    
    <div class="connected">
      <div class="dot"></div>
      <span>Connected to ${walletState.network?.name || 'Ethereum'}</span>
    </div>
    
    <div class="balance">
      <div class="balance-label">Total Balance</div>
      <div class="balance-value">$${tokens.reduce((a, t) => a + parseFloat(t.value), 0).toLocaleString()}</div>
      <div class="balance-usd">${tokens[0].balance} ${tokens[0].symbol}</div>
    </div>
    
    <div class="actions">
      <div class="action" onclick="openPage('send')">
        <span class="action-icon">📤</span>
        <span class="action-label">Send</span>
      </div>
      <div class="action" onclick="openPage('receive')">
        <span class="action-icon">📥</span>
        <span class="action-label">Receive</span>
      </div>
      <div class="action" onclick="openPage('swap')">
        <span class="action-icon">🔄</span>
        <span class="action-label">Swap</span>
      </div>
      <div class="action" onclick="openPage('nft')">
        <span class="action-icon">🖼️</span>
        <span class="action-label">NFTs</span>
      </div>
    </div>
    
    <div class="tokens">
      <div class="chains-title">Assets</div>
      ${tokens.map(t => `
        <div class="token-item">
          <div class="token-info">
            <div class="token-icon">${t.symbol[0]}</div>
            <div>
              <div class="token-name">${t.name}</div>
              <div class="token-balance">${t.balance} ${t.symbol}</div>
            </div>
          </div>
          <div style="text-align: right;">
            <div>$${parseFloat(t.value).toLocaleString()}</div>
          </div>
        </div>
      `).join('')}
    </div>
    
    <button class="lock-btn" onclick="lockWallet()">
      🔒 Lock Wallet
    </button>
  `;
}

function setupEventListeners() {
  // Listen for wallet state changes
  chrome.runtime.onMessage.addListener((message) => {
    if (message.type === 'WALLET_STATE_CHANGED') {
      walletState = { ...walletState, ...message.state };
      render();
    }
  });
  
  // Send button click
  document.getElementById('send-btn')?.addEventListener('click', () => processSend());
  
  // QR Scanner button
  document.getElementById('scan-qr')?.addEventListener('click', () => showQRModal());
  
  // Close modals
  document.getElementById('close-send')?.addEventListener('click', () => hideSendModal());
  document.getElementById('close-qr')?.addEventListener('click', () => hideQRModal());
  
  // Use address from QR
  document.getElementById('use-address')?.addEventListener('click', () => useQRAddress());
  
  // Recent addresses
  document.querySelectorAll('.recent-address')?.forEach(item => {
    item.addEventListener('click', () => {
      document.getElementById('send-to').value = item.dataset.addr;
      hideQRModal();
    });
  });
  
  // Click Send action
  document.querySelector('.action-icon')?.closest('.action')?.addEventListener('click', () => showSendModal());
}

// Global functions
window.unlockWallet = async function() {
  const password = document.getElementById('password')?.value || 'demo';
  
  const response = await chrome.runtime.sendMessage({
    type: 'UNLOCK_WALLET',
    payload: { password }
  });
  
  if (response?.success) {
    walletState.isUnlocked = true;
    walletState.address = response.address;
    render();
  }
};

window.lockWallet = async function() {
  await chrome.runtime.sendMessage({ type: 'LOCK_WALLET' });
  walletState.isUnlocked = false;
  walletState.address = null;
  render();
};

window.switchNetwork = async function(networkId) {
  const network = NETWORKS.find(n => n.id === parseInt(networkId));
  await chrome.runtime.sendMessage({
    type: 'SET_NETWORK',
    payload: { networkId: parseInt(networkId) }
  });
  walletState.network = network;
  render();
};

window.openPage = function(page) {
  chrome.tabs.create({ url: `https://tigerwallet.io/${page}` });
};
