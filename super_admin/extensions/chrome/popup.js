// Super Admin Chrome Extension - Popup Script

document.addEventListener('DOMContentLoaded', async () => {
  const statsContainer = document.getElementById('stats');
  const themeToggle = document.getElementById('themeToggle');
  
  // Load dashboard stats from the real super-admin backend. No fabricated
  // fallback numbers are ever shown; a load failure surfaces an honest error.
  try {
    const stats = await chrome.runtime.sendMessage({ action: 'getDashboard' });
    if (stats && stats.error) {
      displayError(stats.error);
    } else if (stats) {
      displayStats(stats);
    } else {
      displayError('No data returned by the dashboard service.');
    }
  } catch (error) {
    displayError('Failed to load dashboard stats: ' + (error && error.message ? error.message : 'unknown error'));
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
  // Build the stats grid with DOM APIs (createElement + textContent) instead
  // of innerHTML, so backend values cannot inject markup (XSS-safe).
  const container = document.getElementById('stats');
  container.replaceChildren();

  const fields = [
    { key: 'total_users', label: 'Total Users' },
    { key: 'active_users', label: 'Active Users' },
    { key: 'pending_withdrawals', label: 'Pending Withdrawals' },
    { key: 'pending_kyc', label: 'Pending KYC' }
  ];

  for (const f of fields) {
    const value = stats[f.key];
    const div = document.createElement('div');
    div.className = 'stat';

    const valueEl = document.createElement('div');
    valueEl.className = 'stat-value';
    valueEl.textContent =
      typeof value === 'number' ? value.toLocaleString() :
      (value != null ? String(value) : '—');
    div.appendChild(valueEl);

    const labelEl = document.createElement('div');
    labelEl.className = 'stat-label';
    labelEl.textContent = f.label;
    div.appendChild(labelEl);

    container.appendChild(div);
  }
}

function displayError(message) {
  const container = document.getElementById('stats');
  container.replaceChildren();
  const el = document.createElement('div');
  el.className = 'stat-error';
  el.textContent = message;
  container.appendChild(el);
}
