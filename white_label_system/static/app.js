/**
 * TigerWallet White Label System - JavaScript
 * Light/Dark Theme Support & Full Functionality
 */

// API Base URL
const API_BASE = '/api/v1';

// State
let currentUser = null;
let token = localStorage.getItem('token') || null;
let refreshToken = localStorage.getItem('refreshToken') || null;

// =============================================================================
// THEME MANAGEMENT
// =============================================================================

function initTheme() {
    // Check for saved theme preference or default to light
    const savedTheme = localStorage.getItem('theme') || 'light';
    document.documentElement.setAttribute('data-theme', savedTheme);
    updateThemeButton(savedTheme);
}

function toggleTheme() {
    const currentTheme = document.documentElement.getAttribute('data-theme');
    const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
    
    document.documentElement.setAttribute('data-theme', newTheme);
    localStorage.setItem('theme', newTheme);
    updateThemeButton(newTheme);
}

function updateThemeButton(theme) {
    const buttons = document.querySelectorAll('#theme-toggle-btn');
    buttons.forEach(btn => {
        const icon = btn.querySelector('.theme-icon');
        const text = btn.querySelector('.theme-text');
        if (icon) icon.textContent = theme === 'dark' ? '☀️' : '🌙';
        if (text) text.textContent = theme === 'dark' ? 'Switch to Light Mode' : 'Switch to Dark Mode';
    });
}

// =============================================================================
// AUTHENTICATION
// =============================================================================

async function login(email, password, twoFACode = '') {
    try {
        const response = await fetch(`${API_BASE}/auth/login`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ email, password, twoFACode }),
        });

        const data = await response.json();

        if (!response.ok) {
            if (data.requires2FA) {
                document.getElementById('2fa-group').style.display = 'block';
                throw new Error('Please enter your 2FA code');
            }
            throw new Error(data.error || 'Login failed');
        }

        if (data.requires2FA) {
            document.getElementById('2fa-group').style.display = 'block';
            return;
        }

        // Save tokens
        token = data.token;
        refreshToken = data.refreshToken;
        localStorage.setItem('token', token);
        localStorage.setItem('refreshToken', refreshToken);
        currentUser = data.user;

        // Show dashboard
        showDashboard();
        loadDashboardData();

    } catch (error) {
        document.getElementById('login-error').textContent = error.message;
    }
}

async function logout() {
    try {
        await fetch(`${API_BASE}/auth/logout`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${token}`,
            },
        });
    } catch (e) {
        console.error('Logout error:', e);
    }

    // Clear tokens and user
    token = null;
    refreshToken = null;
    currentUser = null;
    localStorage.removeItem('token');
    localStorage.removeItem('refreshToken');

    // Show login
    document.getElementById('login-section').style.display = 'block';
    document.getElementById('dashboard-section').style.display = 'none';
}

async function refreshTokenIfNeeded() {
    if (!refreshToken) return false;

    try {
        const response = await fetch(`${API_BASE}/auth/refresh`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ refreshToken }),
        });

        if (!response.ok) {
            logout();
            return false;
        }

        const data = await response.json();
        token = data.token;
        refreshToken = data.refreshToken;
        localStorage.setItem('token', token);
        localStorage.setItem('refreshToken', refreshToken);

        return true;
    } catch (e) {
        logout();
        return false;
    }
}

// =============================================================================
// UI HELPERS
// =============================================================================

function showDashboard() {
    document.getElementById('login-section').style.display = 'none';
    document.getElementById('dashboard-section').style.display = 'block';
    
    if (currentUser) {
        document.getElementById('user-info').textContent = 
            `${currentUser.name || currentUser.email} (${currentUser.role})`;
    }
}

function showTab(tabId) {
    // Hide all tabs
    document.querySelectorAll('.tab-content').forEach(tab => {
        tab.style.display = 'none';
    });
    
    // Show selected tab
    document.getElementById(tabId).style.display = 'block';
    
    // Update nav
    document.querySelectorAll('.dashboard-nav a').forEach(link => {
        link.classList.remove('active');
    });
    document.querySelector(`.dashboard-nav a[href="#${tabId}"]`).classList.add('active');
    
    // Load tab data
    loadTabData(tabId);
}

function showModal(title, content) {
    document.getElementById('modal-title').textContent = title;
    document.getElementById('modal-body').innerHTML = content;
    document.getElementById('modal-overlay').style.display = 'flex';
}

function closeModal() {
    document.getElementById('modal-overlay').style.display = 'none';
}

// =============================================================================
// DATA LOADING
// =============================================================================

async function loadDashboardData() {
    await Promise.all([
        loadOverviewData(),
        loadClients(),
        loadAdmins(),
        loadProducts(),
        loadRevenue()
    ]);
}

async function loadTabData(tabId) {
    switch(tabId) {
        case 'overview':
            await loadOverviewData();
            break;
        case 'clients':
            await loadClients();
            break;
        case 'admins':
            await loadAdmins();
            break;
        case 'products':
            await loadProducts();
            break;
        case 'fetchers':
            await loadFetchers();
            break;
        case 'revenue':
            await loadRevenue();
            break;
        case 'settings':
            loadSettings();
            break;
    }
}

async function loadOverviewData() {
    try {
        const response = await fetchWithAuth(`${API_BASE}/super-admin/dashboard`);
        const data = await response.json();

        document.getElementById('total-clients').textContent = data.totalClients || 0;
        document.getElementById('authorized-clients').textContent = data.authorizedClients || 0;
        document.getElementById('suspended-clients').textContent = data.suspendedClients || 0;
        document.getElementById('total-revenue').textContent = formatCurrency(data.totalRevenue || 0);
        document.getElementById('super-admin-share').textContent = formatCurrency(data.superAdminShare || 0);

    } catch (error) {
        console.error('Failed to load overview:', error);
    }

    // Load activity log
    try {
        const response = await fetchWithAuth(`${API_BASE}/super-admin/logs`);
        const logs = await response.json();

        const tbody = document.getElementById('activity-log');
        tbody.innerHTML = logs.slice(0, 10).map(log => `
            <tr>
                <td>${formatDate(log.timestamp)}</td>
                <td>${log.action}</td>
                <td>${log.details}</td>
                <td>${log.ipAddress}</td>
            </tr>
        `).join('');
    } catch (error) {
        console.error('Failed to load logs:', error);
    }
}

async function loadClients() {
    try {
        const response = await fetchWithAuth(`${API_BASE}/super-admin/clients`);
        const clients = await response.json();

        const tbody = document.getElementById('clients-list');
        tbody.innerHTML = clients.map(client => `
            <tr>
                <td>${client.name}</td>
                <td>${client.domain}</td>
                <td><span class="status-badge ${client.status}">${client.status}</span></td>
                <td>${formatDate(client.createdAt)}</td>
                <td class="actions">
                    ${client.status === 'pending' ? 
                        `<button class="btn btn-success" onclick="authorizeClient('${client.id}')">Authorize</button>` : ''}
                    ${client.status === 'authorized' ? 
                        `<button class="btn btn-danger" onclick="suspendClient('${client.id}')">Suspend</button>` : ''}
                    ${client.status === 'suspended' ? 
                        `<button class="btn btn-success" onclick="resumeClient('${client.id}')">Resume</button>` : ''}
                    ${client.status !== 'halted' ? 
                        `<button class="btn btn-danger" onclick="haltClient('${client.id}')">Halt</button>` : ''}
                </td>
            </tr>
        `).join('');
    } catch (error) {
        console.error('Failed to load clients:', error);
    }
}

async function loadAdmins() {
    try {
        const response = await fetchWithAuth(`${API_BASE}/white-label/admins`);
        const admins = await response.json();

        const tbody = document.getElementById('admins-list');
        tbody.innerHTML = admins.map(admin => `
            <tr>
                <td>${admin.name}</td>
                <td>${admin.email}</td>
                <td>${admin.role}</td>
                <td><span class="status-badge ${admin.status}">${admin.status}</span></td>
                <td class="actions">
                    <button class="btn btn-danger" onclick="deleteAdmin('${admin.id}')">Delete</button>
                </td>
            </tr>
        `).join('');
    } catch (error) {
        console.error('Failed to load admins:', error);
    }
}

async function loadProducts() {
    try {
        const response = await fetchWithAuth(`${API_BASE}/white-label/products`);
        const products = await response.json();

        const tbody = document.getElementById('products-list');
        tbody.innerHTML = products.map(product => `
            <tr>
                <td>${product.name}</td>
                <td>${product.type}</td>
                <td>${product.fee}%</td>
                <td><span class="status-badge ${product.status}">${product.status}</span></td>
                <td class="actions">
                    <button class="btn btn-danger" onclick="deleteProduct('${product.id}')">Delete</button>
                </td>
            </tr>
        `).join('');
    } catch (error) {
        console.error('Failed to load products:', error);
    }
}

async function loadFetchers() {
    try {
        const response = await fetchWithAuth(`${API_BASE}/white-label/fetchers`);
        const fetchers = await response.json();

        // Update stats for each fetcher
        const statsGrid = document.getElementById('fetcher-stats-grid');
        statsGrid.innerHTML = '';

        for (const fetcher of fetchers) {
            try {
                const statsRes = await fetchWithAuth(`${API_BASE}/white-label/fetchers/${fetcher.name}/stats`);
                const stats = await statsRes.json();
                
                statsGrid.innerHTML += `
                    <div class="stat-card">
                        <div class="stat-value" style="font-size: 1rem;">${fetcher.name}</div>
                        <div class="stat-label">Latency: ${stats.lastLatencyNs}ns</div>
                        <div class="stat-label">Success: ${stats.successRate.toFixed(1)}%</div>
                    </div>
                `;
            } catch (e) {
                console.error(`Failed to load stats for ${fetcher.name}:`, e);
            }
        }
    } catch (error) {
        console.error('Failed to load fetchers:', error);
    }
}

async function loadRevenue() {
    try {
        const response = await fetchWithAuth(`${API_BASE}/super-admin/revenue`);
        const revenues = await response.json();

        let gross = 0, superAdminShare = 0, pending = 0;

        revenues.forEach(r => {
            gross += r.grossRevenue;
            superAdminShare += r.profitShare;
            if (r.status === 'pending') pending += r.profitShare;
        });

        document.getElementById('gross-revenue').textContent = formatCurrency(gross);
        document.getElementById('super-admin-revenue').textContent = formatCurrency(superAdminShare);
        document.getElementById('pending-revenue').textContent = formatCurrency(pending);

        const tbody = document.getElementById('revenue-list');
        tbody.innerHTML = revenues.map(r => `
            <tr>
                <td>${r.period}</td>
                <td>${formatCurrency(r.grossRevenue)}</td>
                <td>${formatCurrency(r.profitShare)}</td>
                <td>${formatCurrency(r.netRevenue)}</td>
                <td><span class="status-badge ${r.status}">${r.status}</span></td>
                <td class="actions">
                    ${r.status === 'pending' ? 
                        `<button class="btn btn-success" onclick="payRevenue('${r.id}')">Pay</button>` : ''}
                </td>
            </tr>
        `).join('');
    } catch (error) {
        console.error('Failed to load revenue:', error);
    }
}

function loadSettings() {
    if (currentUser) {
        document.getElementById('settings-name').value = currentUser.name || '';
        document.getElementById('settings-email').value = currentUser.email || '';
    }
}

// =============================================================================
// CLIENT ACTIONS
// =============================================================================

async function authorizeClient(clientId) {
    try {
        await fetchWithAuth(`${API_BASE}/super-admin/clients/${clientId}/authorize`, { method: 'POST' });
        loadClients();
        loadOverviewData();
    } catch (error) {
        alert('Failed to authorize client');
    }
}

async function suspendClient(clientId) {
    try {
        await fetchWithAuth(`${API_BASE}/super-admin/clients/${clientId}/suspend`, { method: 'POST' });
        loadClients();
        loadOverviewData();
    } catch (error) {
        alert('Failed to suspend client');
    }
}

async function resumeClient(clientId) {
    try {
        await fetchWithAuth(`${API_BASE}/super-admin/clients/${clientId}/resume`, { method: 'POST' });
        loadClients();
        loadOverviewData();
    } catch (error) {
        alert('Failed to resume client');
    }
}

async function haltClient(clientId) {
    if (!confirm('Are you sure you want to halt this client? This will stop all their services.')) return;
    
    try {
        await fetchWithAuth(`${API_BASE}/super-admin/clients/${clientId}/halt`, { method: 'POST' });
        loadClients();
        loadOverviewData();
    } catch (error) {
        alert('Failed to halt client');
    }
}

async function payRevenue(revenueId) {
    try {
        await fetchWithAuth(`${API_BASE}/super-admin/revenue/${revenueId}/pay`, { method: 'POST' });
        loadRevenue();
    } catch (error) {
        alert('Failed to pay revenue');
    }
}

// =============================================================================
// MODAL HELPERS
// =============================================================================

function showCreateClientModal() {
    const content = `
        <form id="create-client-form">
            <div class="form-group">
                <label>Client Name</label>
                <input type="text" name="name" required>
            </div>
            <div class="form-group">
                <label>Domain</label>
                <input type="text" name="domain" placeholder="example.com" required>
            </div>
            <div class="form-group">
                <label>Primary Color</label>
                <input type="color" name="primaryColor" value="#ff6b35">
            </div>
            <div class="form-group">
                <label>Secondary Color</label>
                <input type="color" name="secondaryColor" value="#4361ee">
            </div>
            <button type="submit" class="btn btn-primary btn-block">Create Client</button>
        </form>
    `;
    
    showModal('Create White Label Client', content);
    
    document.getElementById('create-client-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        const formData = new FormData(e.target);
        
        try {
            await fetchWithAuth(`${API_BASE}/white-label/clients`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    name: formData.get('name'),
                    domain: formData.get('domain'),
                    primaryColor: formData.get('primaryColor'),
                    secondaryColor: formData.get('secondaryColor'),
                    blockchainAccess: [1, 56, 137, 42161]
                })
            });
            
            closeModal();
            loadClients();
        } catch (error) {
            alert('Failed to create client');
        }
    });
}

function showCreateAdminModal() {
    const content = `
        <form id="create-admin-form">
            <div class="form-group">
                <label>Email</label>
                <input type="email" name="email" required>
            </div>
            <div class="form-group">
                <label>Password</label>
                <input type="password" name="password" required>
            </div>
            <div class="form-group">
                <label>Name</label>
                <input type="text" name="name" required>
            </div>
            <div class="form-group">
                <label>Role</label>
                <select name="role">
                    <option value="admin">Admin</option>
                    <option value="manager">Manager</option>
                    <option value="support">Support</option>
                </select>
            </div>
            <button type="submit" class="btn btn-primary btn-block">Create Admin</button>
        </form>
    `;
    
    showModal('Create Admin', content);
    
    document.getElementById('create-admin-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        const formData = new FormData(e.target);
        
        try {
            await fetchWithAuth(`${API_BASE}/white-label/admins`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    email: formData.get('email'),
                    password: formData.get('password'),
                    name: formData.get('name'),
                    role: formData.get('role'),
                    clientId: currentUser?.clientId || ''
                })
            });
            
            closeModal();
            loadAdmins();
        } catch (error) {
            alert('Failed to create admin');
        }
    });
}

function showCreateProductModal() {
    const content = `
        <form id="create-product-form">
            <div class="form-group">
                <label>Product Name</label>
                <input type="text" name="name" required>
            </div>
            <div class="form-group">
                <label>Type</label>
                <select name="type" required>
                    <option value="trading">Trading</option>
                    <option value="wallet">Wallet</option>
                    <option value="staking">Staking</option>
                    <option value="nft">NFT</option>
                    <option value="perpetual">Perpetual</option>
                </select>
            </div>
            <div class="form-group">
                <label>Fee (%)</label>
                <input type="number" name="fee" step="0.01" value="0.1" required>
            </div>
            <div class="form-group">
                <label>Min Deposit</label>
                <input type="number" name="minDeposit" value="0">
            </div>
            <div class="form-group">
                <label>Max Deposit</label>
                <input type="number" name="maxDeposit" value="1000000">
            </div>
            <button type="submit" class="btn btn-primary btn-block">Create Product</button>
        </form>
    `;
    
    showModal('Create Product', content);
    
    document.getElementById('create-product-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        const formData = new FormData(e.target);
        
        try {
            await fetchWithAuth(`${API_BASE}/white-label/products`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    name: formData.get('name'),
                    type: formData.get('type'),
                    fee: parseFloat(formData.get('fee')),
                    minDeposit: parseFloat(formData.get('minDeposit')),
                    maxDeposit: parseFloat(formData.get('maxDeposit')),
                    features: []
                })
            });
            
            closeModal();
            loadProducts();
        } catch (error) {
            alert('Failed to create product');
        }
    });
}

// =============================================================================
// UTILITY FUNCTIONS
// =============================================================================

async function fetchWithAuth(url, options = {}) {
    const headers = {
        ...options.headers,
    };
    
    if (token) {
        headers['Authorization'] = `Bearer ${token}`;
    }
    
    if (!(options.body instanceof FormData)) {
        headers['Content-Type'] = 'application/json';
    }
    
    const response = await fetch(url, { ...options, headers });
    
    // Handle 401 - try to refresh token
    if (response.status === 401) {
        const refreshed = await refreshTokenIfNeeded();
        if (refreshed) {
            headers['Authorization'] = `Bearer ${token}`;
            return fetch(url, { ...options, headers });
        }
    }
    
    return response;
}

function formatCurrency(amount) {
    return new Intl.NumberFormat('en-US', {
        style: 'currency',
        currency: 'USD',
    }).format(amount);
}

function formatDate(timestamp) {
    if (!timestamp) return '-';
    return new Date(timestamp).toLocaleString();
}

// =============================================================================
// EVENT LISTENERS
// =============================================================================

// Login form
document.getElementById('login-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    
    const email = document.getElementById('email').value;
    const password = document.getElementById('password').value;
    const twoFACode = document.getElementById('2fa-code').value;
    
    await login(email, password, twoFACode);
});

// Check auth on load
document.addEventListener('DOMContentLoaded', () => {
    initTheme();
    
    if (token) {
        // Try to restore session
        refreshTokenIfNeeded().then(refreshed => {
            if (refreshed) {
                showDashboard();
                loadDashboardData();
            }
        });
    }
});

// Settings form
document.getElementById('settings-form')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    // Implement settings update
});

// Password form
document.getElementById('password-form')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    // Implement password change
});

// Close modal on overlay click
document.getElementById('modal-overlay')?.addEventListener('click', (e) => {
    if (e.target === document.getElementById('modal-overlay')) {
        closeModal();
    }
});
