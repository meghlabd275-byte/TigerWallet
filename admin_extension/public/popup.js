/**
 * TigerWallet Admin - Extension Popup
 * Complete Dark/Light Theme Support
 */

document.addEventListener('DOMContentLoaded', async () => {
    // Load theme
    const theme = await loadTheme();
    applyTheme(theme);

    // Theme toggle
    const themeToggle = document.getElementById('theme-toggle');
    themeToggle.addEventListener('change', (e) => {
        const newTheme = e.target.checked ? 'dark' : 'light';
        applyTheme(newTheme);
        saveTheme(newTheme);
    });

    // Navigation
    const navItems = document.querySelectorAll('.nav-item');
    navItems.forEach(item => {
        item.addEventListener('click', () => {
            navItems.forEach(i => i.classList.remove('active'));
            item.classList.add('active');
        });
    });

    // Open dashboard button
    document.getElementById('open-dashboard').addEventListener('click', () => {
        window.open('https://admin.tigerwallet.com', '_blank');
    });
});

function applyTheme(theme) {
    document.documentElement.setAttribute('data-theme', theme);
    const themeToggle = document.getElementById('theme-toggle');
    if (themeToggle) {
        themeToggle.checked = theme === 'dark';
    }
}

async function loadTheme() {
    return new Promise((resolve) => {
        chrome.storage.local.get(['theme'], (result) => {
            resolve(result.theme || 'light');
        });
    });
}

function saveTheme(theme) {
    chrome.storage.local.set({ theme });
}
