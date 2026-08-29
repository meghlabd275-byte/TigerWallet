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
        document.getElementById('import-wallet-btn')?.addEventListener('click', () => this.showImportWallet());

        // Cloud backup (export encrypted seed blob for Google Drive)
        document.getElementById('backup-btn')?.addEventListener('click', () => this.exportEncryptedBackup());
        
        // Send transaction
        document.getElementById('simulate-btn')?.addEventListener('click', () => this.simulateTransaction());
        document.getElementById('send-btn')?.addEventListener('click', () => this.sendTransaction());

        // Swap — live quote on input change + execute on button click.
        const swapFromAmt = document.getElementById('swap-from-amount');
        const swapFromTok = document.getElementById('swap-from-token');
        const swapToTok = document.getElementById('swap-to-token');
        const swapBtn = document.getElementById('swap-btn');
        [swapFromAmt, swapFromTok, swapToTok].forEach(el => {
            el?.addEventListener('input', () => this.fetchSwapQuote());
            el?.addEventListener('change', () => this.fetchSwapQuote());
        });
        swapBtn?.addEventListener('click', () => this.executeSwap());

        // Staking actions (real backend /staking/{stake,unstake,claim})
        document.getElementById('stake-btn')?.addEventListener('click', () => this.stakingAction('stake'));
        document.getElementById('unstake-btn')?.addEventListener('click', () => this.stakingAction('unstake'));
        document.getElementById('claim-btn')?.addEventListener('click', () => this.stakingAction('claim'));
        document.getElementById('stake-max-btn')?.addEventListener('click', () => {
            const balEl = document.querySelector('.swap-balance');
            const m = (balEl?.textContent || '').match(/([\d.]+)/);
            if (m) document.getElementById('stake-amount').value = m[1];
        });

        // Bridge (real backend /bridge/{quote,transfer})
        document.getElementById('bridge-from-chain')?.addEventListener('change', () => this.fetchBridgeQuote());
        document.getElementById('bridge-to-chain')?.addEventListener('change', () => this.fetchBridgeQuote());
        document.getElementById('bridge-amount')?.addEventListener('input', () => this.fetchBridgeQuote());
        document.getElementById('bridge-btn')?.addEventListener('click', () => this.executeBridge());

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
        } else if (page === 'staking') {
            this.loadStakingAssets();
        } else if (page === 'bridge') {
            this.loadBridgeRoutes();
        } else if (page === 'nft') {
            this.loadNFTs();
        } else if (page === 'swap') {
            this.fetchSwapQuote();
        } else if (page === 'ens') {
            // ENS resolve is input-driven; no auto-load.
        } else if (page === 'kyc') {
            this.loadKYCStatus();
        } else if (page === 'ramp') {
            this.loadFiatProviders();
        } else if (page === 'dapps') {
            this.loadDApps();
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
                wallet_id: wallet.id,          // server-side wallet id required by /api/v1/send
                name: name,
                address: wallet.address,
                balance: '0',
                tokens: [],
                unlocked: false
            });
            localStorage.setItem('tigerwallet-wallets', JSON.stringify(this.wallets));
            localStorage.setItem('tigerwallet-master', await this.hashPassword(password));

            alert('Wallet created successfully!');
            this.showDashboard();
        } catch (e) {
            alert('Wallet creation error: ' + e.message);
        }
    }
    
    // Import an existing wallet from a 24-word BIP-39 mnemonic. Submits the
    // user's mnemonic to the canonical wallet_api (POST /api/v1/wallets with
    // a mnemonic) which re-derives the address server-side. The mnemonic is
    // never persisted client-side beyond this request.
    async showImportWallet() {
        const mnemonic = prompt('Enter your 24-word recovery phrase:');
        if (!mnemonic || mnemonic.trim().split(/\s+/).length < 12) {
            alert('A valid 12/24-word recovery phrase is required');
            return;
        }
        const password = prompt('Set a master password for this imported wallet (min 8 chars):');
        if (!password || password.length < 8) {
            alert('Password must be at least 8 characters');
            return;
        }
        const name = prompt('Wallet name (optional):') || 'Imported Wallet';
        try {
            const res = await fetch('http://localhost:8443/api/v1/wallets', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ label: name, password, mnemonic: mnemonic.trim() })
            });
            if (!res.ok) {
                const err = await res.text();
                alert('Wallet import failed: ' + err);
                return;
            }
            const wallet = await res.json();
            this.wallets.push({
                id: this.generateId(),
                wallet_id: wallet.id,
                name,
                address: wallet.address,
                balance: '0',
                tokens: [],
                unlocked: false
            });
            localStorage.setItem('tigerwallet-wallets', JSON.stringify(this.wallets));
            localStorage.setItem('tigerwallet-master', await this.hashPassword(password));
            // The mnemonic returned by import is empty (only set on create);
            // do not store it. Warn the user to keep their own seed safe.
            alert('Wallet imported successfully! Keep your 24-word seed safe — it is your only recovery method.');
            this.showDashboard();
        } catch (e) {
            alert('Wallet import error: ' + e.message);
        }
    }

    // Export the wallet's AES-256-GCM encrypted seed blob (POST
    // /api/v1/wallets/:id/export-encrypted-seed). The backend returns a
    // password-verified blob the user can save to Google Drive / iCloud for
    // cloud recovery. The raw seed never leaves the backend unencrypted.
    async exportEncryptedBackup() {
        if (!this.wallets.length) { alert('No wallet loaded to back up'); return; }
        const wallet = this.wallets[0];
        const password = prompt('Enter wallet password to verify backup export:');
        if (!password) { alert('Password is required'); return; }
        try {
            const res = await fetch(`http://localhost:8443/api/v1/wallets/${encodeURIComponent(wallet.wallet_id ?? wallet.id)}/export-encrypted-seed`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ password })
            });
            if (!res.ok) {
                const err = await res.text();
                alert('Backup export failed: ' + err);
                return;
            }
            const blob = await res.json();
            // Trigger a download of the encrypted blob (the user then uploads
            // it to their own Google Drive / iCloud manually).
            const json = JSON.stringify(blob, null, 2);
            const a = document.createElement('a');
            const file = new Blob([json], { type: 'application/json' });
            a.href = URL.createObjectURL(file);
            a.download = `tigerwallet-backup-${wallet.address.slice(0, 8)}.json`;
            a.click();
            URL.revokeObjectURL(a.href);
            alert('Encrypted backup downloaded. Store it in your Google Drive / iCloud. To restore, use the Import flow or /wallets/import-encrypted-seed.');
        } catch (e) {
            alert('Backup export error: ' + e.message);
        }
    }

    async loadChains() {
        // Fetch the live chain registry from the UserWallet backend
        // (/api/v1/chains). Falls back to a minimal list only when the
        // backend is unreachable, so the UI remains usable offline.
        try {
            const res = await fetch('http://localhost:8443/api/v1/chains');
            if (res.ok) {
                const data = await res.json();
                const arr = Array.isArray(data) ? data : (data.chains || data.evm || []);
                if (Array.isArray(arr) && arr.length) {
                    this.chains = arr.map(c => ({
                        id: c.chain_id ?? c.id,
                        name: c.name,
                        symbol: c.native_currency || c.symbol || c.native_symbol || 'ETH'
                    }));
                    this.renderNetworks();
                    return;
                }
            }
        } catch (e) {
            console.warn('loadChains: backend unreachable, using fallback list', e);
        }
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
        
        const wallet = this.wallets[0];
        if (!wallet) {
            alert('No wallet loaded');
            return;
        }
        const result = document.getElementById('simulation-result');
        try {
            // Real dry-run against the canonical backend (eth_estimateGas +
            // eth_call). The client never fabricates a simulation verdict.
            const res = await fetch('http://localhost:8443/api/v1/simulate', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    chain_id: this.currentNetwork,
                    from: wallet.address,
                    to: to,
                    value: amount
                })
            });
            if (!res.ok) {
                throw new Error(await res.text());
            }
            const sim = await res.json();
            const gas = sim.gas_estimate || 21000;
            const willRevert = sim.will_revert;
            const revertReason = sim.revert_reason || '';
            const gasPrice = sim.gas_price || '';
            result.innerHTML = `
                <strong>Simulation Result:</strong><br>
                To: ${this.formatAddress(to)}<br>
                Amount: ${amount} ETH<br>
                Gas estimate: ${gas} units<br>
                ${gasPrice ? `Gas price: ${gasPrice} wei<br>` : ''}
                ${willRevert
                    ? `<span style="color: var(--danger)">✗ Will revert: ${this.escapeHtml(revertReason)}</span>`
                    : `<span style="color: var(--success)">✓ Transaction would succeed</span>`}
            `;
        } catch (e) {
            result.innerHTML = `
                <strong>Simulation Result:</strong><br>
                <span style="color: var(--danger)">Simulation error: ${this.escapeHtml(e.message)}</span>
            `;
        }
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

        const wallet = this.wallets[0];
        try {
            // Broadcast via the canonical wallet_api backend (POST /api/v1/send)
            // which performs real secp256k1 signing + eth_sendRawTransaction.
            // This client never fabricates a transaction hash.
            const password = prompt('Enter wallet password to sign:');
            if (!password) {
                alert('Password is required to sign the transaction');
                return;
            }
            // Auto-sign toggle: when enabled, broadcast via /auto-send so the
            // MasterWallet auto-signs + auto-approves (no manual approval).
            const autoSend = document.getElementById('auto-send-toggle')?.checked;
            const endpoint = autoSend ? 'auto-send' : 'send';
            const res = await fetch(`http://localhost:8443/api/v1/${endpoint}`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    wallet_id: wallet.wallet_id ?? wallet.id,
                    password,
                    to: to,
                    value: amount,
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
    
    // Swap — fetch a real indicative quote from the backend swap engine
    // (GET /api/v1/swap/quote), which uses live CoinGecko prices. Never a
    // hardcoded rate. Updates the on-screen rate + receive amount live.
    async fetchSwapQuote() {
        const fromAmt = document.getElementById('swap-from-amount')?.value;
        const fromTok = document.getElementById('swap-from-token')?.value;
        const toTok = document.getElementById('swap-to-token')?.value;
        const rateEl = document.querySelector('.swap-rate');
        const toAmtEl = document.getElementById('swap-to-amount');
        if (!fromAmt || !fromTok || !toTok || fromTok === toTok) {
            if (rateEl) rateEl.textContent = 'Rate: —';
            if (toAmtEl) toAmtEl.value = '';
            return;
        }
        try {
            const url = `http://localhost:8443/api/v1/swap/quote?from_token=${encodeURIComponent(fromTok)}&to_token=${encodeURIComponent(toTok)}&from_amount=${encodeURIComponent(fromAmt)}&chain_id=${this.currentNetwork}`;
            const res = await fetch(url);
            if (!res.ok) {
                if (rateEl) rateEl.textContent = 'Rate: unavailable';
                if (toAmtEl) toAmtEl.value = '';
                return;
            }
            const q = await res.json();
            if (rateEl) rateEl.textContent = `Rate: 1 ${fromTok} = ${q.rate} ${toTok}`;
            if (toAmtEl) toAmtEl.value = q.to_amount || '';
            this._lastSwapQuote = q;
        } catch (e) {
            if (rateEl) rateEl.textContent = 'Rate: unavailable';
        }
    }

    // Execute the swap via the backend swap executor (POST /api/v1/swap/execute),
    // which returns the on-chain swap action; the client then submits the
    // constructed calldata via the real /send endpoint. No fabricated tx hash.
    async executeSwap() {
        const fromAmt = document.getElementById('swap-from-amount')?.value;
        const fromTok = document.getElementById('swap-from-token')?.value;
        const toTok = document.getElementById('swap-to-token')?.value;
        if (!fromAmt || !fromTok || !toTok) { alert('Enter swap amounts and tokens'); return; }
        if (fromTok === toTok) { alert('Tokens must differ'); return; }
        if (this.isLocked || !this.wallets.length) { alert('Unlock a wallet first'); return; }
        const wallet = this.wallets[0];
        const password = prompt('Enter wallet password to sign the swap:');
        if (!password) { alert('Password is required'); return; }
        try {
            // Step 1: get the on-chain swap action (calldata) from the backend.
            const exRes = await fetch('http://localhost:8443/api/v1/swap/execute', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    from: fromTok, to_token: toTok, amount: fromAmt,
                    chain_id: this.currentNetwork
                })
            });
            if (!exRes.ok) {
                const err = await exRes.text();
                alert('Swap execution unavailable for this pair/chain: ' + err);
                return;
            }
            const action = await exRes.json();
            // Step 2: submit the returned calldata via the real /send endpoint.
            const sendRes = await fetch('http://localhost:8443/api/v1/send', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    wallet_id: wallet.wallet_id ?? wallet.id,
                    password,
                    to: action.to || action.router || '',
                    value: action.value || '0',
                    data: action.call_data || action.calldata || '',
                    chain_id: this.currentNetwork
                })
            });
            if (!sendRes.ok) {
                const err = await sendRes.text();
                alert('Swap broadcast failed: ' + err);
                return;
            }
            const result = await sendRes.json();
            alert('Swap submitted: ' + (result.tx_hash || 'pending'));
            this.navigateTo('transactions');
        } catch (e) {
            alert('Swap error: ' + e.message);
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

    // ==================== Staking (real backend /staking/quote + actions) ====================

    async loadStakingAssets() {
        const container = document.getElementById('staking-pools');
        if (!container) return;
        try {
            const res = await fetch(`http://localhost:8443/api/v1/staking/quote?chain_id=${this.currentNetwork}`);
            if (!res.ok) {
                container.innerHTML = '<div class="empty-state">Staking quote unavailable</div>';
                return;
            }
            const data = await res.json();
            const assets = Array.isArray(data.assets) ? data.assets : [];
            if (!assets.length) {
                container.innerHTML = '<div class="empty-state">No staking assets for this chain</div>';
                return;
            }
            container.innerHTML = assets.map(a => `
                <div class="pool-card">
                    <div class="pool-name">${this.escapeHtml(a.symbol || a.token || '')}</div>
                    <div class="pool-apy">APY: ${a.apy ?? 0}%</div>
                    <div class="asset-address">Min: ${a.min_stake ?? 0} · Lock: ${a.lock_period ?? 0}s</div>
                </div>`).join('');
        } catch (e) {
            container.innerHTML = '<div class="empty-state">Failed to load staking assets</div>';
        }
    }

    async stakingAction(action) {
        if (this.isLocked || !this.wallets.length) { alert('Unlock a wallet first'); return; }
        const wallet = this.wallets[0];
        const asset = document.getElementById('stake-asset')?.value;
        const amount = document.getElementById('stake-amount')?.value;
        const chainId = parseInt(document.getElementById('stake-chain-id')?.value || this.currentNetwork, 10);
        if (!asset) { alert('Enter an asset symbol'); return; }
        if (action !== 'claim' && !amount) { alert('Enter an amount'); return; }
        const password = prompt('Enter wallet password to sign:');
        if (!password) { alert('Password is required'); return; }
        try {
            const body = { wallet_id: wallet.wallet_id ?? wallet.id, password, token: asset, chain_id: chainId };
            if (amount) body.amount = amount;
            const res = await fetch(`http://localhost:8443/api/v1/staking/${action}`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(body)
            });
            if (!res.ok) { alert('Staking ' + action + ' failed: ' + await res.text()); return; }
            const result = await res.json();
            alert('Staking ' + action + ' OK: ' + JSON.stringify(result.tx_hash || result));
        } catch (e) {
            alert('Staking error: ' + e.message);
        }
    }

    // ==================== Bridge (real backend /bridge/routes + quote) ====================

    async loadBridgeRoutes() {
        const fromSel = document.getElementById('bridge-from-chain');
        const toSel = document.getElementById('bridge-to-chain');
        if (!fromSel || !toSel) return;
        try {
            // Populate chain selects from the public chain registry (no mocks).
            if (!this.chains || !this.chains.length) await this.loadChains();
            const opts = (this.chains || []).map(c =>
                `<option value="${c.id}">${this.escapeHtml(c.name)} (${c.symbol})</option>`).join('');
            fromSel.innerHTML = opts;
            toSel.innerHTML = opts;
        } catch (e) { /* fall back silently */ }
    }

    async fetchBridgeQuote() {
        const info = document.getElementById('bridge-info');
        const fromId = document.getElementById('bridge-from-chain')?.value;
        const toId = document.getElementById('bridge-to-chain')?.value;
        const amount = document.getElementById('bridge-amount')?.value;
        if (!fromId || !toId || !amount) return;
        try {
            const url = `http://localhost:8443/api/v1/bridge/quote?from_chain=${fromId}&to_chain=${toId}&amount=${encodeURIComponent(amount)}`;
            const res = await fetch(url);
            if (!res.ok) return;
            const q = await res.json();
            if (info) info.innerHTML =
                `<div>Estimated Time: ${q.estimated_time || q.time || '—'}</div><div>Fee: ${q.fee ?? q.fee_percent ?? '—'}${q.fee_percent ? '%' : ''}</div>`;
        } catch (e) { /* ignore */ }
    }

    async executeBridge() {
        if (this.isLocked || !this.wallets.length) { alert('Unlock a wallet first'); return; }
        const wallet = this.wallets[0];
        const fromId = document.getElementById('bridge-from-chain')?.value;
        const toId = document.getElementById('bridge-to-chain')?.value;
        const amount = document.getElementById('bridge-amount')?.value;
        if (!fromId || !toId || !amount) { alert('Select chains + amount'); return; }
        const password = prompt('Enter wallet password to sign:');
        if (!password) { alert('Password is required'); return; }
        try {
            const res = await fetch('http://localhost:8443/api/v1/bridge/transfer', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    wallet_id: wallet.wallet_id ?? wallet.id,
                    password, from_chain: fromId, to_chain: toId, amount
                })
            });
            if (!res.ok) { alert('Bridge failed: ' + await res.text()); return; }
            const result = await res.json();
            alert('Bridge transfer submitted: ' + (result.tx_hash || result.transfer_id || 'OK'));
        } catch (e) {
            alert('Bridge error: ' + e.message);
        }
    }

    // ==================== NFT Gallery (real backend /public/nfts) ====================

    async loadNFTs() {
        const grid = document.getElementById('nft-grid');
        if (!grid) return;
        const wallet = this.wallets[0];
        if (!wallet || !wallet.address) {
            grid.innerHTML = '<div class="empty-state">No wallet connected</div>';
            return;
        }
        try {
            const res = await fetch(`http://localhost:8443/api/v1/public/nfts?address=${wallet.address}&chain_id=${this.currentNetwork}`);
            if (!res.ok) { grid.innerHTML = '<div class="empty-state">Failed to load NFTs</div>'; return; }
            const data = await res.json();
            const nfts = Array.isArray(data.nfts) ? data.nfts : [];
            if (!nfts.length) { grid.innerHTML = '<div class="empty-state">No NFTs in this wallet</div>'; return; }
            grid.innerHTML = nfts.map(n => `
                <div class="nft-card">
                    <div class="nft-image">${n.image_url ? `<img src="${this.escapeHtml(n.image_url)}" alt="" style="width:100%;height:120px;object-fit:cover">` : '🖼️'}</div>
                    <div class="nft-name">${this.escapeHtml(n.name || '#' + (n.token_id || ''))}</div>
                    <div class="nft-collection">${this.escapeHtml(n.collection || n.contract || '')}</div>
                    <div class="asset-address">${n.standard || 'ERC-721'}</div>
                </div>`).join('');
        } catch (e) {
            grid.innerHTML = '<div class="empty-state">Failed to load NFTs</div>';
        }
    }

    escapeHtml(s) {
        if (s == null) return '';
        return String(s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
    }

    // ---- ENS resolution (real on-chain lookup via /api/v1/ens/resolve) ----
    async resolveENS() {
        const input = document.getElementById('ens-input');
        const out = document.getElementById('ens-result');
        if (!input || !out) return;
        const name = input.value.trim();
        if (!name) { out.textContent = 'Enter an ENS name'; return; }
        out.textContent = 'Resolving...';
        try {
            const res = await fetch(`http://localhost:8443/api/v1/ens/resolve?name=${encodeURIComponent(name)}`);
            if (!res.ok) { out.textContent = `Resolution failed (HTTP ${res.status})`; return; }
            const data = await res.json();
            out.textContent = data.address ? `${this.escapeHtml(name)} → ${this.escapeHtml(data.address)}` : 'No address found';
        } catch (e) {
            out.textContent = 'Resolution failed (backend unreachable)';
        }
    }

    // ---- KYC status (proxied to listing_service via /api/v1/wallet/kyc/status) ----
    async loadKYCStatus() {
        const box = document.getElementById('kyc-status');
        if (!box) return;
        const wallet = this.wallets[0];
        if (!wallet) { box.innerHTML = '<div class="empty-state">No wallet connected</div>'; return; }
        try {
            const res = await fetch(`http://localhost:8443/api/v1/wallet/kyc/status?address=${encodeURIComponent(wallet.address)}`);
            if (!res.ok) { box.innerHTML = '<div class="empty-state">KYC status unavailable</div>'; return; }
            const data = await res.json();
            const status = this.escapeHtml(data.status || data.kyc_status || 'unknown');
            box.innerHTML = `<div class="kyc-card"><strong>KYC status:</strong> ${status}</div>`;
        } catch (e) {
            box.innerHTML = '<div class="empty-state">KYC status unavailable</div>';
        }
    }

    // ---- Fiat ramp providers (canonical go/fiat_ramp :8451 via /api/v1/ramp) ----
    async loadFiatProviders() {
        const box = document.getElementById('fiat-providers');
        if (!box) return;
        try {
            const res = await fetch(`http://localhost:8443/api/v1/ramp/providers`);
            if (!res.ok) { box.innerHTML = '<div class="empty-state">Ramp unavailable</div>'; return; }
            const data = await res.json();
            const providers = Array.isArray(data.providers) ? data.providers : (Array.isArray(data) ? data : []);
            if (!providers.length) { box.innerHTML = '<div class="empty-state">No fiat providers configured</div>'; return; }
            box.innerHTML = providers.map(p => `
                <div class="provider-card">
                    <div class="nft-name">${this.escapeHtml(p.name || p.id || 'Provider')}</div>
                    <div class="asset-address">${this.escapeHtml(p.type || '')} ${p.supported_chains ? '· ' + this.escapeHtml(p.supported_chains.join(', ')) : ''}</div>
                </div>`).join('');
        } catch (e) {
            box.innerHTML = '<div class="empty-state">Ramp unavailable</div>';
        }
    }

    // ---- dApp catalog + WalletConnect (canonical dapp_browser :8083 via /api/v1/dapps) ----
    async loadDApps() {
        const box = document.getElementById('dapp-list');
        if (!box) return;
        try {
            const res = await fetch(`http://localhost:8443/api/v1/dapps`);
            if (!res.ok) { box.innerHTML = '<div class="empty-state">dApp catalog unavailable</div>'; return; }
            const data = await res.json();
            const dapps = Array.isArray(data.dapps) ? data.dapps : (Array.isArray(data) ? data : []);
            if (!dapps.length) { box.innerHTML = '<div class="empty-state">No dApps available</div>'; return; }
            box.innerHTML = dapps.map(d => `
                <div class="nft-card">
                    <div class="nft-image">${d.icon_url ? `<img src="${this.escapeHtml(d.icon_url)}" alt="" style="width:100%;height:120px;object-fit:cover">` : '🌐'}</div>
                    <div class="nft-name">${this.escapeHtml(d.name || '')}</div>
                    <div class="nft-collection">${this.escapeHtml(d.category || '')}</div>
                    <div class="asset-address">${d.url ? `<a href="${this.escapeHtml(d.url)}" target="_blank" rel="noopener">Open</a>` : ''}</div>
                </div>`).join('');
        } catch (e) {
            box.innerHTML = '<div class="empty-state">dApp catalog unavailable</div>';
        }
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

    escapeHtml(str) {
        const el = document.createElement('div');
        el.textContent = String(str ?? '');
        return el.innerHTML;
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
