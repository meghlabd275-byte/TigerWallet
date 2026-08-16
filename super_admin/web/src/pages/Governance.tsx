/**
 * TigerWallet Super Admin - Governance Panel
 *
 * SuperAdmin control plane over the license_service: WL client lifecycle
 * (approve/suspend/halt-all/revoke/resume), per-product licenses, per-fetcher
 * feature flags, and the two-party withdrawal co-sign gate.
 *
 * All data is fetched live from the license control plane via licenseApi —
 * no stubs, no fakes, no mocks. Theme-aware via CSS variables
 * (var(--bg-primary), var(--text-primary), ...) set in globals.css.
 */

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { useTheme } from '../context/ThemeContext';
import licenseApi, {
  type WLClient,
  type License,
  type FeatureFlag,
  type WithdrawalApproval,
} from '../services/licenseApi';

type TabKey = 'clients' | 'licenses' | 'flags' | 'withdrawals';

const PRODUCTS = ['master_wallet', 'user_wallet', 'bots', 'project_party'] as const;
const PLANS = ['basic', 'starter', 'professional', 'enterprise'] as const;
const TIERS = ['basic', 'starter', 'professional', 'enterprise'] as const;

export default function Governance() {
  const { resolvedTheme } = useTheme();
  const [tab, setTab] = useState<TabKey>('clients');

  return (
    <div className="p-6" style={{ minHeight: '100vh', backgroundColor: 'var(--bg-primary)' }}>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 style={{ color: 'var(--text-primary)', fontSize: '1.875rem', fontWeight: 700, margin: 0 }}>
            Governance
          </h1>
          <p style={{ color: 'var(--text-secondary)', marginTop: '0.25rem' }}>
            License control plane — WL lifecycle, product licenses, per-fetcher flags, two-party withdrawals.
          </p>
        </div>
        <span
          className="badge badge-neutral"
          title={`Theme: ${resolvedTheme}`}
          style={{ textTransform: 'capitalize' }}
        >
          {resolvedTheme}
        </span>
      </div>

      {/* Tabs */}
      <div
        className="flex gap-2 mb-6"
        style={{ borderBottom: '1px solid var(--border-primary)', flexWrap: 'wrap' }}
      >
        <TabButton active={tab === 'clients'} onClick={() => setTab('clients')}>
          WL Clients
        </TabButton>
        <TabButton active={tab === 'licenses'} onClick={() => setTab('licenses')}>
          Licenses
        </TabButton>
        <TabButton active={tab === 'flags'} onClick={() => setTab('flags')}>
          Feature Flags
        </TabButton>
        <TabButton active={tab === 'withdrawals'} onClick={() => setTab('withdrawals')}>
          Two-Party Withdrawals
        </TabButton>
      </div>

      {tab === 'clients' && <WLClientsTab />}
      {tab === 'licenses' && <LicensesTab />}
      {tab === 'flags' && <FeatureFlagsTab />}
      {tab === 'withdrawals' && <WithdrawalsTab />}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Shared UI primitives (theme-aware via CSS variables)
// ---------------------------------------------------------------------------

function TabButton({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      onClick={onClick}
      style={{
        padding: '0.625rem 1rem',
        background: active ? 'var(--accent-primary)' : 'transparent',
        color: active ? '#ffffff' : 'var(--text-secondary)',
        border: 'none',
        borderBottom: active ? '2px solid var(--accent-primary)' : '2px solid transparent',
        borderRadius: 'var(--radius-md) var(--radius-md) 0 0',
        cursor: 'pointer',
        fontWeight: 600,
        fontSize: '0.875rem',
      }}
    >
      {children}
    </button>
  );
}

function Panel({ children }: { children: React.ReactNode }) {
  return (
    <div className="card" style={{ backgroundColor: 'var(--bg-elevated)' }}>
      <div className="card-body">{children}</div>
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  const map: Record<string, string> = {
    active: 'badge-success',
    approved: 'badge-success',
    pending: 'badge-warning',
    suspended: 'badge-warning',
    halted: 'badge-error',
    revoked: 'badge-error',
    rejected: 'badge-error',
    wl_approved: 'badge-warning',
  };
  const cls = map[status] || 'badge-neutral';
  return <span className={`badge ${cls}`}>{status}</span>;
}

function ActionButton({
  variant,
  onClick,
  disabled,
  children,
  title,
}: {
  variant: 'primary' | 'secondary' | 'success' | 'danger' | 'kill' | 'resume';
  onClick: () => void;
  disabled?: boolean;
  children: React.ReactNode;
  title?: string;
}) {
  const styles: Record<string, React.CSSProperties> = {
    primary: { backgroundColor: 'var(--accent-primary)', color: '#ffffff' },
    secondary: {
      backgroundColor: 'var(--bg-tertiary)',
      color: 'var(--text-primary)',
      border: '1px solid var(--border-primary)',
    },
    success: { backgroundColor: 'var(--success)', color: '#ffffff' },
    danger: { backgroundColor: 'var(--error)', color: '#ffffff' },
    kill: {
      backgroundColor: 'var(--error)',
      color: '#ffffff',
      fontWeight: 700,
      boxShadow: 'var(--shadow-md)',
    },
    resume: {
      backgroundColor: 'var(--success)',
      color: '#ffffff',
      fontWeight: 700,
      boxShadow: 'var(--shadow-md)',
    },
  };
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      title={title}
      style={{
        padding: '0.5rem 0.875rem',
        fontSize: '0.8125rem',
        borderRadius: 'var(--radius-md)',
        border: 'none',
        cursor: disabled ? 'not-allowed' : 'pointer',
        opacity: disabled ? 0.5 : 1,
        ...styles[variant],
      }}
    >
      {children}
    </button>
  );
}

function ErrorBanner({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div className="alert alert-error mb-4">
      <p style={{ color: 'var(--error-border)', margin: 0 }}>{message}</p>
      {onRetry && (
        <button className="btn-secondary mt-2" onClick={onRetry} style={{ marginTop: '0.5rem' }}>
          Retry
        </button>
      )}
    </div>
  );
}

function Loader() {
  return (
    <div className="flex items-center justify-center p-8">
      <div className="loader"></div>
    </div>
  );
}

function EmptyState({ text }: { text: string }) {
  return (
    <Panel>
      <p style={{ color: 'var(--text-secondary)', textAlign: 'center', padding: '2rem 0' }}>
        {text}
      </p>
    </Panel>
  );
}

function FormField({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="form-group" style={{ flex: 1, minWidth: '200px' }}>
      <label
        style={{
          color: 'var(--text-secondary)',
          fontSize: '0.8125rem',
          display: 'block',
          marginBottom: '0.25rem',
        }}
      >
        {label}
      </label>
      {children}
    </div>
  );
}

const inputStyle: React.CSSProperties = {
  width: '100%',
  backgroundColor: 'var(--bg-primary)',
  color: 'var(--text-primary)',
  border: '1px solid var(--border-primary)',
};

const submitButtonStyle: React.CSSProperties = {
  backgroundColor: 'var(--accent-primary)',
  color: '#ffffff',
  border: 'none',
  padding: '0.625rem 1.25rem',
  borderRadius: 'var(--radius-md)',
  cursor: 'pointer',
  fontSize: '0.875rem',
  fontWeight: 500,
  alignSelf: 'flex-start',
};

// ---------------------------------------------------------------------------
// Hook: async action runner with action-level loading guard
// ---------------------------------------------------------------------------

function useActionRunner(reload: () => void) {
  const [actionLoading, setActionLoading] = useState(false);
  const run = useCallback(
    async (fn: () => Promise<any>, reloadAfter = true, confirmMsg?: string) => {
      if (confirmMsg && !window.confirm(confirmMsg)) return;
      setActionLoading(true);
      try {
        await fn();
        if (reloadAfter) reload();
      } catch (err: any) {
        window.alert(err?.message || 'Action failed');
      } finally {
        setActionLoading(false);
      }
    },
    [reload]
  );
  return { actionLoading, run };
}

// ===========================================================================
// Tab 1: WL Clients
// ===========================================================================

function WLClientsTab() {
  const [items, setItems] = useState<WLClient[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({
    name: '',
    slug: '',
    contact_email: '',
    tier: 'basic',
    products: 'master_wallet,user_wallet,bots,project_party',
  });
  const [creating, setCreating] = useState(false);

  const load = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await licenseApi.listWLClients();
      setItems(data);
    } catch (err: any) {
      setError(err?.message || 'Failed to load WL clients');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const { actionLoading, run } = useActionRunner(load);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreating(true);
    try {
      const products = form.products
        .split(',')
        .map((p) => p.trim())
        .filter(Boolean);
      await licenseApi.createWLClient({
        name: form.name,
        slug: form.slug,
        contact_email: form.contact_email,
        tier: form.tier,
        products,
      });
      setShowForm(false);
      setForm({
        name: '',
        slug: '',
        contact_email: '',
        tier: 'basic',
        products: 'master_wallet,user_wallet,bots,project_party',
      });
      load();
    } catch (err: any) {
      window.alert(err?.message || 'Failed to create WL client');
    } finally {
      setCreating(false);
    }
  };

  return (
    <div>
      <div className="flex justify-between items-center mb-4" style={{ flexWrap: 'wrap', gap: '0.5rem' }}>
        <h3 style={{ color: 'var(--text-primary)', margin: 0 }}>WL Clients</h3>
        <ActionButton variant="primary" onClick={() => setShowForm((s) => !s)}>
          {showForm ? 'Cancel' : 'New WL Client'}
        </ActionButton>
      </div>

      {showForm && (
        <Panel>
          <h4 style={{ color: 'var(--text-primary)', marginBottom: '1rem' }}>Create WL Client</h4>
          <form onSubmit={handleCreate} className="flex flex-col gap-3">
            <div className="flex gap-3" style={{ flexWrap: 'wrap' }}>
              <FormField label="Name">
                <input
                  className="input"
                  style={inputStyle}
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  required
                />
              </FormField>
              <FormField label="Slug (domain)">
                <input
                  className="input"
                  style={inputStyle}
                  value={form.slug}
                  onChange={(e) => setForm({ ...form, slug: e.target.value })}
                  placeholder="acme"
                  required
                />
              </FormField>
            </div>
            <div className="flex gap-3" style={{ flexWrap: 'wrap' }}>
              <FormField label="Owner Email">
                <input
                  className="input"
                  style={inputStyle}
                  type="email"
                  value={form.contact_email}
                  onChange={(e) => setForm({ ...form, contact_email: e.target.value })}
                  required
                />
              </FormField>
              <FormField label="Tier / Plan">
                <select
                  className="input"
                  style={inputStyle}
                  value={form.tier}
                  onChange={(e) => setForm({ ...form, tier: e.target.value })}
                >
                  {TIERS.map((t) => (
                    <option key={t} value={t}>
                      {t}
                    </option>
                  ))}
                </select>
              </FormField>
            </div>
            <FormField label="Allowed Products (comma-separated)">
              <input
                className="input"
                style={inputStyle}
                value={form.products}
                onChange={(e) => setForm({ ...form, products: e.target.value })}
              />
            </FormField>
            <button type="submit" disabled={creating} style={submitButtonStyle}>
              {creating ? 'Creating...' : 'Create'}
            </button>
          </form>
        </Panel>
      )}

      {error ? (
        <ErrorBanner message={error} onRetry={load} />
      ) : loading ? (
        <Loader />
      ) : items.length === 0 ? (
        <EmptyState text="No WL clients found." />
      ) : (
        <Panel>
          <div className="overflow-x-auto">
            <table className="table">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Slug</th>
                  <th>Tier</th>
                  <th>Contact</th>
                  <th>Products</th>
                  <th>Status</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {items.map((w) => (
                  <tr key={w.id}>
                    <td style={{ color: 'var(--text-primary)', fontWeight: 600 }}>{w.name}</td>
                    <td style={{ color: 'var(--text-secondary)' }}>{w.slug}</td>
                    <td style={{ color: 'var(--text-secondary)' }}>{w.tier}</td>
                    <td style={{ color: 'var(--text-secondary)' }}>{w.contact_email}</td>
                    <td style={{ color: 'var(--text-secondary)' }}>
                      {(w.allowed_products || []).join(', ') || '—'}
                    </td>
                    <td>
                      <StatusBadge status={w.status} />
                    </td>
                    <td>
                      <div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                        {(w.status === 'pending' || w.status === 'suspended' || w.status === 'halted' || w.status === 'revoked') && (
                          <ActionButton
                            variant="success"
                            disabled={actionLoading}
                            onClick={() => run(() => licenseApi.approveWLClient(w.id))}
                            title="Approve"
                          >
                            Approve
                          </ActionButton>
                        )}
                        {w.status !== 'suspended' && w.status !== 'revoked' && w.status !== 'halted' && (
                          <ActionButton
                            variant="secondary"
                            disabled={actionLoading}
                            onClick={() => run(() => licenseApi.suspendWLClient(w.id), true, 'Suspend this WL client? All its products will be halted.')}
                          >
                            Suspend
                          </ActionButton>
                        )}
                        {w.status !== 'halted' && w.status !== 'revoked' && (
                          <ActionButton
                            variant="kill"
                            disabled={actionLoading}
                            onClick={() =>
                              run(
                                () => licenseApi.haltWLClient(w.id),
                                true,
                                'KILL SWITCH: Halt ALL products for this WL client immediately? Externally-hosted products will stop serving on next heartbeat.'
                              )
                            }
                            title="Halt ALL products (kill switch)"
                          >
                            ⛔ Halt All
                          </ActionButton>
                        )}
                        {w.status !== 'revoked' && (
                          <ActionButton
                            variant="danger"
                            disabled={actionLoading}
                            onClick={() => run(() => licenseApi.revokeWLClient(w.id), true, 'Revoke this WL client license? This is destructive.')}
                          >
                            Revoke
                          </ActionButton>
                        )}
                        {(w.status === 'suspended' || w.status === 'halted' || w.status === 'revoked') && (
                          <ActionButton
                            variant="resume"
                            disabled={actionLoading}
                            onClick={() => run(() => licenseApi.resumeWLClient(w.id), true, 'Resume this WL client and all its licenses?')}
                            title="SuperAdmin-only resume"
                          >
                            ▶ Resume
                          </ActionButton>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Panel>
      )}
    </div>
  );
}

// ===========================================================================
// Tab 2: Licenses
// ===========================================================================

function LicensesTab() {
  const [items, setItems] = useState<License[]>([]);
  const [clients, setClients] = useState<WLClient[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [creating, setCreating] = useState(false);
  const [filterClient, setFilterClient] = useState('');
  const [form, setForm] = useState({
    wl_client_id: '',
    product: 'master_wallet' as string,
    plan: 'basic',
    duration_days: '365',
    max_users: '100',
    max_wallets: '500',
    max_bots: '50',
    features: '',
  });

  const load = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const [cs, ls] = await Promise.all([
        licenseApi.listWLClients(),
        licenseApi.listLicenses(filterClient || undefined),
      ]);
      setClients(cs);
      setItems(ls);
    } catch (err: any) {
      setError(err?.message || 'Failed to load licenses');
    } finally {
      setLoading(false);
    }
  }, [filterClient]);

  useEffect(() => {
    load();
  }, [load]);

  const { actionLoading, run } = useActionRunner(load);

  const clientName = (id: string) => {
    const c = clients.find((x) => x.id === id);
    return c ? c.name : id.slice(0, 8);
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!form.wl_client_id) {
      window.alert('Select a WL client first');
      return;
    }
    setCreating(true);
    try {
      const features = form.features
        .split(',')
        .map((f) => f.trim())
        .filter(Boolean);
      await licenseApi.issueLicense({
        wl_client_id: form.wl_client_id,
        product: form.product,
        plan: form.plan,
        duration_days: Number(form.duration_days) || 365,
        max_users: Number(form.max_users) || 100,
        max_wallets: Number(form.max_wallets) || 500,
        max_bots: Number(form.max_bots) || 50,
        features,
      });
      setShowForm(false);
      setForm({
        wl_client_id: form.wl_client_id,
        product: 'master_wallet',
        plan: 'basic',
        duration_days: '365',
        max_users: '100',
        max_wallets: '500',
        max_bots: '50',
        features: '',
      });
      load();
    } catch (err: any) {
      window.alert(err?.message || 'Failed to issue license');
    } finally {
      setCreating(false);
    }
  };

  return (
    <div>
      <div className="flex justify-between items-center mb-4" style={{ flexWrap: 'wrap', gap: '0.5rem' }}>
        <div className="flex items-center gap-3" style={{ flexWrap: 'wrap' }}>
          <h3 style={{ color: 'var(--text-primary)', margin: 0 }}>Product Licenses</h3>
          <select
            className="input"
            style={{ ...inputStyle, width: 'auto', minWidth: '180px' }}
            value={filterClient}
            onChange={(e) => setFilterClient(e.target.value)}
          >
            <option value="">All WL clients</option>
            {clients.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
        </div>
        <ActionButton variant="primary" onClick={() => setShowForm((s) => !s)}>
          {showForm ? 'Cancel' : 'Issue License'}
        </ActionButton>
      </div>

      {showForm && (
        <Panel>
          <h4 style={{ color: 'var(--text-primary)', marginBottom: '1rem' }}>Issue License</h4>
          <form onSubmit={handleCreate} className="flex flex-col gap-3">
            <div className="flex gap-3" style={{ flexWrap: 'wrap' }}>
              <FormField label="WL Client">
                <select
                  className="input"
                  style={inputStyle}
                  value={form.wl_client_id}
                  onChange={(e) => setForm({ ...form, wl_client_id: e.target.value })}
                  required
                >
                  <option value="">— select —</option>
                  {clients.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.name} ({c.slug})
                    </option>
                  ))}
                </select>
              </FormField>
              <FormField label="Product">
                <select
                  className="input"
                  style={inputStyle}
                  value={form.product}
                  onChange={(e) => setForm({ ...form, product: e.target.value })}
                >
                  {PRODUCTS.map((p) => (
                    <option key={p} value={p}>
                      {p}
                    </option>
                  ))}
                </select>
              </FormField>
              <FormField label="Plan">
                <select
                  className="input"
                  style={inputStyle}
                  value={form.plan}
                  onChange={(e) => setForm({ ...form, plan: e.target.value })}
                >
                  {PLANS.map((p) => (
                    <option key={p} value={p}>
                      {p}
                    </option>
                  ))}
                </select>
              </FormField>
              <FormField label="Duration (days)">
                <input
                  className="input"
                  style={inputStyle}
                  type="number"
                  value={form.duration_days}
                  onChange={(e) => setForm({ ...form, duration_days: e.target.value })}
                />
              </FormField>
            </div>
            <div className="flex gap-3" style={{ flexWrap: 'wrap' }}>
              <FormField label="Max Users">
                <input
                  className="input"
                  style={inputStyle}
                  type="number"
                  value={form.max_users}
                  onChange={(e) => setForm({ ...form, max_users: e.target.value })}
                />
              </FormField>
              <FormField label="Max Wallets">
                <input
                  className="input"
                  style={inputStyle}
                  type="number"
                  value={form.max_wallets}
                  onChange={(e) => setForm({ ...form, max_wallets: e.target.value })}
                />
              </FormField>
              <FormField label="Max Bots">
                <input
                  className="input"
                  style={inputStyle}
                  type="number"
                  value={form.max_bots}
                  onChange={(e) => setForm({ ...form, max_bots: e.target.value })}
                />
              </FormField>
              <FormField label="Features (comma-separated)">
                <input
                  className="input"
                  style={inputStyle}
                  value={form.features}
                  onChange={(e) => setForm({ ...form, features: e.target.value })}
                  placeholder="e.g. vault,trading"
                />
              </FormField>
            </div>
            <button type="submit" disabled={creating} style={submitButtonStyle}>
              {creating ? 'Issuing...' : 'Issue'}
            </button>
          </form>
        </Panel>
      )}

      {error ? (
        <ErrorBanner message={error} onRetry={load} />
      ) : loading ? (
        <Loader />
      ) : items.length === 0 ? (
        <EmptyState text="No licenses found." />
      ) : (
        <Panel>
          <div className="overflow-x-auto">
            <table className="table">
              <thead>
                <tr>
                  <th>WL Client</th>
                  <th>Product</th>
                  <th>Plan</th>
                  <th>Status</th>
                  <th>Valid Until</th>
                  <th>Limits</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {items.map((l) => (
                  <tr key={l.id}>
                    <td style={{ color: 'var(--text-primary)', fontWeight: 600 }}>
                      {clientName(l.wl_client_id)}
                    </td>
                    <td style={{ color: 'var(--text-secondary)' }}>{l.product}</td>
                    <td style={{ color: 'var(--text-secondary)' }}>{l.plan}</td>
                    <td>
                      <StatusBadge status={l.status} />
                    </td>
                    <td style={{ color: 'var(--text-secondary)' }}>
                      {l.valid_until ? new Date(l.valid_until).toLocaleDateString() : '—'}
                    </td>
                    <td style={{ color: 'var(--text-secondary)', fontSize: '0.8125rem' }}>
                      users: {l.max_users} · wallets: {l.max_wallets} · bots: {l.max_bots}
                    </td>
                    <td>
                      <div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                        {l.status !== 'suspended' && l.status !== 'halted' && l.status !== 'revoked' && (
                          <ActionButton
                            variant="secondary"
                            disabled={actionLoading}
                            onClick={() => run(() => licenseApi.suspendLicense(l.id), true, 'Suspend this license?')}
                          >
                            Suspend
                          </ActionButton>
                        )}
                        {l.status !== 'halted' && l.status !== 'revoked' && (
                          <ActionButton
                            variant="kill"
                            disabled={actionLoading}
                            onClick={() => run(() => licenseApi.haltLicense(l.id), true, 'Halt this product license immediately?')}
                            title="Halt product (kill switch)"
                          >
                            ⛔ Halt
                          </ActionButton>
                        )}
                        {l.status !== 'revoked' && (
                          <ActionButton
                            variant="danger"
                            disabled={actionLoading}
                            onClick={() => run(() => licenseApi.revokeLicense(l.id), true, 'Revoke this license? This is destructive.')}
                          >
                            Revoke
                          </ActionButton>
                        )}
                        {(l.status === 'suspended' || l.status === 'halted' || l.status === 'revoked') && (
                          <ActionButton
                            variant="resume"
                            disabled={actionLoading}
                            onClick={() => run(() => licenseApi.resumeLicense(l.id), true, 'Resume this license? (SuperAdmin-only)')}
                            title="SuperAdmin-only resume"
                          >
                            ▶ Resume
                          </ActionButton>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Panel>
      )}
    </div>
  );
}

// ===========================================================================
// Tab 3: Feature Flags (per-fetcher)
// ===========================================================================

function FeatureFlagsTab() {
  const [clients, setClients] = useState<WLClient[]>([]);
  const [flags, setFlags] = useState<FeatureFlag[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filterClient, setFilterClient] = useState('');
  const [filterProduct, setFilterProduct] = useState('');
  const [showForm, setShowForm] = useState(false);
  const [creating, setCreating] = useState(false);
  const [form, setForm] = useState({
    wl_client_id: '',
    product: 'user_wallet' as string,
    fetcher: '',
    enabled: true,
  });

  const load = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const cs = await licenseApi.listWLClients();
      setClients(cs);
      if (filterClient) {
        const fs = await licenseApi.listFeatureFlags(filterClient, filterProduct || undefined);
        setFlags(fs);
      } else {
        setFlags([]);
      }
    } catch (err: any) {
      setError(err?.message || 'Failed to load feature flags');
    } finally {
      setLoading(false);
    }
  }, [filterClient, filterProduct]);

  useEffect(() => {
    load();
  }, [load]);

  const { actionLoading, run } = useActionRunner(load);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!form.wl_client_id) {
      window.alert('Select a WL client first');
      return;
    }
    setCreating(true);
    try {
      await licenseApi.setFeatureFlag({
        wl_client_id: form.wl_client_id,
        product: form.product,
        fetcher: form.fetcher || '*',
        enabled: form.enabled,
      });
      setShowForm(false);
      setForm({ wl_client_id: form.wl_client_id, product: 'user_wallet', fetcher: '', enabled: true });
      if (!filterClient) setFilterClient(form.wl_client_id);
      load();
    } catch (err: any) {
      window.alert(err?.message || 'Failed to set feature flag');
    } finally {
      setCreating(false);
    }
  };

  const toggleFlag = (f: FeatureFlag) => {
    run(() =>
      licenseApi.setFeatureFlag({
        wl_client_id: f.wl_client_id,
        product: f.product,
        fetcher: f.fetcher,
        enabled: !f.enabled,
      })
    );
  };

  return (
    <div>
      <div className="flex justify-between items-center mb-4" style={{ flexWrap: 'wrap', gap: '0.5rem' }}>
        <div className="flex items-center gap-3" style={{ flexWrap: 'wrap' }}>
          <h3 style={{ color: 'var(--text-primary)', margin: 0 }}>Feature Flags</h3>
          <select
            className="input"
            style={{ ...inputStyle, width: 'auto', minWidth: '180px' }}
            value={filterClient}
            onChange={(e) => {
              setFilterClient(e.target.value);
              setFilterProduct('');
            }}
          >
            <option value="">— select WL client —</option>
            {clients.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
          <select
            className="input"
            style={{ ...inputStyle, width: 'auto', minWidth: '160px' }}
            value={filterProduct}
            onChange={(e) => setFilterProduct(e.target.value)}
            disabled={!filterClient}
          >
            <option value="">All products</option>
            {PRODUCTS.map((p) => (
              <option key={p} value={p}>
                {p}
              </option>
            ))}
          </select>
        </div>
        <ActionButton variant="primary" onClick={() => setShowForm((s) => !s)} disabled={!filterClient}>
          {showForm ? 'Cancel' : 'Add Flag'}
        </ActionButton>
      </div>

      <p style={{ color: 'var(--text-secondary)', fontSize: '0.8125rem', marginBottom: '1rem' }}>
        Per-fetcher granularity: disable an individual fetcher (e.g. <code>user_wallet.send</code>) for
        a WL client while leaving <code>user_wallet.balance</code> alive. Absent flags default to enabled.
      </p>

      {showForm && (
        <Panel>
          <h4 style={{ color: 'var(--text-primary)', marginBottom: '1rem' }}>Set Feature Flag</h4>
          <form onSubmit={handleCreate} className="flex flex-col gap-3">
            <div className="flex gap-3" style={{ flexWrap: 'wrap' }}>
              <FormField label="WL Client">
                <select
                  className="input"
                  style={inputStyle}
                  value={form.wl_client_id}
                  onChange={(e) => setForm({ ...form, wl_client_id: e.target.value })}
                  required
                >
                  <option value="">— select —</option>
                  {clients.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.name}
                    </option>
                  ))}
                </select>
              </FormField>
              <FormField label="Product">
                <select
                  className="input"
                  style={inputStyle}
                  value={form.product}
                  onChange={(e) => setForm({ ...form, product: e.target.value })}
                >
                  {PRODUCTS.map((p) => (
                    <option key={p} value={p}>
                      {p}
                    </option>
                  ))}
                </select>
              </FormField>
              <FormField label="Fetcher (use * for whole product)">
                <input
                  className="input"
                  style={inputStyle}
                  value={form.fetcher}
                  onChange={(e) => setForm({ ...form, fetcher: e.target.value })}
                  placeholder="e.g. send"
                />
              </FormField>
              <FormField label="Enabled">
                <select
                  className="input"
                  style={inputStyle}
                  value={form.enabled ? 'true' : 'false'}
                  onChange={(e) => setForm({ ...form, enabled: e.target.value === 'true' })}
                >
                  <option value="true">enabled</option>
                  <option value="false">disabled</option>
                </select>
              </FormField>
            </div>
            <button type="submit" disabled={creating} style={submitButtonStyle}>
              {creating ? 'Saving...' : 'Set Flag'}
            </button>
          </form>
        </Panel>
      )}

      {!filterClient ? (
        <EmptyState text="Select a WL client to view its feature flags." />
      ) : error ? (
        <ErrorBanner message={error} onRetry={load} />
      ) : loading ? (
        <Loader />
      ) : flags.length === 0 ? (
        <EmptyState text="No feature flags set for this WL client. All fetchers default to enabled." />
      ) : (
        <Panel>
          <div className="overflow-x-auto">
            <table className="table">
              <thead>
                <tr>
                  <th>Product</th>
                  <th>Fetcher</th>
                  <th>State</th>
                  <th>Updated</th>
                  <th>Action</th>
                </tr>
              </thead>
              <tbody>
                {flags.map((f) => (
                  <tr key={f.id}>
                    <td style={{ color: 'var(--text-primary)', fontWeight: 600 }}>{f.product}</td>
                    <td style={{ color: 'var(--text-secondary)' }}>
                      <code style={{ backgroundColor: 'var(--bg-tertiary)', padding: '0.125rem 0.375rem', borderRadius: 'var(--radius-sm)' }}>
                        {f.fetcher}
                      </code>
                    </td>
                    <td>
                      <span
                        className={`badge ${f.enabled ? 'badge-success' : 'badge-error'}`}
                      >
                        {f.enabled ? 'enabled' : 'disabled'}
                      </span>
                    </td>
                    <td style={{ color: 'var(--text-secondary)' }}>
                      {f.updated_at ? new Date(f.updated_at).toLocaleString() : '—'}
                    </td>
                    <td>
                      <ActionButton
                        variant={f.enabled ? 'danger' : 'success'}
                        disabled={actionLoading}
                        onClick={() => toggleFlag(f)}
                      >
                        {f.enabled ? 'Disable' : 'Enable'}
                      </ActionButton>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Panel>
      )}
    </div>
  );
}

// ===========================================================================
// Tab 4: Two-Party Withdrawals
// ===========================================================================

function WithdrawalsTab() {
  const [items, setItems] = useState<WithdrawalApproval[]>([]);
  const [clients, setClients] = useState<WLClient[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filterClient, setFilterClient] = useState('');

  const load = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const [cs, ws] = await Promise.all([
        licenseApi.listWLClients(),
        licenseApi.listWithdrawals(filterClient || undefined),
      ]);
      setClients(cs);
      setItems(ws);
    } catch (err: any) {
      setError(err?.message || 'Failed to load withdrawal approvals');
    } finally {
      setLoading(false);
    }
  }, [filterClient]);

  useEffect(() => {
    load();
  }, [load]);

  const { actionLoading, run } = useActionRunner(load);

  const clientName = (id: string) => {
    const c = clients.find((x) => x.id === id);
    return c ? c.name : id.slice(0, 8);
  };

  const pending = useMemo(
    () => items.filter((w) => w.status === 'wl_approved'),
    [items]
  );

  return (
    <div>
      <div className="flex justify-between items-center mb-4" style={{ flexWrap: 'wrap', gap: '0.5rem' }}>
        <div className="flex items-center gap-3" style={{ flexWrap: 'wrap' }}>
          <h3 style={{ color: 'var(--text-primary)', margin: 0 }}>Two-Party Withdrawals</h3>
          <select
            className="input"
            style={{ ...inputStyle, width: 'auto', minWidth: '180px' }}
            value={filterClient}
            onChange={(e) => setFilterClient(e.target.value)}
          >
            <option value="">All WL clients</option>
            {clients.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
        </div>
        <span className="badge badge-warning" title="Awaiting SuperAdmin co-sign">
          {pending.length} pending co-sign
        </span>
      </div>

      <p style={{ color: 'var(--text-secondary)', fontSize: '0.8125rem', marginBottom: '1rem' }}>
        No fund/revenue moves without SuperAdmin approval here. Each withdrawal is filed by a WL client
        (<code>wl_approved</code>) and only becomes executable after the SuperAdmin co-signs
        (<code>approved</code>).
      </p>

      {error ? (
        <ErrorBanner message={error} onRetry={load} />
      ) : loading ? (
        <Loader />
      ) : items.length === 0 ? (
        <EmptyState text="No withdrawal approvals." />
      ) : (
        <Panel>
          <div className="overflow-x-auto">
            <table className="table">
              <thead>
                <tr>
                  <th>WL Client</th>
                  <th>Product</th>
                  <th>Resource</th>
                  <th>To Address</th>
                  <th>Amount (wei)</th>
                  <th>Chain</th>
                  <th>Status</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {items.map((w) => (
                  <tr key={w.id}>
                    <td style={{ color: 'var(--text-primary)', fontWeight: 600 }}>
                      {clientName(w.wl_client_id)}
                    </td>
                    <td style={{ color: 'var(--text-secondary)' }}>{w.product}</td>
                    <td style={{ color: 'var(--text-secondary)', fontSize: '0.8125rem' }}>
                      {w.resource_type}:{w.resource_id}
                    </td>
                    <td style={{ color: 'var(--text-secondary)', fontSize: '0.8125rem' }}>
                      <code>{w.to_address}</code>
                    </td>
                    <td style={{ color: 'var(--text-secondary)', fontVariantNumeric: 'tabular-nums' }}>
                      {w.amount_wei}
                    </td>
                    <td style={{ color: 'var(--text-secondary)' }}>{w.chain_id}</td>
                    <td>
                      <StatusBadge status={w.status} />
                    </td>
                    <td>
                      <div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                        {w.status === 'wl_approved' && (
                          <>
                            <ActionButton
                              variant="success"
                              disabled={actionLoading}
                              onClick={() =>
                                run(
                                  () => licenseApi.approveWithdrawal(w.id),
                                  true,
                                  'Approve this withdrawal? This is the SuperAdmin co-sign — funds become executable.'
                                )
                              }
                            >
                              Approve
                            </ActionButton>
                            <ActionButton
                              variant="danger"
                              disabled={actionLoading}
                              onClick={() => run(() => licenseApi.rejectWithdrawal(w.id), true, 'Reject this withdrawal?')}
                            >
                              Reject
                            </ActionButton>
                          </>
                        )}
                        {w.status === 'approved' && (
                          <span style={{ color: 'var(--success)', fontSize: '0.8125rem' }}>
                            {w.tx_hash ? `tx: ${w.tx_hash.slice(0, 10)}…` : 'awaiting execution'}
                          </span>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Panel>
      )}
    </div>
  );
}
