/**
 * TigerWallet Admin - Trading Control-Plane Page
 *
 * Owner policy surface (RBAC admin tier): full lifecycle (create / add /
 * stop / resume / remove) over the builtin trading engines — trading
 * contracts, liquidity pools, trading pairs, margin markets, options, copy
 * trading — plus whole-vertical halt/resume. Status flips publish to the
 * shared Redis control namespace the wallet engines enforce on. Mirrors
 * /api/v1/trading/* on admin/go (:9093). All builtin TigerWallet; no
 * external broker or exchange.
 */

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../hooks/useTheme';
import { tradingControlAPI } from '../services/api';

const VERTICALS = ['spot', 'perpetual', 'futures', 'margin', 'options', 'copy', 'liquidity'];

type Tab = 'contracts' | 'pools' | 'pairs' | 'margin' | 'verticals' | 'audit';

export const TradingControlPage: React.FC = () => {
  const { isDark } = useTheme();
  const [tab, setTab] = useState<Tab>('contracts');
  const [overview, setOverview] = useState<any>(null);
  const [rows, setRows] = useState<any[]>([]);
  const [audit, setAudit] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [contractForm, setContractForm] = useState({ kind: 'perpetual', symbol: '', base_asset: '', quote_asset: 'USDT', chain_id: '', max_leverage: '10' });
  const [poolForm, setPoolForm] = useState({ chain_id: '1', dex: '', pool_address: '', token0: '', token1: '', fee_bps: '30' });
  const [marginForm, setMarginForm] = useState({ symbol: '', base_asset: '', quote_asset: 'USDT', max_leverage: '3', borrow_cap: '0' });

  const colors = {
    text: isDark ? '#f9fafb' : '#111827',
    textSecondary: isDark ? '#9ca3af' : '#6b7280',
    bgCard: isDark ? '#1e293b' : '#ffffff',
    border: isDark ? '#374151' : '#e5e7eb',
  };

  const load = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      if (tab === 'contracts') {
        const res: any = await tradingControlAPI.contracts.list();
        setRows(res.contracts || []);
      } else if (tab === 'pools') {
        const res: any = await tradingControlAPI.pools.list();
        setRows(res.pools || []);
      } else if (tab === 'pairs') {
        const res: any = await tradingControlAPI.pairs.list();
        setRows(res.pairs || res.trading_pairs || res.data || []);
      } else if (tab === 'margin') {
        const res: any = await tradingControlAPI.marginMarkets.list();
        setRows(res.margin_markets || []);
      } else if (tab === 'audit') {
        const res: any = await tradingControlAPI.audit();
        setAudit(res.audit || []);
      }
      try {
        const ov: any = await tradingControlAPI.overview();
        setOverview(ov);
      } catch {
        /* overview is best-effort */
      }
    } catch (err: any) {
      setError(err?.message || 'Failed to load');
    } finally {
      setLoading(false);
    }
  }, [tab]);

  useEffect(() => { load(); }, [load]);

  const run = async (fn: () => Promise<any>) => {
    setActionLoading(true);
    try {
      await fn();
      await load();
    } catch (err: any) {
      alert(err?.message || 'Action failed');
    } finally {
      setActionLoading(false);
    }
  };

  const badge = (s: string) => (
    <span className={`badge ${s === 'active' ? 'badge-success' : s === 'removed' ? 'badge-error' : 'badge-warning'}`}>{s}</span>
  );

  const lifecycle = (r: any, api: { stop: (id: string) => Promise<any>; resume: (id: string) => Promise<any>; remove: (id: string) => Promise<any> }) => (
    <div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
      <button className="btn btn-secondary" disabled={actionLoading || r.status === 'stopped'} onClick={() => run(() => api.stop(r.id))}>Stop</button>
      <button className="btn btn-success" disabled={actionLoading || r.status === 'active'} onClick={() => run(() => api.resume(r.id))}>Resume</button>
      <button className="btn btn-danger" disabled={actionLoading} onClick={() => { if (confirm('Remove permanently?')) run(() => api.remove(r.id)); }}>Remove</button>
    </div>
  );

  const halts = (overview && overview.vertical_halts) || {};

  const createContract = async (e: React.FormEvent) => {
    e.preventDefault();
    await run(() => tradingControlAPI.contracts.create({
      kind: contractForm.kind,
      symbol: contractForm.symbol,
      base_asset: contractForm.base_asset,
      quote_asset: contractForm.quote_asset,
      chain_id: contractForm.chain_id ? Number(contractForm.chain_id) : 0,
      max_leverage: Number(contractForm.max_leverage) || 1,
    }));
    setShowForm(false);
    setContractForm({ kind: 'perpetual', symbol: '', base_asset: '', quote_asset: 'USDT', chain_id: '', max_leverage: '10' });
  };

  const createPool = async (e: React.FormEvent) => {
    e.preventDefault();
    await run(() => tradingControlAPI.pools.create({
      chain_id: Number(poolForm.chain_id),
      dex: poolForm.dex,
      pool_address: poolForm.pool_address || undefined,
      token0: poolForm.token0,
      token1: poolForm.token1,
      fee_bps: Number(poolForm.fee_bps) || 30,
    }));
    setShowForm(false);
    setPoolForm({ chain_id: '1', dex: '', pool_address: '', token0: '', token1: '', fee_bps: '30' });
  };

  const createMargin = async (e: React.FormEvent) => {
    e.preventDefault();
    await run(() => tradingControlAPI.marginMarkets.create({
      symbol: marginForm.symbol,
      base_asset: marginForm.base_asset,
      quote_asset: marginForm.quote_asset,
      max_leverage: Number(marginForm.max_leverage) || 3,
      borrow_cap: marginForm.borrow_cap,
    }));
    setShowForm(false);
    setMarginForm({ symbol: '', base_asset: '', quote_asset: 'USDT', max_leverage: '3', borrow_cap: '0' });
  };

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-2" style={{ color: colors.text }}>Trading Control-Plane</h1>
      <p className="mb-6" style={{ color: colors.textSecondary }}>
        Builtin TigerWallet trading governance — contracts, liquidity pools, pairs, margin markets, options, copy trading.
        Decisions publish to the shared control plane enforced by every wallet engine.
      </p>

      {overview && (
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4 mb-6">
          <div className="stat-card"><div className="stat-value">{overview.contracts_active ?? 0}</div><div className="stat-label">Contracts</div></div>
          <div className="stat-card"><div className="stat-value">{overview.pools_active ?? 0}</div><div className="stat-label">Pools</div></div>
          <div className="stat-card"><div className="stat-value">{overview.pairs_active ?? 0}</div><div className="stat-label">Pairs</div></div>
          <div className="stat-card"><div className="stat-value">{overview.margin_markets_active ?? 0}</div><div className="stat-label">Margin Mkts</div></div>
          <div className="stat-card"><div className="stat-value">{overview.options_active ?? 0}</div><div className="stat-label">Options</div></div>
          <div className="stat-card"><div className="stat-value">{overview.copy_configs_active ?? 0}</div><div className="stat-label">Copy Configs</div></div>
        </div>
      )}

      <div className="flex gap-2 mb-4" style={{ flexWrap: 'wrap' }}>
        {(['contracts', 'pools', 'pairs', 'margin', 'verticals', 'audit'] as Tab[]).map((t) => (
          <button key={t} className={`btn ${tab === t ? 'btn-primary' : 'btn-secondary'}`} onClick={() => { setTab(t); setShowForm(false); }}>
            {t === 'margin' ? 'Margin Markets' : t.charAt(0).toUpperCase() + t.slice(1)}
          </button>
        ))}
      </div>

      {tab === 'verticals' && (
        <div className="card"><div className="card-body">
          <h3 className="mb-4" style={{ color: colors.text }}>Vertical Halt / Resume</h3>
          <table className="table"><thead><tr><th>Vertical</th><th>State</th><th>Actions</th></tr></thead>
            <tbody>
              {VERTICALS.map((v) => (
                <tr key={v}>
                  <td style={{ color: colors.text }}>{v}</td>
                  <td>{halts[v] ? <span className="badge badge-error">halted</span> : <span className="badge badge-success">running</span>}</td>
                  <td><div className="flex gap-2">
                    <button className="btn btn-danger" disabled={actionLoading || !!halts[v]} onClick={() => run(() => tradingControlAPI.haltVertical(v))}>Halt</button>
                    <button className="btn btn-success" disabled={actionLoading || !halts[v]} onClick={() => run(() => tradingControlAPI.resumeVertical(v))}>Resume</button>
                  </div></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div></div>
      )}

      {tab === 'audit' && (
        <div className="card"><div className="card-body overflow-x-auto">
          <h3 className="mb-4" style={{ color: colors.text }}>Control-Plane Audit</h3>
          {loading ? <div className="flex items-center justify-center p-8"><div className="loader"></div></div> : audit.length === 0 ? (
            <p className="text-center py-8" style={{ color: colors.textSecondary }}>No control-plane actions recorded yet.</p>
          ) : (
            <table className="table"><thead><tr><th>Actor</th><th>Role</th><th>Action</th><th>Kind</th><th>Entity</th><th>When</th></tr></thead>
              <tbody>
                {audit.map((a, i) => (
                  <tr key={a.id || i}>
                    <td style={{ color: colors.textSecondary }}>{a.actor || '—'}</td>
                    <td style={{ color: colors.textSecondary }}>{a.actor_role || '—'}</td>
                    <td style={{ color: colors.text }}>{a.action}</td>
                    <td style={{ color: colors.textSecondary }}>{a.kind}</td>
                    <td style={{ color: colors.text }}>{a.entity}</td>
                    <td style={{ color: colors.textSecondary }}>{a.created_at ? new Date(a.created_at).toLocaleString() : ''}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div></div>
      )}

      {(tab === 'contracts' || tab === 'pools' || tab === 'pairs' || tab === 'margin') && (
        <>
          <div className="flex justify-between items-center mb-4">
            <div />
            {tab !== 'pairs' && (
              <button className="btn btn-primary" onClick={() => setShowForm((s) => !s)}>
                {showForm ? 'Cancel' : tab === 'contracts' ? 'New Contract' : tab === 'pools' ? 'New Pool' : 'New Margin Market'}
              </button>
            )}
          </div>

          {showForm && tab === 'contracts' && (
            <div className="card mb-6"><div className="card-body">
              <h3 className="mb-4" style={{ color: colors.text }}>Create Trading Contract</h3>
              <form onSubmit={createContract} className="flex flex-col gap-3">
                <div className="flex gap-3">
                  <div className="form-group flex-1"><label style={{ color: colors.textSecondary }}>Kind</label>
                    <select className="input w-full" value={contractForm.kind} onChange={(e) => setContractForm({ ...contractForm, kind: e.target.value })}>
                      <option value="perpetual">perpetual</option><option value="futures">futures</option><option value="options">options</option>
                    </select></div>
                  <div className="form-group flex-1"><label style={{ color: colors.textSecondary }}>Symbol</label><input className="input w-full" required value={contractForm.symbol} onChange={(e) => setContractForm({ ...contractForm, symbol: e.target.value })} placeholder="BTC-PERP" /></div>
                </div>
                <div className="flex gap-3">
                  <div className="form-group flex-1"><label style={{ color: colors.textSecondary }}>Base Asset</label><input className="input w-full" required value={contractForm.base_asset} onChange={(e) => setContractForm({ ...contractForm, base_asset: e.target.value })} placeholder="BTC" /></div>
                  <div className="form-group flex-1"><label style={{ color: colors.textSecondary }}>Quote Asset</label><input className="input w-full" required value={contractForm.quote_asset} onChange={(e) => setContractForm({ ...contractForm, quote_asset: e.target.value })} /></div>
                </div>
                <div className="flex gap-3">
                  <div className="form-group flex-1"><label style={{ color: colors.textSecondary }}>Chain ID</label><input className="input w-full" type="number" value={contractForm.chain_id} onChange={(e) => setContractForm({ ...contractForm, chain_id: e.target.value })} /></div>
                  <div className="form-group flex-1"><label style={{ color: colors.textSecondary }}>Max Leverage</label><input className="input w-full" type="number" value={contractForm.max_leverage} onChange={(e) => setContractForm({ ...contractForm, max_leverage: e.target.value })} /></div>
                </div>
                <button className="btn btn-primary" disabled={actionLoading} type="submit">Create</button>
              </form>
            </div></div>
          )}

          {showForm && tab === 'pools' && (
            <div className="card mb-6"><div className="card-body">
              <h3 className="mb-4" style={{ color: colors.text }}>Create Liquidity Pool</h3>
              <form onSubmit={createPool} className="flex flex-col gap-3">
                <div className="flex gap-3">
                  <div className="form-group flex-1"><label style={{ color: colors.textSecondary }}>Chain ID</label><input className="input w-full" type="number" required value={poolForm.chain_id} onChange={(e) => setPoolForm({ ...poolForm, chain_id: e.target.value })} /></div>
                  <div className="form-group flex-1"><label style={{ color: colors.textSecondary }}>DEX</label><input className="input w-full" required value={poolForm.dex} onChange={(e) => setPoolForm({ ...poolForm, dex: e.target.value })} placeholder="uniswap_v2" /></div>
                </div>
                <div className="flex gap-3">
                  <div className="form-group flex-1"><label style={{ color: colors.textSecondary }}>Token0</label><input className="input w-full" required value={poolForm.token0} onChange={(e) => setPoolForm({ ...poolForm, token0: e.target.value })} /></div>
                  <div className="form-group flex-1"><label style={{ color: colors.textSecondary }}>Token1</label><input className="input w-full" required value={poolForm.token1} onChange={(e) => setPoolForm({ ...poolForm, token1: e.target.value })} /></div>
                </div>
                <div className="flex gap-3">
                  <div className="form-group flex-1"><label style={{ color: colors.textSecondary }}>Pool Address (optional)</label><input className="input w-full" value={poolForm.pool_address} onChange={(e) => setPoolForm({ ...poolForm, pool_address: e.target.value })} /></div>
                  <div className="form-group flex-1"><label style={{ color: colors.textSecondary }}>Fee (bps)</label><input className="input w-full" type="number" value={poolForm.fee_bps} onChange={(e) => setPoolForm({ ...poolForm, fee_bps: e.target.value })} /></div>
                </div>
                <button className="btn btn-primary" disabled={actionLoading} type="submit">Create</button>
              </form>
            </div></div>
          )}

          {showForm && tab === 'margin' && (
            <div className="card mb-6"><div className="card-body">
              <h3 className="mb-4" style={{ color: colors.text }}>Create Margin Market</h3>
              <form onSubmit={createMargin} className="flex flex-col gap-3">
                <div className="flex gap-3">
                  <div className="form-group flex-1"><label style={{ color: colors.textSecondary }}>Symbol</label><input className="input w-full" required value={marginForm.symbol} onChange={(e) => setMarginForm({ ...marginForm, symbol: e.target.value })} placeholder="BTC/USDT" /></div>
                  <div className="form-group flex-1"><label style={{ color: colors.textSecondary }}>Max Leverage</label><input className="input w-full" type="number" value={marginForm.max_leverage} onChange={(e) => setMarginForm({ ...marginForm, max_leverage: e.target.value })} /></div>
                </div>
                <div className="flex gap-3">
                  <div className="form-group flex-1"><label style={{ color: colors.textSecondary }}>Base Asset</label><input className="input w-full" required value={marginForm.base_asset} onChange={(e) => setMarginForm({ ...marginForm, base_asset: e.target.value })} /></div>
                  <div className="form-group flex-1"><label style={{ color: colors.textSecondary }}>Quote Asset</label><input className="input w-full" required value={marginForm.quote_asset} onChange={(e) => setMarginForm({ ...marginForm, quote_asset: e.target.value })} /></div>
                </div>
                <div className="form-group"><label style={{ color: colors.textSecondary }}>Borrow Cap</label><input className="input w-full" value={marginForm.borrow_cap} onChange={(e) => setMarginForm({ ...marginForm, borrow_cap: e.target.value })} /></div>
                <button className="btn btn-primary" disabled={actionLoading} type="submit">Create</button>
              </form>
            </div></div>
          )}

          {error ? (
            <div className="alert alert-error mb-4"><p className="text-error">{error}</p><button className="btn btn-secondary mt-2" onClick={load}>Retry</button></div>
          ) : loading ? (
            <div className="flex items-center justify-center p-8"><div className="loader"></div></div>
          ) : rows.length === 0 ? (
            <div className="card"><div className="card-body text-center py-8"><p style={{ color: colors.textSecondary }}>No entries yet.</p></div></div>
          ) : (
            <div className="card"><div className="card-body overflow-x-auto">
              {tab === 'contracts' && (
                <table className="table"><thead><tr><th>Kind</th><th>Symbol</th><th>Assets</th><th>Max Lev</th><th>Status</th><th>Actions</th></tr></thead>
                  <tbody>{rows.map((r) => (
                    <tr key={r.id}>
                      <td style={{ color: colors.textSecondary }}>{r.kind}</td>
                      <td style={{ color: colors.text }}>{r.symbol}</td>
                      <td style={{ color: colors.textSecondary }}>{r.base_asset}/{r.quote_asset}</td>
                      <td style={{ color: colors.textSecondary }}>{r.max_leverage}x</td>
                      <td>{badge(r.status)}</td>
                      <td>{lifecycle(r, tradingControlAPI.contracts)}</td>
                    </tr>))}
                  </tbody></table>
              )}
              {tab === 'pools' && (
                <table className="table"><thead><tr><th>Chain</th><th>DEX</th><th>Tokens</th><th>Fee</th><th>Status</th><th>Actions</th></tr></thead>
                  <tbody>{rows.map((r) => (
                    <tr key={r.id}>
                      <td style={{ color: colors.textSecondary }}>{r.chain_id}</td>
                      <td style={{ color: colors.text }}>{r.dex}</td>
                      <td style={{ color: colors.textSecondary }}>{r.token0}/{r.token1}</td>
                      <td style={{ color: colors.textSecondary }}>{r.fee_bps} bps</td>
                      <td>{badge(r.status)}</td>
                      <td>{lifecycle(r, tradingControlAPI.pools)}</td>
                    </tr>))}
                  </tbody></table>
              )}
              {tab === 'pairs' && (
                <table className="table"><thead><tr><th>Pair</th><th>Base</th><th>Quote</th><th>Status</th><th>Actions</th></tr></thead>
                  <tbody>{rows.map((r) => (
                    <tr key={r.id}>
                      <td style={{ color: colors.text }}>{r.pair_name || r.symbol}</td>
                      <td style={{ color: colors.textSecondary }}>{r.base_token || r.base_asset}</td>
                      <td style={{ color: colors.textSecondary }}>{r.quote_token || r.quote_asset}</td>
                      <td>{badge(r.status)}</td>
                      <td>{lifecycle(r, tradingControlAPI.pairs)}</td>
                    </tr>))}
                  </tbody></table>
              )}
              {tab === 'margin' && (
                <table className="table"><thead><tr><th>Symbol</th><th>Assets</th><th>Max Lev</th><th>Borrow Cap</th><th>Status</th><th>Actions</th></tr></thead>
                  <tbody>{rows.map((r) => (
                    <tr key={r.id}>
                      <td style={{ color: colors.text }}>{r.symbol}</td>
                      <td style={{ color: colors.textSecondary }}>{r.base_asset}/{r.quote_asset}</td>
                      <td style={{ color: colors.textSecondary }}>{r.max_leverage}x</td>
                      <td style={{ color: colors.textSecondary }}>{r.borrow_cap}</td>
                      <td>{badge(r.status)}</td>
                      <td>{lifecycle(r, tradingControlAPI.marginMarkets)}</td>
                    </tr>))}
                  </tbody></table>
              )}
            </div></div>
          )}
        </>
      )}
    </div>
  );
};

export default TradingControlPage;
