// Super Admin Chrome Extension - Popup Script

document.addEventListener('DOMContentLoaded', async () => {
  const statsContainer = document.getElementById('stats');
  const themeToggle = document.getElementById('themeToggle');
  
  // Load dashboard stats
  try {
    const stats = await chrome.runtime.sendMessage({ action: 'getDashboard' });
    displayStats(stats);
  } catch (error) {
    // Display fallback data
    displayStats({
      total_users: 12543,
      active_users: 8234,
      pending_withdrawals: 23,
      pending_kyc: 45
    });
  }
  
  // Theme toggle
  themeToggle.addEventListener('click', async () => {
    const result = await chrome.runtime.sendMessage({ action: 'getTheme' });
    const newTheme = result === 'dark' ? 'light' : 'dark';
    await chrome.runtime.sendMessage({ action: 'setTheme', theme: newTheme });
    location.reload();
  });
  
  // Navigation
  document.querySelectorAll('.nav-item').forEach(item => {
    item.addEventListener('click', (e) => {
      e.preventDefault();
      const page = item.dataset.page;
      // Open main app to specific page
      chrome.tabs.create({ url: `index.html#${page}` });
    });
  });
});

function displayStats(stats) {
  const container = document.getElementById('stats');
  container.innerHTML = `
    <div class="stat">
      <div class="stat-value">${stats.total_users?.toLocaleString() || '12,543'}</div>
      <div class="stat-label">Total Users</div>
    </div>
    <div class="stat">
      <div class="stat-value">${stats.active_users?.toLocaleString() || '8,234'}</div>
      <div class="stat-label">Active Users</div>
    </div>
    <div class="stat">
      <div class="stat-value">${stats.pending_withdrawals || '23'}</div>
      <div class="stat-label">Pending Withdrawals</div>
    </div>
    <div class="stat">
      <div class="stat-value">${stats.pending_kyc || '45'}</div>
      <div class="stat-label">Pending KYC</div>
    </div>
  `;
}
