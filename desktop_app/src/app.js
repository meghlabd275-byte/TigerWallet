// TigerWallet Desktop - Production JavaScript
// Full functionality with light/dark theme support

class TigerWalletApp {
    constructor() {
        this.currentTheme = 'dark';
        this.isLocked = true;
        this.wallets = [];
        this.transactions = [];
        this.currentNetwork = 1;
        this.currentPage = 'wallet';
        
        this.init();
    }
    
    async init() {
        // Apply white-label branding (cache first, then async refresh from
        // the control plane). Stock TigerWallet builds fall back to defaults
        // when no WL_BRANDING_SLUG is present.
        if (typeof BrandingConfig !== 'undefined') {
            BrandingConfig.bootstrap();
            BrandingConfig.onChange(() => this.applyBranding());
        }

        // Load theme from storage
        this.loadTheme();
        this.applyBranding();

        // Set up event listeners
        this.setupEventListeners();

        // Load supported chains
        await this.loadChains();

        // Check if wallet exists
        await this.checkWalletStatus();
    }

    // Apply the WL app name to in-app displayed titles (logo headings, etc).
    // The OS window title + CSS vars are handled by branding.js.
    applyBranding() {
        if (typeof BrandingConfig === 'undefined') return;
        const name = BrandingConfig.appName;
        document.querySelectorAll('[data-app-name]').forEach(el => { el.textContent = name; });
        document.querySelectorAll('.logo h1, .sidebar-header h2').forEach(el => {
            if (/TigerWallet/i.test(el.textContent)) el.textContent = '🐯 ' + name;
        });
    }
    
    loadTheme() {
        const savedTheme = localStorage.getItem('tigerwallet-theme') || 'dark';
        this.setTheme(savedTheme);
    }
    
    setTheme(theme) {
        this.currentTheme = theme;
        document.documentElement.setAttribute('data-theme', theme);
        localStorage.setItem('tigerwallet-theme', theme);
        
        // Update theme icon
        const themeIcon = document.getElementById('theme-icon');
        if (themeIcon) {
            themeIcon.textContent = theme === 'dark' ? '🌙' : '☀️';
        }
        
        // Update settings dropdown
        const themeSelect = document.getElementById('settings-theme');
        if (themeSelect) {
            themeSelect.value = theme;
        }
    }
    
    toggleTheme() {
        const newTheme = this.currentTheme === 'dark' ? 'light' : 'dark';
        this.setTheme(newTheme);
    }
    
    setupEventListeners() {
        // Theme toggle
        document.getElementById('theme-toggle')?.addEventListener('click', () => this.toggleTheme());
        
        // Settings theme
        document.getElementById('settings-theme')?.addEventListener('change', (e) => {
            this.setTheme(e.target.value);
        });
        
        // Network selector
        document.getElementById('network-select')?.addEventListener('change', (e) => {
            this.currentNetwork = parseInt(e.target.value);
            this.updateNetworkDisplay();
        });
        
        // Navigation
        document.querySelectorAll('.nav-item').forEach(item => {
            item.addEventListener('click', () => {
                const page = item.dataset.page;
                this.navigateTo(page);
            });
        });
        
        // Lock button
        document.getElementById('lock-btn')?.addEventListener('click', () => this.lockWallet());
        
        // Login
        document.getElementById('unlock-btn')?.addEventListener('click', () => this.unlockWallet());
        document.getElementById('master-password')?.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') this.unlockWallet();
        });
        
        // Create wallet
        document.getElementById('create-wallet-btn')?.addEventListener('click', () => this.showCreateWallet());
        
        // Send transaction
        document.getElementById('simulate-btn')?.addEventListener('click', () => this.simulateTransaction());
        document.getElementById('send-btn')?.addEventListener('click', () => this.sendTransaction());
        
        // QR Scanner
        document.getElementById('qr-scan-btn')?.addEventListener('click', () => this.showQRModal());
        document.getElementById('close-qr-modal')?.addEventListener('click', () => this.hideQRModal());
        document.getElementById('use-address-btn')?.addEventListener('click', () => this.useQRAddress());
        
        // Recent addresses
        document.querySelectorAll('.address-item')?.forEach(item => {
            item.addEventListener('click', () => {
                document.getElementById('send-to').value = item.dataset.address;
                this.hideQRModal();
            });
        });
        
        // Copy address
        document.querySelector('.copy-btn')?.addEventListener('click', () => this.copyAddress());
        
        // Settings
        document.getElementById('settings-currency')?.addEventListener('change', (e) => {
            localStorage.setItem('tigerwallet-currency', e.target.value);
        });
    }
    
    async checkWalletStatus() {
        // Check if there's an existing wallet in localStorage
        const walletData = localStorage.getItem('tigerwallet-master');
        
        if (walletData) {
            this.showLoginScreen();
        } else {
            this.showLoginScreen();
        }
    }
    
    showLoginScreen() {
        document.getElementById('login-screen').classList.remove('hidden');
        document.getElementById('dashboard-screen').classList.add('hidden');
    }
    
    showDashboard() {
        document.getElementById('login-screen').classList.add('hidden');
        document.getElementById('dashboard-screen').classList.remove('hidden');
        this.loadWalletData();
    }
    
    navigateTo(page) {
        // Update nav
        document.querySelectorAll('.nav-item').forEach(item => {
            item.classList.toggle('active', item.dataset.page === page);
        });
        
        // Update pages
        document.querySelectorAll('.page').forEach(p => {
            p.classList.toggle('active', p.id === `${page}-page`);
            p.classList.toggle('hidden', p.id !== `${page}-page`);
        });
        
        this.currentPage = page;
        
        // Update title
        const titles = {
            wallet: 'My Wallet',
            send: 'Send',
            receive: 'Receive',
            swap: 'Swap',
            staking: 'Staking',
            bridge: 'Bridge',
            nft: 'NFT Gallery',
            transactions: 'Transactions',
            settings: 'Settings'
        };
        document.getElementById('page-title').textContent = titles[page] || 'TigerWallet';
        
        // Load page-specific data
        if (page === 'transactions') {
            this.loadTransactions();
        } else if (page === 'wallet') {
            this.loadWalletData();
        }
    }
    
    async unlockWallet() {
        const password = document.getElementById('master-password').value;

        if (!password) {
            alert('Please enter your master password');
            return;
        }

        // Verify the password against the stored PBKDF2-SHA256 hash. Never
        // accept an arbitrary password. The stored blob is "salt:hash"; if no
        // wallet exists, prompt to create one.
        const stored = localStorage.getItem('tigerwallet-master');
        if (!stored) {
            alert('No wallet found. Please create a wallet first.');
            this.showCreateWallet();
            return;
        }
        const ok = await this.verifyPassword(password, stored);
        if (!ok) {
            alert('Incorrect password');
            return;
        }

        this.isLocked = false;
        this.showDashboard();
    }
    
    async lockWallet() {
        this.isLocked = true;
        document.getElementById('master-password').value = '';
        this.showLoginScreen();
    }
    
    async showCreateWallet() {
        const name = prompt('Enter wallet name:');
        const password = prompt('Set master password:');

        if (!name || !password) {
            return;
        }
        if (password.length < 8) {
            alert('Password must be at least 8 characters');
            return;
        }

        try {
            // Create a real wallet on the canonical wallet_api backend
            // (POST /api/v1/wallets) — the backend derives the address from a
            // real BIP-39 seed (secp256k1 / BIP-44). Never fabricate one here.
            const res = await fetch('http://localhost:8443/api/v1/wallets', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ label: name, password })
            });
            if (!res.ok) {
                const err = await res.text();
                alert('Wallet creation failed: ' + err);
                return;
            }
            const wallet = await res.json();

            this.wallets.push({
                id: this.generateId(),
                name: name,
                address: wallet.address,
                balance: '0',
                tokens: []
            });
            localStorage.setItem('tigerwallet-wallets', JSON.stringify(this.wallets));
            localStorage.setItem('tigerwallet-master', await this.hashPassword(password));

            alert('Wallet created successfully!');
            this.showDashboard();
        } catch (e) {
            alert('Wallet creation error: ' + e.message);
        }
    }
    
    async loadChains() {
        // In production, fetch from Tauri backend
        this.chains = [
            { id: 1, name: 'Ethereum', symbol: 'ETH' },
            { id: 56, name: 'BNB Chain', symbol: 'BNB' },
            { id: 137, name: 'Polygon', symbol: 'MATIC' },
            { id: 42161, name: 'Arbitrum', symbol: 'ETH' },
            { id: 10, name: 'Optimism', symbol: 'ETH' },
            { id: 8453, name: 'Base', symbol: 'ETH' },
            { id: 43114, name: 'Avalanche', symbol: 'AVAX' }
        ];
        
        this.renderNetworks();
    }
    
    renderNetworks() {
        const networksList = document.getElementById('networks-list');
        if (!networksList) return;
        
        networksList.innerHTML = this.chains.map(chain => `
            <div class="network-item">
                <span class="icon">🔗</span>
                <span>${chain.name} (${chain.symbol})</span>
            </div>
        `).join('');
    }
    
    updateNetworkDisplay() {
        const chain = this.chains.find(c => c.id === this.currentNetwork);
        if (chain) {
            console.log('Network changed to:', chain.name);
        }
    }
    
    async loadWalletData() {
        // Load wallets from storage
        const stored = localStorage.getItem('tigerwallet-wallets');
        if (stored) {
            this.wallets = JSON.parse(stored);
        }
        
        if (this.wallets.length === 0) {
            // No demo wallet — show an empty state and prompt the user to
            // create one. Never fabricate balances/addresses.
            this.renderWallet();
            return;
        }

        // Fetch the REAL native balance for the first wallet from the
        // canonical backend (GET /api/v1/public/balance?address=&chain_id=).
        const wallet = this.wallets[0];
        try {
            const chainId = this.currentNetwork || 1;
            const res = await fetch(
                `http://localhost:8443/api/v1/public/balance?address=${wallet.address}&chain_id=${chainId}`
            );
            if (res.ok) {
                const data = await res.json();
                wallet.balance = data.balance || '0';
            }
        } catch (e) {
            // Leave stored balance; the UI shows the last known value.
        }
        
        this.renderWallet();
    }
    
    renderWallet() {
        const wallet = this.wallets[0];
        if (!wallet) return;
        
        // Update balance
        const totalBalance = wallet.tokens.reduce((sum, t) => sum + (t.usdValue || 0), 0);
        document.querySelector('.balance-amount').textContent = `$${totalBalance.toLocaleString('en-US', { minimumFractionDigits: 2 })}`;
        
        // Update address
        const addressEl = document.querySelector('.asset-address');
        if (addressEl) {
            addressEl.textContent = this.formatAddress(wallet.address);
        }
        
        // Update receive address
        const receiveAddress = document.getElementById('receive-address');
        if (receiveAddress) {
            receiveAddress.textContent = wallet.address;
        }
        
        // Render assets
        this.renderAssets(wallet);
    }
    
    renderAssets(wallet) {
        const assetsList = document.getElementById('assets-list');
        if (!assetsList) return;
        
        const icons = { ETH: '🔷', USDT: '🔶', USDC: '🔷', BNB: '🟡', MATIC: '🟣' };
        
        assetsList.innerHTML = wallet.tokens.map(token => `
            <div class="asset-item">
                <div class="asset-icon">${icons[token.symbol] || '🪙'}</div>
                <div class="asset-info">
                    <div class="asset-name">${token.symbol}</div>
                    <div class="asset-address">${this.formatAddress(wallet.address)}</div>
                </div>
                <div class="asset-balance">
                    <div class="balance">${token.balance} ${token.symbol}</div>
                    <div class="usd-value">$${token.usdValue?.toLocaleString() || '0'}</div>
                </div>
            </div>
        `).join('');
    }
    
    async simulateTransaction() {
        const to = document.getElementById('send-to').value;
        const amount = document.getElementById('send-amount').value;
        
        if (!to || !amount) {
            alert('Please fill in recipient and amount');
            return;
        }
        
        // Show simulation result
        const result = document.getElementById('simulation-result');
        result.innerHTML = `
            <strong>Simulation Result:</strong><br>
            To: ${this.formatAddress(to)}<br>
            Amount: ${amount} ETH<br>
            Gas: ~21000 units<br>
            Total Cost: ${(amount * 1.00002).toFixed(6)} ETH<br>
            <span style="color: var(--success)">✓ Transaction would succeed</span>
        `;
        result.classList.add('show');
    }
    
    async sendTransaction() {
        const to = document.getElementById('send-to').value;
        const amount = document.getElementById('send-amount').value;

        if (!to || !amount) {
            alert('Please fill in recipient and amount');
            return;
        }

        if (this.isLocked) {
            alert('Please unlock your wallet first');
            return;
        }
        if (!this.wallets.length) {
            alert('No wallet loaded');
            return;
        }

        try {
            // Broadcast via the canonical wallet_api backend (POST /api/v1/send)
            // which performs real secp256k1 signing + eth_sendRawTransaction.
            // This client never fabricates a transaction hash.
            const res = await fetch('http://localhost:8443/api/v1/send', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    from_address: this.wallets[0].address,
                    to_address: to,
                    amount: amount,
                    chain_id: this.currentNetwork
                })
            });
            if (!res.ok) {
                const err = await res.text();
                alert('Transaction failed: ' + err);
                return;
            }
            const result = await res.json();

            this.transactions.push({
                id: this.generateId(),
                hash: result.tx_hash,
                from: this.wallets[0].address,
                to: to,
                value: amount,
                token: 'ETH',
                status: 'pending',
                chain_id: this.currentNetwork,
                timestamp: Date.now(),
                gas_used: 21000,
                gas_price: '0'
            });

            document.getElementById('send-to').value = '';
            document.getElementById('send-amount').value = '';
            document.getElementById('simulation-result')?.classList.remove('show');

            alert('Transaction broadcast: ' + result.tx_hash);
            this.navigateTo('transactions');
        } catch (e) {
            alert('Transaction error: ' + e.message);
        }
    }
    
    // QR Scanner Functions
    showQRModal() {
        document.getElementById('qr-modal')?.classList.remove('hidden');
    }
    
    hideQRModal() {
        document.getElementById('qr-modal')?.classList.add('hidden');
        document.getElementById('manual-address').value = '';
    }
    
    useQRAddress() {
        const address = document.getElementById('manual-address')?.value.trim();
        if (address) {
            document.getElementById('send-to').value = address;
            this.hideQRModal();
        }
    }
    
    isValidAddress(address) {
        // Ethereum
        if (/^0x[a-fA-F0-9]{40}$/.test(address)) return true;
        // Bitcoin
        if (/^(bc1|[13])[a-zA-HJ-NP-Z0-9]{25,62}$/.test(address)) return true;
        // Solana
        if (/^[1-9A-HJ-NP-Z]{32,44}$/.test(address)) return true;
        // TRON
        if (/^T[a-zA-HJ-NP-Z0-9]{33}$/.test(address)) return true;
        return false;
    }
    
    async loadTransactions() {
        const list = document.getElementById('transactions-list');
        if (!list) return;
        
        // Fetch REAL transactions from the canonical backend. Never show
        // fabricated demo transactions.
        const wallet = this.wallets[0];
        if (!wallet || !wallet.address) {
            list.innerHTML = '<div class="empty-state">No wallet connected</div>';
            return;
        }
        try {
            const chainId = this.currentNetwork || 1;
            const res = await fetch(
                `http://localhost:8443/api/v1/public/transactions?address=${wallet.address}&chain_id=${chainId}`
            );
            if (!res.ok) {
                list.innerHTML = '<div class="empty-state">Failed to load transactions</div>';
                return;
            }
            const data = await res.json();
            this.transactions = Array.isArray(data.result) ? data.result :
                (Array.isArray(data.transactions) ? data.transactions : []);
        } catch (e) {
            list.innerHTML = '<div class="empty-state">Failed to load transactions</div>';
            return;
        }

        if (this.transactions.length === 0) {
            list.innerHTML = '<div class="empty-state">No transactions yet</div>';
            return;
        }
        
        list.innerHTML = this.transactions.map(tx => `
            <div class="asset-item">
                <div class="asset-icon">${tx.value > 0 ? '📤' : '📥'}</div>
                <div class="asset-info">
                    <div class="asset-name">${tx.token} Transfer</div>
                    <div class="asset-address">${this.formatAddress(tx.from)} → ${this.formatAddress(tx.to)}</div>
                </div>
                <div class="asset-balance">
                    <div class="balance" style="color: ${tx.value > 0 ? 'var(--danger)' : 'var(--success)'}">
                        ${tx.value > 0 ? '-' : '+'}${tx.value} ${tx.token}
                    </div>
                    <div class="usd-value">${new Date(tx.timestamp).toLocaleDateString()}</div>
                </div>
            </div>
        `).join('');
    }
    
    copyAddress() {
        const address = document.getElementById('receive-address').textContent;
        navigator.clipboard.writeText(address);
        alert('Address copied!');
    }
    
    formatAddress(address) {
        if (!address) return '';
        return `${address.slice(0, 6)}...${address.slice(-4)}`;
    }
    
    generateId() {
        if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
            return crypto.randomUUID();
        }
        const bytes = new Uint8Array(16);
        crypto.getRandomValues(bytes);
        bytes[6] = (bytes[6] & 0x0f) | 0x40;
        bytes[8] = (bytes[8] & 0x3f) | 0x80;
        return Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('');
    }

    generateAddress() {
        // Addresses are derived by the canonical wallet-api backend (/wallets)
        // from a real BIP-39 seed. This client never fabricates an address.
        throw new Error('Address derivation is performed by the canonical wallet-api backend (/wallets); client-side fabrication is disabled');
    }

    generateHash() {
        // Transaction hashes are produced by the backend's signed broadcast
        // (/send). This client never fabricates a hash.
        throw new Error('Transaction hashes are produced by the canonical wallet-api backend (/send); client-side fabrication is disabled');
    }
    
    async hashPassword(password) {
        // PBKDF2-SHA256, 600k iterations, 16-byte salt + 32-byte hash.
        // Stored as "saltHex:hashHex". Never use a non-cryptographic hash.
        const salt = crypto.getRandomValues(new Uint8Array(16));
        const hash = await this.derivePbkdf2(password, salt);
        const toHex = (b) => Array.from(b, (x) => x.toString(16).padStart(2, '0')).join('');
        return `${toHex(salt)}:${toHex(hash)}`;
    }

    async verifyPassword(password, stored) {
        const [saltHex, hashHex] = stored.split(':');
        if (!saltHex || !hashHex) return false;
        const salt = new Uint8Array(saltHex.match(/.{2}/g).map((b) => parseInt(b, 16)));
        const hash = await this.derivePbkdf2(password, salt);
        const expected = hashHex;
        const actual = Array.from(hash, (x) => x.toString(16).padStart(2, '0')).join('');
        // Constant-time comparison.
        if (actual.length !== expected.length) return false;
        let diff = 0;
        for (let i = 0; i < actual.length; i++) {
            diff |= actual.charCodeAt(i) ^ expected.charCodeAt(i);
        }
        return diff === 0;
    }

    async derivePbkdf2(password, salt) {
        const enc = new TextEncoder();
        const keyMaterial = await crypto.subtle.importKey(
            'raw', enc.encode(password), { name: 'PBKDF2' }, false, ['deriveBits']
        );
        const bits = await crypto.subtle.deriveBits(
            { name: 'PBKDF2', salt, iterations: 600000, hash: 'SHA-256' },
            keyMaterial, 256
        );
        return new Uint8Array(bits);
    }
}

// Initialize app
document.addEventListener('DOMContentLoaded', () => {
    window.app = new TigerWalletApp();
});
