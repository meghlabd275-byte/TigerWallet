/**
 * TigerWallet Admin Firefox Extension - Background Script
 * Handles background tasks, notifications, and API communication
 */

// Configuration
const API_BASE_URL = localStorage.getItem('api_base_url') || 'http://localhost:9093/api/v1';
const WS_URL = localStorage.getItem('ws_url') || 'ws://localhost:9093/ws';
const CHECK_INTERVAL = 60000; // 1 minute

// State
let websocket = null;
let reconnectTimer = null;
let checkTimer = null;

// Initialize
browser.runtime.onInstalled.addListener(() => {
    console.log('TigerWallet Admin Extension installed');
    initializeStorage();
    startPeriodicCheck();
    connectWebSocket();
});

// Message handling
browser.runtime.onMessage.addListener((message, sender, sendResponse) => {
    handleMessage(message, sender).then(sendResponse);
    return true;
});

// Handle messages from popup and content scripts
async function handleMessage(message, sender) {
    switch (message.type) {
        case 'GET_AUTH_TOKEN':
            return localStorage.getItem('admin_token');
            
        case 'SET_AUTH_TOKEN':
            localStorage.setItem('admin_token', message.token);
            return true;
            
        case 'API_REQUEST':
            return await apiRequest(message.endpoint, message.options);
            
        case 'OPEN_USER_DETAIL':
            openUserDetailPage(message.userId);
            return true;
            
        case 'OPEN_USER_EDIT':
            openUserEditPage(message.userId);
            return true;
            
        case 'OPEN_KYC_DETAIL':
            openKYCDetailPage(message.kycId);
            return true;
            
        case 'OPEN_ADD_TOKEN':
            openAddTokenPage();
            return true;
            
        case 'THEME_CHANGED':
            broadcastThemeToAllTabs(message.theme);
            return true;
            
        case 'PING':
            return { status: 'ok', timestamp: Date.now() };
            
        default:
            console.warn('Unknown message type:', message.type);
            return null;
    }
}

// API Request Handler
async function apiRequest(endpoint, options = {}) {
    const token = localStorage.getItem('admin_token');
    
    try {
        const response = await fetch(`${API_BASE_URL}${endpoint}`, {
            ...options,
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json',
                ...options.headers
            }
        });
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}`);
        }
        
        return await response.json();
    } catch (error) {
        console.error('API request failed:', error);
        throw error;
    }
}

// Storage Initialization
function initializeStorage() {
    // Set defaults if not already set
    if (!localStorage.getItem('admin_theme')) {
        localStorage.setItem('admin_theme', 'dark');
    }
    
    if (!localStorage.getItem('api_base_url')) {
        localStorage.setItem('api_base_url', 'http://localhost:9093/api/v1');
    }
    
    if (!localStorage.getItem('ws_url')) {
        localStorage.setItem('ws_url', 'ws://localhost:9093/ws');
    }
}

// WebSocket Connection
function connectWebSocket() {
    const token = localStorage.getItem('admin_token');
    if (!token) {
        console.log('No auth token, skipping WebSocket connection');
        return;
    }

    try {
        websocket = new WebSocket(`${WS_URL}?token=${token}`);

        websocket.onopen = () => {
            console.log('WebSocket connected');
            if (reconnectTimer) {
                clearTimeout(reconnectTimer);
                reconnectTimer = null;
            }
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
            scheduleReconnect();
        };

        websocket.onerror = (error) => {
            console.error('WebSocket error:', error);
        };
    } catch (error) {
        console.error('Failed to connect WebSocket:', error);
        scheduleReconnect();
    }
}

function scheduleReconnect() {
    if (reconnectTimer) return;
    
    reconnectTimer = setTimeout(() => {
        reconnectTimer = null;
        connectWebSocket();
    }, 5000);
}

// Handle WebSocket messages
function handleWebSocketMessage(data) {
    switch (data.type) {
        case 'DASHBOARD_UPDATE':
            broadcastToPopup({ type: 'DASHBOARD_UPDATE', data: data.data });
            break;
            
        case 'NEW_USER':
            showNotification('New User', `${data.data.email} has registered`);
            broadcastToPopup({ type: 'NEW_USER', data: data.data });
            break;
            
        case 'NEW_TRANSACTION':
            if (data.data.amount > 10000) {
                showNotification('Large Transaction', `$${data.data.amount} ${data.data.type}`);
            }
            broadcastToPopup({ type: 'NEW_TRANSACTION', data: data.data });
            break;
            
        case 'KYC_SUBMITTED':
            showNotification('KYC Submitted', `${data.data.email} submitted KYC documents`);
            broadcastToPopup({ type: 'KYC_SUBMITTED', data: data.data });
            break;
            
        case 'SYSTEM_ALERT':
            showNotification('System Alert', data.data.message, 'warning');
            broadcastToPopup({ type: 'SYSTEM_ALERT', data: data.data });
            break;
            
        case 'SERVICE_STATUS':
            broadcastToPopup({ type: 'SERVICE_STATUS', data: data.data });
            break;
    }
}

// Broadcast to popup
function broadcastToPopup(message) {
    browser.runtime.sendMessage(message).catch(() => {
        // Popup might not be open, that's ok
    });
}

// Broadcast theme to all tabs
function broadcastThemeToAllTabs(theme) {
    browser.tabs.query({}).then(tabs => {
        tabs.forEach(tab => {
            browser.tabs.sendMessage(tab.id, {
                type: 'THEME_CHANGED',
                theme: theme
            }).catch(() => {});
        });
    });
}

// Show notification
function showNotification(title, message, type = 'info') {
    browser.notifications.create({
        type: 'basic',
        iconUrl: browser.runtime.getURL('icons/icon48.png'),
        title: title,
        message: message,
        priority: type === 'warning' ? 2 : 1
    });
}

// Open pages
function openUserDetailPage(userId) {
    const url = `${getBaseAdminURL()}/users/${userId}`;
    browser.tabs.create({ url });
}

function openUserEditPage(userId) {
    const url = `${getBaseAdminURL()}/users/${userId}/edit`;
    browser.tabs.create({ url });
}

function openKYCDetailPage(kycId) {
    const url = `${getBaseAdminURL()}/kyc/${kycId}`;
    browser.tabs.create({ url });
}

function openAddTokenPage() {
    const url = `${getBaseAdminURL()}/tokens/add`;
    browser.tabs.create({ url });
}

function getBaseAdminURL() {
    return localStorage.getItem('admin_panel_url') || 'http://localhost:3000/admin';
}

// Periodic Check
function startPeriodicCheck() {
    if (checkTimer) {
        clearInterval(checkTimer);
    }
    
    checkTimer = setInterval(async () => {
        await performHealthCheck();
    }, CHECK_INTERVAL);
    
    // Initial check
    performHealthCheck();
}

async function performHealthCheck() {
    try {
        const token = localStorage.getItem('admin_token');
        if (!token) return;
        
        const response = await fetch(`${API_BASE_URL}/admin/health`, {
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });
        
        if (!response.ok) {
            if (response.status === 401) {
                // Token expired
                handleTokenExpired();
            }
        }
        
        const data = await response.json();
        
        // Update badge
        if (data.pendingKYC > 0) {
            browser.action.setBadgeText({ text: String(data.pendingKYC) });
            browser.action.setBadgeBackgroundColor({ color: '#f59e0b' });
        } else {
            browser.action.setBadgeText({ text: '' });
        }
        
    } catch (error) {
        console.error('Health check failed:', error);
    }
}

function handleTokenExpired() {
    localStorage.removeItem('admin_token');
    broadcastToPopup({ type: 'TOKEN_EXPIRED' });
    
    showNotification('Session Expired', 'Please log in again', 'warning');
}

// Context menu
browser.contextMenus?.create({
    id: 'admin-tools',
    title: 'Admin Tools',
    contexts: ['page', 'selection']
});

browser.contextMenus?.create({
    id: 'view-on-admin',
    parentId: 'admin-tools',
    title: 'View on Admin Panel',
    contexts: ['page']
});

browser.contextMenus?.onClicked.addListener((info, tab) => {
    if (info.menuItemId === 'view-on-admin') {
        const url = new URL(tab.url);
        const baseUrl = getBaseAdminURL();
        
        if (url.pathname.includes('/user/')) {
            const userId = url.pathname.split('/user/')[1];
            openUserDetailPage(userId);
        }
    }
});

// Storage change listener
browser.storage.onChanged.addListener((changes, areaName) => {
    if (changes.admin_theme) {
        broadcastThemeToAllTabs(changes.admin_theme.newValue);
    }
});

console.log('TigerWallet Admin Background Script loaded');
