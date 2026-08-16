/**
 * TigerWallet Super Admin - Renderer
 * Drives the 12 governance domain screens via the preload IPC bridge that
 * calls the real super_admin/go backend on :8082. Every screen has loading,
 * error, and empty states; no fabricated data is ever shown.
 */
(function () {
  'use strict';

  const tiger = window.tiger;
  const screensEl = document.getElementById('screens');
  const navEl = document.getElementById('domainNav');
  const titleEl = document.getElementById('screenTitle');
  const tokenInput = document.getElementById('tokenInput');
  const saveTokenBtn = document.getElementById('saveTokenBtn');
  const themeToggleBtn = document.getElementById('themeToggleBtn');

  const modalBackdrop = document.getElementById('modalBackdrop');
  const modalTitle = document.getElementById('modalTitle');
  const modalBody = document.getElementById('modalBody');
  const modalCancel = document.getElementById('modalCancel');
  const modalSave = document.getElementById('modalSave');

  let domains = [];
  let activeDomain = null;
  let modalCallback = null;

  // ---- Theme ----
  async function initTheme() {
    const dark = await tiger.getTheme();
    document.documentElement.setAttribute('data-theme', dark ? 'dark' : 'light');
  }
  tiger.onThemeChanged((dark) => {
    document.documentElement.setAttribute('data-theme', dark ? 'dark' : 'light');
  });
  themeToggleBtn.addEventListener('click', async () => {
    const dark = await tiger.getTheme();
    await tiger.setTheme(!dark);
    document.documentElement.setAttribute('data-theme', !dark ? 'dark' : 'light');
  });

  // ---- Token ----
  saveTokenBtn.addEventListener('click', async () => {
    await tiger.setToken(tokenInput.value.trim());
    flash('Token saved.');
    if (activeDomain) refresh(activeDomain);
  });

  // ---- Navigation + screens ----
  async function init() {
    await initTheme();
    domains = await tiger.domains();
    buildNav();
    if (domains.length > 0) selectDomain(domains[0]);
  }

  function buildNav() {
    navEl.replaceChildren();
    domains.forEach((d) => {
      const btn = document.createElement('button');
      btn.className = 'nav-item';
      btn.textContent = d.label;
      btn.dataset.domain = d.id;
      btn.addEventListener('click', () => selectDomain(d));
      navEl.appendChild(btn);
    });
  }

  function selectDomain(d) {
    activeDomain = d;
    titleEl.textContent = d.label;
    document.querySelectorAll('.nav-item[data-domain]').forEach((b) => {
      b.classList.toggle('active', b.dataset.domain === d.id);
    });
    document.querySelectorAll('.screen').forEach((s) => s.classList.remove('active'));
    let screen = document.getElementById('screen-' + d.id);
    if (!screen) {
      screen = buildScreen(d);
      screensEl.appendChild(screen);
    }
    screen.classList.add('active');
    refresh(d);
  }

  function buildScreen(d) {
    const screen = document.createElement('section');
    screen.className = 'screen';
    screen.id = 'screen-' + d.id;

    const card = document.createElement('div');
    card.className = 'card';

    const toolbar = document.createElement('div');
    toolbar.className = 'toolbar';
    const left = document.createElement('div');
    left.className = 'count';
    left.textContent = d.label;
    const right = document.createElement('div');
    const createBtn = document.createElement('button');
    createBtn.className = 'btn btn-primary';
    createBtn.textContent = '+ New';
    createBtn.addEventListener('click', () => openCreate(d));
    const refreshBtn = document.createElement('button');
    refreshBtn.className = 'btn';
    refreshBtn.textContent = '↻ Refresh';
    refreshBtn.addEventListener('click', () => refresh(d));
    right.append(createBtn, ' ', refreshBtn);
    toolbar.append(left, right);

    const body = document.createElement('div');
    body.className = 'screen-body';
    body.id = 'body-' + d.id;
    body.innerHTML = '<div class="state">Loading...</div>';

    card.append(toolbar, body);
    screen.appendChild(card);
    return screen;
  }

  async function refresh(d) {
    const body = document.getElementById('body-' + d.id);
    if (!body) return;
    setLoading(body, 'Loading ' + d.label + '...');
    const result = await tiger.domainCall({ domain: d.id, op: 'list' });
    if (result && result.error) {
      setError(body, result.error);
      return;
    }
    const rows = extractArray(result && result.data);
    renderTable(d, body, rows);
  }

  function renderTable(d, body, rows) {
    body.replaceChildren();
    const count = document.createElement('div');
    count.className = 'state';
    if (!rows || rows.length === 0) {
      const empty = document.createElement('div');
      empty.className = 'state';
      empty.textContent = 'No ' + d.label.toLowerCase() + ' records found.';
      body.appendChild(empty);
      return;
    }

    const tableWrap = document.createElement('div');
    const table = document.createElement('table');
    const cols = pickColumns(rows);

    const thead = document.createElement('thead');
    const headRow = document.createElement('tr');
    cols.forEach((c) => {
      const th = document.createElement('th');
      th.textContent = c;
      headRow.appendChild(th);
    });
    const actionTh = document.createElement('th');
    actionTh.textContent = 'Actions';
    headRow.appendChild(actionTh);
    thead.appendChild(headRow);
    table.appendChild(thead);

    const tbody = document.createElement('tbody');
    rows.forEach((row) => {
      const tr = document.createElement('tr');
      cols.forEach((c) => {
        const td = document.createElement('td');
        td.textContent = formatValue(row[c]);
        tr.appendChild(td);
      });
      const actionsTd = document.createElement('td');
      actionsTd.className = 'actions';
      appendActions(d, actionsTd, row);
      tr.appendChild(actionsTd);
      tbody.appendChild(tr);
    });
    table.appendChild(tbody);
    tableWrap.appendChild(table);
    body.appendChild(tableWrap);
  }

  function appendActions(d, container, row) {
    const id = idOf(row);
    const editBtn = document.createElement('button');
    editBtn.className = 'btn';
    editBtn.textContent = 'Edit';
    editBtn.addEventListener('click', () => openEdit(d, row));
    container.appendChild(editBtn);

    const delBtn = document.createElement('button');
    delBtn.className = 'btn btn-danger';
    delBtn.textContent = 'Delete';
    delBtn.style.marginLeft = '4px';
    delBtn.addEventListener('click', () => doDelete(d, id));
    container.appendChild(delBtn);

    if (d.actions.includes('status')) {
      const statusBtn = document.createElement('button');
      statusBtn.className = 'btn';
      statusBtn.textContent = 'Set Status';
      statusBtn.style.marginLeft = '4px';
      statusBtn.addEventListener('click', () => openStatus(d, id, row.status));
      container.appendChild(statusBtn);
    }
    if (d.actions.includes('approve')) {
      const ap = document.createElement('button');
      ap.className = 'btn';
      ap.textContent = 'Approve';
      ap.style.marginLeft = '4px';
      ap.addEventListener('click', () => doAction(d, 'approve', id));
      container.appendChild(ap);
    }
    if (d.actions.includes('reject')) {
      const rj = document.createElement('button');
      rj.className = 'btn btn-danger';
      rj.textContent = 'Reject';
      rj.style.marginLeft = '4px';
      rj.addEventListener('click', () => openReject(d, id));
      container.appendChild(rj);
    }
  }

  // ---- Modal-driven create / edit / status / reject ----
  function openModal(title, fieldsHtml, onSave) {
    modalTitle.textContent = title;
    modalBody.innerHTML = fieldsHtml;
    modalCallback = onSave;
    modalBackdrop.classList.add('show');
  }
  function closeModal() {
    modalBackdrop.classList.remove('show');
    modalCallback = null;
    modalBody.innerHTML = '';
  }
  modalCancel.addEventListener('click', closeModal);
  modalSave.addEventListener('click', () => {
    if (!modalCallback) return;
    const values = collectForm();
    modalCallback(values);
  });

  function collectForm() {
    const values = {};
    modalBody.querySelectorAll('[data-field]').forEach((el) => {
      values[el.dataset.field] = el.value;
    });
    return values;
  }

  function openCreate(d) {
    const fields = guessFields(d, null);
    openModal('New ' + d.label, fieldsHtml(fields), async (values) => {
      const result = await tiger.domainCall({ domain: d.id, op: 'create', body: values });
      if (result && result.error) { flash(result.error, true); return; }
      closeModal();
      refresh(d);
    });
  }

  function openEdit(d, row) {
    const fields = guessFields(d, row);
    openModal('Edit ' + d.label, fieldsHtml(fields), async (values) => {
      const result = await tiger.domainCall({ domain: d.id, op: 'update', id: idOf(row), body: values });
      if (result && result.error) { flash(result.error, true); return; }
      closeModal();
      refresh(d);
    });
  }

  function openStatus(d, id, current) {
    const html =
      label('status') +
      selectField('status', ['pending', 'active', 'paused', 'inactive', 'suspended', 'completed', 'failed'], current || 'pending');
    openModal('Set Status', html, async (values) => {
      const result = await tiger.domainAction({ domain: d.id, action: 'status', id, status: values.status });
      if (result && result.error) { flash(result.error, true); return; }
      closeModal();
      refresh(d);
    });
  }

  function openReject(d, id) {
    const html = label('reason') + textField('reason', '');
    openModal('Reject record', html, async (values) => {
      if (!values.reason || !values.reason.trim()) { flash('A reason is required.', true); return; }
      const result = await tiger.domainAction({ domain: d.id, action: 'reject', id, reason: values.reason });
      if (result && result.error) { flash(result.error, true); return; }
      closeModal();
      refresh(d);
    });
  }

  async function doAction(d, action, id) {
    const result = await tiger.domainAction({ domain: d.id, action, id });
    if (result && result.error) { flash(result.error, true); return; }
    refresh(d);
  }

  async function doDelete(d, id) {
    if (!confirm('Delete this ' + d.label.toLowerCase() + ' record?')) return;
    const result = await tiger.domainCall({ domain: d.id, op: 'delete', id });
    if (result && result.error) { flash(result.error, true); return; }
    refresh(d);
  }

  // ---- Helpers ----
  function setLoading(body, msg) {
    body.replaceChildren();
    const el = document.createElement('div');
    el.className = 'state';
    el.textContent = msg;
    body.appendChild(el);
  }
  function setError(body, msg) {
    body.replaceChildren();
    const el = document.createElement('div');
    el.className = 'state error';
    el.textContent = msg;
    body.appendChild(el);
  }

  function flash(msg, isError) {
    const toast = document.createElement('div');
    toast.className = 'state' + (isError ? ' error' : '');
    toast.textContent = msg;
    toast.style.position = 'fixed';
    toast.style.bottom = '16px';
    toast.style.right = '16px';
    toast.style.background = 'var(--surface)';
    toast.style.border = '1px solid var(--border)';
    toast.style.padding = '10px 14px';
    toast.style.borderRadius = '8px';
    toast.style.zIndex = '20';
    document.body.appendChild(toast);
    setTimeout(() => toast.remove(), 3000);
  }

  function extractArray(data) {
    if (Array.isArray(data)) return data;
    if (data && typeof data === 'object') {
      for (const v of Object.values(data)) {
        if (Array.isArray(v)) return v;
      }
    }
    return [];
  }

  function pickColumns(rows) {
    const seen = {};
    rows.forEach((r) => {
      Object.keys(r || {}).forEach((k) => { seen[k] = (seen[k] || 0) + 1; });
    });
    return Object.keys(seen).sort((a, b) => {
      // id-like columns first
      const ai = a.toLowerCase() === 'id' || a.toLowerCase().endsWith('_id') ? 0 : 1;
      const bi = b.toLowerCase() === 'id' || b.toLowerCase().endsWith('_id') ? 0 : 1;
      if (ai !== bi) return ai - bi;
      return seen[b] - seen[a];
    }).slice(0, 6);
  }

  function idOf(row) {
    if (!row) return null;
    return row.id || row.ID || row.uuid || row.Uuid || row._id || null;
  }

  function formatValue(v) {
    if (v === null || v === undefined) return '—';
    if (typeof v === 'object') return JSON.stringify(v);
    if (typeof v === 'number') return Number.isInteger(v) ? String(v) : String(v);
    if (typeof v === 'boolean') return v ? 'yes' : 'no';
    return String(v);
  }

  function label(text) {
    return '<div class="form-row"><label>' + escapeHtml(text) + '</label></div>';
  }
  function textField(name, value) {
    return '<div class="form-row"><input data-field="' + name + '" value="' + escapeAttr(value) + '" /></div>';
  }
  function selectField(name, options, current) {
    const opts = options.map((o) =>
      '<option value="' + escapeAttr(o) + '"' + (o === current ? ' selected' : '') + '>' + escapeHtml(o) + '</option>'
    ).join('');
    return '<div class="form-row"><select data-field="' + name + '">' + opts + '</select></div>';
  }
  function fieldsHtml(fields) {
    return fields.map((f) =>
      label(f.name) + textField(f.name, f.value || '')
    ).join('');
  }

  function guessFields(d, row) {
    // Default editable fields per domain. These are guidance fields; the Go
    // backend validates and ignores unknown keys.
    const presets = {
      futures: ['pair', 'side', 'size', 'leverage', 'entry_price', 'liquidation_price', 'margin', 'chain_id'],
      options: ['underlying', 'option_type', 'strike', 'expiry', 'premium', 'size', 'chain_id'],
      'copy-trading': ['follower_id', 'leader_id', 'allocation', 'max_leverage'],
      convert: ['user_id', 'from_token', 'to_token', 'from_amount', 'to_amount', 'rate', 'chain_id'],
      onramp: ['user_id', 'provider', 'fiat_currency', 'crypto_token', 'fiat_amount', 'crypto_amount'],
      offramp: ['user_id', 'provider', 'crypto_token', 'fiat_currency', 'crypto_amount', 'fiat_amount'],
      'p2p-clients': ['user_id', 'username'],
      partners: ['name', 'contact_email', 'revenue_share'],
      rewards: ['name', 'reward_type', 'amount', 'token', 'start_at', 'end_at'],
      marketing: ['name', 'channel', 'budget', 'start_at', 'end_at'],
      'admin-roles': ['name', 'description'],
      'wl-control': ['name', 'domain']
    };
    const names = presets[d.id] || ['name'];
    return names.map((n) => ({ name: n, value: row ? formatValue(row[n]) : '' }));
  }

  function escapeHtml(s) {
    return String(s).replace(/[&<>"']/g, (c) => ({
      '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
    }[c]));
  }
  function escapeAttr(s) {
    return escapeHtml(s).replace(/"/g, '&quot;');
  }

  // Navigate handler (tray menu) -> select domain by route name.
  tiger.onNavigate((route) => {
    const d = domains.find((x) => x.id === route);
    if (d) selectDomain(d);
  });

  init();
})();
