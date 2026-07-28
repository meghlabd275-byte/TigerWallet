// TigerWallet Popup - Main UI Logic

// ============================================================================
// STATE
// ============================================================================

let currentState = {
  isUnlocked: false,
  hasWallet: false,
  chain: 'ethereum',
  address: '',
  balance: '0',
  transactions: [],
};

// ============================================================================
// DOM ELEMENTS
// ============================================================================

const views = {
  locked: document.getElementById('locked-view'),
  unlocked: document.getElementById('unlocked-view'),
  noWallet: document.getElementById('no-wallet-view'),
};

const elements = {
  unlockForm: document.getElementById('unlock-form'),
  unlockPassword: document.getElementById('unlock-password'),
  importBtn: document.getElementById('import-btn'),
  createWalletBtn: document.getElementById('create-wallet-btn'),
  importExistingBtn: document.getElementById('import-existing-btn'),
  networkBtn: document.getElementById('network-btn'),
  lockBtn: document.getElementById('lock-btn'),
  settingsBtn: document.getElementById('settings-btn'),
  balanceAmount: document.querySelector('.balance-amount'),
  address: document.querySelector('.address'),
  copyBtn: document.querySelector('.copy-btn'),
  transactionsList: document.getElementById('transactions-list'),
};

// ============================================================================
// MESSAGE HANDLING
// ============================================================================

function sendMessage(message) {
  return new Promise((resolve, reject) => {
    chrome.runtime.sendMessage(message, response => {
      if (chrome.runtime.lastError) {
        reject(new Error(chrome.runtime.lastError.message));
      } else {
        resolve(response);
      }
    });
  });
}

// ============================================================================
// UI FUNCTIONS
// ============================================================================

function showView(viewName) {
  Object.values(views).forEach(view => view.classList.add('hidden'));
  views[viewName]?.classList.remove('hidden');
}

function updateUI() {
  if (!currentState.hasWallet) {
    showView('noWallet');
    return;
  }
  
  if (!currentState.isUnlocked) {
    showView('locked');
    return;
  }
  
  showView('unlocked');
  
  // Update address
  if (elements.address) {
    elements.address.textContent = currentState.address 
      ? `${currentState.address.slice(0, 6)}...${currentState.address.slice(-4)}`
      : 'No address';
  }
  
  // Update balance
  if (elements.balanceAmount) {
    elements.balanceAmount.textContent = `$${currentState.balance}`;
  }
  
  // Update network
  if (elements.networkBtn) {
    const networkName = getNetworkName(currentState.chain);
    elements.networkBtn.querySelector('.network-name').textContent = networkName;
  }
  
  // Update transactions
  renderTransactions();
}

function getNetworkName(chainId) {
  const networks = {
    ethereum: 'Ethereum',
    sepolia: 'Sepolia',
    bsc: 'BNB Chain',
    polygon: 'Polygon',
    arbitrum: 'Arbitrum',
    optimism: 'Optimism',
    base: 'Base',
    avalanche: 'Avalanche',
    fantom: 'Fantom',
  };
  return networks[chainId] || chainId;
}

function renderTransactions() {
  if (!elements.transactionsList) return;
  
  if (currentState.transactions.length === 0) {
    elements.transactionsList.innerHTML = `
      <div class="empty-state">
        <p>No transactions yet</p>
      </div>
    `;
    return;
  }
  
  elements.transactionsList.innerHTML = currentState.transactions
    .slice(0, 5)
    .map(tx => `
      <div class="transaction-item">
        <div class="tx-icon ${tx.type === 'in' ? 'receive' : 'send'}">
          ${tx.type === 'in' ? '↓' : '↑'}
        </div>
        <div class="tx-details">
          <span class="tx-type">${tx.type === 'in' ? 'Received' : 'Sent'}</span>
          <span class="tx-amount">${tx.amount} ${tx.symbol}</span>
        </div>
        <div class="tx-status ${tx.status}">
          ${tx.status === 'confirmed' ? '✓' : '...'}
        </div>
      </div>
    `).join('');
}

// ============================================================================
// EVENT HANDLERS
// ============================================================================

// Unlock form submission
elements.unlockForm?.addEventListener('submit', async (e) => {
  e.preventDefault();
  
  const password = elements.unlockPassword.value;
  if (!password) return;
  
  try {
    const response = await sendMessage({
      type: 'UNLOCK_WALLET',
      password,
    });
    
    if (response.success) {
      currentState = {
        ...currentState,
        isUnlocked: true,
        address: response.data.addresses?.[currentState.chain] || '',
      };
      updateUI();
    } else {
      alert(response.error || 'Failed to unlock wallet');
    }
  } catch (error) {
    console.error('Unlock error:', error);
    alert('Failed to unlock wallet');
  }
  
  elements.unlockPassword.value = '';
});

// Lock button
elements.lockBtn?.addEventListener('click', async () => {
  try {
    await sendMessage({ type: 'LOCK_WALLET' });
    currentState.isUnlocked = false;
    updateUI();
  } catch (error) {
    console.error('Lock error:', error);
  }
});

// Copy address
elements.copyBtn?.addEventListener('click', async () => {
  if (currentState.address) {
    await navigator.clipboard.writeText(currentState.address);
    // Show toast or feedback
  }
});

// Network selector
elements.networkBtn?.addEventListener('click', async () => {
  // Show network selector modal
  const networks = ['ethereum', 'bsc', 'polygon', 'arbitrum', 'optimism', 'base', 'avalanche', 'fantom'];
  const chain = await showNetworkSelector(networks);
  
  if (chain && chain !== currentState.chain) {
    try {
      await sendMessage({
        type: 'SWITCH_CHAIN',
        chainId: chain,
      });
      
      currentState.chain = chain;
      currentState.address = ''; // Will be updated
      updateUI();
      
      // Get new address
      const response = await sendMessage({
        type: 'GET_ADDRESS',
        chain: chain,
      });
      
      if (response.success) {
        currentState.address = response.data;
        updateUI();
      }
    } catch (error) {
      console.error('Chain switch error:', error);
    }
  }
});

// Action buttons
document.querySelectorAll('.action-btn').forEach(btn => {
  btn.addEventListener('click', async () => {
    const action = btn.dataset.action;
    
    switch (action) {
      case 'send':
        // Open send view or tab
        break;
      case 'receive':
        // Show receive QR
        break;
      case 'swap':
        // Open swap interface
        break;
      case 'buy':
        // Open buy crypto
        break;
    }
  });
});

// ============================================================================
// NETWORK SELECTOR MODAL
// ============================================================================

async function showNetworkSelector(networks) {
  // Simple prompt for now - in production would be a proper modal
  const networkNames = networks.map(n => `${n} (${getNetworkName(n)})`).join('\n');
  const selected = prompt(`Select network:\n${networkNames}\n\nEnter network name:`);
  return networks.find(n => n === selected?.toLowerCase()) || networks[0];
}

// ============================================================================
// INITIALIZATION
// ============================================================================

async function initialize() {
  try {
    // Get wallet state
    const response = await sendMessage({ type: 'GET_STATE' });
    
    if (response.success) {
      currentState = {
        ...currentState,
        hasWallet: true,
        isUnlocked: response.data.isUnlocked,
        chain: response.data.currentChain || 'ethereum',
        address: response.data.addresses?.[response.data.currentChain] || '',
      };
    } else {
      currentState.hasWallet = false;
    }
    
    updateUI();
    
    // If unlocked, get address for current chain
    if (currentState.isUnlocked) {
      try {
        const addrResponse = await sendMessage({
          type: 'GET_ADDRESS',
          chain: currentState.chain,
        });
        
        if (addrResponse.success) {
          currentState.address = addrResponse.data;
          updateUI();
        }
      } catch (e) {
        console.error('Failed to get address:', e);
      }
    }
  } catch (error) {
    console.error('Initialization error:', error);
    currentState.hasWallet = false;
    updateUI();
  }
  
  // Listen for state changes
  chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    if (message.type === 'STATE_CHANGED') {
      currentState = {
        ...currentState,
        ...message.data,
      };
      updateUI();
    }
  });
}

// Start
initialize();
