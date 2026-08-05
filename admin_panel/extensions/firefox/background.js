// Super Admin Chrome Extension - Background Service Worker
const API_BASE_URL = 'http://localhost:9090/api/v1';

// Handle messages from popup and content scripts
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  switch (request.action) {
    case 'getDashboard':
      fetchDashboard().then(sendResponse);
      return true;
    case 'getUsers':
      fetchUsers(request.params).then(sendResponse);
      return true;
    case 'getTransactions':
      fetchTransactions(request.params).then(sendResponse);
      return true;
    case 'getStats':
      fetchStats().then(sendResponse);
      return true;
    case 'getTheme':
      chrome.storage.local.get(['theme']).then(result => {
        sendResponse(result.theme || 'system');
      });
      return true;
    case 'setTheme':
      chrome.storage.local.set({ theme: request.theme });
      sendResponse({ success: true });
      return true;
    default:
      sendResponse({ error: 'Unknown action' });
  }
});

// API Functions
async function fetchDashboard() {
  try {
    const token = await getToken();
    const response = await fetch(`${API_BASE_URL}/dashboard/stats`, {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    return await response.json();
  } catch (error) {
    return { error: error.message };
  }
}

async function fetchUsers(params = {}) {
  try {
    const token = await getToken();
    const query = new URLSearchParams(params).toString();
    const response = await fetch(`${API_BASE_URL}/users?${query}`, {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    return await response.json();
  } catch (error) {
    return { error: error.message };
  }
}

async function fetchTransactions(params = {}) {
  try {
    const token = await getToken();
    const query = new URLSearchParams(params).toString();
    const response = await fetch(`${API_BASE_URL}/transactions?${query}`, {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    return await response.json();
  } catch (error) {
    return { error: error.message };
  }
}

async function fetchStats() {
  try {
    const token = await getToken();
    const response = await fetch(`${API_BASE_URL}/stats`, {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    return await response.json();
  } catch (error) {
    return { error: error.message };
  }
}

async function getToken() {
  const result = await chrome.storage.local.get(['token']);
  return result.token;
}

// Handle extension install
chrome.runtime.onInstalled.addListener((details) => {
  if (details.reason === 'install') {
    console.log('TigerWallet Super Admin extension installed');
  }
});

// Badge update function
function updateBadge(count, color = '#ef4444') {
  chrome.action.setBadgeText({ text: count > 0 ? String(count) : '' });
  chrome.action.setBadgeBackgroundColor({ color });
}
