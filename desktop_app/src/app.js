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

        // Public live price feed (WebSocket /api/v1/ws) for the wallet page.
        this.connectLiveFeed();

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
        
        // Create wallet (from unlock screen)
        document.getElementById('create-wallet-btn')?.addEventListener('click', () => this.showOnboardingScreen());
        document.getElementById('import-wallet-btn')?.addEventListener('click', () => this.showOnboardingScreen());

        // Onboarding navigation
        document.getElementById('onboard-create-btn')?.addEventListener('click', () => this.showCreateWalletScreen());
        document.getElementById('onboard-import-btn')?.addEventListener('click', () => this.showImportWalletScreen());
        document.getElementById('onboard-back-btn')?.addEventListener('click', () => this.showLoginScreen());

        // Create wallet form
        document.getElementById('create-wallet-submit-btn')?.addEventListener('click', () => this.submitCreateWallet());
        document.getElementById('create-wallet-back-btn')?.addEventListener('click', () => this.showOnboardingScreen());

        // Import wallet form
        document.getElementById('import-wallet-submit-btn')?.addEventListener('click', () => this.submitImportWallet());
        document.getElementById('import-wallet-back-btn')?.addEventListener('click', () => this.showOnboardingScreen());

        // Backup screen
        document.getElementById('backup-copy-btn')?.addEventListener('click', () => this.copyBackupMnemonic());
        document.getElementById('backup-drive-btn')?.addEventListener('click', () => this.saveBackupToDrive());
        document.getElementById('backup-download-btn')?.addEventListener('click', () => this.downloadEncryptedBackup());
        document.getElementById('backup-confirm')?.addEventListener('change', (e) => {
            document.getElementById('backup-continue-btn').disabled = !e.target.checked;
        });
        document.getElementById('backup-continue-btn')?.addEventListener('click', () => this.confirmBackupAndContinue());

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
        document.getElementById('amm-quote-btn')?.addEventListener('click', () => this.fetchAmmQuote());
        document.getElementById('amm-swap-btn')?.addEventListener('click', () => this.executeAmmSwap());

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

        // New feature pages
        document.getElementById('trading-open-btn')?.addEventListener('click', () => this.openTradingPosition());
        document.getElementById('trading-market')?.addEventListener('change', () => this.loadTradingPositions());
        document.getElementById('launchpool-stake-btn')?.addEventListener('click', () => this.launchpoolAction('stake'));
        document.getElementById('launchpool-unstake-btn')?.addEventListener('click', () => this.launchpoolAction('unstake'));
        document.getElementById('p2p-order-btn')?.addEventListener('click', () => this.createP2POrder());
        document.getElementById('alert-create-btn')?.addEventListener('click', () => this.createPriceAlert());
        document.getElementById('contact-add-btn')?.addEventListener('click', () => this.addContact());
        document.getElementById('ens-resolve-btn')?.addEventListener('click', () => this.resolveENS());
        document.getElementById('nft-transfer-btn')?.addEventListener('click', () => this.transferNFT());
        document.getElementById('security-check-btn')?.addEventListener('click', () => this.securityCheck());
        document.getElementById('security-scan-btn')?.addEventListener('click', () => this.securityScan());
        document.getElementById('terminal-load-btn')?.addEventListener('click', () => { this.loadTerminalChart(); this.loadChartHistory(); });

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
        document.getElementById('settings-api-base-save')?.addEventListener('click', () => this.saveApiBase());

        // KYC
        document.getElementById('kyc-register-btn')?.addEventListener('click', () => this.kycRegister());
        document.getElementById('kyc-submit-btn')?.addEventListener('click', () => this.kycSubmit());
        document.getElementById('kyc-upload-btn')?.addEventListener('click', () => this.kycUploadDocument());

        // Multisig
        document.getElementById('ms-create-btn')?.addEventListener('click', () => this.createMultisigWallet());
        document.getElementById('ms-tx-create-btn')?.addEventListener('click', () => this.createMultisigTx());
        document.getElementById('ms-tx-refresh-btn')?.addEventListener('click', () => this.loadMultisigTxs());

        // Non-EVM
        document.getElementById('nevm-derive-btn')?.addEventListener('click', () => this.deriveNonEvmAddress());
        document.getElementById('nevm-send-btn')?.addEventListener('click', () => this.sendNonEvm());

        // Lending + prediction
        document.getElementById('lending-action-btn')?.addEventListener('click', () => this.lendingAction());
        document.getElementById('prediction-bet-btn')?.addEventListener('click', () => this.placePrediction());

        // Keystore
        document.getElementById('keystore-export-btn')?.addEventListener('click', () => this.exportKeystore());
        document.getElementById('keystore-import-btn')?.addEventListener('click', () => this.importKeystore());

        // Hardware wallet
        document.getElementById('hw-detect-btn')?.addEventListener('click', () => this.hwDetect());
        document.getElementById('hw-address-btn')?.addEventListener('click', () => this.hwGetAddress());
        document.getElementById('hw-sign-btn')?.addEventListener('click', () => this.hwSignMessage());
        document.getElementById('hw-tx-sign-btn')?.addEventListener('click', () => this.hwSignTransaction());

        // Copy trading (trading page)
        document.getElementById('copy-follow-btn')?.addEventListener('click', () => this.followTrader());

        // Watch-only wallet
        document.getElementById('watch-only-add-btn')?.addEventListener('click', () => this.addWatchOnlyWallet());

        // WalletConnect pairing (dApps page)
        document.getElementById('wc-pair-btn')?.addEventListener('click', () => this.pairWalletConnect());

        // KYC session detail refresh
        document.getElementById('kyc-session-btn')?.addEventListener('click', () => this.loadKycSession());

        // Passkey wallet registration (WebAuthn)
        document.getElementById('passkey-register-btn')?.addEventListener('click', () => this.registerPasskeyWallet());

        // Health badge: poll backend health every 30s
        this.updateHealthBadge();
        setInterval(() => this.updateHealthBadge(), 30000);
    }
    
    async checkWalletStatus() {
        // No mandatory registration: if a wallet exists locally, show the
        // password unlock screen; otherwise show onboarding (create/import).
        const walletData = localStorage.getItem('tigerwallet-master');
        if (walletData && this.wallets.length > 0) {
            this.showLoginScreen();
        } else {
            this.showOnboardingScreen();
        }
    }

    showLoginScreen() {
        document.getElementById('login-screen').classList.remove('hidden');
        document.getElementById('onboarding-screen').classList.add('hidden');
        document.getElementById('dashboard-screen').classList.add('hidden');
        document.getElementById('create-wallet-screen').classList.add('hidden');
        document.getElementById('import-wallet-screen').classList.add('hidden');
        document.getElementById('backup-screen').classList.add('hidden');
    }

    showOnboardingScreen() {
        document.getElementById('login-screen').classList.add('hidden');
        document.getElementById('onboarding-screen').classList.remove('hidden');
        document.getElementById('dashboard-screen').classList.add('hidden');
        document.getElementById('create-wallet-screen').classList.add('hidden');
        document.getElementById('import-wallet-screen').classList.add('hidden');
        document.getElementById('backup-screen').classList.add('hidden');
    }

    showCreateWalletScreen() {
        document.getElementById('onboarding-screen').classList.add('hidden');
        document.getElementById('create-wallet-screen').classList.remove('hidden');
    }

    showImportWalletScreen() {
        document.getElementById('onboarding-screen').classList.add('hidden');
        document.getElementById('import-wallet-screen').classList.remove('hidden');
    }

    showBackupScreen(mnemonic, wallet) {
        document.getElementById('create-wallet-screen').classList.add('hidden');
        document.getElementById('backup-screen').classList.remove('hidden');
        const grid = document.getElementById('backup-mnemonic-display');
        grid.innerHTML = '';
        const words = mnemonic.split(' ');
        words.forEach((word, i) => {
            const div = document.createElement('div');
            div.className = 'mnemonic-word';
            div.textContent = (i + 1) + '. ' + word;
            grid.appendChild(div);
        });
        this._pendingBackup = { mnemonic: mnemonic, wallet: wallet };
        document.getElementById('backup-confirm').checked = false;
        document.getElementById('backup-continue-btn').disabled = true;
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
            defi: 'DeFi',
            trading: 'Trading',
            launchpool: 'Launchpool',
            'token-sales': 'Token Sales',
            p2p: 'P2P Trading',
            cards: 'Crypto Card',
            'price-alerts': 'Price Alerts',
            dao: 'DAO Governance',
            ens: 'ENS',
            security: 'Security Center',
            terminal: 'Trading Terminal',
            ramp: 'Fiat Ramp',
            dapps: 'dApps',
            approvals: 'Token Approvals',
            'address-book': 'Address Book',
            devices: 'Linked Devices',
            kyc: 'KYC Verification',
            multisig: 'Multisig Wallets',
            'non-evm': 'Non-EVM Chains',
            lending: 'Lending',
            prediction: 'Prediction Markets',
            keystore: 'Keystore',
            'hardware-wallet': 'Hardware Wallet',
            transactions: 'Transactions',
            fees: 'Fees',
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
        } else if (page === 'terminal') {
            this.loadTerminalChart();
            this.loadChartHistory();
        } else if (page === 'kyc') {
            this.loadKYCStatus();
        } else if (page === 'ramp') {
            this.loadFiatProviders();
        } else if (page === 'dapps') {
            this.loadDappCategories();
            this.loadDApps();
            this.loadWcPairings();
            this.loadWcSessions();
        } else if (page === 'fees') {
            this.loadFees();
        } else if (page === 'defi') {
            this.loadDefiProtocols();
        } else if (page === 'trading') {
            this.loadTradingPositions();
            this.loadFuturesPairs();
            this.loadCopyTraders();
        } else if (page === 'launchpool') {
            this.loadLaunchpool();
        } else if (page === 'token-sales') {
            this.loadTokenSales();
        } else if (page === 'p2p') {
            this.loadP2PAdverts();
        } else if (page === 'cards') {
            this.loadCardInfo();
        } else if (page === 'price-alerts') {
            this.loadPriceAlerts();
        } else if (page === 'dao') {
            this.loadDao();
        } else if (page === 'approvals') {
            this.loadApprovals();
        } else if (page === 'address-book') {
            this.loadContacts();
        } else if (page === 'devices') {
            this.loadDevices();
        } else if (page === 'multisig') {
            this.loadMultisigWallets();
        } else if (page === 'lending') {
            this.loadLending();
        } else if (page === 'prediction') {
            this.loadPredictionMarkets();
        } else if (page === 'keystore') {
            this.populateWalletSelects();
        } else if (page === 'settings') {
            const input = document.getElementById('settings-api-base');
            if (input) input.value = twApiOrigin();
            this.loadNetworkStatus();
            this.checkReadiness();
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
    
    // Show the create-wallet form (replaces prompt-based flow).
    showCreateWallet() {
        this.showCreateWalletScreen();
    }

    async submitCreateWallet() {
        const name = document.getElementById('create-wallet-name').value.trim() || 'Main Wallet';
        const password = document.getElementById('create-wallet-password').value;
        const password2 = document.getElementById('create-wallet-password2').value;

        if (!password || password.length < 8) {
            alert('Password must be at least 8 characters');
            return;
        }
        if (password !== password2) {
            alert('Passwords do not match');
            return;
        }

        try {
            // Create a real wallet on the canonical wallet_api backend
            // (POST /api/v1/wallets) — the backend derives the address from a
            // real BIP-39 seed (secp256k1 / BIP-44). Never fabricate one here.
            const res = await twFetch(`${twApiBase()}/wallets`, {
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

            // Show backup screen with the mnemonic so the user can copy /
            // save to Google Drive before continuing.
            this.showBackupScreen(wallet.mnemonic, wallet);
        } catch (e) {
            alert('Wallet creation error: ' + e.message);
        }
    }
    
    // Show the import-wallet form (replaces prompt-based flow).
    showImportWallet() {
        this.showImportWalletScreen();
    }

    async submitImportWallet() {
        const mnemonic = document.getElementById('import-wallet-mnemonic').value.trim();
        if (!mnemonic || mnemonic.split(/\s+/).length < 12) {
            alert('A valid 12/24-word recovery phrase is required');
            return;
        }
        const password = document.getElementById('import-wallet-password').value;
        if (!password || password.length < 8) {
            alert('Password must be at least 8 characters');
            return;
        }
        const name = document.getElementById('import-wallet-name').value.trim() || 'Imported Wallet';
        try {
            const res = await twFetch(`${twApiBase()}/wallets`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ label: name, password, mnemonic })
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
            const res = await twFetch(`${twApiBase()}/wallets/${encodeURIComponent(wallet.wallet_id ?? wallet.id)}/export-encrypted-seed`, {
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
            const res = await twFetch(`${twApiBase()}/chains`);
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
            const res = await twFetch(`${twApiBase()}/public/balance?address=${wallet.address}&chain_id=${chainId}`
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
            const res = await twFetch(`${twApiBase()}/simulate`, {
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
            const res = await twFetch(`${twApiBase()}/${endpoint}`, {
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
    
    // ---- On-chain AMM (Uniswap V2-style routers, /amm/quote + /amm/swap) ----
    // Quote is a real eth_call getAmountsOut; swap returns calldata that is
    // broadcast via the real /send endpoint. No fabricated tx hash.
    async fetchAmmQuote() {
        const tokenIn = document.getElementById('amm-token-in')?.value?.trim();
        const tokenOut = document.getElementById('amm-token-out')?.value?.trim();
        const amountIn = document.getElementById('amm-amount-in')?.value?.trim();
        const out = document.getElementById('amm-quote-result');
        if (!tokenIn || !tokenOut || !amountIn) {
            if (out) out.textContent = 'Enter token in/out addresses and amount';
            return;
        }
        try {
            const res = await twFetch(`${twApiBase()}/amm/quote?chain_id=${this.currentNetwork}&token_in=${encodeURIComponent(tokenIn)}&token_out=${encodeURIComponent(tokenOut)}&amount_in=${encodeURIComponent(amountIn)}`);
            if (!res.ok) {
                const err = await res.json().catch(() => ({}));
                if (out) out.textContent = 'Quote unavailable: ' + (err.error || `HTTP ${res.status}`);
                return;
            }
            const q = await res.json();
            this._lastAmmQuote = q;
            if (out) out.textContent = `${amountIn} in → ${q.amount_out || '?'} out (router ${q.router || '?'})`;
        } catch (e) {
            if (out) out.textContent = 'Quote unavailable: ' + e.message;
        }
    }

    async executeAmmSwap() {
        const tokenIn = document.getElementById('amm-token-in')?.value?.trim();
        const tokenOut = document.getElementById('amm-token-out')?.value?.trim();
        const amountIn = document.getElementById('amm-amount-in')?.value?.trim();
        if (!tokenIn || !tokenOut || !amountIn) { alert('Enter token in/out addresses and amount'); return; }
        if (this.isLocked || !this.wallets.length) { alert('Unlock a wallet first'); return; }
        const wallet = this.wallets[0];
        const from = wallet.address;
        const password = prompt('Enter wallet password to sign the AMM swap:');
        if (!password) { alert('Password is required'); return; }
        try {
            const exRes = await twFetch(`${twApiBase()}/amm/swap`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    from, chain_id: this.currentNetwork,
                    token_in: tokenIn, token_out: tokenOut,
                    amount_in: amountIn, amount_out_min: ''
                })
            });
            if (!exRes.ok) {
                const err = await exRes.json().catch(() => ({}));
                alert('AMM swap construction failed: ' + (err.error || `HTTP ${exRes.status}`));
                return;
            }
            const action = await exRes.json();
            const tx = action.tx || {};
            const sendRes = await twFetch(`${twApiBase()}/send`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    wallet_id: wallet.wallet_id ?? wallet.id,
                    password,
                    to: tx.to || action.router || '',
                    value: tx.value || '0',
                    chain_id: this.currentNetwork,
                    data: tx.data || ''
                })
            });
            if (!sendRes.ok) {
                const err = await sendRes.json().catch(() => ({}));
                alert('Broadcast failed: ' + (err.error || `HTTP ${sendRes.status}`));
                return;
            }
            const sent = await sendRes.json();
            alert('AMM swap submitted to the blockchain network: ' + (sent.tx_hash || ''));
        } catch (e) {
            alert('AMM swap failed: ' + e.message);
        }
    }

    async fetchSwapQuote() {
        // GET /api/v1/swap/quote — real indicative quote (live CoinGecko
        // prices server-side). Never a hardcoded rate.
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
            const url = `${twApiBase()}/swap/quote?from_token=${encodeURIComponent(fromTok)}&to_token=${encodeURIComponent(toTok)}&from_amount=${encodeURIComponent(fromAmt)}&chain_id=${this.currentNetwork}`;
            const res = await twFetch(url);
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
            const exRes = await twFetch(`${twApiBase()}/swap/execute`, {
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
            const sendRes = await twFetch(`${twApiBase()}/send`, {
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
            const res = await twFetch(`${twApiBase()}/public/transactions?address=${wallet.address}&chain_id=${chainId}`
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
            <div class="asset-item tx-row" data-txhash="${this.escapeHtml(tx.hash || '')}" style="cursor: pointer" title="Tap to check on-chain status">
                <div class="asset-icon">${tx.value > 0 ? '📤' : '📥'}</div>
                <div class="asset-info">
                    <div class="asset-name">${tx.token} Transfer</div>
                    <div class="asset-address">${this.formatAddress(tx.from)} → ${this.formatAddress(tx.to)}</div>
                    <div class="asset-address tx-receipt-status" data-txreceipt="${this.escapeHtml(tx.hash || '')}"></div>
                </div>
                <div class="asset-balance">
                    <div class="balance" style="color: ${tx.value > 0 ? 'var(--danger)' : 'var(--success)'}">
                        ${tx.value > 0 ? '-' : '+'}${tx.value} ${tx.token}
                    </div>
                    <div class="usd-value">${new Date(tx.timestamp).toLocaleDateString()}</div>
                </div>
            </div>
        `).join('');

        list.querySelectorAll('.tx-row').forEach(row => {
            row.addEventListener('click', () => this.showTransactionReceipt(row.dataset.txhash));
        });
    }

    // GET /transactions/:txHash?chain_id=N -> real on-chain receipt via the
    // chain explorer (status / block / confirmations). Fail-closed.
    async showTransactionReceipt(txHash) {
        if (!txHash) return;
        const slot = document.querySelector(`[data-txreceipt="${txHash}"]`);
        if (!slot) return;
        slot.textContent = 'Checking on-chain status…';
        try {
            const res = await twFetch(`${twApiBase()}/transactions/${encodeURIComponent(txHash)}?chain_id=${this.currentNetwork || 1}`);
            if (!res.ok) {
                slot.textContent = 'Receipt unavailable';
                return;
            }
            const data = await res.json();
            const info = data.result || data;
            const status = info.status || info.txreceipt_status || info.result?.status;
            const block = info.blockNumber || info.block_number;
            if (status === '0x1' || status === '1') {
                slot.textContent = `Confirmed on-chain${block ? ` in block ${parseInt(block, 16) || block}` : ''}`;
                slot.style.color = 'var(--success)';
            } else if (status === '0x0' || status === '0') {
                slot.textContent = 'Failed on-chain';
                slot.style.color = 'var(--danger)';
            } else {
                slot.textContent = 'Pending confirmation';
            }
        } catch (e) {
            slot.textContent = 'Receipt unavailable';
        }
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
            const res = await twFetch(`${twApiBase()}/staking/quote?chain_id=${this.currentNetwork}`);
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
            const res = await twFetch(`${twApiBase()}/staking/${action}`, {
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
            const url = `${twApiBase()}/bridge/quote?from_chain=${fromId}&to_chain=${toId}&amount=${encodeURIComponent(amount)}`;
            const res = await twFetch(url);
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
            const res = await twFetch(`${twApiBase()}/bridge/transfer`, {
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
            const res = await twFetch(`${twApiBase()}/public/nfts?address=${wallet.address}&chain_id=${this.currentNetwork}`);
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
            const res = await twFetch(`${twApiBase()}/ens/resolve?name=${encodeURIComponent(name)}`);
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
            const res = await twFetch(`${twApiBase()}/wallet/kyc/status?address=${encodeURIComponent(wallet.address)}`);
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
            const res = await twFetch(`${twApiBase()}/ramp/providers`);
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
    async loadDApps(category) {
        const box = document.getElementById('dapp-list');
        if (!box) return;
        if (category !== undefined) this._dappCategory = category;
        try {
            const qs = this._dappCategory ? `?category=${encodeURIComponent(this._dappCategory)}` : '';
            const res = await twFetch(`${twApiBase()}/dapps${qs}`);
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

    // ---- dApp categories (GET /api/v1/dapps/categories) — filter chips ----
    async loadDappCategories() {
        const box = document.getElementById('dapp-categories');
        if (!box) return;
        try {
            const res = await twFetch(`${twApiBase()}/dapps/categories`);
            if (!res.ok) { box.innerHTML = ''; return; }
            const data = await res.json();
            const cats = Array.isArray(data.categories) ? data.categories : (Array.isArray(data) ? data : []);
            const all = [''].concat(cats);
            box.innerHTML = all.map(c =>
                `<button class="category-chip" data-cat="${this.escapeHtml(c)}" style="margin-right:6px;padding:4px 12px;border-radius:12px;border:1px solid var(--border);background:${(this._dappCategory || '') === c ? 'var(--accent)' : 'transparent'};color:var(--text);cursor:pointer">${c === '' ? 'All' : this.escapeHtml(c)}</button>`
            ).join('');
            box.querySelectorAll('.category-chip').forEach(chip => {
                chip.addEventListener('click', () => this.loadDApps(chip.dataset.cat || ''));
            });
        } catch (e) {
            box.innerHTML = '';
        }
    }

    // ---- WalletConnect pairing (/dapp/* proxied to dapp_browser :8083) ----
    async pairWalletConnect() {
        const input = document.getElementById('wc-uri');
        const uri = (input?.value || '').trim();
        if (!uri) { alert('Paste a WalletConnect URI (wc:…)'); return; }
        try {
            const res = await twFetch(`${twApiBase()}/dapp/pairings`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ uri })
            });
            if (!res.ok) throw new Error(`HTTP ${res.status}`);
            input.value = '';
            this.loadWcPairings();
        } catch (e) {
            alert('Pairing failed: ' + e.message);
        }
    }

    async loadWcPairings() {
        const box = document.getElementById('wc-pairings');
        if (!box) return;
        try {
            const res = await twFetch(`${twApiBase()}/dapp/pairings`);
            if (!res.ok) { box.innerHTML = ''; return; }
            const data = await res.json();
            const list = data.pairings || data.data || [];
            if (!list.length) { box.innerHTML = '<div class="empty-state">No pending pairings</div>'; return; }
            box.innerHTML = list.map(p => {
                const topic = this.escapeHtml(p.topic || '');
                const name = this.escapeHtml(p.peer_name || p.name || topic);
                const status = this.escapeHtml(p.status || 'pending');
                return `<div class="asset-item" style="display:flex;justify-content:space-between;align-items:center;padding:8px 0">
                    <span>${name} · ${status}</span>
                    <span>
                        <button class="btn-primary wc-approve" data-topic="${topic}" style="margin-right:6px">Approve</button>
                        <button class="btn-secondary wc-reject" data-topic="${topic}">Reject</button>
                    </span>
                </div>`;
            }).join('');
            box.querySelectorAll('.wc-approve').forEach(b => b.addEventListener('click', () => this.wcPairingAction(b.dataset.topic, true)));
            box.querySelectorAll('.wc-reject').forEach(b => b.addEventListener('click', () => this.wcPairingAction(b.dataset.topic, false)));
        } catch (e) {
            box.innerHTML = '';
        }
    }

    async loadWcSessions() {
        const box = document.getElementById('wc-sessions');
        if (!box) return;
        try {
            const res = await twFetch(`${twApiBase()}/dapp/sessions`);
            if (!res.ok) { box.innerHTML = ''; return; }
            const data = await res.json();
            const list = data.sessions || data.data || [];
            if (!list.length) { box.innerHTML = '<div class="empty-state">No active sessions</div>'; return; }
            box.innerHTML = list.map(s =>
                `<div class="asset-item" style="padding:8px 0">${this.escapeHtml(s.peer_name || s.name || '')} · ${this.escapeHtml(s.topic || '')}</div>`
            ).join('');
        } catch (e) {
            box.innerHTML = '';
        }
    }

    async wcPairingAction(topic, approve) {
        try {
            const res = await twFetch(`${twApiBase()}/dapp/pairings/${encodeURIComponent(topic)}/${approve ? 'approve' : 'reject'}`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: '{}'
            });
            if (!res.ok) throw new Error(`HTTP ${res.status}`);
            this.loadWcPairings();
            this.loadWcSessions();
        } catch (e) {
            alert('Action failed: ' + e.message);
        }
    }

    // ---- Fees (public fee transparency) ----
    async loadFees() {
        const tiersBox = document.getElementById('fees-tiers');
        const txBox = document.getElementById('fees-transactions');
        try {
            const res = await twFetch(`${twApiBase()}/public/fees`);
            if (tiersBox) {
                if (!res.ok) { tiersBox.innerHTML = '<div class="empty-state">Fee schedule unavailable</div>'; }
                else {
                    const data = await res.json();
                    const tiers = data.fees || data.data || [];
                    tiersBox.innerHTML = tiers.length
                        ? tiers.map(t => `<div class="asset-item" style="padding:8px 0">${this.escapeHtml(t.name || t.tier || '')} — ${this.escapeHtml(String(t.rate_bps ?? t.rate ?? ''))} bps</div>`).join('')
                        : '<div class="empty-state">No fee tiers configured</div>';
                }
            }
        } catch (e) {
            if (tiersBox) tiersBox.innerHTML = '<div class="empty-state">Fee schedule unavailable</div>';
        }
        try {
            const res = await twFetch(`${twApiBase()}/public/fees/transactions`);
            if (txBox) {
                if (!res.ok) { txBox.innerHTML = '<div class="empty-state">No settled fee transactions</div>'; }
                else {
                    const data = await res.json();
                    const txs = data.transactions || data.data || [];
                    txBox.innerHTML = txs.length
                        ? txs.slice(0, 25).map(t => `<div class="asset-item" style="padding:8px 0">${this.escapeHtml(String(t.tx_hash || '')).slice(0, 18)}… ${this.escapeHtml(String(t.fee_amount ?? t.amount ?? ''))} ${this.escapeHtml(t.token || '')}</div>`).join('')
                        : '<div class="empty-state">No settled fee transactions</div>';
                }
            }
        } catch (e) {
            if (txBox) txBox.innerHTML = '<div class="empty-state">No settled fee transactions</div>';
        }
    }

    // ---- Network status (GET /api/v1/network-status) + readiness probe ----
    async loadNetworkStatus() {
        const box = document.getElementById('network-status');
        if (!box) return;
        try {
            const res = await twFetch(`${twApiBase()}/network-status`);
            if (!res.ok) { box.innerHTML = '<div class="empty-state">Network status unavailable</div>'; return; }
            const data = await res.json();
            const bn = data.block_number != null ? String(data.block_number) : (data.note || '?');
            box.innerHTML = `<div class="asset-item" style="padding:8px 0">Chain ${this.escapeHtml(String(data.chain_id ?? ''))} — latest block: ${this.escapeHtml(bn)}${data.latency_ms != null ? ' · ' + this.escapeHtml(String(data.latency_ms)) + 'ms' : ''}</div>`;
        } catch (e) {
            box.innerHTML = '<div class="empty-state">Network status unavailable</div>';
        }
    }

    async checkReadiness() {
        const el = document.getElementById('backend-readiness');
        if (!el) return;
        try {
            const res = await fetch(`${twApiBase()}/health/ready`);
            const ok = res.ok;
            el.textContent = ok ? 'ready' : 'degraded';
            el.style.background = ok ? 'var(--success, #16a34a)' : 'var(--danger, #dc2626)';
        } catch (e) {
            el.textContent = 'unreachable';
            el.style.background = 'var(--danger, #dc2626)';
        }
    }

    // ---- KYC session detail (GET /kyc/session/:id) ----
    async loadKycSession() {
        const idInput = document.getElementById('kyc-session-id');
        const out = document.getElementById('kyc-result');
        const id = (idInput?.value || '').trim();
        if (!id) { alert('Enter a KYC session ID'); return; }
        try {
            const res = await twFetch(`${twApiBase()}/kyc/session/${encodeURIComponent(id)}`);
            if (!out) return;
            if (!res.ok) { out.innerHTML = '<div class="empty-state">Session not found</div>'; return; }
            const data = await res.json();
            out.innerHTML = `<div class="kyc-card"><strong>Session ${this.escapeHtml(id)}</strong><br>Status: ${this.escapeHtml(data.status || 'unknown')}${data.reviewed_at ? '<br>Reviewed: ' + this.escapeHtml(data.reviewed_at) : ''}</div>`;
        } catch (e) {
            if (out) out.innerHTML = '<div class="empty-state">Session unavailable</div>';
        }
    }

    // ---- Passkey wallet registration (POST /passkey/wallet via WebAuthn) ----
    async registerPasskeyWallet() {
        const out = document.getElementById('passkey-result');
        if (!window.PublicKeyCredential || !navigator.credentials) {
            if (out) out.innerHTML = '<div class="empty-state">Passkeys are not supported in this environment</div>';
            return;
        }
        try {
            const challenge = new Uint8Array(32);
            crypto.getRandomValues(challenge);
            const credential = await navigator.credentials.create({
                publicKey: {
                    challenge,
                    rp: { name: 'TigerWallet' },
                    user: {
                        id: new Uint8Array(16),
                        name: 'tigerwallet-user',
                        displayName: 'TigerWallet User'
                    },
                    pubKeyCredParams: [
                        { type: 'public-key', alg: -7 },
                        { type: 'public-key', alg: -257 }
                    ],
                    authenticatorSelection: {
                        authenticatorAttachment: 'platform',
                        residentKey: 'preferred',
                        userVerification: 'preferred'
                    },
                    timeout: 60000,
                    attestation: 'none'
                }
            });
            if (!credential || credential.type !== 'public-key') throw new Error('passkey creation failed');
            const b64u = (buf) => {
                const bytes = new Uint8Array(buf);
                let s = '';
                for (let i = 0; i < bytes.length; i++) s += String.fromCharCode(bytes[i]);
                return btoa(s).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
            };
            let publicKey = '';
            if (typeof credential.response.getPublicKey === 'function') {
                const spki = credential.response.getPublicKey();
                if (spki) publicKey = b64u(spki);
            }
            const res = await twFetch(`${twApiBase()}/passkey/wallet`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    label: 'Passkey Wallet',
                    chain_id: 1,
                    credential_id: b64u(credential.rawId),
                    public_key: publicKey,
                    sign_count: 0
                })
            });
            if (!res.ok) {
                const err = await res.json().catch(() => ({}));
                throw new Error(err.error || `HTTP ${res.status}`);
            }
            const data = await res.json();
            if (out) out.innerHTML = `<div class="success-banner">Passkey wallet created: ${this.escapeHtml(data.address || data.wallet_id || '')}</div>`;
            this.loadWalletData();
        } catch (e) {
            if (out) out.innerHTML = `<div class="empty-state">Passkey registration failed: ${this.escapeHtml(e.message)}</div>`;
        }
    }

    // ---- Price history (GET /api/v1/chart/history?coin=&days=) — line chart ----
    async loadChartHistory() {
        const canvas = document.getElementById('chart-history-canvas');
        if (!canvas) return;
        const symbolInput = document.getElementById('terminal-symbol');
        const daysInput = document.getElementById('terminal-days');
        const symbol = ((symbolInput?.value || 'ETH').trim() || 'ETH').toLowerCase();
        const coinMap = { btc: 'bitcoin', eth: 'ethereum', bnb: 'binancecoin', sol: 'solana', matic: 'matic-network', avax: 'avalanche-2', usdt: 'tether', usdc: 'usd-coin' };
        const coin = coinMap[symbol] || symbol;
        const days = daysInput?.value || '30';
        try {
            const res = await twFetch(`${twApiBase()}/chart/history?coin=${encodeURIComponent(coin)}&days=${encodeURIComponent(days)}`);
            if (!res.ok) { this.drawLineChart(canvas, []); return; }
            const data = await res.json();
            const candles = data.candles || [];
            const points = candles.map(c => Array.isArray(c) ? +c[1] : +(c.price ?? c.close ?? c.c)).filter(v => isFinite(v));
            this.drawLineChart(canvas, points);
        } catch (e) {
            this.drawLineChart(canvas, []);
        }
    }

    drawLineChart(canvas, points) {
        const ctx = canvas.getContext('2d');
        if (!ctx) return;
        const w = (canvas.width = canvas.offsetWidth || 800);
        const h = (canvas.height = 200);
        const muted = getComputedStyle(canvas).getPropertyValue('--text-secondary') || '#8b949e';
        ctx.clearRect(0, 0, w, h);
        if (!points.length) {
            ctx.fillStyle = muted;
            ctx.font = '13px sans-serif';
            ctx.fillText('No price history for this asset/range.', 16, 30);
            return;
        }
        const padX = 50;
        let min = Infinity, max = -Infinity;
        for (const v of points) { if (v < min) min = v; if (v > max) max = v; }
        const span = max - min || 1;
        const x = (i) => padX + (i / (points.length - 1 || 1)) * (w - padX - 10);
        const y = (v) => h - ((v - min) / span) * (h - 24) - 12;
        ctx.strokeStyle = getComputedStyle(canvas).getPropertyValue('--accent') || '#4f8cff';
        ctx.lineWidth = 2;
        ctx.beginPath();
        points.forEach((v, i) => { if (i === 0) ctx.moveTo(x(i), y(v)); else ctx.lineTo(x(i), y(v)); });
        ctx.stroke();
        ctx.fillStyle = muted;
        ctx.font = '11px sans-serif';
        ctx.fillText(max.toFixed(2), 4, y(max) + 4);
        ctx.fillText(min.toFixed(2), 4, y(min) + 4);
    }

    // ---- DeFi protocols (/api/v1/defi/protocols) ----
    async loadDefiProtocols() {
        const box = document.getElementById('defi-protocols');
        if (!box) return;
        try {
            const res = await twFetch(`${twApiBase()}/defi/protocols`);
            if (!res.ok) { box.innerHTML = '<div class="empty-state">DeFi protocols unavailable</div>'; return; }
            const data = await res.json();
            const list = Array.isArray(data.protocols) ? data.protocols : (Array.isArray(data) ? data : []);
            if (!list.length) { box.innerHTML = '<div class="empty-state">No DeFi protocols available</div>'; return; }
            box.innerHTML = list.map(p => `
                <div class="provider-card">
                    <div class="nft-name">${this.escapeHtml(p.name || p.protocol || '')}</div>
                    <div class="asset-address">${this.escapeHtml(p.category || p.chain || '')}${p.tvl != null ? ' · TVL: ' + this.escapeHtml(p.tvl) : ''}${p.apy != null ? ' · APY: ' + this.escapeHtml(p.apy) + '%' : ''}</div>
                </div>`).join('');
        } catch (e) {
            box.innerHTML = '<div class="empty-state">DeFi protocols unavailable</div>';
        }
    }

    // ---- Trading: perpetual + margin positions (/api/v1/{perpetual,margin}/positions) ----
    tradingMarket() {
        return document.getElementById('trading-market')?.value === 'margin' ? 'margin' : 'perpetual';
    }

    async loadTradingPositions() {
        const box = document.getElementById('trading-positions');
        if (!box) return;
        const market = this.tradingMarket();
        try {
            const res = await twFetch(`${twApiBase()}/${market}/positions`);
            if (!res.ok) { box.innerHTML = '<div class="empty-state">Positions unavailable</div>'; return; }
            const data = await res.json();
            const list = Array.isArray(data.positions) ? data.positions : (Array.isArray(data) ? data : []);
            if (!list.length) { box.innerHTML = '<div class="empty-state">No open positions</div>'; return; }
            box.innerHTML = list.map(p => {
                const id = this.escapeHtml(p.id ?? p.position_id ?? '');
                return `
                <div class="provider-card">
                    <div class="nft-name">${this.escapeHtml(p.pair || '')} ${this.escapeHtml(p.side || '')}</div>
                    <div class="asset-address">size=${this.escapeHtml(p.size ?? '')} lev=${this.escapeHtml(p.leverage ?? '')} pnl=${this.escapeHtml(p.pnl ?? p.unrealized_pnl ?? '0')}</div>
                    <button class="btn-secondary" onclick="window.app.closeTradingPosition('${id}')">Close</button>
                </div>`;
            }).join('');
        } catch (e) {
            box.innerHTML = '<div class="empty-state">Positions unavailable</div>';
        }
    }

    async openTradingPosition() {
        if (this.isLocked || !this.wallets.length) { alert('Unlock a wallet first'); return; }
        const market = this.tradingMarket();
        const pair = document.getElementById('trading-pair')?.value;
        const side = document.getElementById('trading-side')?.value;
        const size = document.getElementById('trading-size')?.value;
        const leverage = parseInt(document.getElementById('trading-leverage')?.value || '1', 10);
        if (!pair || !size) { alert('Enter a pair and size'); return; }
        try {
            const res = await twFetch(`${twApiBase()}/${market}/positions`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ pair, side, size, leverage, chain_id: this.currentNetwork })
            });
            if (!res.ok) { alert('Open position failed: ' + await res.text()); return; }
            const result = await res.json();
            alert('Position submitted to the blockchain network: ' + JSON.stringify(result.id || result.position_id || result));
            this.loadTradingPositions();
        } catch (e) {
            alert('Trading error: ' + e.message);
        }
    }

    async closeTradingPosition(id) {
        const market = this.tradingMarket();
        try {
            const res = await twFetch(`${twApiBase()}/${market}/positions/${encodeURIComponent(id)}/close`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: '{}'
            });
            if (!res.ok) { alert('Close position failed: ' + await res.text()); return; }
            alert('Position close submitted to the blockchain network');
            this.loadTradingPositions();
        } catch (e) {
            alert('Trading error: ' + e.message);
        }
    }

    // ---- Launchpool (/api/v1/launchpool/*) ----
    async loadLaunchpool() {
        const poolsBox = document.getElementById('launchpool-pools');
        const stakesBox = document.getElementById('launchpool-stakes');
        try {
            const res = await twFetch(`${twApiBase()}/launchpool`);
            if (poolsBox) {
                if (!res.ok) { poolsBox.innerHTML = '<div class="empty-state">Launchpool unavailable</div>'; }
                else {
                    const data = await res.json();
                    const list = Array.isArray(data.pools) ? data.pools : (Array.isArray(data) ? data : []);
                    poolsBox.innerHTML = list.length
                        ? list.map(p => `<div class="provider-card"><div class="nft-name">${this.escapeHtml(p.name || p.symbol || '')}</div><div class="asset-address">${this.escapeHtml(JSON.stringify(p))}</div></div>`).join('')
                        : '<div class="empty-state">No launchpool pools</div>';
                }
            }
        } catch (e) {
            if (poolsBox) poolsBox.innerHTML = '<div class="empty-state">Launchpool unavailable</div>';
        }
        try {
            const res = await twFetch(`${twApiBase()}/launchpool/stakes`);
            if (stakesBox) {
                if (!res.ok) { stakesBox.innerHTML = '<div class="empty-state">Stakes unavailable</div>'; return; }
                const data = await res.json();
                const list = Array.isArray(data.stakes) ? data.stakes : (Array.isArray(data) ? data : []);
                stakesBox.innerHTML = list.length
                    ? list.map(s => `<div class="provider-card"><div class="asset-address">${this.escapeHtml(JSON.stringify(s))}</div></div>`).join('')
                    : '<div class="empty-state">No stakes yet</div>';
            }
        } catch (e) {
            if (stakesBox) stakesBox.innerHTML = '<div class="empty-state">Stakes unavailable</div>';
        }
    }

    async launchpoolAction(action) {
        if (this.isLocked || !this.wallets.length) { alert('Unlock a wallet first'); return; }
        const wallet = this.wallets[0];
        const amount = document.getElementById('launchpool-amount')?.value;
        if (!amount) { alert('Enter an amount'); return; }
        const password = prompt('Enter wallet password to sign:');
        if (!password) { alert('Password is required'); return; }
        try {
            const res = await twFetch(`${twApiBase()}/launchpool/${action}`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ wallet_id: wallet.wallet_id ?? wallet.id, password, amount })
            });
            if (!res.ok) { alert('Launchpool ' + action + ' failed: ' + await res.text()); return; }
            const result = await res.json();
            alert('Launchpool ' + action + ' submitted to the blockchain network: ' + JSON.stringify(result.tx_hash || result));
            this.loadLaunchpool();
        } catch (e) {
            alert('Launchpool error: ' + e.message);
        }
    }

    // ---- Token sales (/api/v1/token-sales/*) ----
    async loadTokenSales() {
        const box = document.getElementById('token-sales-list');
        if (!box) return;
        try {
            const res = await twFetch(`${twApiBase()}/token-sales`);
            if (!res.ok) { box.innerHTML = '<div class="empty-state">Token sales unavailable</div>'; return; }
            const data = await res.json();
            const list = Array.isArray(data.sales) ? data.sales : (Array.isArray(data) ? data : []);
            if (!list.length) { box.innerHTML = '<div class="empty-state">No active token sales</div>'; return; }
            box.innerHTML = list.map(s => {
                const id = this.escapeHtml(s.id ?? s.sale_id ?? '');
                return `
                <div class="provider-card">
                    <div class="nft-name">${this.escapeHtml(s.name || s.symbol || id)}</div>
                    <div class="asset-address">${this.escapeHtml(JSON.stringify(s))}</div>
                    <div class="stake-input-row">
                        <input type="text" id="sale-amount-${id}" placeholder="Amount" class="stake-input">
                        <button class="btn-primary" onclick="window.app.participateTokenSale('${id}')">Participate</button>
                    </div>
                </div>`;
            }).join('');
        } catch (e) {
            box.innerHTML = '<div class="empty-state">Token sales unavailable</div>';
        }
    }

    async participateTokenSale(saleId) {
        const amount = document.getElementById(`sale-amount-${saleId}`)?.value;
        if (!amount) { alert('Enter an amount'); return; }
        try {
            const res = await twFetch(`${twApiBase()}/token-sales/${encodeURIComponent(saleId)}/participate`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ amount })
            });
            if (!res.ok) { alert('Participation failed: ' + await res.text()); return; }
            const result = await res.json();
            alert('Participation submitted to the blockchain network: ' + JSON.stringify(result.tx_hash || result));
            this.loadTokenSales();
        } catch (e) {
            alert('Token sale error: ' + e.message);
        }
    }

    // ---- P2P trading (/api/v1/p2p/*) ----
    async loadP2PAdverts() {
        const box = document.getElementById('p2p-adverts');
        if (!box) return;
        try {
            const res = await twFetch(`${twApiBase()}/p2p/adverts`);
            if (!res.ok) { box.innerHTML = '<div class="empty-state">P2P adverts unavailable</div>'; return; }
            const data = await res.json();
            const list = Array.isArray(data.adverts) ? data.adverts : (Array.isArray(data) ? data : []);
            if (!list.length) { box.innerHTML = '<div class="empty-state">No P2P adverts</div>'; return; }
            box.innerHTML = list.map(a => `
                <div class="provider-card">
                    <div class="nft-name">${this.escapeHtml(a.asset || a.token || '')} @ ${this.escapeHtml(a.price ?? '')}</div>
                    <div class="asset-address">id=${this.escapeHtml(a.id ?? a.advert_id ?? '')} · ${this.escapeHtml(a.side || a.type || '')}</div>
                </div>`).join('');
        } catch (e) {
            box.innerHTML = '<div class="empty-state">P2P adverts unavailable</div>';
        }
    }

    async createP2POrder() {
        const advertId = document.getElementById('p2p-advert-id')?.value;
        const amount = document.getElementById('p2p-amount')?.value;
        if (!advertId || !amount) { alert('Enter an advert id and amount'); return; }
        try {
            const res = await twFetch(`${twApiBase()}/p2p/orders`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ advert_id: advertId, amount })
            });
            if (!res.ok) { alert('P2P order failed: ' + await res.text()); return; }
            const result = await res.json();
            alert('P2P order submitted: ' + JSON.stringify(result.id || result.order_id || result));
        } catch (e) {
            alert('P2P error: ' + e.message);
        }
    }

    // ---- Crypto card (/api/v1/cards/*) ----
    async loadCardInfo() {
        const balBox = document.getElementById('card-balance');
        const ratesBox = document.getElementById('card-rates');
        const txBox = document.getElementById('card-transactions');
        try {
            const res = await twFetch(`${twApiBase()}/cards/default/balance`);
            if (balBox) {
                if (!res.ok) { balBox.innerHTML = '<div class="empty-state">Card balance unavailable</div>'; }
                else {
                    const data = await res.json();
                    balBox.innerHTML = `<div class="provider-card"><div class="asset-address">${this.escapeHtml(JSON.stringify(data))}</div></div>`;
                }
            }
        } catch (e) {
            if (balBox) balBox.innerHTML = '<div class="empty-state">Card balance unavailable</div>';
        }
        try {
            const res = await twFetch(`${twApiBase()}/cards/rates`);
            if (ratesBox) {
                if (!res.ok) { ratesBox.innerHTML = '<div class="empty-state">Rates unavailable</div>'; return; }
                const data = await res.json();
                const entries = Object.entries(data.rates || data);
                ratesBox.innerHTML = entries.length
                    ? entries.map(([k, v]) => `<div class="provider-card"><div class="nft-name">${this.escapeHtml(k)}</div><div class="asset-address">$${this.escapeHtml(v)}</div></div>`).join('')
                    : '<div class="empty-state">Rates unavailable</div>';
            }
        } catch (e) {
            if (ratesBox) ratesBox.innerHTML = '<div class="empty-state">Rates unavailable</div>';
        }
        try {
            const res = await twFetch(`${twApiBase()}/cards/default/transactions`);
            if (txBox) {
                if (!res.ok) { txBox.innerHTML = '<div class="no-transactions">Card transactions unavailable</div>'; return; }
                const data = await res.json();
                const list = Array.isArray(data.transactions) ? data.transactions : (Array.isArray(data) ? data : []);
                txBox.innerHTML = list.length
                    ? list.map(t => `<div class="provider-card"><div class="asset-address">${this.escapeHtml(JSON.stringify(t))}</div></div>`).join('')
                    : '<div class="no-transactions">No card transactions</div>';
            }
        } catch (e) {
            if (txBox) txBox.innerHTML = '<div class="no-transactions">Card transactions unavailable</div>';
        }
    }

    // ---- Price alerts (/api/v1/price-alerts/*) ----
    async loadPriceAlerts() {
        const box = document.getElementById('price-alerts-list');
        if (!box) return;
        try {
            const res = await twFetch(`${twApiBase()}/price-alerts`);
            if (!res.ok) { box.innerHTML = '<div class="empty-state">Price alerts unavailable</div>'; return; }
            const data = await res.json();
            const list = Array.isArray(data.alerts) ? data.alerts : (Array.isArray(data) ? data : []);
            if (!list.length) { box.innerHTML = '<div class="empty-state">No price alerts</div>'; return; }
            box.innerHTML = list.map(a => {
                const id = this.escapeHtml(a.id ?? '');
                return `
                <div class="provider-card">
                    <div class="nft-name">${this.escapeHtml(a.symbol || '')} ${this.escapeHtml(a.direction || '')} $${this.escapeHtml(a.target_price ?? '')}</div>
                    <button class="btn-secondary" onclick="window.app.deletePriceAlert('${id}')">Delete</button>
                </div>`;
            }).join('');
        } catch (e) {
            box.innerHTML = '<div class="empty-state">Price alerts unavailable</div>';
        }
    }

    async createPriceAlert() {
        const symbol = document.getElementById('alert-symbol')?.value;
        const target = document.getElementById('alert-target')?.value;
        const direction = document.getElementById('alert-direction')?.value;
        if (!symbol || !target) { alert('Enter a symbol and target price'); return; }
        try {
            const res = await twFetch(`${twApiBase()}/price-alerts`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ symbol: symbol.toUpperCase(), target_price: target, direction })
            });
            if (!res.ok) { alert('Create alert failed: ' + await res.text()); return; }
            alert('Price alert created');
            this.loadPriceAlerts();
        } catch (e) {
            alert('Price alert error: ' + e.message);
        }
    }

    async deletePriceAlert(id) {
        try {
            const res = await twFetch(`${twApiBase()}/price-alerts/${encodeURIComponent(id)}`, { method: 'DELETE' });
            if (!res.ok) { alert('Delete alert failed: ' + await res.text()); return; }
            this.loadPriceAlerts();
        } catch (e) {
            alert('Price alert error: ' + e.message);
        }
    }

    // ---- DAO governance (/api/v1/dao/*) ----
    async loadDao() {
        const propBox = document.getElementById('dao-proposals');
        const delBox = document.getElementById('dao-delegates');
        try {
            const res = await twFetch(`${twApiBase()}/dao/proposals`);
            if (propBox) {
                if (!res.ok) { propBox.innerHTML = '<div class="empty-state">Proposals unavailable</div>'; }
                else {
                    const data = await res.json();
                    const list = Array.isArray(data.proposals) ? data.proposals : (Array.isArray(data) ? data : []);
                    propBox.innerHTML = list.length ? list.map(p => {
                        const id = this.escapeHtml(p.id ?? p.proposal_id ?? '');
                        return `
                        <div class="provider-card">
                            <div class="nft-name">${this.escapeHtml(p.title || id)}</div>
                            <div class="asset-address">${this.escapeHtml(p.description || '')}</div>
                            <button class="btn-primary" onclick="window.app.daoVote('${id}', true)">For</button>
                            <button class="btn-secondary" onclick="window.app.daoVote('${id}', false)">Against</button>
                        </div>`;
                    }).join('') : '<div class="empty-state">No active proposals</div>';
                }
            }
        } catch (e) {
            if (propBox) propBox.innerHTML = '<div class="empty-state">Proposals unavailable</div>';
        }
        try {
            const res = await twFetch(`${twApiBase()}/dao/delegates`);
            if (delBox) {
                if (!res.ok) { delBox.innerHTML = '<div class="empty-state">Delegates unavailable</div>'; return; }
                const data = await res.json();
                const list = Array.isArray(data.delegates) ? data.delegates : (Array.isArray(data) ? data : []);
                delBox.innerHTML = list.length
                    ? list.map(d => `<div class="provider-card"><div class="asset-address">${this.escapeHtml(JSON.stringify(d))}</div></div>`).join('')
                    : '<div class="empty-state">No delegates</div>';
            }
        } catch (e) {
            if (delBox) delBox.innerHTML = '<div class="empty-state">Delegates unavailable</div>';
        }
    }

    async daoVote(proposalId, support) {
        try {
            const res = await twFetch(`${twApiBase()}/dao/proposals/${encodeURIComponent(proposalId)}/vote`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ support })
            });
            if (!res.ok) { alert('Vote failed: ' + await res.text()); return; }
            alert('Vote submitted to the blockchain network');
            this.loadDao();
        } catch (e) {
            alert('DAO error: ' + e.message);
        }
    }

    // ---- Token approvals (/api/v1/approvals/*) ----
    async loadApprovals() {
        const box = document.getElementById('approvals-list');
        if (!box) return;
        try {
            const res = await twFetch(`${twApiBase()}/approvals`);
            if (!res.ok) { box.innerHTML = '<div class="empty-state">Approvals unavailable</div>'; return; }
            const data = await res.json();
            const list = Array.isArray(data.approvals) ? data.approvals : (Array.isArray(data) ? data : []);
            if (!list.length) { box.innerHTML = '<div class="empty-state">No token approvals</div>'; return; }
            box.innerHTML = list.map(a => {
                const id = this.escapeHtml(a.id ?? '');
                return `
                <div class="provider-card">
                    <div class="nft-name">${this.escapeHtml(a.token || a.token_symbol || '')} → ${this.escapeHtml(a.spender || '')}</div>
                    <div class="asset-address">allowance=${this.escapeHtml(a.allowance ?? a.amount ?? '')}</div>
                    <button class="btn-secondary" onclick="window.app.revokeApproval('${id}')">Revoke</button>
                </div>`;
            }).join('');
        } catch (e) {
            box.innerHTML = '<div class="empty-state">Approvals unavailable</div>';
        }
    }

    async revokeApproval(id) {
        try {
            const res = await twFetch(`${twApiBase()}/approvals/${encodeURIComponent(id)}`, { method: 'DELETE' });
            if (!res.ok) { alert('Revoke failed: ' + await res.text()); return; }
            alert('Revocation submitted to the blockchain network');
            this.loadApprovals();
        } catch (e) {
            alert('Approvals error: ' + e.message);
        }
    }

    // ---- Address book (/api/v1/address-book/contacts/*) ----
    async loadContacts() {
        const box = document.getElementById('contacts-list');
        if (!box) return;
        try {
            const res = await twFetch(`${twApiBase()}/address-book/contacts`);
            if (!res.ok) { box.innerHTML = '<div class="empty-state">Contacts unavailable</div>'; return; }
            const data = await res.json();
            const list = Array.isArray(data.contacts) ? data.contacts : (Array.isArray(data) ? data : []);
            if (!list.length) { box.innerHTML = '<div class="empty-state">No contacts</div>'; return; }
            box.innerHTML = list.map(c => {
                const id = this.escapeHtml(c.id ?? '');
                return `
                <div class="provider-card">
                    <div class="nft-name">${this.escapeHtml(c.label || c.name || '')}</div>
                    <div class="asset-address">${this.escapeHtml(c.address || '')}</div>
                    <button class="btn-secondary" onclick="window.app.deleteContact('${id}')">Delete</button>
                </div>`;
            }).join('');
        } catch (e) {
            box.innerHTML = '<div class="empty-state">Contacts unavailable</div>';
        }
    }

    async addContact() {
        const label = document.getElementById('contact-label')?.value;
        const address = document.getElementById('contact-address')?.value;
        if (!label || !address) { alert('Enter a label and address'); return; }
        try {
            const res = await twFetch(`${twApiBase()}/address-book/contacts`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ label, address })
            });
            if (!res.ok) { alert('Add contact failed: ' + await res.text()); return; }
            this.loadContacts();
        } catch (e) {
            alert('Address book error: ' + e.message);
        }
    }

    async deleteContact(id) {
        try {
            const res = await twFetch(`${twApiBase()}/address-book/contacts/${encodeURIComponent(id)}`, { method: 'DELETE' });
            if (!res.ok) { alert('Delete contact failed: ' + await res.text()); return; }
            this.loadContacts();
        } catch (e) {
            alert('Address book error: ' + e.message);
        }
    }

    // ---- Linked devices (/api/v1/devices/*) ----
    async loadDevices() {
        const box = document.getElementById('devices-list');
        if (!box) return;
        try {
            const res = await twFetch(`${twApiBase()}/devices`);
            if (!res.ok) { box.innerHTML = '<div class="empty-state">Devices unavailable</div>'; return; }
            const data = await res.json();
            const list = Array.isArray(data.devices) ? data.devices : (Array.isArray(data) ? data : []);
            if (!list.length) { box.innerHTML = '<div class="empty-state">No linked devices</div>'; return; }
            box.innerHTML = list.map(d => `
                <div class="provider-card">
                    <div class="nft-name">${this.escapeHtml(d.name || d.device_name || d.platform || '')}</div>
                    <div class="asset-address">${this.escapeHtml(d.last_seen || d.synced_at || '')}</div>
                </div>`).join('');
        } catch (e) {
            box.innerHTML = '<div class="empty-state">Devices unavailable</div>';
        }
    }

    // ---- Security Center (/api/v1/security/{check-url,check-address,scan}) ----
    async securityCheck() {
        const input = document.getElementById('security-target');
        const out = document.getElementById('security-result');
        if (!input || !out) return;
        const target = input.value.trim();
        if (!target) { out.textContent = 'Enter a URL or address'; return; }
        out.textContent = 'Checking...';
        const isUrl = target.startsWith('http://') || target.startsWith('https://');
        const path = isUrl
            ? `${twApiBase()}/security/check-url?url=${encodeURIComponent(target)}`
            : `${twApiBase()}/security/check-address?address=${encodeURIComponent(target)}`;
        try {
            const res = await twFetch(path);
            if (!res.ok) { out.textContent = `Check failed (HTTP ${res.status})`; return; }
            const data = await res.json();
            out.textContent = data.safe ? `✓ Safe: ${this.escapeHtml(data.reason || 'no threats')}` : `⚠ Flagged: ${this.escapeHtml(data.reason || 'threat detected')}`;
        } catch (e) {
            out.textContent = 'Check failed (backend unreachable)';
        }
    }

    async securityScan() {
        const input = document.getElementById('security-target');
        const out = document.getElementById('security-result');
        if (!input || !out) return;
        const target = input.value.trim();
        if (!target) { out.textContent = 'Enter a URL or address'; return; }
        out.textContent = 'Scanning...';
        try {
            const res = await twFetch(`${twApiBase()}/security/scan`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ target })
            });
            if (!res.ok) { out.textContent = `Scan failed (HTTP ${res.status})`; return; }
            const data = await res.json();
            const threats = Array.isArray(data.threats) ? data.threats : [];
            out.textContent = threats.length
                ? '⚠ Threats: ' + threats.map(t => this.escapeHtml(JSON.stringify(t))).join('; ')
                : '✓ Safe: no threats detected';
        } catch (e) {
            out.textContent = 'Scan failed (backend unreachable)';
        }
    }

    // ---- Trading Terminal (/api/v1/terminal/{kline,ticker}) — real OHLC canvas ----
    async loadTerminalChart() {
        const symbolInput = document.getElementById('terminal-symbol');
        const daysInput = document.getElementById('terminal-days');
        const tickerOut = document.getElementById('terminal-ticker');
        const canvas = document.getElementById('terminal-canvas');
        if (!symbolInput || !daysInput || !canvas) return;
        const symbol = (symbolInput.value.trim() || 'ETH').toUpperCase();
        const days = parseInt(daysInput.value, 10) || 1;
        try {
            const res = await twFetch(`${twApiBase()}/terminal/ticker/${encodeURIComponent(symbol)}`);
            if (tickerOut) tickerOut.textContent = res.ok ? JSON.stringify(await res.json()) : 'Ticker unavailable';
        } catch (e) {
            if (tickerOut) tickerOut.textContent = 'Ticker unavailable';
        }
        try {
            const res = await twFetch(`${twApiBase()}/terminal/kline/${encodeURIComponent(symbol)}?days=${days}`);
            const raw = res.ok ? await res.json() : null;
            const list = Array.isArray(raw) ? raw : (raw?.candles ?? raw?.kline ?? []);
            const candles = list.map((c) => Array.isArray(c)
                ? { t: +c[0], o: +c[1], h: +c[2], l: +c[3], c: +c[4] }
                : { t: +c.time ?? +c.t ?? +c.timestamp, o: +c.open ?? +c.o, h: +c.high ?? +c.h, l: +c.low ?? +c.l, c: +c.close ?? +c.c })
                .filter((c) => isFinite(c.o) && isFinite(c.h) && isFinite(c.l) && isFinite(c.c));
            this.drawCandles(canvas, candles);
        } catch (e) {
            this.drawCandles(canvas, []);
        }
    }

    drawCandles(canvas, candles) {
        const ctx = canvas.getContext('2d');
        if (!ctx) return;
        const w = (canvas.width = canvas.offsetWidth || 800);
        const h = (canvas.height = 320);
        const muted = getComputedStyle(canvas).getPropertyValue('--text-secondary') || '#8b949e';
        ctx.clearRect(0, 0, w, h);
        if (!candles.length) {
            ctx.fillStyle = muted;
            ctx.font = '14px sans-serif';
            ctx.fillText('No candle data for this symbol/range.', 16, 30);
            return;
        }
        const padX = 60;
        let min = Infinity, max = -Infinity;
        for (const c of candles) { if (c.l < min) min = c.l; if (c.h > max) max = c.h; }
        const span = max - min || 1;
        const bw = Math.max(2, (w - padX) / candles.length - 2);
        const y = (v) => h - ((v - min) / span) * (h - 20) - 10;
        candles.forEach((c, i) => {
            const up = c.c >= c.o;
            ctx.strokeStyle = ctx.fillStyle = up ? '#16a34a' : '#dc2626';
            const x = padX + i * (bw + 2);
            ctx.beginPath();
            ctx.moveTo(x + bw / 2, y(c.h));
            ctx.lineTo(x + bw / 2, y(c.l));
            ctx.stroke();
            const top = y(Math.max(c.o, c.c));
            const height = Math.max(1, Math.abs(y(c.o) - y(c.c)));
            ctx.fillRect(x, top, bw, height);
        });
        ctx.fillStyle = muted;
        ctx.font = '11px sans-serif';
        ctx.fillText(max.toFixed(2), 4, y(max) + 4);
        ctx.fillText(min.toFixed(2), 4, y(min) + 4);
    }

    // ---- NFT transfer (/api/v1/nft/transfer — real ERC-721 safeTransferFrom) ----
    async transferNFT() {
        if (this.isLocked || !this.wallets.length) { alert('Unlock a wallet first'); return; }
        const wallet = this.wallets[0];
        const contract = document.getElementById('nft-transfer-contract')?.value;
        const tokenId = document.getElementById('nft-transfer-token-id')?.value;
        const to = document.getElementById('nft-transfer-to')?.value;
        if (!contract || !tokenId || !to) { alert('Enter contract, token id, and recipient'); return; }
        const password = prompt('Enter wallet password to sign:');
        if (!password) { alert('Password is required'); return; }
        try {
            const res = await twFetch(`${twApiBase()}/nft/transfer`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    wallet_id: wallet.wallet_id ?? wallet.id,
                    password,
                    to,
                    token_id: tokenId,
                    contract_address: contract,
                    chain_id: this.currentNetwork
                })
            });
            if (!res.ok) { alert('NFT transfer failed: ' + await res.text()); return; }
            const result = await res.json();
            alert('NFT transfer submitted to the blockchain network: ' + (result.transaction_hash || result.tx_hash || JSON.stringify(result)));
            this.loadNFTs();
        } catch (e) {
            alert('NFT transfer error: ' + e.message);
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

    // ---- Backend server configuration ----
    saveApiBase() {
        const input = document.getElementById('settings-api-base');
        const status = document.getElementById('settings-api-base-status');
        if (!input || !input.value.trim()) { alert('Enter a backend URL'); return; }
        if (!twSetApiBase(input.value)) { alert('Invalid backend URL'); return; }
        if (status) status.innerHTML = '<div class="asset-item"><div class="asset-name">Backend set to ' + this.escapeHtml(twApiOrigin()) + ' — reloading…</div></div>';
        setTimeout(() => location.reload(), 600);
    }

    // ---- Live price feed (WebSocket /api/v1/ws) ----
    connectLiveFeed() {
        const box = document.getElementById('live-ticker');
        if (!box || typeof WebSocket === 'undefined') return;
        if (this._liveFeed) { try { this._liveFeed.close(); } catch (e) {} }
        let ws;
        try { ws = new WebSocket(twWsUrl()); } catch (e) { return; }
        this._liveFeed = ws;
        const prices = {};
        const render = () => {
            const rows = Object.entries(prices).map(([sym, t]) =>
                `<div class="asset-item"><div class="asset-info"><div class="asset-name">${this.escapeHtml(sym)}</div></div>` +
                `<div class="asset-balance"><div class="balance">$${Number(t.last_price).toLocaleString()}</div>` +
                `<div class="usd-value">${Number(t.change_24h_pct || 0).toFixed(2)}% (24h)</div></div></div>`).join('');
            box.innerHTML = rows || '';
        };
        ws.onopen = () => ws.send(JSON.stringify({ action: 'subscribe', symbols: ['BTC', 'ETH'] }));
        ws.onmessage = (ev) => {
            try {
                const msg = JSON.parse(ev.data);
                if (msg.type === 'ticker' && msg.symbol) { prices[msg.symbol] = msg; render(); }
            } catch (e) { /* ignore malformed frame */ }
        };
    }

    // ---- KYC (proxied to the canonical listing_service via /kyc/*) ----
    async kycRegister() {
        const name = document.getElementById('kyc-name')?.value;
        const country = document.getElementById('kyc-country')?.value;
        if (!name || !country) { alert('Enter name and country'); return; }
        const out = document.getElementById('kyc-result');
        try {
            const res = await twFetch(`${twApiBase()}/kyc/register`, {
                method: 'POST', headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ full_name: name, country })
            });
            if (!res.ok) { alert('KYC register failed: ' + await res.text()); return; }
            const data = await res.json();
            if (out) out.innerHTML = '<div class="asset-item"><div class="asset-name">Registered: ' + this.escapeHtml(JSON.stringify(data)) + '</div></div>';
            this.loadKYCStatus();
        } catch (e) { alert('KYC register error: ' + e.message); }
    }

    async kycSubmit() {
        const dob = document.getElementById('kyc-dob')?.value;
        if (!dob) { alert('Enter date of birth'); return; }
        try {
            const res = await twFetch(`${twApiBase()}/kyc/submit`, {
                method: 'POST', headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ date_of_birth: dob })
            });
            if (!res.ok) { alert('KYC submit failed: ' + await res.text()); return; }
            alert('KYC application submitted');
            this.loadKYCStatus();
        } catch (e) { alert('KYC submit error: ' + e.message); }
    }

    async kycUploadDocument() {
        const docType = document.getElementById('kyc-doc-type')?.value;
        const file = document.getElementById('kyc-doc-file')?.files?.[0];
        if (!docType || !file) { alert('Choose a document type and file'); return; }
        const form = new FormData();
        form.append('document_type', docType);
        form.append('document', file);
        try {
            const res = await twFetch(`${twApiBase()}/kyc/document`, { method: 'POST', body: form });
            if (!res.ok) { alert('KYC upload failed: ' + await res.text()); return; }
            alert('KYC document uploaded');
            this.loadKYCStatus();
        } catch (e) { alert('KYC upload error: ' + e.message); }
    }

    // ---- Multisig (proxied to MasterWallet :8450 via /wallet/multisig/*) ----
    async loadMultisigWallets() {
        const box = document.getElementById('multisig-list');
        if (!box) return;
        box.innerHTML = '<div class="no-transactions">Loading multisig wallets…</div>';
        try {
            const res = await twFetch(`${twApiBase()}/wallet/multisig/wallets`);
            if (!res.ok) { box.innerHTML = '<div class="no-transactions">Multisig unavailable: ' + this.escapeHtml(await res.text()) + '</div>'; return; }
            const data = await res.json();
            const list = data.multisig_wallets || data.wallets || [];
            box.innerHTML = list.length ? list.map((w) =>
                `<div class="asset-item"><div class="asset-info"><div class="asset-name">${this.escapeHtml(w.name || w.id)}</div>` +
                `<div class="asset-address">${this.escapeHtml(w.id)} — chain ${w.chain_id}, ${w.threshold}-of-${(w.owners || []).length}</div></div></div>`).join('')
                : '<div class="no-transactions">No multisig wallets</div>';
        } catch (e) {
            box.innerHTML = '<div class="no-transactions">Multisig unavailable: ' + this.escapeHtml(e.message) + '</div>';
        }
    }

    async createMultisigWallet() {
        const name = document.getElementById('ms-name')?.value;
        const owners = (document.getElementById('ms-owners')?.value || '').split(',').map((s) => s.trim()).filter(Boolean);
        const threshold = parseInt(document.getElementById('ms-threshold')?.value || '0', 10);
        if (!name || !owners.length || !threshold) { alert('Enter name, owners and threshold'); return; }
        try {
            const res = await twFetch(`${twApiBase()}/wallet/multisig/wallets`, {
                method: 'POST', headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name, owners, threshold, chain_id: this.currentNetwork })
            });
            if (!res.ok) { alert('Create multisig failed: ' + await res.text()); return; }
            alert('Multisig wallet created');
            this.loadMultisigWallets();
        } catch (e) { alert('Multisig error: ' + e.message); }
    }

    async createMultisigTx() {
        const wid = document.getElementById('ms-tx-wallet')?.value;
        const to = document.getElementById('ms-tx-to')?.value;
        const value = document.getElementById('ms-tx-value')?.value;
        const data = document.getElementById('ms-tx-data')?.value || '';
        if (!wid || !to || !value) { alert('Enter wallet id, to address and value'); return; }
        try {
            const res = await twFetch(`${twApiBase()}/wallet/multisig/wallets/${encodeURIComponent(wid)}/transactions`, {
                method: 'POST', headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ to_address: to, value, data })
            });
            if (!res.ok) { alert('Create multisig tx failed: ' + await res.text()); return; }
            alert('Multisig transaction created — pending signatures');
            this.loadMultisigTxs();
        } catch (e) { alert('Multisig tx error: ' + e.message); }
    }

    async loadMultisigTxs() {
        const wid = document.getElementById('ms-tx-list-wallet')?.value || document.getElementById('ms-tx-wallet')?.value;
        const box = document.getElementById('multisig-tx-list');
        if (!wid || !box) { alert('Enter a multisig wallet id'); return; }
        box.innerHTML = '<div class="no-transactions">Loading transactions…</div>';
        try {
            const res = await twFetch(`${twApiBase()}/wallet/multisig/wallets/${encodeURIComponent(wid)}/transactions`);
            if (!res.ok) { box.innerHTML = '<div class="no-transactions">Failed: ' + this.escapeHtml(await res.text()) + '</div>'; return; }
            const data = await res.json();
            const list = data.transactions || data.multisig_transactions || [];
            box.innerHTML = list.length ? list.map((t) =>
                `<div class="asset-item"><div class="asset-info"><div class="asset-name">${this.escapeHtml(t.id)} → ${this.escapeHtml(t.to_address || '')}</div>` +
                `<div class="asset-address">${this.escapeHtml(t.status || '')} — sigs ${t.signatures_collected ?? 0}</div></div>` +
                `<div class="asset-balance"><button class="btn-secondary" onclick="window.app.multisigSign('${t.id}')">Sign</button> ` +
                `<button class="btn-primary" onclick="window.app.multisigExecute('${t.id}')">Execute</button></div></div>`).join('')
                : '<div class="no-transactions">No multisig transactions</div>';
        } catch (e) {
            box.innerHTML = '<div class="no-transactions">Failed: ' + this.escapeHtml(e.message) + '</div>';
        }
    }

    async multisigSign(tid) {
        try {
            const res = await twFetch(`${twApiBase()}/wallet/multisig/transactions/${encodeURIComponent(tid)}/sign`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{}' });
            if (!res.ok) { alert('Sign failed: ' + await res.text()); return; }
            alert('Multisig transaction signed');
            this.loadMultisigTxs();
        } catch (e) { alert('Sign error: ' + e.message); }
    }

    async multisigExecute(tid) {
        try {
            const res = await twFetch(`${twApiBase()}/wallet/multisig/transactions/${encodeURIComponent(tid)}/execute`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{}' });
            if (!res.ok) { alert('Execute failed: ' + await res.text()); return; }
            const data = await res.json();
            alert('Transaction submitted to the blockchain network: ' + (data.tx_hash || data.status || 'broadcast'));
            this.loadMultisigTxs();
        } catch (e) { alert('Execute error: ' + e.message); }
    }

    // ---- Non-EVM chains (real key derivation + signing; mainnet only) ----
    async deriveNonEvmAddress() {
        if (this.isLocked || !this.wallets.length) { alert('Unlock a wallet first'); return; }
        const chain = (document.getElementById('nevm-chain')?.value || '').trim().toLowerCase();
        if (!chain) { alert('Enter a chain type'); return; }
        const wallet = this.wallets[0];
        const password = prompt('Enter wallet password to derive the address:');
        if (!password) { alert('Password is required'); return; }
        const out = document.getElementById('nevm-address-result');
        try {
            const res = await twFetch(`${twApiBase()}/non_evm/address`, {
                method: 'POST', headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ wallet_id: wallet.wallet_id ?? wallet.id, password, chain_type: chain })
            });
            if (!res.ok) { alert('Derive failed: ' + await res.text()); return; }
            const data = await res.json();
            if (out) out.innerHTML = '<div class="asset-item"><div class="asset-info"><div class="asset-name">' + this.escapeHtml(chain) + '</div><div class="asset-address">' + this.escapeHtml(data.address || JSON.stringify(data)) + '</div></div></div>';
        } catch (e) { alert('Derive error: ' + e.message); }
    }

    async sendNonEvm() {
        if (this.isLocked || !this.wallets.length) { alert('Unlock a wallet first'); return; }
        const chain = (document.getElementById('nevm-send-chain')?.value || '').trim().toLowerCase();
        const to = document.getElementById('nevm-send-to')?.value;
        const amount = document.getElementById('nevm-send-amount')?.value;
        const password = document.getElementById('nevm-send-password')?.value;
        if (!chain || !to || !amount || !password) { alert('Enter chain, destination, amount and password'); return; }
        const wallet = this.wallets[0];
        const out = document.getElementById('nevm-send-result');
        // Non-EVM send is a signing operation: bitcoin requires explicit UTXO
        // inputs/outputs and cosmos a sign doc. Build the chain-appropriate
        // request; the backend returns the raw signed payload for broadcast.
        const body = { wallet_id: wallet.wallet_id ?? wallet.id, password, chain_type: chain };
        if (chain === 'bitcoin') {
            body.bitcoin_inputs = [{ txid: to, vout: 0, amount_sats: Math.round(parseFloat(amount) * 1e8) }];
            body.bitcoin_outputs = [{ address: to, amount_sats: Math.round(parseFloat(amount) * 1e8) }];
        } else if (chain === 'cosmos') {
            body.cosmos_sign_doc = { chain_id: 'cosmoshub-4', account_number: '0', body_bytes: '', auth_info_bytes: '' };
        }
        try {
            const res = await twFetch(`${twApiBase()}/non_evm/send`, {
                method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body)
            });
            if (!res.ok) { alert('Non-EVM send failed: ' + await res.text()); return; }
            const data = await res.json();
            if (out) out.innerHTML = '<div class="asset-item"><div class="asset-info"><div class="asset-name">Signed payload</div><div class="asset-address">' + this.escapeHtml((data.raw_tx || data.signature || '') + '') + '</div></div></div>';
            alert('Signed. ' + (data.action || 'Transaction submitted to the blockchain network'));
        } catch (e) { alert('Non-EVM send error: ' + e.message); }
    }

    // ---- Lending (proxied to lending_service :8009 via /lending/*) ----
    async loadLending() {
        const marketsBox = document.getElementById('lending-markets');
        const posBox = document.getElementById('lending-positions');
        if (marketsBox) {
            try {
                const res = await twFetch(`${twApiBase()}/lending/markets`);
                const data = res.ok ? await res.json() : {};
                const list = data.markets || data.data || (Array.isArray(data) ? data : []);
                marketsBox.innerHTML = list.length ? list.map((m) =>
                    `<div class="asset-item"><div class="asset-info"><div class="asset-name">${this.escapeHtml(m.asset || m.symbol || m.id || 'market')}</div>` +
                    `<div class="asset-address">supply APY ${m.supply_apy ?? m.apy ?? '—'}%</div></div></div>`).join('')
                    : '<div class="no-transactions">No lending markets</div>';
            } catch (e) {
                marketsBox.innerHTML = '<div class="no-transactions">Lending unavailable: ' + this.escapeHtml(e.message) + '</div>';
            }
        }
        if (posBox) {
            try {
                const res = await twFetch(`${twApiBase()}/lending/positions`);
                const data = res.ok ? await res.json() : {};
                const list = data.positions || data.data || (Array.isArray(data) ? data : []);
                posBox.innerHTML = list.length ? list.map((p) =>
                    `<div class="asset-item"><div class="asset-info"><div class="asset-name">${this.escapeHtml(p.asset || p.id || 'position')}</div>` +
                    `<div class="asset-address">${this.escapeHtml(p.amount ?? '')} (${this.escapeHtml(p.kind || p.side || '')})</div></div></div>`).join('')
                    : '<div class="no-transactions">No lending positions</div>';
            } catch (e) {
                posBox.innerHTML = '<div class="no-transactions">Positions unavailable: ' + this.escapeHtml(e.message) + '</div>';
            }
        }
    }

    async lendingAction() {
        if (this.isLocked || !this.wallets.length) { alert('Unlock a wallet first'); return; }
        const action = document.getElementById('lending-action')?.value;
        const asset = document.getElementById('lending-asset')?.value;
        const amount = document.getElementById('lending-amount')?.value;
        const password = document.getElementById('lending-password')?.value;
        if (!asset || !amount || !password) { alert('Enter asset, amount and password'); return; }
        const wallet = this.wallets[0];
        try {
            const res = await twFetch(`${twApiBase()}/lending/${action}`, {
                method: 'POST', headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ wallet_id: wallet.wallet_id ?? wallet.id, password, asset, amount, chain_id: this.currentNetwork })
            });
            if (!res.ok) { alert('Lending ' + action + ' failed: ' + await res.text()); return; }
            const data = await res.json();
            alert('Lending ' + action + ' submitted to the blockchain network: ' + (data.tx_hash || data.id || 'accepted'));
            this.loadLending();
        } catch (e) { alert('Lending error: ' + e.message); }
    }

    // ---- Prediction markets (proxied to prediction_service :8455) ----
    async loadPredictionMarkets() {
        const box = document.getElementById('prediction-markets');
        if (!box) return;
        try {
            const res = await twFetch(`${twApiBase()}/prediction/markets`);
            const data = res.ok ? await res.json() : {};
            const list = data.markets || data.data || (Array.isArray(data) ? data : []);
            box.innerHTML = list.length ? list.map((m) =>
                `<div class="asset-item"><div class="asset-info"><div class="asset-name">${this.escapeHtml(m.question || m.title || m.id || 'market')}</div>` +
                `<div class="asset-address">id ${this.escapeHtml(String(m.id || ''))} — ${this.escapeHtml(m.status || '')}</div></div></div>`).join('')
                : '<div class="no-transactions">No prediction markets</div>';
        } catch (e) {
            box.innerHTML = '<div class="no-transactions">Prediction unavailable: ' + this.escapeHtml(e.message) + '</div>';
        }
    }

    async placePrediction() {
        const marketId = document.getElementById('prediction-market-id')?.value;
        const outcome = document.getElementById('prediction-outcome')?.value;
        const amount = document.getElementById('prediction-amount')?.value;
        if (!marketId || !outcome || !amount) { alert('Enter market id, outcome and amount'); return; }
        try {
            const res = await twFetch(`${twApiBase()}/prediction/markets/${encodeURIComponent(marketId)}/bet`, {
                method: 'POST', headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ outcome, amount })
            });
            if (!res.ok) { alert('Prediction failed: ' + await res.text()); return; }
            alert('Position submitted to the blockchain network');
            this.loadPredictionMarkets();
        } catch (e) { alert('Prediction error: ' + e.message); }
    }

    // ---- Keystore (Web3 Secret Storage export/import) ----
    populateWalletSelects() {
        const sel = document.getElementById('keystore-export-wallet');
        if (!sel) return;
        sel.innerHTML = this.wallets.map((w) =>
            `<option value="${this.escapeHtml(String(w.wallet_id ?? w.id))}">${this.escapeHtml(w.label || w.address || (w.wallet_id ?? w.id))}</option>`).join('');
    }

    async exportKeystore() {
        const walletId = document.getElementById('keystore-export-wallet')?.value;
        const password = document.getElementById('keystore-export-password')?.value;
        const exportPassword = document.getElementById('keystore-export-new-password')?.value;
        if (!walletId || !password || !exportPassword) { alert('Choose a wallet and enter both passwords'); return; }
        const out = document.getElementById('keystore-export-result');
        try {
            const res = await twFetch(`${twApiBase()}/keystore/export`, {
                method: 'POST', headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ wallet_id: walletId, password, export_password: exportPassword })
            });
            if (!res.ok) { alert('Keystore export failed: ' + await res.text()); return; }
            const data = await res.json();
            const ks = JSON.stringify(data.keystore || data, null, 2);
            const blob = new Blob([ks], { type: 'application/json' });
            const a = document.createElement('a');
            a.href = URL.createObjectURL(blob);
            a.download = 'keystore-' + walletId + '.json';
            a.click();
            URL.revokeObjectURL(a.href);
            if (out) out.innerHTML = '<div class="asset-item"><div class="asset-name">Keystore exported (downloaded)</div></div>';
        } catch (e) { alert('Keystore export error: ' + e.message); }
    }

    async importKeystore() {
        const json = document.getElementById('keystore-import-json')?.value;
        const password = document.getElementById('keystore-import-password')?.value;
        const label = document.getElementById('keystore-import-label')?.value || 'Imported wallet';
        if (!json || !password) { alert('Paste the keystore JSON and enter its password'); return; }
        const out = document.getElementById('keystore-import-result');
        try {
            const res = await twFetch(`${twApiBase()}/keystore/import`, {
                method: 'POST', headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ keystore_json: json, password, label })
            });
            if (!res.ok) { alert('Keystore import failed: ' + await res.text()); return; }
            const data = await res.json();
            if (out) out.innerHTML = '<div class="asset-item"><div class="asset-info"><div class="asset-name">Imported: ' + this.escapeHtml(data.address || '') + '</div><div class="asset-address">' + this.escapeHtml(data.wallet_id || '') + '</div></div></div>';
            this.loadWalletData();
        } catch (e) { alert('Keystore import error: ' + e.message); }
    }

    // ---- Hardware wallet (real WebHID/WebUSB via hardwareWallet.js) ----
    getHwManager() {
        if (!this._hw) this._hw = new HardwareWalletManager();
        return this._hw;
    }

    async hwDetect() {
        const box = document.getElementById('hw-device');
        if (!box) return;
        box.innerHTML = '<div class="no-transactions">Detecting…</div>';
        const device = await this.getHwManager().detectDevice();
        box.innerHTML = device
            ? '<div class="asset-item"><div class="asset-info"><div class="asset-name">' + this.escapeHtml(device.name) + ' (' + this.escapeHtml(device.model || '') + ')</div><div class="asset-address">vendor 0x' + (device.vendorId || 0).toString(16) + '</div></div></div>'
            : '<div class="no-transactions">No hardware wallet detected (WebHID/WebUSB required)</div>';
    }

    async hwGetAddress() {
        const chain = (document.getElementById('hw-chain')?.value || 'ethereum').trim().toLowerCase();
        const out = document.getElementById('hw-address-result');
        try {
            const address = await this.getHwManager().getAddress(chain);
            if (out) out.innerHTML = '<div class="asset-item"><div class="asset-info"><div class="asset-name">' + this.escapeHtml(chain) + '</div><div class="asset-address">' + this.escapeHtml(address || 'unavailable') + '</div></div></div>';
        } catch (e) { alert('Hardware address error: ' + e.message); }
    }

    async hwSignMessage() {
        const message = document.getElementById('hw-message')?.value;
        const out = document.getElementById('hw-sign-result');
        if (!message) { alert('Enter a message'); return; }
        try {
            const sig = await this.getHwManager().signMessage(message);
            if (out) out.innerHTML = '<div class="asset-item"><div class="asset-info"><div class="asset-name">Signature</div><div class="asset-address">' + this.escapeHtml(sig || 'unavailable') + '</div></div></div>';
        } catch (e) { alert('Hardware sign error: ' + e.message); }
    }

    async hwSignTransaction() {
        const g = (id) => (document.getElementById(id)?.value || '').trim();
        const out = document.getElementById('hw-tx-result');
        const to = g('hw-tx-to');
        if (!/^0x[0-9a-fA-F]{40}$/.test(to)) { alert('Enter a valid recipient address'); return; }
        const gasPrice = g('hw-tx-gasprice');
        const maxFee = g('hw-tx-maxfee');
        const tx = {
            to,
            value: g('hw-tx-value') || '0',
            nonce: g('hw-tx-nonce') || '0',
            gasLimit: g('hw-tx-gaslimit') || '21000',
            chainId: g('hw-tx-chainid') || '1',
            data: g('hw-tx-data') || '0x',
        };
        // Legacy EIP-155 when a gas price is given, EIP-1559 type-2 otherwise.
        if (gasPrice) tx.gasPrice = gasPrice;
        else { tx.maxFeePerGas = maxFee || '0'; tx.maxPriorityFeePerGas = g('hw-tx-maxprio') || '0'; }
        try {
            const res = await this.getHwManager().signTransaction(tx);
            if (out) out.innerHTML = '<div class="asset-item"><div class="asset-info"><div class="asset-name">Signed raw transaction</div>' +
                '<div class="asset-address">' + this.escapeHtml(res.rawTransaction) + '</div>' +
                '<div class="asset-address">v=' + res.v + ' r=' + this.escapeHtml(res.r) + ' s=' + this.escapeHtml(res.s) + '</div>' +
                '<div class="asset-address">Broadcast via eth_sendRawTransaction on chain ' + this.escapeHtml(tx.chainId) + '</div></div></div>';
        } catch (e) { alert('Hardware transaction sign error: ' + e.message); }
    }

    // ---- Trading page extras: futures catalog + copy trading (tradingFeatures.js) ----
    async loadFuturesPairs() {
        const box = document.getElementById('futures-pairs');
        if (!box || typeof FuturesService === 'undefined') return;
        if (!this._futures) this._futures = new FuturesService();
        const pairs = await this._futures.loadPairs();
        box.innerHTML = pairs.length ? pairs.slice(0, 50).map((p) =>
            `<div class="asset-item"><div class="asset-info"><div class="asset-name">${this.escapeHtml(p.symbol)}</div></div>` +
            `<div class="asset-balance"><div class="balance">$${Number(p.price || 0).toLocaleString()}</div><div class="usd-value">${Number(p.change24h || 0).toFixed(2)}%</div></div></div>`).join('')
            : '<div class="no-transactions">No futures markets</div>';
    }

    async loadCopyTraders() {
        const box = document.getElementById('copy-traders');
        if (!box || typeof CopyTradingService === 'undefined') return;
        if (!this._copy) this._copy = new CopyTradingService();
        const traders = await this._copy.loadTraders();
        box.innerHTML = traders.length ? traders.map((t) =>
            `<div class="asset-item"><div class="asset-info"><div class="asset-name">${this.escapeHtml(t.name || t.id)}</div>` +
            `<div class="asset-address">${this.escapeHtml(String(t.id))}</div></div></div>`).join('')
            : '<div class="no-transactions">No copy traders</div>';
    }

    async followTrader() {
        const traderId = document.getElementById('copy-trader-id')?.value;
        if (!traderId) { alert('Enter a trader id'); return; }
        const wallet = this.wallets[0];
        try {
            const res = await twFetch(`${twApiBase()}/copytrading/follow`, {
                method: 'POST', headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ trader_id: traderId, wallet_id: wallet ? (wallet.wallet_id ?? wallet.id) : undefined, allocation: document.getElementById('copy-allocation')?.value })
            });
            if (!res.ok) { alert('Follow failed: ' + await res.text()); return; }
            alert('Following trader — copy trades will execute automatically');
        } catch (e) { alert('Follow error: ' + e.message); }
    }
    // ==================== Backup Actions ====================

    copyBackupMnemonic() {
        if (!this._pendingBackup) return;
        navigator.clipboard.writeText(this._pendingBackup.mnemonic).then(() => {
            alert('Recovery phrase copied to clipboard. Store it safely!');
        }).catch(() => {
            alert('Copy failed. Please write down the phrase manually.');
        });
    }

    async saveBackupToDrive() {
        if (!this._pendingBackup) return;
        // Google Drive backup requires a real OAuth client_id configured
        // via the VITE_GOOGLE_DRIVE_CLIENT_ID env var. Fail-closed if unset.
        try {
            const { uploadBackupToDrive } = await import('./services/googleDriveBackup.js');
            await uploadBackupToDrive(this._pendingBackup.wallet, this._pendingBackup.mnemonic);
            alert('Encrypted backup saved to Google Drive (appDataFolder).');
        } catch (e) {
            alert('Google Drive backup not available: ' + e.message);
        }
    }

    async downloadEncryptedBackup() {
        if (!this._pendingBackup) return;
        const wallet = this._pendingBackup.wallet;
        const password = prompt('Enter wallet password to verify backup export:');
        if (!password) { alert('Password is required'); return; }
        try {
            const res = await twFetch(`${twApiBase()}/wallets/${encodeURIComponent(wallet.id)}/export-encrypted-seed`, {
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
            const json = JSON.stringify(blob, null, 2);
            const a = document.createElement('a');
            const file = new Blob([json], { type: 'application/json' });
            a.href = URL.createObjectURL(file);
            a.download = `tigerwallet-backup-${wallet.address.slice(0, 8)}.json`;
            a.click();
            URL.revokeObjectURL(a.href);
        } catch (e) {
            alert('Backup export error: ' + e.message);
        }
    }

    confirmBackupAndContinue() {
        if (!this._pendingBackup) return;
        this._pendingBackup = null;
        this.showDashboard();
    }

    async addWatchOnlyWallet() {
        const input = document.getElementById('watch-only-address');
        const address = input && input.value ? input.value.trim() : '';
        if (!address || !address.startsWith('0x') || address.length !== 42) {
            alert('Enter a valid EVM address (0x...)');
            return;
        }
        try {
            const res = await twFetch(twApiBase() + '/wallets/watch-only', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ address: address, chain_id: this.currentNetwork, label: 'Watch: ' + address.slice(0, 8) })
            });
            if (!res.ok) {
                const err = await res.text();
                alert('Watch-only wallet failed: ' + err);
                return;
            }
            const w = await res.json();
            this.wallets.push({
                id: this.generateId(),
                wallet_id: w.id,
                name: w.label || 'Watch-only',
                address: w.address,
                balance: '0',
                tokens: [],
                unlocked: false,
                watchOnly: true
            });
            localStorage.setItem('tigerwallet-wallets', JSON.stringify(this.wallets));
            input.value = '';
            alert('Watch-only wallet added. You can track balances but cannot send.');
            this.loadWalletData();
        } catch (e) {
            alert('Watch-only error: ' + e.message);
        }
    }

    async updateHealthBadge() {
        const badge = document.getElementById('health-badge');
        if (!badge) return;
        try {
            const res = await fetch(twApiOrigin() + '/health');
            if (res.ok) {
                badge.className = 'health-badge health-ok';
                badge.title = 'Backend: healthy';
            } else {
                badge.className = 'health-badge health-degraded';
                badge.title = 'Backend: degraded (HTTP ' + res.status + ')';
            }
        } catch (e) {
            badge.className = 'health-badge health-down';
            badge.title = 'Backend: unreachable';
        }
    }

    async openMarginPosition() {
        if (this.isLocked || !this.wallets.length) { alert('Unlock a wallet first'); return; }
        const wallet = this.wallets[0];
        const pair = prompt('Trading pair (e.g. ETH/USDT):');
        if (!pair) return;
        const side = prompt('Side (long/short):', 'long');
        if (!side || ['long', 'short'].indexOf(side) === -1) { alert('Side must be long or short'); return; }
        const collateral = prompt('Collateral amount:');
        if (!collateral || isNaN(parseFloat(collateral))) { alert('Invalid collateral'); return; }
        const leverage = prompt('Leverage (e.g. 2):', '2');
        if (!leverage || isNaN(parseFloat(leverage))) { alert('Invalid leverage'); return; }
        try {
            const res = await twFetch(twApiBase() + '/margin/positions', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    pair_symbol: pair,
                    side: side,
                    collateral: collateral,
                    leverage: leverage,
                    chain_id: this.currentNetwork,
                    wallet_id: wallet.wallet_id !== undefined ? wallet.wallet_id : wallet.id
                })
            });
            if (!res.ok) { alert('Margin open failed: ' + await res.text()); return; }
            const pos = await res.json();
            alert('Margin position opened: ' + (pos.id || pos.position_id || 'confirmed') + ' — submitted to the blockchain network');
            this.loadTradingPositions();
        } catch (e) {
            alert('Margin open error: ' + e.message);
        }
    }

}

// Initialize app
document.addEventListener('DOMContentLoaded', () => {
    window.app = new TigerWalletApp();


});
