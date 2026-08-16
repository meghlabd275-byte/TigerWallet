/**
 * Listings (listing_admin) — coin/token listing + trading pair management +
 * blockchains (listingManager / partner). Expansion of the Tokens page.
 * Backend: GET /api/v1/admin/tokens, /pairs, /blockchains (+ create/status).
 */

import React, { useEffect, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';
import { useTheme } from '../context/ThemeContext';

type Tab = 'tokens' | 'pairs' | 'blockchains';

export default function Listings() {
  const { isDark } = useTheme();
  const [tab, setTab] = useState<Tab>('tokens');
  const [tokens, setTokens] = useState<any[]>([]);
  const [pairs, setPairs] = useState<any[]>([]);
  const [blockchains, setBlockchains] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showPair, setShowPair] = useState(false);
  const [pairForm, setPairForm] = useState({ base_token_id: '', quote_token_id: '', pair_name: '', chain_id: 0 });

  useEffect(() => { loadAll(); }, []);

  const loadAll = async () => {
    setLoading(true); setError('');
    try {
      const [t, p, b] = await Promise.all([
        whiteLabelAdminApi.getTokens(),
        whiteLabelAdminApi.getPairs(),
        whiteLabelAdminApi.getBlockchains(),
      ]);
      setTokens(t.tokens || []);
      setPairs(p.pairs || []);
      setBlockchains(b.blockchains || []);
    } catch (e: any) { setError(e.message || 'Failed to load listings'); }
    finally { setLoading(false); }
  };

  const toggleToken = async (tok: any) => {
    try { await whiteLabelAdminApi.updateToken(tok.id, { is_active: !tok.is_active }); loadAll(); }
    catch (e: any) { setError(e.message || 'Failed to update token'); }
  };

  const verifyToken = async (tok: any) => {
    try { await whiteLabelAdminApi.updateToken(tok.id, { is_verified: true }); loadAll(); }
    catch (e: any) { setError(e.message || 'Failed to verify token'); }
  };

  const setPairStatus = async (id: string, status: string) => {
    try { await whiteLabelAdminApi.updatePairStatus(id, status); loadAll(); }
    catch (e: any) { setError(e.message || 'Failed to update pair'); }
  };

  const createPair = async () => {
    try { await whiteLabelAdminApi.createPair({ ...pairForm, chain_id: Number(pairForm.chain_id) || 0 }); setShowPair(false); setPairForm({ base_token_id: '', quote_token_id: '', pair_name: '', chain_id: 0 }); loadAll(); }
    catch (e: any) { setError(e.message || 'Failed to create pair'); }
  };

  const toggleChain = async (ch: any) => {
    try { await whiteLabelAdminApi.setBlockchainStatus(ch.id, !ch.is_active); loadAll(); }
    catch (e: any) { setError(e.message || 'Failed to update blockchain'); }
  };

  const cardBg = isDark ? 'bg-gray-800' : 'bg-white';
  const cardText = isDark ? 'text-white' : 'text-gray-900';
  const border = isDark ? 'border-gray-700' : 'border-gray-200';
  const muted = isDark ? 'text-gray-400' : 'text-gray-500';
  const thBg = isDark ? 'bg-gray-700' : 'bg-gray-50';
  const inputCls = `w-full px-3 py-2 rounded border ${border} ${cardBg} ${cardText}`;
  const short = (v: string) => (v ? `${v.substring(0, 8)}…` : '—');

  const pill = (on: boolean, label: string) => (
    <span className={`px-2 py-0.5 text-xs rounded ${on ? (isDark ? 'bg-green-900 text-green-200' : 'bg-green-100 text-green-800') : (isDark ? 'bg-gray-700 text-gray-300' : 'bg-gray-200 text-gray-600')}`}>{label}</span>
  );

  return (
    <div className="p-6">
      <div className="flex flex-wrap justify-between items-center mb-4">
        <div>
          <h1 className={`text-2xl font-bold ${cardText}`}>Listings</h1>
          <p className={muted}>Tokens · Trading pairs · Blockchains</p>
        </div>
        {tab === 'pairs' && <button onClick={() => setShowPair(true)} className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">+ Add Pair</button>}
      </div>

      <div className="flex gap-2 mb-4">
        {(['tokens', 'pairs', 'blockchains'] as Tab[]).map((t) => (
          <button key={t} onClick={() => setTab(t)} className={`px-3 py-1 rounded text-sm capitalize ${tab === t ? 'bg-blue-600 text-white' : (isDark ? 'bg-gray-700 text-gray-200' : 'bg-gray-200 text-gray-700')}`}>{t}</button>
        ))}
      </div>

      {error && <div className={`mb-3 p-2 rounded text-sm ${isDark ? 'bg-red-900/50 text-red-200' : 'bg-red-50 text-red-700'}`}>{error}</div>}
      {loading && <div className={muted}>Loading…</div>}

      {!loading && tab === 'tokens' && (
        <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
          <table className="w-full">
            <thead className={thBg}><tr>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Name</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Symbol</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Contract</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Status</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Actions</th>
            </tr></thead>
            <tbody className={`divide-y ${border}`}>
              {tokens.length === 0 && <tr><td colSpan={5} className={`px-6 py-8 text-center ${muted}`}>No tokens.</td></tr>}
              {tokens.map((t) => (
                <tr key={t.id}>
                  <td className={`px-6 py-4 ${cardText}`}>{t.name}</td>
                  <td className={`px-6 py-4 ${cardText}`}>{t.symbol}</td>
                  <td className={`px-6 py-4 font-mono text-xs ${muted}`}>{short(t.contract_address)}</td>
                  <td className="px-6 py-4 space-x-1">{pill(t.is_active, t.is_active ? 'Active' : 'Inactive')}{t.is_verified && pill(true, 'Verified')}</td>
                  <td className="px-6 py-4 space-x-2">
                    {!t.is_verified && <button onClick={() => verifyToken(t)} className="text-blue-600 hover:underline text-sm">Verify</button>}
                    <button onClick={() => toggleToken(t)} className="text-yellow-600 hover:underline text-sm">{t.is_active ? 'Disable' : 'Enable'}</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {!loading && tab === 'pairs' && (
        <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
          <table className="w-full">
            <thead className={thBg}><tr>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Pair</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Price</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>24h Vol</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Liquidity</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Status</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Actions</th>
            </tr></thead>
            <tbody className={`divide-y ${border}`}>
              {pairs.length === 0 && <tr><td colSpan={6} className={`px-6 py-8 text-center ${muted}`}>No trading pairs.</td></tr>}
              {pairs.map((p) => (
                <tr key={p.id}>
                  <td className={`px-6 py-4 ${cardText}`}>{p.pair_name}</td>
                  <td className={`px-6 py-4 ${cardText}`}>{p.price}</td>
                  <td className={`px-6 py-4 ${muted}`}>{p.volume_24h}</td>
                  <td className={`px-6 py-4 ${muted}`}>{p.liquidity}</td>
                  <td className="px-6 py-4"><span className={`px-2 py-0.5 text-xs rounded ${p.status === 'active' ? (isDark ? 'bg-green-900 text-green-200' : 'bg-green-100 text-green-800') : (isDark ? 'bg-gray-700 text-gray-300' : 'bg-gray-200 text-gray-600')}`}>{p.status}</span></td>
                  <td className="px-6 py-4 space-x-2">
                    {p.status !== 'active' && <button onClick={() => setPairStatus(p.id, 'active')} className="text-green-600 hover:underline text-sm">Activate</button>}
                    {p.status === 'active' && <button onClick={() => setPairStatus(p.id, 'suspended')} className="text-yellow-600 hover:underline text-sm">Suspend</button>}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {!loading && tab === 'blockchains' && (
        <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
          <table className="w-full">
            <thead className={thBg}><tr>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Name</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Chain ID</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>EVM</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Native</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Gas (gwei)</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Status</th>
              <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Actions</th>
            </tr></thead>
            <tbody className={`divide-y ${border}`}>
              {blockchains.length === 0 && <tr><td colSpan={7} className={`px-6 py-8 text-center ${muted}`}>No blockchains.</td></tr>}
              {blockchains.map((ch) => (
                <tr key={ch.id}>
                  <td className={`px-6 py-4 ${cardText}`}>{ch.name} <span className={muted}>({ch.symbol})</span></td>
                  <td className={`px-6 py-4 ${muted}`}>{ch.chain_id}</td>
                  <td className="px-6 py-4">{ch.is_evm ? '✓' : '—'}</td>
                  <td className={`px-6 py-4 ${cardText}`}>{ch.native_token || '—'}</td>
                  <td className={`px-6 py-4 ${muted}`}>{ch.avg_gas_price_gwei}</td>
                  <td className="px-6 py-4">{pill(ch.is_active, ch.is_active ? 'Active' : 'Inactive')}</td>
                  <td className="px-6 py-4"><button onClick={() => toggleChain(ch)} className="text-yellow-600 hover:underline text-sm">{ch.is_active ? 'Disable' : 'Enable'}</button></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {showPair && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className={`${cardBg} rounded-lg p-6 w-full max-w-md`}>
            <h2 className={`text-lg font-bold mb-4 ${cardText}`}>Add Trading Pair</h2>
            <div className="space-y-3">
              <input placeholder="Base token ID" value={pairForm.base_token_id} onChange={e => setPairForm({ ...pairForm, base_token_id: e.target.value })} className={inputCls} />
              <input placeholder="Quote token ID" value={pairForm.quote_token_id} onChange={e => setPairForm({ ...pairForm, quote_token_id: e.target.value })} className={inputCls} />
              <input placeholder="Pair name (e.g. BTC/USDT)" value={pairForm.pair_name} onChange={e => setPairForm({ ...pairForm, pair_name: e.target.value })} className={inputCls} />
              <input type="number" placeholder="Chain ID (optional)" value={pairForm.chain_id || ''} onChange={e => setPairForm({ ...pairForm, chain_id: Number(e.target.value) })} className={inputCls} />
            </div>
            <div className="flex justify-end gap-2 mt-4">
              <button onClick={() => setShowPair(false)} className={`px-4 py-2 rounded ${isDark ? 'bg-gray-700 text-white' : 'bg-gray-200 text-gray-800'}`}>Cancel</button>
              <button onClick={createPair} className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">Create</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
