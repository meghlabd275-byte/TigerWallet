/**
 * Trading Control — white-label client's tenant-scoped trading control-plane.
 *
 * Owner policy surface: create / add / stop / resume / remove trading
 * contracts, liquidity pools, trading pairs, and margin markets inside this
 * tenancy, plus tenant-scoped vertical halt/resume. Backend:
 * /api/v1/admin/trading/* (tenant-isolated, scope-gated).
 */

import React, { useEffect, useState, useCallback } from 'react';
import { whiteLabelAdminApi } from '../services/api';
import { useTheme } from '../context/ThemeContext';

const VERTICALS = ['spot', 'perpetual', 'futures', 'margin', 'options', 'copy', 'liquidity'];

type Tab = 'contracts' | 'pools' | 'pairs' | 'margin' | 'verticals' | 'audit';

export default function TradingControl() {
  const { isDark } = useTheme();
  const [tab, setTab] = useState<Tab>('contracts');
  const [overview, setOverview] = useState<any>(null);
  const [rows, setRows] = useState<any[]>([]);
  const [audit, setAudit] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [contractForm, setContractForm] = useState({ kind: 'perpetual', symbol: '', base_asset: '', quote_asset: 'USDT', max_leverage: '10' });
  const [poolForm, setPoolForm] = useState({ chain_id: '1', dex: '', token0: '', token1: '', fee_bps: '30' });
  const [pairForm, setPairForm] = useState({ symbol: '', base_asset: '', quote_asset: 'USDT', market: 'spot' });
  const [marginForm, setMarginForm] = useState({ symbol: '', base_asset: '', quote_asset: 'USDT', max_leverage: '3' });

  const load = useCallback(async () => {
    setLoading(true); setError('');
    try {
      if (tab === 'contracts') { const d = await whiteLabelAdminApi.getTradingContracts(); setRows(d.contracts || []); }
      else if (tab === 'pools') { const d = await whiteLabelAdminApi.getTradingPools(); setRows(d.pools || []); }
      else if (tab === 'pairs') { const d = await whiteLabelAdminApi.getTradingPairsList(); setRows(d.pairs || []); }
      else if (tab === 'margin') { const d = await whiteLabelAdminApi.getTradingMarginMarkets(); setRows(d.margin_markets || []); }
      else if (tab === 'audit') { const d = await whiteLabelAdminApi.getTradingAudit(); setAudit(d.audit || []); }
      const ov = await whiteLabelAdminApi.getTradingOverview();
      setOverview(ov);
    } catch (e: any) { setError(e.message || 'Failed to load'); }
    finally { setLoading(false); }
  }, [tab]);

  useEffect(() => { load(); }, [load]);

  const run = async (fn: () => Promise<any>) => {
    try { await fn(); await load(); } catch (e: any) { setError(e.message || 'Action failed'); }
  };

  const cardBg = isDark ? 'bg-gray-800' : 'bg-white';
  const cardText = isDark ? 'text-white' : 'text-gray-900';
  const border = isDark ? 'border-gray-700' : 'border-gray-200';
  const muted = isDark ? 'text-gray-400' : 'text-gray-500';
  const thBg = isDark ? 'bg-gray-700' : 'bg-gray-50';
  const inputCls = isDark ? 'bg-gray-700 text-white border-gray-600' : 'bg-white text-gray-900 border-gray-300';

  const badge = (s: string) => (
    <span className={`px-2 py-1 rounded text-xs ${s === 'active' ? 'bg-green-100 text-green-700' : s === 'removed' ? 'bg-red-100 text-red-700' : 'bg-yellow-100 text-yellow-700'}`}>{s}</span>
  );

  const lifecycle = (r: any, api: { stop: (id: string) => Promise<any>; resume: (id: string) => Promise<any>; remove: (id: string) => Promise<any> }) => (
    <div className="flex gap-2 flex-wrap">
      <button onClick={() => run(() => api.stop(r.id))} disabled={r.status === 'stopped'} className="px-2 py-1 rounded bg-yellow-500 text-white text-xs disabled:opacity-40">Stop</button>
      <button onClick={() => run(() => api.resume(r.id))} disabled={r.status === 'active'} className="px-2 py-1 rounded bg-green-600 text-white text-xs disabled:opacity-40">Resume</button>
      <button onClick={() => { if (confirm('Remove permanently?')) run(() => api.remove(r.id)); }} className="px-2 py-1 rounded bg-red-600 text-white text-xs">Remove</button>
    </div>
  );

  const halts = (overview && overview.vertical_halts) || {};

  return (
    <div className="p-6">
      <h1 className={`text-2xl font-bold mb-2 ${cardText}`}>Trading Control</h1>
      <p className={`mb-4 ${muted}`}>Tenant-scoped builtin trading governance — contracts, pools, pairs, margin markets, vertical halts.</p>

      {overview && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-6">
          <div className={`${cardBg} p-3 rounded-lg border ${border}`}><div className={`text-xl font-bold ${cardText}`}>{overview.contracts_active ?? 0}</div><div className={`text-xs ${muted}`}>Contracts</div></div>
          <div className={`${cardBg} p-3 rounded-lg border ${border}`}><div className={`text-xl font-bold ${cardText}`}>{overview.pools_active ?? 0}</div><div className={`text-xs ${muted}`}>Pools</div></div>
          <div className={`${cardBg} p-3 rounded-lg border ${border}`}><div className={`text-xl font-bold ${cardText}`}>{overview.pairs_active ?? 0}</div><div className={`text-xs ${muted}`}>Pairs</div></div>
          <div className={`${cardBg} p-3 rounded-lg border ${border}`}><div className={`text-xl font-bold ${cardText}`}>{overview.margin_markets_active ?? 0}</div><div className={`text-xs ${muted}`}>Margin Markets</div></div>
        </div>
      )}

      <div className="flex gap-2 mb-4 flex-wrap">
        {(['contracts', 'pools', 'pairs', 'margin', 'verticals', 'audit'] as Tab[]).map((t) => (
          <button key={t} onClick={() => setTab(t)} className={`px-3 py-2 rounded text-sm ${tab === t ? 'bg-blue-600 text-white' : `${cardBg} ${cardText} border ${border}`}`}>
            {t === 'margin' ? 'Margin Markets' : t.charAt(0).toUpperCase() + t.slice(1)}
          </button>
        ))}
      </div>

      {error && <div className={`mb-3 p-2 rounded text-sm ${isDark ? 'bg-red-900/50 text-red-200' : 'bg-red-50 text-red-700'}`}>{error}</div>}
      {loading && <div className={muted}>Loading…</div>}

      {!loading && tab === 'verticals' && (
        <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
          <table className="w-full">
            <thead className={thBg}><tr>{['Vertical', 'State', 'Actions'].map((h) => <th key={h} className={`px-4 py-2 text-left text-xs font-semibold ${muted}`}>{h}</th>)}</tr></thead>
            <tbody className={`divide-y ${border}`}>
              {VERTICALS.map((v) => (
                <tr key={v}>
                  <td className={`px-4 py-3 ${cardText}`}>{v}</td>
                  <td className="px-4 py-3">{halts[v] ? <span className="px-2 py-1 rounded text-xs bg-red-100 text-red-700">halted</span> : <span className="px-2 py-1 rounded text-xs bg-green-100 text-green-700">running</span>}</td>
                  <td className="px-4 py-3"><div className="flex gap-2">
                    <button onClick={() => run(() => whiteLabelAdminApi.haltTradingVertical(v))} disabled={!!halts[v]} className="px-2 py-1 rounded bg-red-600 text-white text-xs disabled:opacity-40">Halt</button>
                    <button onClick={() => run(() => whiteLabelAdminApi.resumeTradingVertical(v))} disabled={!halts[v]} className="px-2 py-1 rounded bg-green-600 text-white text-xs disabled:opacity-40">Resume</button>
                  </div></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {!loading && tab === 'audit' && (
        <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
          <table className="w-full">
            <thead className={thBg}><tr>{['Actor', 'Action', 'Kind', 'Entity', 'When'].map((h) => <th key={h} className={`px-4 py-2 text-left text-xs font-semibold ${muted}`}>{h}</th>)}</tr></thead>
            <tbody className={`divide-y ${border}`}>
              {audit.length === 0 && <tr><td colSpan={5} className={`px-4 py-8 text-center ${muted}`}>No control-plane actions recorded yet.</td></tr>}
              {audit.map((a, i) => (
                <tr key={a.id || i}>
                  <td className={`px-4 py-3 ${muted}`}>{a.actor || '—'}</td>
                  <td className={`px-4 py-3 ${cardText}`}>{a.action}</td>
                  <td className={`px-4 py-3 ${muted}`}>{a.kind}</td>
                  <td className={`px-4 py-3 ${cardText}`}>{a.entity}</td>
                  <td className={`px-4 py-3 ${muted}`}>{a.created_at ? new Date(a.created_at).toLocaleString() : ''}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {!loading && tab === 'contracts' && (
        <>
          <div className={`${cardBg} p-4 rounded-lg shadow border ${border} mb-4`}>
            <h2 className={`text-sm font-semibold mb-3 ${cardText}`}>New Contract</h2>
            <div className="flex flex-wrap gap-2">
              <select value={contractForm.kind} onChange={(e) => setContractForm({ ...contractForm, kind: e.target.value })} className={`px-3 py-2 rounded border ${inputCls}`}>
                <option value="perpetual">perpetual</option><option value="futures">futures</option><option value="options">options</option>
              </select>
              <input value={contractForm.symbol} onChange={(e) => setContractForm({ ...contractForm, symbol: e.target.value })} placeholder="Symbol (BTC-PERP)" className={`px-3 py-2 rounded border ${inputCls}`} />
              <input value={contractForm.base_asset} onChange={(e) => setContractForm({ ...contractForm, base_asset: e.target.value })} placeholder="Base (BTC)" className={`px-3 py-2 rounded border ${inputCls} w-28`} />
              <input value={contractForm.quote_asset} onChange={(e) => setContractForm({ ...contractForm, quote_asset: e.target.value })} placeholder="Quote" className={`px-3 py-2 rounded border ${inputCls} w-24`} />
              <input value={contractForm.max_leverage} onChange={(e) => setContractForm({ ...contractForm, max_leverage: e.target.value })} type="number" min="1" placeholder="Max lev" className={`px-3 py-2 rounded border ${inputCls} w-24`} />
              <button onClick={() => run(() => whiteLabelAdminApi.createTradingContract({ ...contractForm, max_leverage: Number(contractForm.max_leverage) || 1 }))} className="px-4 py-2 rounded bg-blue-600 text-white">Create</button>
            </div>
          </div>
          <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
            <table className="w-full">
              <thead className={thBg}><tr>{['Kind', 'Symbol', 'Assets', 'Max Lev', 'Status', 'Actions'].map((h) => <th key={h} className={`px-4 py-2 text-left text-xs font-semibold ${muted}`}>{h}</th>)}</tr></thead>
              <tbody className={`divide-y ${border}`}>
                {rows.length === 0 && <tr><td colSpan={6} className={`px-4 py-8 text-center ${muted}`}>No contracts yet.</td></tr>}
                {rows.map((r) => (
                  <tr key={r.id}>
                    <td className={`px-4 py-3 ${muted}`}>{r.kind}</td>
                    <td className={`px-4 py-3 ${cardText}`}>{r.symbol}</td>
                    <td className={`px-4 py-3 ${muted}`}>{r.base_asset}/{r.quote_asset}</td>
                    <td className={`px-4 py-3 ${muted}`}>{r.max_leverage}x</td>
                    <td className="px-4 py-3">{badge(r.status)}</td>
                    <td className="px-4 py-3">{lifecycle(r, { stop: whiteLabelAdminApi.stopTradingContract, resume: whiteLabelAdminApi.resumeTradingContract, remove: whiteLabelAdminApi.deleteTradingContract })}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}

      {!loading && tab === 'pools' && (
        <>
          <div className={`${cardBg} p-4 rounded-lg shadow border ${border} mb-4`}>
            <h2 className={`text-sm font-semibold mb-3 ${cardText}`}>New Liquidity Pool</h2>
            <div className="flex flex-wrap gap-2">
              <input value={poolForm.chain_id} onChange={(e) => setPoolForm({ ...poolForm, chain_id: e.target.value })} type="number" placeholder="Chain ID" className={`px-3 py-2 rounded border ${inputCls} w-24`} />
              <input value={poolForm.dex} onChange={(e) => setPoolForm({ ...poolForm, dex: e.target.value })} placeholder="DEX" className={`px-3 py-2 rounded border ${inputCls}`} />
              <input value={poolForm.token0} onChange={(e) => setPoolForm({ ...poolForm, token0: e.target.value })} placeholder="Token0" className={`px-3 py-2 rounded border ${inputCls}`} />
              <input value={poolForm.token1} onChange={(e) => setPoolForm({ ...poolForm, token1: e.target.value })} placeholder="Token1" className={`px-3 py-2 rounded border ${inputCls}`} />
              <input value={poolForm.fee_bps} onChange={(e) => setPoolForm({ ...poolForm, fee_bps: e.target.value })} type="number" placeholder="Fee bps" className={`px-3 py-2 rounded border ${inputCls} w-24`} />
              <button onClick={() => run(() => whiteLabelAdminApi.createTradingPool({ chain_id: Number(poolForm.chain_id), dex: poolForm.dex, token0: poolForm.token0, token1: poolForm.token1, fee_bps: Number(poolForm.fee_bps) || 30 }))} className="px-4 py-2 rounded bg-blue-600 text-white">Create</button>
            </div>
          </div>
          <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
            <table className="w-full">
              <thead className={thBg}><tr>{['Chain', 'DEX', 'Tokens', 'Fee', 'Status', 'Actions'].map((h) => <th key={h} className={`px-4 py-2 text-left text-xs font-semibold ${muted}`}>{h}</th>)}</tr></thead>
              <tbody className={`divide-y ${border}`}>
                {rows.length === 0 && <tr><td colSpan={6} className={`px-4 py-8 text-center ${muted}`}>No pools yet.</td></tr>}
                {rows.map((r) => (
                  <tr key={r.id}>
                    <td className={`px-4 py-3 ${muted}`}>{r.chain_id}</td>
                    <td className={`px-4 py-3 ${cardText}`}>{r.dex}</td>
                    <td className={`px-4 py-3 ${muted}`}>{r.token0}/{r.token1}</td>
                    <td className={`px-4 py-3 ${muted}`}>{r.fee_bps} bps</td>
                    <td className="px-4 py-3">{badge(r.status)}</td>
                    <td className="px-4 py-3">{lifecycle(r, { stop: whiteLabelAdminApi.stopTradingPool, resume: whiteLabelAdminApi.resumeTradingPool, remove: whiteLabelAdminApi.deleteTradingPool })}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}

      {!loading && tab === 'pairs' && (
        <>
          <div className={`${cardBg} p-4 rounded-lg shadow border ${border} mb-4`}>
            <h2 className={`text-sm font-semibold mb-3 ${cardText}`}>New Trading Pair</h2>
            <div className="flex flex-wrap gap-2">
              <input value={pairForm.symbol} onChange={(e) => setPairForm({ ...pairForm, symbol: e.target.value })} placeholder="Symbol (BTC/USDT)" className={`px-3 py-2 rounded border ${inputCls}`} />
              <input value={pairForm.base_asset} onChange={(e) => setPairForm({ ...pairForm, base_asset: e.target.value })} placeholder="Base" className={`px-3 py-2 rounded border ${inputCls} w-24`} />
              <input value={pairForm.quote_asset} onChange={(e) => setPairForm({ ...pairForm, quote_asset: e.target.value })} placeholder="Quote" className={`px-3 py-2 rounded border ${inputCls} w-24`} />
              <select value={pairForm.market} onChange={(e) => setPairForm({ ...pairForm, market: e.target.value })} className={`px-3 py-2 rounded border ${inputCls}`}>
                <option value="spot">spot</option><option value="perpetual">perpetual</option><option value="margin">margin</option>
              </select>
              <button onClick={() => run(() => whiteLabelAdminApi.createTradingPair(pairForm))} className="px-4 py-2 rounded bg-blue-600 text-white">Create</button>
            </div>
          </div>
          <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
            <table className="w-full">
              <thead className={thBg}><tr>{['Symbol', 'Assets', 'Market', 'Status', 'Actions'].map((h) => <th key={h} className={`px-4 py-2 text-left text-xs font-semibold ${muted}`}>{h}</th>)}</tr></thead>
              <tbody className={`divide-y ${border}`}>
                {rows.length === 0 && <tr><td colSpan={5} className={`px-4 py-8 text-center ${muted}`}>No pairs yet.</td></tr>}
                {rows.map((r) => (
                  <tr key={r.id}>
                    <td className={`px-4 py-3 ${cardText}`}>{r.symbol}</td>
                    <td className={`px-4 py-3 ${muted}`}>{r.base_asset}/{r.quote_asset}</td>
                    <td className={`px-4 py-3 ${muted}`}>{r.market}</td>
                    <td className="px-4 py-3">{badge(r.status)}</td>
                    <td className="px-4 py-3">{lifecycle(r, { stop: whiteLabelAdminApi.stopTradingPair, resume: whiteLabelAdminApi.resumeTradingPair, remove: whiteLabelAdminApi.deleteTradingPair })}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}

      {!loading && tab === 'margin' && (
        <>
          <div className={`${cardBg} p-4 rounded-lg shadow border ${border} mb-4`}>
            <h2 className={`text-sm font-semibold mb-3 ${cardText}`}>New Margin Market</h2>
            <div className="flex flex-wrap gap-2">
              <input value={marginForm.symbol} onChange={(e) => setMarginForm({ ...marginForm, symbol: e.target.value })} placeholder="Symbol (BTC/USDT)" className={`px-3 py-2 rounded border ${inputCls}`} />
              <input value={marginForm.base_asset} onChange={(e) => setMarginForm({ ...marginForm, base_asset: e.target.value })} placeholder="Base" className={`px-3 py-2 rounded border ${inputCls} w-24`} />
              <input value={marginForm.quote_asset} onChange={(e) => setMarginForm({ ...marginForm, quote_asset: e.target.value })} placeholder="Quote" className={`px-3 py-2 rounded border ${inputCls} w-24`} />
              <input value={marginForm.max_leverage} onChange={(e) => setMarginForm({ ...marginForm, max_leverage: e.target.value })} type="number" min="1" placeholder="Max lev" className={`px-3 py-2 rounded border ${inputCls} w-24`} />
              <button onClick={() => run(() => whiteLabelAdminApi.createTradingMarginMarket({ ...marginForm, max_leverage: Number(marginForm.max_leverage) || 3 }))} className="px-4 py-2 rounded bg-blue-600 text-white">Create</button>
            </div>
          </div>
          <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
            <table className="w-full">
              <thead className={thBg}><tr>{['Symbol', 'Assets', 'Max Lev', 'Status', 'Actions'].map((h) => <th key={h} className={`px-4 py-2 text-left text-xs font-semibold ${muted}`}>{h}</th>)}</tr></thead>
              <tbody className={`divide-y ${border}`}>
                {rows.length === 0 && <tr><td colSpan={5} className={`px-4 py-8 text-center ${muted}`}>No margin markets yet.</td></tr>}
                {rows.map((r) => (
                  <tr key={r.id}>
                    <td className={`px-4 py-3 ${cardText}`}>{r.symbol}</td>
                    <td className={`px-4 py-3 ${muted}`}>{r.base_asset}/{r.quote_asset}</td>
                    <td className={`px-4 py-3 ${muted}`}>{r.max_leverage}x</td>
                    <td className="px-4 py-3">{badge(r.status)}</td>
                    <td className="px-4 py-3">{lifecycle(r, { stop: whiteLabelAdminApi.stopTradingMarginMarket, resume: whiteLabelAdminApi.resumeTradingMarginMarket, remove: whiteLabelAdminApi.deleteTradingMarginMarket })}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </div>
  );
}
