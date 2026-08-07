/**
 * TigerWallet Admin - Background Service Worker
 */

chrome.runtime.onInstalled.addListener(() => {
    console.log('TigerWallet Admin Extension installed');
    
    // Set default settings
    chrome.storage.local.set({
        theme: 'light',
        notifications: true
    });
});

// Handle messages from popup or other scripts
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    if (message.action === 'getTheme') {
        chrome.storage.local.get(['theme'], (result) => {
            sendResponse({ theme: result.theme });
        });
        return true;
    }

    if (message.action === 'setTheme') {
        chrome.storage.local.set({ theme: message.theme });
        sendResponse({ success: true });
        return true;
    }

    if (message.action === 'openDashboard') {
        chrome.tabs.create({ url: 'https://admin.tigerwallet.com' });
        sendResponse({ success: true });
        return true;
    }
});

// Handle extension icon click
chrome.action.onClicked.addListener((tab) => {
    chrome.tabs.create({ url: 'https://admin.tigerwallet.com' });
});
