/**
 * TigerWallet Admin Safari Extension - Background Service Worker
 * Handles background tasks, notifications, and API communication
 */

// Configuration
const API_BASE_URL = localStorage.getItem('api_base_url') || 'http://localhost:9093/api/v1';
const WS_URL = localStorage.getItem('ws_url') || 'ws://localhost:9093/ws';
const CHECK_INTERVAL = 60000;

// State
let websocket = null;
let reconnectTimer = null;
let checkTimer = null;

// Initialize
self.addEventListener('install', (event) => {
    console.log('TigerWallet Admin Extension installed');
    initializeStorage();
    startPeriodicCheck();
    connectWebSocket();
    self.skipWaiting();
});

self.addEventListener('activate', (event) => {
    console.log('TigerWallet Admin Extension activated');
    event.waitUntil(clients.claim());
});

// Message handling
self.addEventListener('message', (event) => {
    handleMessage(event.data).then(response => {
        if (response !== undefined) {
            event.source.postMessage(response);
        }
    });
});

// Handle messages from popup and content scripts
async function handleMessage(message) {
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
async function broadcastToPopup(message) {
    const clients = await self.clients.matchAll({ type: 'popup' });
    clients.forEach(client => {
        client.postMessage(message);
    });
}

// Broadcast theme to all tabs
async function broadcastThemeToAllTabs(theme) {
    const allClients = await self.clients.matchAll({ type: 'all' });
    allClients.forEach(client => {
        client.postMessage({
            type: 'THEME_CHANGED',
            theme: theme
        });
    });
}

// Show notification
function showNotification(title, message, type = 'info') {
    self.registration.showNotification(title, {
        body: message,
        icon: 'icons/icon48.png',
        badge: 'icons/icon48.png',
        tag: 'tigerwallet-admin',
        renotify: true,
        priority: type === 'warning' ? 2 : 1
    });
}

// Open pages
function openUserDetailPage(userId) {
    const url = `${getBaseAdminURL()}/users/${userId}`;
    self.clients.openWindow(url);
}

function openUserEditPage(userId) {
    const url = `${getBaseAdminURL()}/users/${userId}/edit`;
    self.clients.openWindow(url);
}

function openKYCDetailPage(kycId) {
    const url = `${getBaseAdminURL()}/kyc/${kycId}`;
    self.clients.openWindow(url);
}

function openAddTokenPage() {
    const url = `${getBaseAdminURL()}/tokens/add`;
    self.clients.openWindow(url);
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
                handleTokenExpired();
            }
        }
        
        const data = await response.json();
        
        // Update badge using the Safari-specific approach
        if (data.pendingKYC > 0) {
            // For Safari, we'd use the badge API
            try {
                await self.registration.set(data.pendingKYC.toString());
            } catch (e) {
                console.log('Badge not supported');
            }
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

// Push notifications (for future Safari push notification support)
self.addEventListener('push', (event) => {
    const data = event.data?.json() || {};
    
    const options = {
        body: data.body || 'New notification',
        icon: 'icons/icon48.png',
        badge: 'icons/icon48.png',
        tag: data.tag || 'tigerwallet-admin',
        data: data.data || {}
    };
    
    event.waitUntil(
        self.registration.showNotification(data.title || 'TigerWallet Admin', options)
    );
});

// Notification click handler
self.addEventListener('notificationclick', (event) => {
    event.notification.close();
    
    const data = event.notification.data || {};
    
    if (data.url) {
        event.waitUntil(
            self.clients.openWindow(data.url)
        );
    }
});

console.log('TigerWallet Admin Background Service Worker loaded');
