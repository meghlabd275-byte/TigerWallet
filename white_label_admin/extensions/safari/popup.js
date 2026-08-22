// Popup script - 11 domain read-only sections + tabs. Identical across
// chrome/firefox/safari (only the tab-open API differs; see openDashboard()).
// Read-only: governance record metadata, no fund movement.

const DOMAINS = [
  { key: 'futures', label: 'Futures', endpoints: [['GET', '/futures'], ['PUT', '/futures/:id/status', true]] },
  { key: 'options', label: 'Options', endpoints: [['GET', '/options'], ['PUT', '/options/:id/status', true]] },
  { key: 'copy-trading', label: 'Copy', endpoints: [['GET', '/copy-trading'], ['PUT', '/copy-trading/:id/status', true]] },
  { key: 'convert', label: 'Convert', endpoints: [['GET', '/convert'], ['PUT', '/convert/:id/status', true]] },
  { key: 'onramp', label: 'On-Ramp', endpoints: [['GET', '/onramp'], ['POST', '/onramp/:id/approve', true], ['POST', '/onramp/:id/reject {reason}', true]] },
  { key: 'offramp', label: 'Off-Ramp', endpoints: [['GET', '/offramp'], ['POST', '/offramp/:id/approve', true], ['POST', '/offramp/:id/reject {reason}', true]] },
  { key: 'p2p-clients', label: 'P2P', endpoints: [['GET', '/p2p-clients'], ['PUT', '/p2p-clients/:id/status', true]] },
  { key: 'partners', label: 'Partners', endpoints: [['GET', '/partners'], ['PUT', '/partners/:id/status', true], ['POST', '/partners/:id/approve', true], ['POST', '/partners/:id/reject {reason}', true]] },
  { key: 'rewards', label: 'Rewards', endpoints: [['GET', '/rewards'], ['PUT', '/rewards/:id/status', true]] },
  { key: 'marketing', label: 'Marketing', endpoints: [['GET', '/marketing'], ['PUT', '/marketing/:id/status', true]] },
  { key: 'liquidity', label: 'Liquidity', endpoints: [['GET', '/wl-liquidity/sources'], ['POST', '/wl-liquidity/sources'], ['PUT', '/wl-liquidity/sources/:id'], ['DELETE', '/wl-liquidity/sources/:id'], ['GET', '/wl-liquidity/allocations'], ['POST', '/wl-liquidity/allocations'], ['GET', '/wl-liquidity/stats']] },
  { key: 'crypto-card', label: 'Cards', endpoints: [['GET', '/wl-cards'], ['POST', '/wl-cards'], ['GET', '/wl-cards/transactions'], ['GET', '/wl-cards/stats'], ['PUT', '/wl-cards/:id/status', true]] },
  { key: 'bots', label: 'Bots', endpoints: [['GET', '/wl-bots/operators'], ['POST', '/wl-bots/operators'], ['GET', '/wl-bots/config'], ['GET', '/wl-bots/stats'], ['PUT', '/wl-bots/operators/:id/status', true]] },
  { key: 'kyc', label: 'KYC', endpoints: [['GET', '/kyc'], ['POST', '/kyc/:id/approve', true], ['POST', '/kyc/:id/reject {reason}', true]] },
  { key: 'tickets', label: 'Tickets', endpoints: [['GET', '/tickets'], ['GET', '/tickets/:id'], ['POST', '/tickets'], ['PUT', '/tickets/:id/status', true], ['POST', '/tickets/:id/messages', true], ['PUT', '/tickets/:id/assign', true]] },
  { key: 'ip-whitelist', label: 'IP WL', endpoints: [['GET', '/ip-whitelist'], ['POST', '/ip-whitelist'], ['DELETE', '/ip-whitelist/:id', true]] },
  { key: 'audit-logs', label: 'Audit', endpoints: [['GET', '/audit-logs'], ['POST', '/audit-logs/export', true]] },
  { key: 'wallet-management', label: 'Wallets', endpoints: [['GET', '/withdrawals'], ['GET', '/fees'], ['POST', '/fees'], ['PUT', '/fees/:id'], ['PUT', '/users/:id/status', true], ['POST', '/withdrawals/:id/approve', true], ['POST', '/withdrawals/:id/reject {reason}', true], ['POST', '/withdrawals/:id/process', true]] },
  { key: 'withdrawals', label: 'Withdrawals', endpoints: [['GET', '/withdrawals'], ['POST', '/withdrawals/:id/approve', true], ['POST', '/withdrawals/:id/reject {reason}', true], ['POST', '/withdrawals/:id/process', true]] },
  { key: 'rbac', label: 'RBAC', endpoints: [['GET', '/admin-roles'], ['GET', '/admin-permissions'], ['POST', '/admins/:id/role', true], ['GET', '/admins/:id/permissions']] },
];

function buildTabs() {
  const tabs = document.getElementById('tabs');
  const sections = document.getElementById('sections');
  tabs.innerHTML = '';
  sections.innerHTML = '';
  DOMAINS.forEach((d, i) => {
    const tab = document.createElement('div');
    tab.className = 'tab' + (i === 0 ? ' active' : '');
    tab.textContent = d.label;
    tab.dataset.key = d.key;
    tab.addEventListener('click', () => {
      document.querySelectorAll('.tab').forEach((t) => t.classList.remove('active'));
      document.querySelectorAll('.section').forEach((s) => s.classList.remove('active'));
      tab.classList.add('active');
      const sec = sections.querySelector(`[data-section="${d.key}"]`);
      if (sec) sec.classList.add('active');
    });
    tabs.appendChild(tab);

    const sec = document.createElement('div');
    sec.className = 'section' + (i === 0 ? ' active' : '');
    sec.dataset.section = d.key;
    sec.innerHTML =
      '<div class="card"><h3>' + d.label + ' - read-only</h3>' +
      d.endpoints.map(([m, p, gov]) => '<div class="ep' + (gov ? ' gov' : '') + '">' + m + ' ' + p + '</div>').join('') +
      '<div class="note">Governance records only - no fund movement. WL backend :8082.</div></div>';
    sections.appendChild(sec);
  });
}

function openDashboard(url) {
  // chrome / firefox expose their own tab API; safari uses application.activeBrowserWindow.
  if (typeof chrome !== 'undefined' && chrome.tabs && chrome.tabs.create) {
    chrome.tabs.create({ url });
  } else if (typeof browser !== 'undefined' && browser.tabs && browser.tabs.create) {
    browser.tabs.create({ url });
  } else if (typeof safari !== 'undefined' && safari.application && safari.application.activeBrowserWindow) {
    safari.application.activeBrowserWindow.openTab().url = url;
  } else {
    window.open(url, '_blank');
  }
}

function initTheme() {
  const toggle = document.getElementById('theme-toggle');
  const apply = (dark) => {
    document.body.classList.toggle('dark', dark);
    toggle.textContent = dark ? 'Light' : 'Dark';
  };
  const store = (typeof chrome !== 'undefined' && chrome.storage) ? chrome.storage.local
    : (typeof browser !== 'undefined' && browser.storage) ? browser.storage.local : null;
  if (store) {
    store.get(['darkMode'], (res) => apply(!!(res && res.darkMode)));
    toggle.addEventListener('click', () => {
      const dark = !document.body.classList.contains('dark');
      store.set({ darkMode: dark });
      apply(dark);
    });
  } else {
    toggle.addEventListener('click', () => apply(!document.body.classList.contains('dark')));
  }
}

document.addEventListener('DOMContentLoaded', () => {
  buildTabs();
  initTheme();
  document.getElementById('open-dashboard').addEventListener('click', () => openDashboard('http://localhost:8082'));
});
