// MasterWallet Web — additional pages that wire the remaining api.ts methods
// to real screens (Policies, Fees, Notifications, Webhooks, Audit, Multisig,
// Chains, Tokens, Feature Flags, Passkeys). Every page fetches live data from
// the canonical MasterWallet backend (:8450) with loading/error/empty states
// and full light/dark theme support via useTheme(). No mock data.

import React, { useCallback, useEffect, useState, type FormEvent } from 'react';
import {
  masterWalletAPI,
  ApiError,
  type FeeConfig,
  type AuditLog,
  type NotificationItem,
  type Webhook,
  type MultisigWallet,
} from './api';

// ---------------- shared helpers ----------------

const card = (isDark: boolean) =>
  isDark ? 'bg-gray-800 border-gray-700 text-white' : 'bg-white border-gray-200 text-gray-900';
const muted = (isDark: boolean) => (isDark ? 'text-gray-400' : 'text-gray-500');
const inputCls = (isDark: boolean) =>
  `w-full px-3 py-2 rounded-lg border ${isDark ? 'bg-gray-900 border-gray-700 text-white' : 'bg-white border-gray-300 text-gray-900'} focus:outline-none focus:ring-2 focus:ring-blue-500`;
const btn = (isDark: boolean, variant: 'primary' | 'danger' | 'ghost' = 'primary') => {
  if (variant === 'danger') return 'px-3 py-1.5 rounded-lg text-sm bg-red-600 text-white hover:bg-red-700';
  if (variant === 'ghost')
    return `px-3 py-1.5 rounded-lg text-sm ${isDark ? 'bg-gray-700 text-gray-200 hover:bg-gray-600' : 'bg-gray-100 text-gray-700 hover:bg-gray-200'}`;
  return 'px-3 py-1.5 rounded-lg text-sm bg-blue-600 text-white hover:bg-blue-700';
};

function Banner({ isDark, loading, error }: { isDark: boolean; loading: boolean; error: string | null }) {
  if (loading)
    return <div className={`mb-4 p-3 rounded-lg ${isDark ? 'bg-gray-800 text-gray-300' : 'bg-blue-50 text-blue-700'}`}>Loading…</div>;
  if (error)
    return <div className="mb-4 p-3 rounded-lg bg-red-100 text-red-700">{error}</div>;
  return null;
}

const PageShell = ({ isDark, title, children }: { isDark: boolean; title: string; children: React.ReactNode }) => (
  <div className="space-y-4">
    <h2 className={`text-2xl font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>{title}</h2>
    {children}
  </div>
);

function useFetch<T>(loader: () => Promise<T>, deps: unknown[] = []): {
  data: T | null;
  loading: boolean;
  error: string | null;
  reload: () => Promise<void>;
  setData: React.Dispatch<React.SetStateAction<T | null>>;
} {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const run = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      setData(await loader());
    } catch (e) {
      setError(e instanceof ApiError ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);
  useEffect(() => { void run(); }, [run]);
  return { data, loading, error, reload: run, setData };
}

// ---------------- Policies ----------------

interface Policy {
  id: string;
  name: string;
  policy_type: string;
  is_active: boolean;
  priority: number;
  conditions?: Record<string, unknown>;
  actions?: Record<string, unknown>;
  created_at?: string;
}

export const PoliciesPage = ({ isDark, masterId }: { isDark: boolean; masterId?: string }) => {
  const { data, loading, error, reload } = useFetch(async () => {
    if (!masterId) return [] as Policy[];
    return masterWalletAPI.getPolicies(masterId);
  }, [masterId]);
  const [name, setName] = useState('');
  const [ptype, setPtype] = useState('spending_limit');
  const [priority, setPriority] = useState('1');
  const [submitting, setSubmitting] = useState(false);

  const create = async (e: FormEvent) => {
    e.preventDefault();
    if (!masterId || !name.trim()) return;
    setSubmitting(true);
    try {
      await masterWalletAPI.createPolicy(masterId, {
        name: name.trim(),
        policy_type: ptype,
        priority: Number(priority) || 0,
        conditions: {},
        actions: {},
      });
      setName('');
      void reload();
    } catch (err) {
      setError2(err instanceof ApiError ? err.message : String(err));
    } finally {
      setSubmitting(false);
    }
  };
  const [error2, setError2] = useState<string | null>(null);
  const remove = async (pid: string) => {
    if (!masterId) return;
    try {
      await masterWalletAPI.deletePolicy(masterId, pid);
      void reload();
    } catch (err) {
      setError2(err instanceof ApiError ? err.message : String(err));
    }
  };

  return (
    <PageShell isDark={isDark} title="Policies">
      <Banner isDark={isDark} loading={loading} error={error || error2} />
      <form onSubmit={create} className={`p-4 rounded-lg border space-y-3 ${card(isDark)}`}>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
          <input className={inputCls(isDark)} placeholder="Policy name" value={name} onChange={(e) => setName(e.target.value)} required />
          <select className={inputCls(isDark)} value={ptype} onChange={(e) => setPtype(e.target.value)}>
            <option value="spending_limit">spending_limit</option>
            <option value="whitelist">whitelist</option>
            <option value="rate_limit">rate_limit</option>
            <option value="approval_threshold">approval_threshold</option>
          </select>
          <input className={inputCls(isDark)} type="number" placeholder="Priority" value={priority} onChange={(e) => setPriority(e.target.value)} />
        </div>
        <button className={btn(isDark)} disabled={submitting || !masterId}>{submitting ? 'Creating…' : 'Create Policy'}</button>
      </form>
      <div className="space-y-2">
        {(data ?? []).length === 0 ? (
          <div className={`p-4 rounded-lg border ${card(isDark)} ${muted(isDark)}`}>No policies configured.</div>
        ) : (
          (data ?? []).map((p) => (
            <div key={p.id} className={`p-4 rounded-lg border flex items-center justify-between ${card(isDark)}`}>
              <div>
                <div className="font-semibold">{p.name}</div>
                <div className={`text-sm ${muted(isDark)}`}>{p.policy_type} · priority {p.priority} · {p.is_active ? 'active' : 'inactive'}</div>
              </div>
              <button className={btn(isDark, 'danger')} onClick={() => remove(p.id)}>Delete</button>
            </div>
          ))
        )}
      </div>
    </PageShell>
  );
};

// ---------------- Fees ----------------

export const FeesPage = ({ isDark, masterId }: { isDark: boolean; masterId?: string }) => {
  const { data, loading, error, reload } = useFetch<FeeConfig[]>(async () => {
    if (!masterId) return [] as FeeConfig[];
    return masterWalletAPI.getFeeConfigs(masterId);
  }, [masterId]);
  const [name, setName] = useState('');
  const [pct, setPct] = useState('0.5');
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const create = async (e: FormEvent) => {
    e.preventDefault();
    if (!masterId || !name.trim()) return;
    setSubmitting(true);
    try {
      await masterWalletAPI.createFeeConfig(masterId, {
        name: name.trim(),
        fee_type: 'percentage',
        fee_percentage: Number(pct) || 0,
      } as any);
      setName('');
      void reload();
    } catch (e2) {
      setErr(e2 instanceof ApiError ? e2.message : String(e2));
    } finally {
      setSubmitting(false);
    }
  };
  const remove = async (fid: string) => {
    if (!masterId) return;
    try {
      await masterWalletAPI.deleteFeeConfig(masterId, fid);
      void reload();
    } catch (e2) {
      setErr(e2 instanceof ApiError ? e2.message : String(e2));
    }
  };

  return (
    <PageShell isDark={isDark} title="Fees">
      <Banner isDark={isDark} loading={loading} error={error || err} />
      <form onSubmit={create} className={`p-4 rounded-lg border space-y-3 ${card(isDark)}`}>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          <input className={inputCls(isDark)} placeholder="Fee name" value={name} onChange={(e) => setName(e.target.value)} required />
          <input className={inputCls(isDark)} type="number" step="0.01" placeholder="Percentage %" value={pct} onChange={(e) => setPct(e.target.value)} />
        </div>
        <button className={btn(isDark)} disabled={submitting || !masterId}>{submitting ? 'Creating…' : 'Create Fee'}</button>
      </form>
      <div className="space-y-2">
        {(data ?? []).length === 0 ? (
          <div className={`p-4 rounded-lg border ${card(isDark)} ${muted(isDark)}`}>No fee configs.</div>
        ) : (
          ((data ?? []) as any[]).map((f) => (
            <div key={f.id} className={`p-4 rounded-lg border flex items-center justify-between ${card(isDark)}`}>
              <div>
                <div className="font-semibold">{f.fee_type ?? 'Fee'}</div>
                <div className={`text-sm ${muted(isDark)}`}>{f.fee_percentage != null ? `${f.fee_percentage}%` : (f.fee_fixed ?? '—')}</div>
              </div>
              <button className={btn(isDark, 'danger')} onClick={() => remove(f.id)}>Delete</button>
            </div>
          ))
        )}
      </div>
    </PageShell>
  );
};

// ---------------- Notifications ----------------

export const NotificationsPage = ({ isDark, masterId }: { isDark: boolean; masterId?: string }) => {
  const { data, loading, error, reload } = useFetch<NotificationItem[]>(async () => {
    if (!masterId) return [] as NotificationItem[];
    return masterWalletAPI.getNotifications(masterId);
  }, [masterId]);
  const [title, setTitle] = useState('');
  const [message, setMessage] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const create = async (e: FormEvent) => {
    e.preventDefault();
    if (!masterId || !title.trim()) return;
    setSubmitting(true);
    try {
      await masterWalletAPI.createNotification(masterId, { title: title.trim(), message: message.trim(), type: 'info' } as any);
      setTitle(''); setMessage('');
      void reload();
    } catch (e2) {
      setErr(e2 instanceof ApiError ? e2.message : String(e2));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <PageShell isDark={isDark} title="Notifications">
      <Banner isDark={isDark} loading={loading} error={error || err} />
      <form onSubmit={create} className={`p-4 rounded-lg border space-y-3 ${card(isDark)}`}>
        <input className={inputCls(isDark)} placeholder="Title" value={title} onChange={(e) => setTitle(e.target.value)} required />
        <textarea className={inputCls(isDark)} placeholder="Message" value={message} onChange={(e) => setMessage(e.target.value)} rows={2} />
        <button className={btn(isDark)} disabled={submitting || !masterId}>{submitting ? 'Sending…' : 'Send Notification'}</button>
      </form>
      <div className="space-y-2">
        {(data ?? []).length === 0 ? (
          <div className={`p-4 rounded-lg border ${card(isDark)} ${muted(isDark)}`}>No notifications.</div>
        ) : (
          (data ?? []).map((n) => (
            <div key={n.id} className={`p-4 rounded-lg border ${card(isDark)}`}>
              <div className="font-semibold">{n.title ?? 'Notification'}</div>
              <div className={`text-sm ${muted(isDark)}`}>{n.message}</div>
            </div>
          ))
        )}
      </div>
    </PageShell>
  );
};

// ---------------- Webhooks ----------------

export const WebhooksPage = ({ isDark, masterId }: { isDark: boolean; masterId?: string }) => {
  const { data, loading, error, reload } = useFetch<Webhook[]>(async () => {
    if (!masterId) return [] as Webhook[];
    return masterWalletAPI.getWebhooks(masterId);
  }, [masterId]);
  const [url, setUrl] = useState('');
  const [events, setEvents] = useState('transaction.confirmed,balance.updated');
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const create = async (e: FormEvent) => {
    e.preventDefault();
    if (!masterId || !url.trim()) return;
    setSubmitting(true);
    try {
      await masterWalletAPI.createWebhook(masterId, {
        url: url.trim(),
        events: events.split(',').map((s) => s.trim()).filter(Boolean),
      } as any);
      setUrl('');
      void reload();
    } catch (e2) {
      setErr(e2 instanceof ApiError ? e2.message : String(e2));
    } finally {
      setSubmitting(false);
    }
  };
  const remove = async (wid: string) => {
    if (!masterId) return;
    try {
      await masterWalletAPI.deleteWebhook(masterId, wid);
      void reload();
    } catch (e2) {
      setErr(e2 instanceof ApiError ? e2.message : String(e2));
    }
  };

  return (
    <PageShell isDark={isDark} title="Webhooks">
      <Banner isDark={isDark} loading={loading} error={error || err} />
      <form onSubmit={create} className={`p-4 rounded-lg border space-y-3 ${card(isDark)}`}>
        <input className={inputCls(isDark)} placeholder="https://example.com/webhook" value={url} onChange={(e) => setUrl(e.target.value)} required />
        <input className={inputCls(isDark)} placeholder="event types (comma separated)" value={events} onChange={(e) => setEvents(e.target.value)} />
        <button className={btn(isDark)} disabled={submitting || !masterId}>{submitting ? 'Creating…' : 'Add Webhook'}</button>
      </form>
      <div className="space-y-2">
        {(data ?? []).length === 0 ? (
          <div className={`p-4 rounded-lg border ${card(isDark)} ${muted(isDark)}`}>No webhooks.</div>
        ) : (
          (data ?? []).map((w) => (
            <div key={w.id} className={`p-4 rounded-lg border flex items-center justify-between ${card(isDark)}`}>
              <div>
                <div className="font-mono text-sm">{w.url}</div>
                <div className={`text-xs ${muted(isDark)}`}>{(w.events ?? []).join(', ')}</div>
              </div>
              <button className={btn(isDark, 'danger')} onClick={() => remove(w.id)}>Delete</button>
            </div>
          ))
        )}
      </div>
    </PageShell>
  );
};

// ---------------- Audit ----------------

export const AuditPage = ({ isDark, masterId }: { isDark: boolean; masterId?: string }) => {
  const { data, loading, error } = useFetch<AuditLog[]>(async () => {
    if (!masterId) return [] as AuditLog[];
    return masterWalletAPI.getAuditLogs(masterId);
  }, [masterId]);
  const logs = data ?? [];
  return (
    <PageShell isDark={isDark} title="Audit Log">
      <Banner isDark={isDark} loading={loading} error={error} />
      {logs.length === 0 ? (
        <div className={`p-4 rounded-lg border ${card(isDark)} ${muted(isDark)}`}>No audit entries.</div>
      ) : (
        <div className={`rounded-lg border overflow-hidden ${card(isDark)}`}>
          <table className="w-full text-sm">
            <thead className={isDark ? 'bg-gray-900' : 'bg-gray-50'}>
              <tr>
                <th className="text-left p-3">Event</th>
                <th className="text-left p-3">Actor</th>
                <th className="text-left p-3">Target</th>
                <th className="text-left p-3">Severity</th>
                <th className="text-left p-3">Time</th>
              </tr>
            </thead>
            <tbody>
              {logs.map((l) => (
                <tr key={l.id ?? l.event_type + String(l.created_at)} className={isDark ? 'border-t border-gray-700' : 'border-t border-gray-200'}>
                  <td className="p-3">{l.event_type}</td>
                  <td className="p-3">{l.actor_id ?? '—'}</td>
                  <td className="p-3">{l.target_id ?? '—'}</td>
                  <td className="p-3">{l.severity ?? '—'}</td>
                  <td className="p-3">{l.created_at ?? '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </PageShell>
  );
};

// ---------------- Multisig ----------------

export const MultisigPage = ({ isDark, masterId }: { isDark: boolean; masterId?: string }) => {
  const [view, setView] = useState<'wallets' | 'txs'>('wallets');
  const [selectedWallet, setSelectedWallet] = useState<string | null>(null);
  const { data: wallets, loading: lw, error: ew, reload: rw } = useFetch<MultisigWallet[]>(async () => {
    if (!masterId) return [] as MultisigWallet[];
    return masterWalletAPI.getMultisigWallets(masterId);
  }, [masterId]);
  const { data: txs, loading: lt, error: et } = useFetch<unknown[]>(async () => {
    if (!masterId || !selectedWallet) return [];
    return masterWalletAPI.getMultisigTransactions(masterId, selectedWallet);
  }, [masterId, selectedWallet]);
  const [name, setName] = useState('');
  const [owners, setOwners] = useState('');
  const [threshold, setThreshold] = useState('1');
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const create = async (e: FormEvent) => {
    e.preventDefault();
    if (!masterId || !name.trim()) return;
    setSubmitting(true);
    try {
      await masterWalletAPI.createMultisigWallet(masterId, {
        name: name.trim(),
        owners: owners.split(',').map((s) => s.trim()).filter(Boolean),
        threshold: Number(threshold) || 1,
      });
      setName(''); setOwners(''); setThreshold('1');
      void rw();
    } catch (e2) {
      setErr(e2 instanceof ApiError ? e2.message : String(e2));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <PageShell isDark={isDark} title="Multisig Wallets">
      <Banner isDark={isDark} loading={lw || lt} error={ew || et || err} />
      <div className="flex gap-2 mb-4">
        <button className={btn(isDark, view === 'wallets' ? 'primary' : 'ghost')} onClick={() => setView('wallets')}>Wallets</button>
        <button className={btn(isDark, view === 'txs' ? 'primary' : 'ghost')} onClick={() => setView('txs')} disabled={!selectedWallet}>Transactions</button>
      </div>
      <form onSubmit={create} className={`p-4 rounded-lg border space-y-3 mb-4 ${card(isDark)}`}>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
          <input className={inputCls(isDark)} placeholder="Wallet name" value={name} onChange={(e) => setName(e.target.value)} required />
          <input className={inputCls(isDark)} placeholder="owners (comma separated 0x…)" value={owners} onChange={(e) => setOwners(e.target.value)} />
          <input className={inputCls(isDark)} type="number" placeholder="threshold" value={threshold} onChange={(e) => setThreshold(e.target.value)} />
        </div>
        <button className={btn(isDark)} disabled={submitting || !masterId}>{submitting ? 'Creating…' : 'Create Multisig'}</button>
      </form>
      {view === 'wallets' ? (
        <div className="space-y-2">
          {(wallets ?? []).length === 0 ? (
            <div className={`p-4 rounded-lg border ${card(isDark)} ${muted(isDark)}`}>No multisig wallets.</div>
          ) : (
            (wallets ?? []).map((w: any) => (
              <div key={w.id} className={`p-4 rounded-lg border flex items-center justify-between ${card(isDark)}`}>
                <div>
                  <div className="font-semibold">{w.name}</div>
                  <div className={`text-sm ${muted(isDark)}`}>threshold {w.threshold} · {w.owners?.length ?? 0} owners · chain {w.chain_id ?? 1}</div>
                </div>
                <button className={btn(isDark, 'ghost')} onClick={() => { setSelectedWallet(w.id); setView('txs'); }}>View txs</button>
              </div>
            ))
          )}
        </div>
      ) : (
        <div className="space-y-2">
          {(txs ?? []).length === 0 ? (
            <div className={`p-4 rounded-lg border ${card(isDark)} ${muted(isDark)}`}>No transactions for this multisig wallet.</div>
          ) : (
            ((txs ?? []) as any[]).map((t: any) => (
              <div key={t.id} className={`p-4 rounded-lg border ${card(isDark)}`}>
                <div className="font-mono text-sm">→ {t.to_address} · {t.value}</div>
                <div className={`text-xs ${muted(isDark)}`}>status {t.status} · nonce {t.nonce}</div>
              </div>
            ))
          )}
        </div>
      )}
    </PageShell>
  );
};

// ---------------- Chains (UserWallet governance) ----------------


export const ChainsPage = ({ isDark, masterId }: { isDark: boolean; masterId?: string }) => {
  const { data, loading, error, reload } = useFetch<unknown[]>(async () => {
    if (!masterId) return [];
    return masterWalletAPI.listUserEVMChains(masterId);
  }, [masterId]);
  const [chainId, setChainId] = useState('');
  const [name, setName] = useState('');
  const [symbol, setSymbol] = useState('');
  const [rpc, setRpc] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const add = async (e: FormEvent) => {
    e.preventDefault();
    if (!masterId || !chainId.trim() || !name.trim()) return;
    setSubmitting(true);
    try {
      await masterWalletAPI.addUserEVMChain(masterId, {
        chain_id: Number(chainId),
        name: name.trim(),
        symbol: symbol.trim(),
        rpc_endpoint: rpc.trim(),
      } as any);
      setChainId(''); setName(''); setSymbol(''); setRpc('');
      void reload();
    } catch (e2) {
      setErr(e2 instanceof ApiError ? e2.message : String(e2));
    } finally {
      setSubmitting(false);
    }
  };
  const remove = async (cid: number) => {
    if (!masterId) return;
    try {
      await masterWalletAPI.removeUserEVMChain(masterId, cid);
      void reload();
    } catch (e2) {
      setErr(e2 instanceof ApiError ? e2.message : String(e2));
    }
  };

  return (
    <PageShell isDark={isDark} title="UserWallet EVM Chains">
      <Banner isDark={isDark} loading={loading} error={error || err} />
      <form onSubmit={add} className={`p-4 rounded-lg border space-y-3 ${card(isDark)}`}>
        <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
          <input className={inputCls(isDark)} type="number" placeholder="Chain ID" value={chainId} onChange={(e) => setChainId(e.target.value)} required />
          <input className={inputCls(isDark)} placeholder="Name" value={name} onChange={(e) => setName(e.target.value)} required />
          <input className={inputCls(isDark)} placeholder="Symbol" value={symbol} onChange={(e) => setSymbol(e.target.value)} />
          <input className={inputCls(isDark)} placeholder="RPC endpoint" value={rpc} onChange={(e) => setRpc(e.target.value)} />
        </div>
        <button className={btn(isDark)} disabled={submitting || !masterId}>{submitting ? 'Adding…' : 'Add Chain'}</button>
      </form>
      <div className="space-y-2">
        {(data ?? []).length === 0 ? (
          <div className={`p-4 rounded-lg border ${card(isDark)} ${muted(isDark)}`}>No user EVM chains configured.</div>
        ) : (
          ((data ?? []) as any[]).map((c) => (
            <div key={c.chain_id} className={`p-4 rounded-lg border flex items-center justify-between ${card(isDark)}`}>
              <div>
                <div className="font-semibold">{c.name} <span className={muted(isDark)}>(#{c.chain_id})</span></div>
                <div className={`text-sm ${muted(isDark)}`}>{c.symbol} · {c.rpc_endpoint ?? '—'}</div>
              </div>
              <button className={btn(isDark, 'danger')} onClick={() => remove(c.chain_id)}>Remove</button>
            </div>
          ))
        )}
      </div>
    </PageShell>
  );
};

// ---------------- Tokens (UserWallet governance) ----------------


export const TokensPage = ({ isDark, masterId }: { isDark: boolean; masterId?: string }) => {
  const { data, loading, error, reload } = useFetch<unknown[]>(async () => {
    if (!masterId) return [];
    return masterWalletAPI.listUserTokens(masterId);
  }, [masterId]);
  const [addr, setAddr] = useState('');
  const [symbol, setSymbol] = useState('');
  const [decimals, setDecimals] = useState('18');
  const [chainId, setChainId] = useState('1');
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const add = async (e: FormEvent) => {
    e.preventDefault();
    if (!masterId || !addr.trim()) return;
    setSubmitting(true);
    try {
      await masterWalletAPI.addUserToken(masterId, {
        contract_address: addr.trim(),
        symbol: symbol.trim(),
        decimals: Number(decimals) || 18,
        chain_id: Number(chainId) || 1,
      } as any);
      setAddr(''); setSymbol('');
      void reload();
    } catch (e2) {
      setErr(e2 instanceof ApiError ? e2.message : String(e2));
    } finally {
      setSubmitting(false);
    }
  };
  const remove = async (tid: string) => {
    if (!masterId) return;
    try {
      await masterWalletAPI.removeUserToken(masterId, tid);
      void reload();
    } catch (e2) {
      setErr(e2 instanceof ApiError ? e2.message : String(e2));
    }
  };

  return (
    <PageShell isDark={isDark} title="UserWallet Tokens">
      <Banner isDark={isDark} loading={loading} error={error || err} />
      <form onSubmit={add} className={`p-4 rounded-lg border space-y-3 ${card(isDark)}`}>
        <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
          <input className={inputCls(isDark)} placeholder="Contract address 0x…" value={addr} onChange={(e) => setAddr(e.target.value)} required />
          <input className={inputCls(isDark)} placeholder="Symbol" value={symbol} onChange={(e) => setSymbol(e.target.value)} />
          <input className={inputCls(isDark)} type="number" placeholder="Decimals" value={decimals} onChange={(e) => setDecimals(e.target.value)} />
          <input className={inputCls(isDark)} type="number" placeholder="Chain ID" value={chainId} onChange={(e) => setChainId(e.target.value)} />
        </div>
        <button className={btn(isDark)} disabled={submitting || !masterId}>{submitting ? 'Adding…' : 'Add Token'}</button>
      </form>
      <div className="space-y-2">
        {(data ?? []).length === 0 ? (
          <div className={`p-4 rounded-lg border ${card(isDark)} ${muted(isDark)}`}>No user tokens configured.</div>
        ) : (
          ((data ?? []) as any[]).map((t) => (
            <div key={t.id} className={`p-4 rounded-lg border flex items-center justify-between ${card(isDark)}`}>
              <div>
                <div className="font-semibold">{t.symbol ?? 'Token'}</div>
                <div className={`text-sm font-mono ${muted(isDark)}`}>{t.contract_address}</div>
              </div>
              <button className={btn(isDark, 'danger')} onClick={() => remove(t.id)}>Remove</button>
            </div>
          ))
        )}
      </div>
    </PageShell>
  );
};

// ---------------- Feature Flags ----------------


export const FeatureFlagsPage = ({ isDark, masterId }: { isDark: boolean; masterId?: string }) => {
  const { data, loading, error, reload } = useFetch<unknown[]>(async () => {
    if (!masterId) return [];
    return masterWalletAPI.listFeatureFlags(masterId);
  }, [masterId]);
  const [key, setKey] = useState('');
  const [desc, setDesc] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const add = async (e: FormEvent) => {
    e.preventDefault();
    if (!masterId || !key.trim()) return;
    setSubmitting(true);
    try {
      await masterWalletAPI.addFeatureFlag(masterId, { flag_key: key.trim(), enabled: true, description: desc.trim() } as any);
      setKey(''); setDesc('');
      void reload();
    } catch (e2) {
      setErr(e2 instanceof ApiError ? e2.message : String(e2));
    } finally {
      setSubmitting(false);
    }
  };
  const remove = async (fid: string | number) => {
    if (!masterId) return;
    try {
      await masterWalletAPI.removeFeatureFlag(masterId, fid);
      void reload();
    } catch (e2) {
      setErr(e2 instanceof ApiError ? e2.message : String(e2));
    }
  };

  return (
    <PageShell isDark={isDark} title="Feature Flags">
      <Banner isDark={isDark} loading={loading} error={error || err} />
      <form onSubmit={add} className={`p-4 rounded-lg border space-y-3 ${card(isDark)}`}>
        <input className={inputCls(isDark)} placeholder="flag_key (e.g. user_wallet.swap)" value={key} onChange={(e) => setKey(e.target.value)} required />
        <input className={inputCls(isDark)} placeholder="description" value={desc} onChange={(e) => setDesc(e.target.value)} />
        <button className={btn(isDark)} disabled={submitting || !masterId}>{submitting ? 'Adding…' : 'Add Flag'}</button>
      </form>
      <div className="space-y-2">
        {(data ?? []).length === 0 ? (
          <div className={`p-4 rounded-lg border ${card(isDark)} ${muted(isDark)}`}>No feature flags.</div>
        ) : (
          ((data ?? []) as any[]).map((f) => (
            <div key={f.id} className={`p-4 rounded-lg border flex items-center justify-between ${card(isDark)}`}>
              <div>
                <div className="font-semibold">{f.flag_key ?? f.id}</div>
                <div className={`text-sm ${muted(isDark)}`}>{f.description ?? ''} · {f.enabled ? 'enabled' : 'disabled'}</div>
              </div>
              <button className={btn(isDark, 'danger')} onClick={() => remove(f.id)}>Remove</button>
            </div>
          ))
        )}
      </div>
    </PageShell>
  );
};

// ---------------- Passkeys ----------------

import PasskeyService from './services/PasskeyService';

export const PasskeysPage = ({ isDark, masterId }: { isDark: boolean; masterId?: string }) => {
  const { data, loading, error, reload } = useFetch(async () => {
    if (!masterId) return [];
    return PasskeyService.listRegistered(masterId);
  }, [masterId]);
  const [label, setLabel] = useState('');
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<string | null>(null);
  const [err, setErr] = useState<string | null>(null);

  const register = async (e: FormEvent) => {
    e.preventDefault();
    if (!masterId) return;
    setBusy(true); setResult(null); setErr(null);
    try {
      const challenge = crypto.getRandomValues(new Uint8Array(32));
      const challengeB64 = btoa(String.fromCharCode(...challenge)).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
      const res = await PasskeyService.register(
        masterId,
        { name: 'TigerWallet MasterWallet' },
        { id: masterId, name: label || 'master-owner', displayName: label || 'Master Owner' },
        challengeB64,
        label.trim() || undefined,
      );
      if (res.success) setResult(`Registered passkey ${res.passkeyId ?? res.credential?.id}`);
      else setErr(res.error ?? 'Registration failed');
      void reload();
    } catch (e2) {
      setErr(e2 instanceof Error ? e2.message : String(e2));
    } finally {
      setBusy(false);
    }
  };

  const remove = async (credId: string) => {
    if (!masterId) return;
    try {
      await PasskeyService.remove(masterId, credId);
      void reload();
    } catch (e2) {
      setErr(e2 instanceof Error ? e2.message : String(e2));
    }
  };

  return (
    <PageShell isDark={isDark} title="Passkeys (WebAuthn)">
      <Banner isDark={isDark} loading={loading} error={error || err} />
      {result && <div className="mb-4 p-3 rounded-lg bg-green-100 text-green-800">{result}</div>}
      <form onSubmit={register} className={`p-4 rounded-lg border space-y-3 ${card(isDark)}`}>
        <input className={inputCls(isDark)} placeholder="Label (e.g. iPhone, YubiKey)" value={label} onChange={(e) => setLabel(e.target.value)} />
        <button className={btn(isDark)} disabled={busy || !masterId || !PasskeyService.isSupported()}>
          {busy ? 'Ceremony in progress…' : !PasskeyService.isSupported() ? 'WebAuthn not supported' : 'Register Passkey'}
        </button>
      </form>
      <div className="space-y-2">
        {(data ?? []).length === 0 ? (
          <div className={`p-4 rounded-lg border ${card(isDark)} ${muted(isDark)}`}>No passkeys registered.</div>
        ) : (
          (data ?? []).map((p: any) => (
            <div key={p.id} className={`p-4 rounded-lg border flex items-center justify-between ${card(isDark)}`}>
              <div>
                <div className="font-semibold">{p.label ?? 'Passkey'}</div>
                <div className={`text-xs font-mono ${muted(isDark)}`}>{p.credential_id}</div>
                <div className={`text-xs ${muted(isDark)}`}>sign count {p.sign_count}</div>
              </div>
              <button className={btn(isDark, 'danger')} onClick={() => remove(p.credential_id)}>Revoke</button>
            </div>
          ))
        )}
      </div>
    </PageShell>
  );
};
