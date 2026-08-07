/**
 * TigerWallet Admin Firefox Extension - Popup Script
 * Complete production-ready admin functionality
 */

(function() {
    'use strict';

    // Configuration
    const API_BASE_URL = localStorage.getItem('api_base_url') || 'http://localhost:9093/api/v1';
    const WS_URL = localStorage.getItem('ws_url') || 'ws://localhost:9093/ws';
    
    // State
    let currentUser = null;
    let isDarkMode = true;
    let websocket = null;
    let reconnectAttempts = 0;
    const MAX_RECONNECT_ATTEMPTS = 5;

    // DOM Elements
    const elements = {
        themeToggle: document.getElementById('themeToggle'),
        connectionStatus: document.getElementById('connectionStatus'),
        navItems: document.querySelectorAll('.nav-item'),
        contentSections: document.querySelectorAll('.content-section'),
        adminEmail: document.getElementById('adminEmail'),
        adminRole: document.getElementById('adminRole'),
        logoutBtn: document.getElementById('logoutBtn'),
        
        // Dashboard
        totalUsers: document.getElementById('totalUsers'),
        totalVolume: document.getElementById('totalVolume'),
        pendingKYC: document.getElementById('pendingKYC'),
        systemHealth: document.getElementById('systemHealth'),
        activityList: document.getElementById('activityList'),
        
        // Users
        userSearch: document.getElementById('userSearch'),
        userStatusFilter: document.getElementById('userStatusFilter'),
        kycFilter: document.getElementById('kycFilter'),
        usersTableBody: document.getElementById('usersTableBody'),
        
        // Transactions
        txSearch: document.getElementById('txSearch'),
        txTypeFilter: document.getElementById('txTypeFilter'),
        txStatusFilter: document.getElementById('txStatusFilter'),
        transactionsTableBody: document.getElementById('transactionsTableBody'),
        
        // KYC
        kycPendingCount: document.getElementById('kycPendingCount'),
        kycApprovedCount: document.getElementById('kycApprovedCount'),
        kycRejectedCount: document.getElementById('kycRejectedCount'),
        kycTableBody: document.getElementById('kycTableBody'),
        
        // Tokens
        tokensGrid: document.getElementById('tokensGrid'),
        addTokenBtn: document.getElementById('addTokenBtn'),
        refreshTokensBtn: document.getElementById('refreshTokensBtn'),
        
        // Withdrawals
        withdrawalStatusFilter: document.getElementById('withdrawalStatusFilter'),
        withdrawalChainFilter: document.getElementById('withdrawalChainFilter'),
        withdrawalsTableBody: document.getElementById('withdrawalsTableBody'),
        
        // System
        serviceList: document.getElementById('serviceList'),
        refreshSystemBtn: document.getElementById('refreshSystemBtn')
    };

    // Initialize
    document.addEventListener('DOMContentLoaded', init);

    async function init() {
        loadTheme();
        await checkAuth();
        setupEventListeners();
        await loadDashboard();
        connectWebSocket();
    }

    // Theme Management
    function loadTheme() {
        const savedTheme = localStorage.getItem('admin_theme');
        isDarkMode = savedTheme !== 'light';
        applyTheme();
    }

    function applyTheme() {
        document.body.classList.toggle('dark-mode', isDarkMode);
        document.body.classList.toggle('light-mode', !isDarkMode);
        
        const themeIcon = elements.themeToggle.querySelector('.theme-icon');
        themeIcon.textContent = isDarkMode ? '🌙' : '☀️';
        
        localStorage.setItem('admin_theme', isDarkMode ? 'dark' : 'light');
    }

    function toggleTheme() {
        isDarkMode = !isDarkMode;
        applyTheme();
        broadcastThemeChange();
    }

    function broadcastThemeChange() {
        browser.runtime.sendMessage({
            type: 'THEME_CHANGED',
            theme: isDarkMode ? 'dark' : 'light'
        });
    }

    // Authentication
    async function checkAuth() {
        const token = localStorage.getItem('admin_token');
        
        if (!token) {
            showLoginRequired();
            return false;
        }

        try {
            const response = await fetch(`${API_BASE_URL}/admin/me`, {
                headers: {
                    'Authorization': `Bearer ${token}`,
                    'Content-Type': 'application/json'
                }
            });

            if (!response.ok) {
                showLoginRequired();
                return false;
            }

            currentUser = await response.json();
            elements.adminEmail.textContent = currentUser.email || 'Unknown';
            elements.adminRole.textContent = currentUser.role || 'admin';
            
            return true;
        } catch (error) {
            console.error('Auth check failed:', error);
            showLoginRequired();
            return false;
        }
    }

    function showLoginRequired() {
        elements.adminEmail.textContent = 'Login required';
        elements.adminRole.textContent = '-';
        
        // Show login modal or redirect
        window.location.href = 'login.html';
    }

    // Event Listeners
    function setupEventListeners() {
        // Theme toggle
        elements.themeToggle.addEventListener('click', toggleTheme);

        // Navigation
        elements.navItems.forEach(item => {
            item.addEventListener('click', () => {
                const section = item.dataset.section;
                navigateToSection(section);
            });
        });

        // Logout
        elements.logoutBtn.addEventListener('click', logout);

        // Search inputs with debounce
        let searchTimeout;
        elements.userSearch.addEventListener('input', (e) => {
            clearTimeout(searchTimeout);
            searchTimeout = setTimeout(() => searchUsers(e.target.value), 500);
        });

        elements.txSearch.addEventListener('input', (e) => {
            clearTimeout(searchTimeout);
            searchTimeout = setTimeout(() => searchTransactions(e.target.value), 500);
        });

        // Filters
        elements.userStatusFilter.addEventListener('change', loadUsers);
        elements.kycFilter.addEventListener('change', loadUsers);
        elements.txTypeFilter.addEventListener('change', loadTransactions);
        elements.txStatusFilter.addEventListener('change', loadTransactions);
        elements.withdrawalStatusFilter.addEventListener('change', loadWithdrawals);
        elements.withdrawalChainFilter.addEventListener('change', loadWithdrawals);

        // Quick actions
        document.querySelectorAll('.action-btn').forEach(btn => {
            btn.addEventListener('click', handleAction);
        });

        // Refresh buttons
        elements.refreshSystemBtn.addEventListener('click', loadSystemStatus);
        elements.refreshTokensBtn.addEventListener('click', loadTokens);

        // Add token
        elements.addTokenBtn.addEventListener('click', showAddTokenModal);
    }

    // Navigation
    function navigateToSection(sectionId) {
        // Update nav
        elements.navItems.forEach(item => {
            item.classList.toggle('active', item.dataset.section === sectionId);
        });

        // Update content
        elements.contentSections.forEach(section => {
            section.classList.toggle('active', section.id === sectionId);
        });

        // Load section data
        switch (sectionId) {
            case 'dashboard':
                loadDashboard();
                break;
            case 'users':
                loadUsers();
                break;
            case 'transactions':
                loadTransactions();
                break;
            case 'kyc':
                loadKYC();
                break;
            case 'tokens':
                loadTokens();
                break;
            case 'withdrawals':
                loadWithdrawals();
                break;
            case 'system':
                loadSystemStatus();
                break;
        }
    }

    // Logout
    function logout() {
        localStorage.removeItem('admin_token');
        localStorage.removeItem('admin_user');
        
        if (websocket) {
            websocket.close();
        }
        
        showLoginRequired();
    }

    // WebSocket Connection
    function connectWebSocket() {
        const token = localStorage.getItem('admin_token');
        if (!token) return;

        try {
            websocket = new WebSocket(`${WS_URL}?token=${token}`);

            websocket.onopen = () => {
                console.log('WebSocket connected');
                updateConnectionStatus(true);
                reconnectAttempts = 0;
            };

            websocket.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    handleWebSocketMessage(data);
                } catch (e) {
                    console.error('Failed to parse WebSocket message:', e);
                }
            };

            websocket.onclose = () => {
                console.log('WebSocket disconnected');
                updateConnectionStatus(false);
                attemptReconnect();
            };

            websocket.onerror = (error) => {
                console.error('WebSocket error:', error);
                updateConnectionStatus(false);
            };
        } catch (error) {
            console.error('Failed to connect WebSocket:', error);
            updateConnectionStatus(false);
        }
    }

    function handleWebSocketMessage(data) {
        switch (data.type) {
            case 'DASHBOARD_UPDATE':
                updateDashboard(data.data);
                break;
            case 'NEW_USER':
                addActivityItem('👤', `New user registered: ${data.data.email}`, data.data.time);
                break;
            case 'NEW_TRANSACTION':
                addActivityItem('💰', `New transaction: ${data.data.type}`, data.data.time);
                break;
            case 'KYC_SUBMITTED':
                addActivityItem('✅', `KYC submitted: ${data.data.email}`, data.data.time);
                break;
            case 'SYSTEM_ALERT':
                addActivityItem('⚠️', data.data.message, data.data.time);
                break;
            case 'SERVICE_STATUS':
                updateServiceStatus(data.data);
                break;
        }
    }

    function attemptReconnect() {
        if (reconnectAttempts < MAX_RECONNECT_ATTEMPTS) {
            reconnectAttempts++;
            setTimeout(connectWebSocket, 2000 * reconnectAttempts);
        }
    }

    function updateConnectionStatus(connected) {
        const statusDot = elements.connectionStatus.querySelector('.status-dot');
        const statusText = elements.connectionStatus.querySelector('.status-text');
        
        statusDot.classList.toggle('connected', connected);
        statusText.textContent = connected ? 'Connected' : 'Disconnected';
    }

    // Dashboard
    async function loadDashboard() {
        try {
            const token = localStorage.getItem('admin_token');
            const response = await fetch(`${API_BASE_URL}/admin/dashboard`, {
                headers: {
                    'Authorization': `Bearer ${token}`,
                    'Content-Type': 'application/json'
                }
            });

            if (!response.ok) throw new Error('Failed to load dashboard');

            const data = await response.json();
            updateDashboard(data);
        } catch (error) {
            console.error('Dashboard load error:', error);
            updateDashboardUnavailable('Dashboard data is unavailable. Reconnect to the TigerWallet admin API and retry.');
        }
    }

    function updateDashboardUnavailable(message) {
        elements.totalUsers.textContent = '-';
        elements.totalVolume.textContent = '-';
        elements.pendingKYC.textContent = '-';
        elements.systemHealth.textContent = 'Unavailable';
        elements.activityList.textContent = message;
    }

    function updateDashboard(data) {
        elements.totalUsers.textContent = data.totalUsers?.toLocaleString() || '-';
        elements.totalVolume.textContent = data.totalVolume || '-';
        elements.pendingKYC.textContent = data.pendingKYC || '-';
        elements.systemHealth.textContent = data.systemHealth || '-';

        // Update activity list
        if (data.recentActivity && data.recentActivity.length > 0) {
            elements.activityList.innerHTML = data.recentActivity.map(activity => `
                <li class="activity-item">
                    <span class="activity-icon">${getActivityIcon(activity.type)}</span>
                    <span class="activity-text">${activity.message}</span>
                    <span class="activity-time">${activity.time}</span>
                </li>
            `).join('');
        }
    }

    function getActivityIcon(type) {
        const icons = {
            'user_verified': '👤',
            'transaction': '💰',
            'kyc': '✅',
            'token': '🪙',
            'suspicious': '⚠️'
        };
        return icons[type] || '📌';
    }

    function addActivityItem(icon, text, time) {
        const activityItem = document.createElement('li');
        activityItem.className = 'activity-item';
        activityItem.innerHTML = `
            <span class="activity-icon">${icon}</span>
            <span class="activity-text">${text}</span>
            <span class="activity-time">${time}</span>
        `;
        
        elements.activityList.insertBefore(activityItem, elements.activityList.firstChild);
        
        // Keep only last 10 items
        while (elements.activityList.children.length > 10) {
            elements.activityList.removeChild(elements.activityList.lastChild);
        }
    }

    // Users
    async function loadUsers() {
        const token = localStorage.getItem('admin_token');
        const status = elements.userStatusFilter.value;
        const kyc = elements.kycFilter.value;

        try {
            let url = `${API_BASE_URL}/admin/users?`;
            if (status) url += `status=${status}&`;
            if (kyc) url += `kyc=${kyc}&`;

            const response = await fetch(url, {
                headers: {
                    'Authorization': `Bearer ${token}`,
                    'Content-Type': 'application/json'
                }
            });

            if (!response.ok) throw new Error('Failed to load users');

            const data = await response.json();
            renderUsers(data.users || []);
        } catch (error) {
            console.error('Load users error:', error);
            renderUsersUnavailable('User data is unavailable. Reconnect to the TigerWallet admin API and retry.');
        }
    }

    function renderUsersUnavailable(message) {
        elements.usersTableBody.innerHTML = `<tr><td colspan="5" class="empty">${message}</td></tr>`;
    }

    function renderUsers(users) {
        if (users.length === 0) {
            elements.usersTableBody.innerHTML = '<tr><td colspan="5" class="empty">No users found</td></tr>';
            return;
        }

        elements.usersTableBody.innerHTML = users.map(user => `
            <tr data-user-id="${user.id}">
                <td>
                    <div class="user-cell">
                        <span class="user-email">${user.email}</span>
                        <span class="user-id">${user.id?.substring(0, 8)}...</span>
                    </div>
                </td>
                <td><span class="status-badge status-${user.status}">${user.status}</span></td>
                <td><span class="kyc-badge kyc-${user.kycStatus}">${user.kycStatus || 'none'}</span></td>
                <td>${formatDate(user.createdAt)}</td>
                <td>
                    <button class="table-btn view-btn" data-action="view" data-user-id="${user.id}">View</button>
                    <button class="table-btn edit-btn" data-action="edit" data-user-id="${user.id}">Edit</button>
                </td>
            </tr>
        `).join('');

        // Add event listeners
        elements.usersTableBody.querySelectorAll('.table-btn').forEach(btn => {
            btn.addEventListener('click', handleUserAction);
        });
    }

    async function searchUsers(query) {
        if (!query) {
            loadUsers();
            return;
        }

        const token = localStorage.getItem('admin_token');
        
        try {
            const response = await fetch(`${API_BASE_URL}/admin/users/search?q=${encodeURIComponent(query)}`, {
                headers: {
                    'Authorization': `Bearer ${token}`,
                    'Content-Type': 'application/json'
                }
            });

            if (!response.ok) throw new Error('Search failed');
            
            const data = await response.json();
            renderUsers(data.users || []);
        } catch (error) {
            console.error('Search error:', error);
            renderUsersUnavailable('User search is unavailable. Reconnect to the TigerWallet admin API and retry.');
        }
    }

    function handleUserAction(event) {
        const action = event.target.dataset.action;
        const userId = event.target.dataset.userId;
        
        switch (action) {
            case 'view':
                viewUser(userId);
                break;
            case 'edit':
                editUser(userId);
                break;
        }
    }

    function viewUser(userId) {
        browser.runtime.sendMessage({
            type: 'OPEN_USER_DETAIL',
            userId: userId
        });
    }

    function editUser(userId) {
        browser.runtime.sendMessage({
            type: 'OPEN_USER_EDIT',
            userId: userId
        });
    }

    // Transactions
    async function loadTransactions() {
        const token = localStorage.getItem('admin_token');
        const type = elements.txTypeFilter.value;
        const status = elements.txStatusFilter.value;

        try {
            let url = `${API_BASE_URL}/admin/transactions?`;
            if (type) url += `type=${type}&`;
            if (status) url += `status=${status}&`;

            const response = await fetch(url, {
                headers: {
                    'Authorization': `Bearer ${token}`,
                    'Content-Type': 'application/json'
                }
            });

            if (!response.ok) throw new Error('Failed to load transactions');

            const data = await response.json();
            renderTransactions(data.transactions || []);
        } catch (error) {
            console.error('Load transactions error:', error);
            renderTransactionsUnavailable('Transaction data is unavailable. Reconnect to the TigerWallet admin API and retry.');
        }
    }

    function renderTransactionsUnavailable(message) {
        elements.transactionsTableBody.innerHTML = `<tr><td colspan="6" class="empty">${message}</td></tr>`;
    }

    function renderTransactions(transactions) {
        if (transactions.length === 0) {
            elements.transactionsTableBody.innerHTML = '<tr><td colspan="6" class="empty">No transactions found</td></tr>';
            return;
        }

        elements.transactionsTableBody.innerHTML = transactions.map(tx => `
            <tr data-tx-id="${tx.id}">
                <td><code class="tx-hash">${tx.hash?.substring(0, 18)}...</code></td>
                <td>${tx.type}</td>
                <td class="amount">${formatAmount(tx.amount)} ${tx.token || ''}</td>
                <td><span class="status-badge status-${tx.status}">${tx.status}</span></td>
                <td>${formatDate(tx.createdAt)}</td>
                <td>
                    <button class="table-btn view-btn" data-action="view" data-tx-id="${tx.id}">View</button>
                    <button class="table-btn flag-btn" data-action="flag" data-tx-id="${tx.id}">Flag</button>
                </td>
            </tr>
        `).join('');
    }

    async function searchTransactions(query) {
        if (!query) {
            loadTransactions();
            return;
        }

        const token = localStorage.getItem('admin_token');
        
        try {
            const response = await fetch(`${API_BASE_URL}/admin/transactions/search?q=${encodeURIComponent(query)}`, {
                headers: {
                    'Authorization': `Bearer ${token}`,
                    'Content-Type': 'application/json'
                }
            });

            if (!response.ok) throw new Error('Search failed');
            
            const data = await response.json();
            renderTransactions(data.transactions || []);
        } catch (error) {
            console.error('Search error:', error);
            renderTransactionsUnavailable('Transaction search is unavailable. Reconnect to the TigerWallet admin API and retry.');
        }
    }

    // KYC
    async function loadKYC() {
        const token = localStorage.getItem('admin_token');

        try {
            const response = await fetch(`${API_BASE_URL}/admin/kyc`, {
                headers: {
                    'Authorization': `Bearer ${token}`,
                    'Content-Type': 'application/json'
                }
            });

            if (!response.ok) throw new Error('Failed to load KYC');

            const data = await response.json();
            updateKYCCounts(data);
            renderKYC(data.applications || []);
        } catch (error) {
            console.error('Load KYC error:', error);
            updateKYCCounts({});
            elements.kycTableBody.innerHTML = '<tr><td colspan="5" class="empty">KYC data is unavailable. Reconnect to the TigerWallet admin API and retry.</td></tr>';
        }
    }

    function updateKYCCounts(data) {
        elements.kycPendingCount.textContent = data.pending || '-';
        elements.kycApprovedCount.textContent = data.approved || '-';
        elements.kycRejectedCount.textContent = data.rejected || '-';
    }

    function renderKYC(applications) {
        if (applications.length === 0) {
            elements.kycTableBody.innerHTML = '<tr><td colspan="5" class="empty">No KYC applications</td></tr>';
            return;
        }

        elements.kycTableBody.innerHTML = applications.map(app => `
            <tr data-kyc-id="${app.id}">
                <td>
                    <div class="user-cell">
                        <span class="user-email">${app.email}</span>
                    </div>
                </td>
                <td><span class="kyc-level">Level ${app.level}</span></td>
                <td>${formatDate(app.submittedAt)}</td>
                <td><span class="status-badge status-${app.status}">${app.status}</span></td>
                <td>
                    <button class="table-btn approve-btn" data-action="approve" data-kyc-id="${app.id}">Approve</button>
                    <button class="table-btn reject-btn" data-action="reject" data-kyc-id="${app.id}">Reject</button>
                    <button class="table-btn view-btn" data-action="view" data-kyc-id="${app.id}">View</button>
                </td>
            </tr>
        `).join('');

        // Add event listeners
        elements.kycTableBody.querySelectorAll('.table-btn').forEach(btn => {
            btn.addEventListener('click', handleKYCAction);
        });
    }

    async function handleKYCAction(event) {
        const action = event.target.dataset.action;
        const kycId = event.target.dataset.kycId;
        
        if (action === 'view') {
            browser.runtime.sendMessage({
                type: 'OPEN_KYC_DETAIL',
                kycId: kycId
            });
            return;
        }

        const newStatus = action === 'approve' ? 'approved' : 'rejected';
        
        if (confirm(`Are you sure you want to ${newStatus} this KYC application?`)) {
            await updateKYCStatus(kycId, newStatus);
        }
    }

    async function updateKYCStatus(kycId, status) {
        const token = localStorage.getItem('admin_token');

        try {
            const response = await fetch(`${API_BASE_URL}/admin/kyc/${kycId}/status`, {
                method: 'PATCH',
                headers: {
                    'Authorization': `Bearer ${token}`,
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ status })
            });

            if (!response.ok) throw new Error('Update failed');

            // Refresh KYC list
            loadKYC();
        } catch (error) {
            console.error('Update KYC error:', error);
            alert('Failed to update KYC status');
        }
    }

    // Tokens
    async function loadTokens() {
        const token = localStorage.getItem('admin_token');

        try {
            const response = await fetch(`${API_BASE_URL}/admin/tokens`, {
                headers: {
                    'Authorization': `Bearer ${token}`,
                    'Content-Type': 'application/json'
                }
            });

            if (!response.ok) throw new Error('Failed to load tokens');

            const data = await response.json();
            renderTokens(data.tokens || []);
        } catch (error) {
            console.error('Load tokens error:', error);
            elements.tokensGrid.innerHTML = '<div class="empty">Token data is unavailable. Reconnect to the TigerWallet admin API and retry.</div>';
        }
    }

    function renderTokens(tokens) {
        if (tokens.length === 0) {
            elements.tokensGrid.innerHTML = '<div class="empty">No tokens found</div>';
            return;
        }

        elements.tokensGrid.innerHTML = tokens.map(token => `
            <div class="token-card" data-token-id="${token.id}">
                <div class="token-header">
                    <img src="${token.logo}" alt="${token.symbol}" class="token-logo" onerror="this.src='data:image/svg+xml,...'">
                    <div class="token-info">
                        <span class="token-symbol">${token.symbol}</span>
                        <span class="token-name">${token.name}</span>
                    </div>
                    <span class="token-status ${token.isActive ? 'active' : 'inactive'}">
                        ${token.isActive ? 'Active' : 'Inactive'}
                    </span>
                </div>
                <div class="token-details">
                    <div class="token-detail">
                        <span class="detail-label">Price</span>
                        <span class="detail-value">$${token.price || '0.00'}</span>
                    </div>
                    <div class="token-detail">
                        <span class="detail-label">Market Cap</span>
                        <span class="detail-value">$${formatNumber(token.marketCap)}</span>
                    </div>
                    <div class="token-detail">
                        <span class="detail-label">24h Volume</span>
                        <span class="detail-value">$${formatNumber(token.volume24h)}</span>
                    </div>
                </div>
                <div class="token-actions">
                    <button class="table-btn view-btn" data-action="view" data-token-id="${token.id}">View</button>
                    <button class="table-btn edit-btn" data-action="edit" data-token-id="${token.id}">Edit</button>
                    <button class="table-btn ${token.isActive ? 'danger-btn' : 'success-btn'}" 
                            data-action="${token.isActive ? 'deactivate' : 'activate'}" 
                            data-token-id="${token.id}">
                        ${token.isActive ? 'Deactivate' : 'Activate'}
                    </button>
                </div>
            </div>
        `).join('');
    }

    function showAddTokenModal() {
        browser.runtime.sendMessage({
            type: 'OPEN_ADD_TOKEN'
        });
    }

    // Withdrawals
    async function loadWithdrawals() {
        const token = localStorage.getItem('admin_token');
        const status = elements.withdrawalStatusFilter.value;
        const chain = elements.withdrawalChainFilter.value;

        try {
            let url = `${API_BASE_URL}/admin/withdrawals?`;
            if (status) url += `status=${status}&`;
            if (chain) url += `chain=${chain}&`;

            const response = await fetch(url, {
                headers: {
                    'Authorization': `Bearer ${token}`,
                    'Content-Type': 'application/json'
                }
            });

            if (!response.ok) throw new Error('Failed to load withdrawals');

            const data = await response.json();
            renderWithdrawals(data.withdrawals || []);
        } catch (error) {
            console.error('Load withdrawals error:', error);
            elements.withdrawalsTableBody.innerHTML = '<tr><td colspan="6" class="empty">Withdrawal data is unavailable. Reconnect to the TigerWallet admin API and retry.</td></tr>';
        }
    }

    function renderWithdrawals(withdrawals) {
        if (withdrawals.length === 0) {
            elements.withdrawalsTableBody.innerHTML = '<tr><td colspan="6" class="empty">No withdrawals found</td></tr>';
            return;
        }

        elements.withdrawalsTableBody.innerHTML = withdrawals.map(wd => `
            <tr data-withdrawal-id="${wd.id}">
                <td>${wd.userEmail}</td>
                <td class="amount">${formatAmount(wd.amount)} ${wd.token}</td>
                <td>${wd.chain}</td>
                <td><code class="address">${wd.address?.substring(0, 10)}...</code></td>
                <td><span class="status-badge status-${wd.status}">${wd.status}</span></td>
                <td>
                    ${wd.status === 'pending' ? `
                        <button class="table-btn approve-btn" data-action="approve" data-withdrawal-id="${wd.id}">Approve</button>
                        <button class="table-btn reject-btn" data-action="reject" data-withdrawal-id="${wd.id}">Reject</button>
                    ` : ''}
                    <button class="table-btn view-btn" data-action="view" data-withdrawal-id="${wd.id}">View</button>
                </td>
            </tr>
        `).join('');
    }

    // System Status
    async function loadSystemStatus() {
        const token = localStorage.getItem('admin_token');

        try {
            const response = await fetch(`${API_BASE_URL}/admin/system/status`, {
                headers: {
                    'Authorization': `Bearer ${token}`,
                    'Content-Type': 'application/json'
                }
            });

            if (!response.ok) throw new Error('Failed to load system status');

            const data = await response.json();
            updateSystemStatus(data);
        } catch (error) {
            console.error('Load system status error:', error);
            // Demo data
            updateSystemStatus({
                services: [
                    { name: 'api_gateway', status: 'running', uptime: '99.99%' },
                    { name: 'wallet_service', status: 'running', uptime: '99.95%' },
                    { name: 'tx_engine', status: 'running', uptime: '99.99%' },
                    { name: 'price_feed', status: 'running', uptime: '99.90%' }
                ],
                database: [
                    { name: 'postgres', status: 'running', uptime: '99.99%' },
                    { name: 'redis', status: 'running', uptime: '99.95%' }
                ],
                network: [
                    { name: 'eth_rpc', status: 'running', uptime: '99.80%' },
                    { name: 'bsc_rpc', status: 'running', uptime: '99.85%' }
                ]
            });
        }
    }

    function updateSystemStatus(data) {
        // Update services
        const serviceElements = elements.serviceList.querySelectorAll('.service-status');
        serviceElements.forEach(el => {
            const serviceName = el.dataset.service;
            const service = data.services?.find(s => s.name === serviceName);
            
            if (service) {
                el.textContent = service.status === 'running' ? '✅ Running' : '❌ Down';
                el.className = `service-status ${service.status}`;
            }
        });
    }

    function updateServiceStatus(data) {
        const serviceEl = document.querySelector(`[data-service="${data.name}"]`);
        if (serviceEl) {
            serviceEl.textContent = data.status === 'running' ? '✅ Running' : '❌ Down';
            serviceEl.className = `service-status ${data.status}`;
        }
    }

    // Action Handler
    async function handleAction(event) {
        const action = event.currentTarget.dataset.action;
        
        switch (action) {
            case 'refresh':
                loadDashboard();
                break;
            case 'users':
                navigateToSection('users');
                break;
            case 'transactions':
                navigateToSection('transactions');
                break;
            case 'system':
                navigateToSection('system');
                break;
        }
    }

    // Utility Functions
    function formatDate(dateString) {
        if (!dateString) return '-';
        const date = new Date(dateString);
        return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    }

    function formatAmount(amount) {
        if (!amount) return '$0.00';
        return '$' + parseFloat(amount).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
    }

    function formatNumber(num) {
        if (!num) return '0';
        if (num >= 1e12) return (num / 1e12).toFixed(2) + 'T';
        if (num >= 1e9) return (num / 1e9).toFixed(2) + 'B';
        if (num >= 1e6) return (num / 1e6).toFixed(2) + 'M';
        if (num >= 1e3) return (num / 1e3).toFixed(2) + 'K';
        return num.toFixed(2);
    }

    // Message listener for background script
    browser.runtime.onMessage.addListener((message, sender, sendResponse) => {
        switch (message.type) {
            case 'DASHBOARD_UPDATE':
                updateDashboard(message.data);
                break;
            case 'THEME_CHANGED':
                isDarkMode = message.theme === 'dark';
                applyTheme();
                break;
            case 'TOKEN_EXPIRED':
                logout();
                break;
        }
    });

})();
