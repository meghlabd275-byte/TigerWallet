/**
 * TigerWallet Admin - Desktop Renderer
 * Complete Dark/Light Theme Support and Navigation
 */

class TigerAdminApp {
  constructor() {
    this.currentScreen = 'dashboard';
    this.theme = 'light';
    this.init();
  }

  async init() {
    await this.initTheme();
    this.setupNavigation();
    this.setupEventListeners();
    this.loadData();
  }

  async initTheme() {
    // Get theme from electron or localStorage
    if (window.electronAPI) {
      this.theme = await window.electronAPI.getTheme();
    } else {
      this.theme = localStorage.getItem('theme') || 'light';
    }
    this.applyTheme();
  }

  applyTheme() {
    document.documentElement.setAttribute('data-theme', this.theme);
    localStorage.setItem('theme', this.theme);
    
    const themeBtn = document.getElementById('theme-toggle');
    const darkModeToggle = document.getElementById('dark-mode-toggle');
    
    if (themeBtn) {
      themeBtn.querySelector('.label').textContent = this.theme === 'dark' ? 'Light Mode' : 'Dark Mode';
      themeBtn.querySelector('.icon').textContent = this.theme === 'dark' ? '☀️' : '🌙';
    }
    
    if (darkModeToggle) {
      darkModeToggle.checked = this.theme === 'dark';
    }
  }

  toggleTheme() {
    this.theme = this.theme === 'dark' ? 'light' : 'dark';
    this.applyTheme();
    
    if (window.electronAPI) {
      window.electronAPI.setTheme(this.theme);
    }
  }

  setupNavigation() {
    const navItems = document.querySelectorAll('.nav-item');
    navItems.forEach(item => {
      item.addEventListener('click', (e) => {
        e.preventDefault();
        const screen = item.dataset.screen;
        this.navigateTo(screen);
      });
    });
  }

  navigateTo(screenName) {
    // Update nav
    document.querySelectorAll('.nav-item').forEach(item => {
      item.classList.remove('active');
      if (item.dataset.screen === screenName) {
        item.classList.add('active');
      }
    });

    // Update screen
    document.querySelectorAll('.screen').forEach(screen => {
      screen.classList.remove('active');
    });
    document.getElementById(`screen-${screenName}`)?.classList.add('active');

    // Update title
    const title = document.getElementById('page-title');
    const screenTitle = screenName.charAt(0).toUpperCase() + screenName.slice(1);
    title.textContent = screenTitle;

    this.currentScreen = screenName;
  }

  setupEventListeners() {
    // Theme toggle
    document.getElementById('theme-toggle')?.addEventListener('click', () => this.toggleTheme());
    document.getElementById('dark-mode-toggle')?.addEventListener('change', (e) => {
      this.theme = e.target.checked ? 'dark' : 'light';
      this.applyTheme();
    });

    // Tabs
    document.querySelectorAll('.tab').forEach(tab => {
      tab.addEventListener('click', () => {
        document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
        tab.classList.add('active');
      });
    });
  }

  loadData() {
    this.loadUsers();
    this.loadTransactions();
    this.loadKYC();
  }

  loadUsers() {
    const tbody = document.getElementById('users-table-body');
    if (!tbody) return;

    const users = [
      { id: 'USR001', name: 'John Doe', email: 'john@example.com', status: 'Active', kyc: 'Level 2' },
      { id: 'USR002', name: 'Jane Smith', email: 'jane@example.com', status: 'Active', kyc: 'Level 3' },
      { id: 'USR003', name: 'Bob Wilson', email: 'bob@example.com', status: 'Suspended', kyc: 'Level 1' },
      { id: 'USR004', name: 'Alice Brown', email: 'alice@example.com', status: 'Active', kyc: 'Level 2' },
      { id: 'USR005', name: 'Charlie Davis', email: 'charlie@example.com', status: 'Banned', kyc: 'Pending' }
    ];

    tbody.innerHTML = users.map(user => `
      <tr>
        <td>${user.id}</td>
        <td>${user.name}</td>
        <td>${user.email}</td>
        <td><span class="status-badge status-${user.status.toLowerCase()}">${user.status}</span></td>
        <td>${user.kyc}</td>
        <td>
          <button class="btn-action">View</button>
          <button class="btn-action">Edit</button>
        </td>
      </tr>
    `).join('');
  }

  loadTransactions() {
    const tbody = document.getElementById('transactions-table-body');
    if (!tbody) return;

    const transactions = [
      { hash: '0x1234...abcd', type: 'Deposit', amount: '$1,000', status: 'Confirmed', time: '2 min ago' },
      { hash: '0x5678...efgh', type: 'Withdraw', amount: '$500', status: 'Pending', time: '5 min ago' },
      { hash: '0x9abc...ijkl', type: 'Transfer', amount: '$250', status: 'Confirmed', time: '10 min ago' },
      { hash: '0xdef0...mnop', type: 'Swap', amount: '$1,500', status: 'Confirmed', time: '15 min ago' }
    ];

    tbody.innerHTML = transactions.map(tx => `
      <tr>
        <td>${tx.hash}</td>
        <td>${tx.type}</td>
        <td>${tx.amount}</td>
        <td><span class="status-badge status-${tx.status.toLowerCase()}">${tx.status}</span></td>
        <td>${tx.time}</td>
      </tr>
    `).join('');
  }

  loadKYC() {
    const list = document.getElementById('kyc-list');
    if (!list) return;

    const requests = [
      { name: 'John Doe', level: 'Level 2', country: 'US', type: 'Passport' },
      { name: 'Jane Smith', level: 'Level 3', country: 'UK', type: 'ID Card' },
      { name: 'Bob Wilson', level: 'Level 1', country: 'CA', type: 'Driver License' }
    ];

    list.innerHTML = requests.map(req => `
      <div class="kyc-card">
        <div class="kyc-info">
          <h3>${req.name}</h3>
          <p>${req.country} - ${req.type}</p>
        </div>
        <div class="kyc-actions">
          <button class="btn-approve">Approve</button>
          <button class="btn-reject">Reject</button>
        </div>
      </div>
    `).join('');
  }
}

// Initialize app
document.addEventListener('DOMContentLoaded', () => {
  window.app = new TigerAdminApp();
});
