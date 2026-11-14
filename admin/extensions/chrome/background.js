/**
 * TigerWallet Admin — background service worker (MV3, Chrome + Firefox).
 *
 * Replaces the previous copy which used `localStorage` (undefined in a
 * service worker) and the `browser` namespace (absent in Chrome), so it threw
 * on load and the extension had no working background context.
 */
importScripts('js/store.js');

// WebExtension API compatibility shim.
// Firefox exposes the promise-based `browser.*`; Chromium exposes `chrome.*`.
if (typeof browser === 'undefined' && typeof chrome !== 'undefined') {
    // eslint-disable-next-line no-global-assign
    self.browser = chrome;
}

const DEFAULT_API_BASE = 'http://localhost:9093/api/v1';
const DEFAULT_WS_URL = 'ws://localhost:9093/ws';
const CHECK_INTERVAL = 60000;

let websocket = null;
let reconnectTimer = null;
let checkTimer = null;

function getApiBase() {
    return TWStore.getAsync('api_base_url', DEFAULT_API_BASE);
}

function getWsUrl() {
    return TWStore.getAsync('ws_url', DEFAULT_WS_URL);
}

browser.runtime.onInstalled.addListener(() => {
    console.log('TigerWallet Admin Extension installed');
    initializeStorage();
    startPeriodicCheck();
    connectWebSocket();
});

browser.runtime.onMessage.addListener((message, sender, sendResponse) => {
    handleMessage(message, sender).then(sendResponse).catch((err) => {
        sendResponse({ error: String(err) });
    });
    return true;
});

async function handleMessage(message) {
    switch (message.type) {
        case 'GET_AUTH_TOKEN':
            return TWStore.getAsync('admin_token', null);

        case 'SET_AUTH_TOKEN':
            await TWStore.setAsync('admin_token', message.token);
            connectWebSocket();
            return true;

        case 'CLEAR_AUTH_TOKEN':
            await TWStore.removeAsync('admin_token');
            disconnectWebSocket();
            return true;

        case 'API_REQUEST':
            return await apiRequest(message.endpoint, message.options);

        case 'OPEN_USER_DETAIL':
            openPage(`/users/${message.userId}`);
            return true;

        case 'OPEN_USER_EDIT':
            openPage(`/users/${message.userId}/edit`);
            return true;

        case 'OPEN_KYC_DETAIL':
            openPage(`/kyc/${message.kycId}`);
            return true;

        case 'OPEN_ADD_TOKEN':
            openPage('/tokens/add');
            return true;

        case 'THEME_CHANGED':
            broadcastThemeToAllTabs(message.theme);
            return true;

        case 'PING':
            return { status: 'ok', timestamp: Date.now() };

        case 'ENSURE_HOST_PERMISSION': {
            // A user-configured remote API host is an optional permission;
            // MV3 blocks the fetch until it is granted.
            const origin = new URL(message.url).origin + '/*';
            const has = await browser.permissions.contains({ origins: [origin] });
            if (has) return true;
            return await browser.permissions.request({ origins: [origin] });
        }

        default:
            console.warn('Unknown message type:', message.type);
            return null;
    }
}

async function apiRequest(endpoint, options = {}) {
    const base = await getApiBase();
    const token = await TWStore.getAsync('admin_token', null);

    const response = await fetch(`${base}${endpoint}`, {
        ...options,
        headers: {
            Authorization: `Bearer ${token}`,
            'Content-Type': 'application/json',
            ...options.headers
        }
    });

    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    return await response.json();
}

async function initializeStorage() {
    const base = await TWStore.getAsync('api_base_url', null);
    if (!base) await TWStore.setAsync('api_base_url', DEFAULT_API_BASE);
    const ws = await TWStore.getAsync('ws_url', null);
    if (!ws) await TWStore.setAsync('ws_url', DEFAULT_WS_URL);
    const theme = await TWStore.getAsync('admin_theme', null);
    if (!theme) await TWStore.setAsync('admin_theme', 'dark');
}

async function connectWebSocket() {
    const token = await TWStore.getAsync('admin_token', null);
    if (!token) return;
    if (websocket && websocket.readyState <= WebSocket.OPEN) return;

    const wsUrl = await getWsUrl();
    try {
        websocket = new WebSocket(`${wsUrl}?token=${token}`);

        websocket.onopen = () => {
            if (reconnectTimer) {
                clearTimeout(reconnectTimer);
                reconnectTimer = null;
            }
        };

        websocket.onmessage = (event) => {
            try {
                handleWebSocketMessage(JSON.parse(event.data));
            } catch (e) {
                console.error('Failed to parse WebSocket message:', e);
            }
        };

        websocket.onclose = () => scheduleReconnect();
        websocket.onerror = (error) => console.error('WebSocket error:', error);
    } catch (error) {
        console.error('Failed to connect WebSocket:', error);
        scheduleReconnect();
    }
}

function disconnectWebSocket() {
    if (websocket) {
        websocket.onclose = null;
        websocket.close();
        websocket = null;
    }
}

function scheduleReconnect() {
    if (reconnectTimer) return;
    reconnectTimer = setTimeout(() => {
        reconnectTimer = null;
        connectWebSocket();
    }, 5000);
}

function handleWebSocketMessage(data) {
    switch (data.type) {
        case 'DASHBOARD_UPDATE':
        case 'SERVICE_STATUS':
            broadcastToPopup({ type: data.type, data: data.data });
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
    }
}

function broadcastToPopup(message) {
    browser.runtime.sendMessage(message).catch(() => {
        // Popup is usually closed; that is not an error.
    });
}

function broadcastThemeToAllTabs(theme) {
    browser.tabs.query({}).then((tabs) => {
        tabs.forEach((tab) => {
            browser.tabs.sendMessage(tab.id, { type: 'THEME_CHANGED', theme }).catch(() => {});
        });
    });
}

function showNotification(title, message, type = 'info') {
    browser.notifications.create({
        type: 'basic',
        iconUrl: browser.runtime.getURL('icons/icon48.png'),
        title: title,
        message: message,
        priority: type === 'warning' ? 2 : 1
    });
}

async function openPage(path) {
    const base = await TWStore.getAsync('admin_panel_url', 'http://localhost:3000/admin');
    browser.tabs.create({ url: `${base.replace(/\/$/, '')}${path}` });
}

function startPeriodicCheck() {
    if (checkTimer) clearInterval(checkTimer);
    checkTimer = setInterval(performHealthCheck, CHECK_INTERVAL);
    performHealthCheck();
}

async function performHealthCheck() {
    try {
        const token = await TWStore.getAsync('admin_token', null);
        if (!token) return;

        const base = await getApiBase();
        const response = await fetch(`${base}/admin/health`, {
            headers: { Authorization: `Bearer ${token}` }
        });

        if (response.status === 401) {
            await TWStore.removeAsync('admin_token');
            broadcastToPopup({ type: 'TOKEN_EXPIRED' });
            showNotification('Session Expired', 'Please log in again', 'warning');
            return;
        }
        if (!response.ok) return;

        const data = await response.json();
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
    if (info.menuItemId !== 'view-on-admin' || !tab || !tab.url) return;
    try {
        const url = new URL(tab.url);
        if (url.pathname.includes('/user/')) {
            openPage(`/users/${url.pathname.split('/user/')[1]}`);
        }
    } catch (e) {
        // Non-HTTP tab.
    }
});

browser.storage.onChanged.addListener((changes) => {
    if (changes.admin_theme) broadcastThemeToAllTabs(changes.admin_theme.newValue);
});

console.log('TigerWallet Admin Background Script loaded');
