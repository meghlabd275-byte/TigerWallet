/**
 * TigerWallet Admin Safari Extension - Content Script
 * Injected into web pages for enhanced admin functionality
 */

(function() {
    'use strict';

    // Listen for messages from background script
    browser.runtime.onMessage.addListener((message, sender, sendResponse) => {
        switch (message.type) {
            case 'THEME_CHANGED':
                document.dispatchEvent(new CustomEvent('tigerwallet-theme-changed', {
                    detail: { theme: message.theme }
                }));
                break;
                
            case 'INJECT_ADMIN_CONTROLS':
                injectAdminControls(message.data);
                break;
                
            case 'GET_PAGE_INFO':
                sendResponse(getPageInfo());
                break;
        }
    });

    // Inject admin controls into admin panel pages
    function injectAdminControls(data) {
        // Only inject on admin panel pages
        if (!isAdminPanelPage()) return;
        
        // Add keyboard shortcuts
        document.addEventListener('keydown', handleKeyboardShortcuts);
        
        // Add context menu items
        setupContextMenus();
    }

    // Check if current page is admin panel
    function isAdminPanelPage() {
        const path = window.location.pathname;
        return path.includes('/admin') || path.includes('/admin-panel');
    }

    // Handle keyboard shortcuts
    function handleKeyboardShortcuts(event) {
        // Ctrl/Cmd + K: Quick search
        if ((event.ctrlKey || event.metaKey) && event.key === 'k') {
            event.preventDefault();
            const searchInput = document.querySelector('input[type="search"], input[placeholder*="search"]');
            if (searchInput) {
                searchInput.focus();
            }
        }
        
        // Ctrl/Cmd + D: Toggle dark mode
        if ((event.ctrlKey || event.metaKey) && event.key === 'd') {
            event.preventDefault();
            togglePageTheme();
        }
        
        // R: Refresh (when not in input)
        if (event.key === 'r' && !isInputFocused()) {
            event.preventDefault();
            window.location.reload();
        }
    }

    // Check if an input is focused
    function isInputFocused() {
        const focused = document.activeElement;
        return focused && (
            focused.tagName === 'INPUT' ||
            focused.tagName === 'TEXTAREA' ||
            focused.isContentEditable
        );
    }

    // Toggle page theme
    function togglePageTheme() {
        const isDark = document.body.classList.contains('dark-mode');
        document.body.classList.toggle('dark-mode', !isDark);
        document.body.classList.toggle('light-mode', isDark);
        
        // Notify background script
        browser.runtime.sendMessage({
            type: 'THEME_CHANGED',
            theme: isDark ? 'light' : 'dark'
        });
    }

    // Setup context menus
    function setupContextMenus() {
        // Add admin quick actions on user elements
        document.addEventListener('contextmenu', (event) => {
            const userElement = event.target.closest('[data-user-id], [data-user-email]');
            if (userElement) {
                event.preventDefault();
                
                const userId = userElement.dataset.userId || userElement.dataset.userEmail;
                showAdminContextMenu(event.pageX, event.pageY, userId);
            }
        });
    }

    // Show admin context menu
    function showAdminContextMenu(x, y, userId) {
        // Remove existing menu
        const existingMenu = document.querySelector('.tiger-admin-context-menu');
        if (existingMenu) {
            existingMenu.remove();
        }

        const menu = document.createElement('div');
        menu.className = 'tiger-admin-context-menu';
        menu.innerHTML = `
            <div class="menu-item" data-action="view-profile">View Profile</div>
            <div class="menu-item" data-action="edit-user">Edit User</div>
            <div class="menu-item" data-action="kyc-review">Review KYC</div>
            <div class="menu-item" data-action="view-transactions">View Transactions</div>
            <div class="menu-divider"></div>
            <div class="menu-item danger" data-action="suspend-user">Suspend User</div>
            <div class="menu-item danger" data-action="ban-user">Ban User</div>
        `;

        menu.style.cssText = `
            position: fixed;
            left: ${x}px;
            top: ${y}px;
            background: var(--bg-card, #1f2937);
            border: 1px solid var(--border-color, #374151);
            border-radius: 8px;
            padding: 8px 0;
            z-index: 10000;
            min-width: 180px;
            box-shadow: 0 10px 25px rgba(0,0,0,0.3);
        `;

        document.body.appendChild(menu);

        // Handle menu item clicks
        menu.querySelectorAll('.menu-item').forEach(item => {
            item.addEventListener('click', () => {
                const action = item.dataset.action;
                handleMenuAction(action, userId);
                menu.remove();
            });
        });

        // Close on click outside
        setTimeout(() => {
            document.addEventListener('click', function closeMenu() {
                menu.remove();
                document.removeEventListener('click', closeMenu);
            });
        }, 100);
    }

    // Handle context menu actions
    function handleMenuAction(action, userId) {
        browser.runtime.sendMessage({
            type: 'ADMIN_CONTEXT_ACTION',
            action: action,
            userId: userId
        });
    }

    // Get page information
    function getPageInfo() {
        return {
            url: window.location.href,
            path: window.location.pathname,
            title: document.title,
            userId: getCurrentUserId(),
            timestamp: Date.now()
        };
    }

    // Get current user ID from page
    function getCurrentUserId() {
        // Try various methods to find user ID
        const userElement = document.querySelector('[data-user-id]');
        if (userElement) {
            return userElement.dataset.userId;
        }
        
        // Try from URL
        const urlMatch = window.location.pathname.match(/\/user[sz]?\/([^/]+)/);
        if (urlMatch) {
            return urlMatch[1];
        }
        
        return null;
    }

    // Theme change listener
    document.addEventListener('tigerwallet-theme-changed', (event) => {
        const { theme } = event.detail;
        // Apply theme to page if needed
        console.log('Theme changed to:', theme);
    });

    // Initialize
    console.log('TigerWallet Admin Content Script loaded');

})();
