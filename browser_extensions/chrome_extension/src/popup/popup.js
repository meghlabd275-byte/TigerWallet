/**
 * TigerWallet Chrome Extension - Popup Script
 * 
 * Handles:
 * - Wallet creation/import
 * - Network switching
 * - Balance display
 * - Transaction signing
 * - DApp connections
 */

class TigerWalletPopup {
  constructor() {
    this.wallet = null;
    this.currentNetwork = 'ethereum';
    this.networks = {
      ethereum: { chainId: 1, name: 'Ethereum', symbol: 'ETH', rpc: 'https://eth.llamarpc.com' },
      bsc: { chainId: 56, name: 'BNB Chain', symbol: 'BNB', rpc: 'https://bsc-dataseed.binance.org' },
      polygon: { chainId: 137, name: 'Polygon', symbol: 'MATIC', rpc: 'https://polygon-rpc.com' },
      arbitrum: { chainId: 42161, name: 'Arbitrum', symbol: 'ETH', rpc: 'https://arb1.arbitrum.io/rpc' },
      optimism: { chainId: 10, name: 'Optimism', symbol: 'ETH', rpc: 'https://mainnet.optimism.io' },
      base: { chainId: 8453, name: 'Base', symbol: 'ETH', rpc: 'https://mainnet.base.org' },
      avalanche: { chainId: 43114, name: 'Avalanche', symbol: 'AVAX', rpc: 'https://api.avax.network' },
      solana: { chainId: 101, name: 'Solana', symbol: 'SOL', rpc: 'https://api.mainnet-beta.solana.com' },
    };
    
    this.init();
  }

  async init() {
    this.bindEvents();
    await this.checkWalletStatus();
  }

  bindEvents() {
    // Create wallet button
    document.getElementById('createWalletBtn')?.addEventListener('click', () => {
      this.showModal('createWalletModal');
    });

    // Import wallet button
    document.getElementById('importWalletBtn')?.addEventListener('click', () => {
      this.showModal('importWalletModal');
    });

    // Generate seed phrase
    document.getElementById('generateBtn')?.addEventListener('click', () => {
      this.createWallet();
    });

    // Import wallet
    document.getElementById('importBtn')?.addEventListener('click', () => {
      this.importWallet();
    });

    // Close modals
    document.querySelectorAll('.close-btn').forEach(btn => {
      btn.addEventListener('click', () => {
        this.hideAllModals();
      });
    });

    // Network selector
    document.getElementById('networkBtn')?.addEventListener('click', () => {
      this.showNetworkSelector();
    });
  }

  async checkWalletStatus() {
    try {
      const result = await this.sendMessage({ type: 'GET_WALLET' });
      
      if (result && result.address) {
        this.wallet = result;
        this.showConnectedState();
        await this.updateBalance();
      } else {
        this.showNotConnectedState();
      }
    } catch (error) {
      console.error('Error checking wallet status:', error);
      this.showNotConnectedState();
    }
  }

  showNotConnectedState() {
    document.getElementById('notConnected')?.classList.remove('hidden');
    document.getElementById('connected')?.classList.add('hidden');
  }

  showConnectedState() {
    document.getElementById('notConnected')?.classList.add('hidden');
    document.getElementById('connected')?.classList.remove('hidden');
    
    // Update address display
    if (this.wallet?.address) {
      const shortAddress = this.wallet.address.substring(0, 6) + '...' + this.wallet.address.substring(38);
      document.getElementById('addressDisplay') && (document.getElementById('addressDisplay').textContent = shortAddress);
    }
  }

  async createWallet() {
    const name = document.getElementById('walletName')?.value;
    const password = document.getElementById('walletPassword')?.value;
    const confirmPassword = document.getElementById('confirmPassword')?.value;

    if (!name || !password || !confirmPassword) {
      this.showError('Please fill in all fields');
      return;
    }

    if (password !== confirmPassword) {
      this.showError('Passwords do not match');
      return;
    }

    if (password.length < 8) {
      this.showError('Password must be at least 8 characters');
      return;
    }

    try {
      const seedPhrase = await this.generateSeedPhrase();
      
      // Store wallet
      const walletData = {
        name,
        seedPhrase,
        createdAt: Date.now(),
      };
      
      await this.sendMessage({ 
        type: 'CREATE_WALLET', 
        data: walletData 
      });
      
      this.wallet = await this.deriveAddress(seedPhrase, this.networks[this.currentNetwork].chainId);
      this.showConnectedState();
      this.hideAllModals();
      this.showSuccess('Wallet created successfully!');
      
    } catch (error) {
      this.showError('Failed to create wallet: ' + error.message);
    }
  }

  async importWallet() {
    const seedPhrase = document.getElementById('importSeed')?.value.trim();
    const name = document.getElementById('importWalletName')?.value;
    const password = document.getElementById('importPassword')?.value;

    if (!seedPhrase || !name || !password) {
      this.showError('Please fill in all fields');
      return;
    }

    if (!this.validateSeedPhrase(seedPhrase)) {
      this.showError('Invalid seed phrase');
      return;
    }

    try {
      const walletData = {
        name,
        seedPhrase,
        importedAt: Date.now(),
      };
      
      await this.sendMessage({ 
        type: 'IMPORT_WALLET', 
        data: walletData 
      });
      
      this.wallet = await this.deriveAddress(seedPhrase, this.networks[this.currentNetwork].chainId);
      this.showConnectedState();
      this.hideAllModals();
      this.showSuccess('Wallet imported successfully!');
      
    } catch (error) {
      this.showError('Failed to import wallet: ' + error.message);
    }
  }

  async generateSeedPhrase() {
    const words = WORDLIST;
    const random = new Uint8Array(24);
    crypto.getRandomValues(random);
    
    const phrase = [];
    for (let i = 0; i < 24; i++) {
      const index = random[i] % words.length;
      phrase.push(words[index]);
    }
    
    return phrase.join(' ');
  }

  validateSeedPhrase(phrase) {
    const words = phrase.trim().split(/\s+/);
    return words.length === 12 || words.length === 24;
  }

  async deriveAddress(seedPhrase, chainId) {
    // Simplified address derivation
    const hash = await this.sha256(seedPhrase + chainId);
    const address = '0x' + hash.substring(0, 40);
    
    return {
      address,
      seedPhrase,
      chainId
    };
  }

  async sha256(message) {
    const msgBuffer = new TextEncoder().encode(message);
    const hashBuffer = await crypto.subtle.digest('SHA-256', msgBuffer);
    const hashArray = Array.from(new Uint8Array(hashBuffer));
    return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
  }

  showNetworkSelector() {
    // Show network dropdown
    const networkBtn = document.getElementById('networkBtn');
    if (!networkBtn) return;
    
    // Simple network switch
    const networks = Object.keys(this.networks);
    const currentIndex = networks.indexOf(this.currentNetwork);
    const nextIndex = (currentIndex + 1) % networks.length;
    this.currentNetwork = networks[nextIndex];
    
    // Update UI
    const network = this.networks[this.currentNetwork];
    networkBtn.querySelector('.network-name').textContent = network.name;
    
    // Re-derive address for new network
    if (this.wallet?.seedPhrase) {
      this.wallet = await this.deriveAddress(this.wallet.seedPhrase, network.chainId);
      this.showConnectedState();
    }
  }

  async updateBalance() {
    // Simplified balance update - in production would fetch from RPC
    const balanceEl = document.querySelector('.balance-amount');
    if (balanceEl) {
      balanceEl.textContent = '$0.00';
    }
  }

  showModal(modalId) {
    document.getElementById(modalId)?.classList.remove('hidden');
  }

  hideAllModals() {
    document.querySelectorAll('.modal').forEach(modal => {
      modal.classList.add('hidden');
    });
  }

  showError(message) {
    alert('Error: ' + message);
  }

  showSuccess(message) {
    alert(message);
  }

  sendMessage(message) {
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
}

// BIP-39 Wordlist (abbreviated)
const WORDLIST = [
  'abandon', 'ability', 'able', 'about', 'above', 'absent', 'absorb', 'abstract',
  'absurd', 'abuse', 'access', 'accident', 'account', 'accuse', 'achieve', 'acid',
  'acoustic', 'acquire', 'across', 'act', 'action', 'actor', 'actress', 'actual',
  'adapt', 'add', 'addict', 'address', 'adjust', 'admit', 'adult', 'advance',
  'advice', 'aerobic', 'affair', 'afford', 'afraid', 'again', 'age', 'agent',
  'agree', 'ahead', 'aim', 'air', 'airport', 'aisle', 'alarm', 'album',
];

// Initialize popup
document.addEventListener('DOMContentLoaded', () => {
  window.tigerWallet = new TigerWalletPopup();
});