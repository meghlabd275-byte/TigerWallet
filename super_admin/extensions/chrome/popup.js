// WebExtension API compatibility shim.
// Firefox MV2 exposes the promise-based `browser.*` namespace but not
// `chrome.*` promises; Chromium MV3 exposes `chrome.*` (with promises in
// service workers) but not `browser.*`. Alias whichever is present so the
// same source runs in both engines without per-browser code copies.
if (typeof browser !== "undefined" && typeof chrome === "undefined") {
  // eslint-disable-next-line no-global-assign
  chrome = browser;
} else if (typeof chrome !== "undefined" && typeof browser === "undefined") {
  // eslint-disable-next-line no-global-assign
  browser = chrome;
}
// Super Admin Chrome Extension - Popup Script
// Renders the dashboard plus 12 read-only governance domain sections (GET
// only). All data comes from the real super_admin/go backend on :8082; a load
// failure surfaces an honest error, an empty list shows an empty state.

document.addEventListener('DOMContentLoaded', async () => {
  await applyStoredTheme();

  const themeToggle = document.getElementById('themeToggle');
  const tabsBar = document.getElementById('domainTabs');
  const domainPanel = document.getElementById('domainPanel');

  // Load dashboard stats from the real super-admin backend.
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
    const current = document.documentElement.getAttribute('data-theme') || 'light';
    const newTheme = current === 'dark' ? 'light' : 'dark';
    await chrome.runtime.sendMessage({ action: 'setTheme', theme: newTheme });
    await applyStoredTheme();
  });

  // Build the 12 domain tabs + load the first one.
  const domains = await chrome.runtime.sendMessage({ action: 'getDomains' });
  const domainList = (domains && domains.domains) || [];
  buildTabs(domainList);

  if (domainList.length > 0) {
    loadDomain(domainList[0]);
  }

  function buildTabs(list) {
    tabsBar.replaceChildren();
    list.forEach((d, idx) => {
      const btn = document.createElement('button');
      btn.className = 'domain-tab' + (idx === 0 ? ' active' : '');
      btn.textContent = d.label;
      btn.dataset.domain = d.id;
      btn.dataset.resource = d.resource;
      btn.addEventListener('click', () => {
        document.querySelectorAll('.domain-tab').forEach((b) => b.classList.remove('active'));
        btn.classList.add('active');
        loadDomain(d);
      });
      tabsBar.appendChild(btn);
    });
  }

  async function loadDomain(d) {
    domainPanel.replaceChildren();
    const loading = document.createElement('div');
    loading.className = 'domain-loading';
    loading.textContent = 'Loading ' + d.label + '...';
    domainPanel.appendChild(loading);

    let result;
    try {
      result = await chrome.runtime.sendMessage({ action: 'getDomain', domain: d.resource });
    } catch (error) {
      renderDomainError(d, 'Failed to load ' + d.label + ': ' + (error && error.message ? error.message : 'unknown error'));
      return;
    }

    if (!result) {
      renderDomainError(d, 'No data returned by the ' + d.label + ' service.');
      return;
    }
    if (result.error) {
      renderDomainError(d, result.error);
      return;
    }
    renderDomainList(d, result);
  }

  function renderDomainList(d, data) {
    domainPanel.replaceChildren();
    const rows = extractArray(data);
    const header = document.createElement('div');
    header.className = 'domain-header';
    header.textContent = d.label + ' (' + rows.length + ')';
    domainPanel.appendChild(header);

    if (rows.length === 0) {
      const empty = document.createElement('div');
      empty.className = 'domain-empty';
      empty.textContent = 'No ' + d.label.toLowerCase() + ' records found.';
      domainPanel.appendChild(empty);
      return;
    }

    const list = document.createElement('div');
    list.className = 'domain-list';
    rows.slice(0, 25).forEach((row) => {
      const item = document.createElement('div');
      item.className = 'domain-row';
      const entries = Object.entries(row || {});
      if (entries.length === 0) {
        item.textContent = '\u2014';
      } else {
        entries.slice(0, 6).forEach(([k, v]) => {
          const field = document.createElement('div');
          field.className = 'domain-field';
          const lbl = document.createElement('span');
          lbl.className = 'domain-field-label';
          lbl.textContent = k + ': ';
          const val = document.createElement('span');
          val.className = 'domain-field-value';
          val.textContent = formatValue(v);
          field.appendChild(lbl);
          field.appendChild(val);
          item.appendChild(field);
        });
      }
      list.appendChild(item);
    });
    domainPanel.appendChild(list);
  }

  function renderDomainError(d, message) {
    domainPanel.replaceChildren();
    const header = document.createElement('div');
    header.className = 'domain-header';
    header.textContent = d.label;
    domainPanel.appendChild(header);
    const err = document.createElement('div');
    err.className = 'stat-error';
    err.textContent = message;
    domainPanel.appendChild(err);
  }

  // Navigation
  document.querySelectorAll('.nav-item').forEach(item => {
    item.addEventListener('click', (e) => {
      e.preventDefault();
      const page = item.dataset.page;
      // Open main app to specific page
      chrome.tabs.create({ url: 'index.html#' + page });
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
      (value != null ? String(value) : '\u2014');
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

// Pull the first JSON array out of an arbitrary response object. The Go
// backend uses different per-domain keys (positions, contracts, clients...).
function extractArray(data) {
  if (Array.isArray(data)) return data;
  if (data && typeof data === 'object') {
    for (const v of Object.values(data)) {
      if (Array.isArray(v)) return v;
    }
  }
  return [];
}

function formatValue(v) {
  if (v === null || v === undefined) return '\u2014';
  if (typeof v === 'object') return JSON.stringify(v);
  if (typeof v === 'number') return Number.isInteger(v) ? String(v) : v.toFixed(4).replace(/\.?0+$/, '');
  return String(v);
}

async function applyStoredTheme() {
  let theme;
  try {
    theme = await chrome.runtime.sendMessage({ action: 'getTheme' });
  } catch (_) {
    theme = 'system';
  }
  const resolved = theme === 'dark' ? 'dark'
    : theme === 'light' ? 'light'
    : (matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
  document.documentElement.setAttribute('data-theme', resolved);
}
