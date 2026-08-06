// Popup script
document.addEventListener('DOMContentLoaded', () => {
  // Load theme
  chrome.storage.local.get(['darkMode'], (result) => {
    if (result.darkMode) {
      document.body.classList.add('dark');
    }
  });

  // Theme toggle
  document.getElementById('theme-toggle').addEventListener('click', () => {
    document.body.classList.toggle('dark');
    chrome.storage.local.set({ darkMode: document.body.classList.contains('dark') });
  });

  // Open dashboard
  document.getElementById('open-dashboard').addEventListener('click', () => {
    chrome.tabs.create({ url: 'http://localhost:9091' });
  });

  // Load stats (simulated)
  document.getElementById('wl-count').textContent = '0';
  document.getElementById('user-count').textContent = '0';
  document.getElementById('tx-count').textContent = '0';
  document.getElementById('pending-count').textContent = '0';
});
