// Popup script for UserWallet extension
document.addEventListener('DOMContentLoaded', () => {
  // Theme toggle
  document.getElementById('toggleTheme').addEventListener('click', async () => {
    const isDark = await browser.storage.local.get('theme');
    const newTheme = isDark.theme === 'dark' ? 'light' : 'dark';
    await browser.storage.local.set({ theme: newTheme });
    document.documentElement.setAttribute('data-theme', newTheme);
  });
  
  // Button handlers
  document.getElementById('viewDashboard').addEventListener('click', () => {
    // Open dashboard in new tab
    window.open('http://localhost:8105', '_blank');
  });
  
  document.getElementById('viewWallets').addEventListener('click', () => {
    window.open('http://localhost:8105/wallets', '_blank');
  });
  
  document.getElementById('viewTransactions').addEventListener('click', () => {
    window.open('http://localhost:8105/transactions', '_blank');
  });
  
  // Load theme
  browser.storage.local.get('theme').then(result => {
    if (result.theme) {
      document.documentElement.setAttribute('data-theme', result.theme);
    }
  });
});
