/**
 * TigerWallet Chrome Extension - DApp Browser Service
 * Production-ready decentralized application browser
 */

class DAppBrowserService {
    constructor() {
        this.dapps = new Map();
        this.history = [];
        this.bookmarks = new Set();
        this.tabs = new Map();
        this.activeTabId = null;
        this.whitelist = new Set();
        this.initialized = false;
    }

    async initialize() {
        if (this.initialized) return true;
        
        await this.loadData();
        this.initialized = true;
        console.log('[DAppBrowser] Service initialized');
        return true;
    }

    async loadData() {
        try {
            const result = await chrome.storage.local.get([
                'dapp_history',
                'dapp_bookmarks',
                'dapp_whitelist'
            ]);
            
            if (result.dapp_history) {
                this.history = result.dapp_history;
            }
            
            if (result.dapp_bookmarks) {
                this.bookmarks = new Set(result.dapp_bookmarks);
            }
            
            if (result.dapp_whitelist) {
                this.whitelist = new Set(result.dapp_whitelist);
            }
        } catch (error) {
            console.error('[DAppBrowser] Load failed:', error);
        }
    }

    async saveData() {
        try {
            await chrome.storage.local.set({
                dapp_history: this.history.slice(0, 100),
                dapp_bookmarks: Array.from(this.bookmarks),
                dapp_whitelist: Array.from(this.whitelist)
            });
        } catch (error) {
            console.error('[DAppBrowser] Save failed:', error);
        }
    }

    /**
     * Open DApp in new tab
     */
    async openTab(url, title = '') {
        const tabId = 'tab_' + Date.now();
        
        const tab = {
            id: tabId,
            url: this.normalizeUrl(url),
            title: title || this.extractDomain(url),
            favicon: await this.getFavicon(url),
            createdAt: Date.now(),
            lastActiveAt: Date.now(),
            history: [],
            isLoading: false
        };

        this.tabs.set(tabId, tab);
        this.activeTabId = tabId;

        // Add to history
        this.addToHistory(tab.url, tab.title);

        await this.saveData();
        return tab;
    }

    /**
     * Close tab
     */
    async closeTab(tabId) {
        if (this.tabs.has(tabId)) {
            this.tabs.delete(tabId);
            
            if (this.activeTabId === tabId) {
                const tabs = Array.from(this.tabs.keys());
                this.activeTabId = tabs.length > 0 ? tabs[0] : null;
            }
            
            await this.saveData();
            return true;
        }
        return false;
    }

    /**
     * Get all tabs
     */
    getTabs() {
        return Array.from(this.tabs.values());
    }

    /**
     * Get active tab
     */
    getActiveTab() {
        return this.activeTabId ? this.tabs.get(this.activeTabId) : null;
    }

    /**
     * Set active tab
     */
    setActiveTab(tabId) {
        if (this.tabs.has(tabId)) {
            const tab = this.tabs.get(tabId);
            tab.lastActiveAt = Date.now();
            this.activeTabId = tabId;
            return tab;
        }
        return null;
    }

    /**
     * Navigate tab to URL
     */
    async navigateTab(tabId, url) {
        const tab = this.tabs.get(tabId);
        if (!tab) throw new Error('Tab not found');

        tab.url = this.normalizeUrl(url);
        tab.title = this.extractDomain(url);
        tab.favicon = await this.getFavicon(url);
        tab.isLoading = false;

        this.addToHistory(tab.url, tab.title);
        await this.saveData();

        return tab;
    }

    /**
     * Get history
     */
    getHistory(limit = 50) {
        return this.history.slice(0, limit);
    }

    /**
     * Clear history
     */
    async clearHistory() {
        this.history = [];
        await this.saveData();
    }

    /**
     * Add to history
     */
    addToHistory(url, title) {
        const entry = {
            url,
            title,
            timestamp: Date.now()
        };

        // Remove duplicate if exists
        const existingIndex = this.history.findIndex(h => h.url === url);
        if (existingIndex !== -1) {
            this.history.splice(existingIndex, 1);
        }

        // Add to beginning
        this.history.unshift(entry);

        // Keep only last 100
        if (this.history.length > 100) {
            this.history = this.history.slice(0, 100);
        }
    }

    /**
     * Add bookmark
     */
    async addBookmark(url, title) {
        const bookmark = {
            url: this.normalizeUrl(url),
            title: title || this.extractDomain(url),
            favicon: await this.getFavicon(url),
            addedAt: Date.now()
        };

        this.bookmarks.add(bookmark.url);
        await this.saveData();

        return bookmark;
    }

    /**
     * Remove bookmark
     */
    async removeBookmark(url) {
        if (this.bookmarks.has(url)) {
            this.bookmarks.delete(url);
            await this.saveData();
            return true;
        }
        return false;
    }

    /**
     * Get bookmarks
     */
    getBookmarks() {
        return Array.from(this.bookmarks);
    }

    /**
     * Check if bookmarked
     */
    isBookmarked(url) {
        return this.bookmarks.has(this.normalizeUrl(url));
    }

    /**
     * Add to whitelist
     */
    async addToWhitelist(url) {
        this.whitelist.add(this.normalizeUrl(url));
        await this.saveData();
    }

    /**
     * Remove from whitelist
     */
    async removeFromWhitelist(url) {
        if (this.whitelist.has(url)) {
            this.whitelist.delete(url);
            await this.saveData();
            return true;
        }
        return false;
    }

    /**
     * Check if whitelisted
     */
    isWhitelisted(url) {
        const normalizedUrl = this.normalizeUrl(url);
        return this.whitelist.has(normalizedUrl);
    }

    /**
     * Get whitelist
     */
    getWhitelist() {
        return Array.from(this.whitelist);
    }

    /**
     * Connect to DApp (WalletConnect style)
     */
    async connectDApp(dappUrl, chainId = 1) {
        const connection = {
            id: this.generateId(),
            dappUrl: this.normalizeUrl(dappUrl),
            chainId,
            connected: true,
            accounts: [],
            connectedAt: Date.now()
        };

        // In production, handle proper wallet connection
        return connection;
    }

    /**
     * Disconnect from DApp
     */
    async disconnectDApp(dappUrl) {
        // Handle disconnection
        return true;
    }

    /**
     * Get connected DApps
     */
    getConnectedDApps() {
        // Return connected DApps
        return [];
    }

    /**
     * Sign transaction request from DApp
     */
    async signTransactionRequest(dappUrl, txParams) {
        // Check whitelist
        if (!this.isWhitelisted(dappUrl)) {
            throw new Error('DApp not whitelisted');
        }

        const request = {
            id: this.generateId(),
            dappUrl: this.normalizeUrl(dappUrl),
            type: 'transaction',
            params: txParams,
            status: 'pending',
            createdAt: Date.now()
        };

        // Notify user and wait for approval
        return request;
    }

    /**
     * Sign message request from DApp
     */
    async signMessageRequest(dappUrl, message) {
        if (!this.isWhitelisted(dappUrl)) {
            throw new Error('DApp not whitelisted');
        }

        const request = {
            id: this.generateId(),
            dappUrl: this.normalizeUrl(dappUrl),
            type: 'sign',
            message,
            status: 'pending',
            createdAt: Date.now()
        };

        return request;
    }

    /**
     * Handle DApp request
     */
    async handleDAppRequest(request, approved) {
        if (approved) {
            // Execute the request
            if (request.type === 'transaction') {
                return await this.executeTransaction(request.params);
            } else if (request.type === 'sign') {
                return await this.signMessage(request.message);
            }
        } else {
            throw new Error('Request rejected');
        }
    }

    /**
     * Execute transaction (placeholder)
     */
    async executeTransaction(params) {
        // In production, sign and broadcast transaction
        return {
            txHash: '0x' + this.generateId(),
            status: 'pending'
        };
    }

    /**
     * Sign message (placeholder)
     */
    async signMessage(message) {
        // In production, use wallet to sign
        return {
            signature: '0x' + this.generateId()
        };
    }

    /**
     * Get popular DApps
     */
    getPopularDApps() {
        return [
            { name: 'Uniswap', url: 'https://app.uniswap.org', category: 'DEX' },
            { name: 'OpenSea', url: 'https://opensea.io', category: 'NFT' },
            { name: 'Aave', url: 'https://app.aave.com', category: 'Lending' },
            { name: 'Compound', url: 'https://app.compound.finance', category: 'Lending' },
            { name: 'Curve', url: 'https://curve.fi', category: 'DEX' },
            { name: '1inch', url: 'https://app.1inch.io', category: 'DEX' },
            { name: ' ENS', url: 'https://app.ens.domains', category: 'Domain' },
            { name: 'LooksRare', url: 'https://looksrare.org', category: 'NFT' },
            { name: 'Magic Eden', url: 'https://magiceden.io', category: 'NFT' },
            { name: 'Sushiswap', url: 'https://app.sushi.com', category: 'DEX' }
        ];
    }

    /**
     * Normalize URL
     */
    normalizeUrl(url) {
        try {
            if (!url.startsWith('http://') && !url.startsWith('https://')) {
                url = 'https://' + url;
            }
            return new URL(url).href;
        } catch {
            return url;
        }
    }

    /**
     * Extract domain from URL
     */
    extractDomain(url) {
        try {
            return new URL(this.normalizeUrl(url)).hostname;
        } catch {
            return url;
        }
    }

    /**
     * Get favicon URL
     */
    async getFavicon(url) {
        const domain = this.extractDomain(url);
        return `https://www.google.com/s2/favicons?domain=${domain}&sz=64`;
    }

    generateId() {
        return Array.from(crypto.getRandomValues(new Uint8Array(16)))
            .map(b => b.toString(16).padStart(2, '0'))
            .join('');
    }
}

window.DAppBrowserService = new DAppBrowserService();
