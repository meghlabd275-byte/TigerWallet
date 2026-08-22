/**
 * Crypto Card (card_admin) — WL-branded crypto card management.
 * Backend: GET/POST /api/v1/admin/wl-cards, PUT /:id/status, POST /:id/block,
 * POST /:id/activate, PUT /:id/limit, GET /transactions, GET /stats.
 */

import React, { useEffect, useMemo, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';
import { useTheme } from '../context/ThemeContext';

export default function CryptoCard() {
  const { isDark } = useTheme();
  const [cards, setCards] = useState<any[]>([]);
  const [txs, setTxs] = useState<any[]>([]);
  const [stats, setStats] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [tab, setTab] = useState<'cards' | 'transactions'>('cards');
  const [userId, setUserId] = useState('');
  const [cardType, setCardType] = useState('virtual');
  const [limitId, setLimitId] = useState<string | null>(null);
  const [limit, setLimit] = useState('');

  useEffect(() => { load(); }, []);

  const load = async () => {
    setLoading(true); setError('');
    try {
      const [c, t, s] = await Promise.all([
        whiteLabelAdminApi.getWLCards(),
        whiteLabelAdminApi.getWLCardTransactions(),
        whiteLabelAdminApi.getWLCardStats(),
      ]);
      setCards(c.cards || c || []);
      setTxs(t.transactions || t || []);
      setStats(s);
    } catch (e: any) { setError(e.message || 'Failed to load card data'); }
    finally { setLoading(false); }
  };

  const cardVolume = useMemo(() => {
    let v = 0;
    txs.forEach((t) => { v += parseFloat(t.amount) || 0; });
    return v;
  }, [txs]);

  const issue = async () => {
    if (!userId) { setError('User ID required'); return; }
    try { await whiteLabelAdminApi.issueWLCard({ user_id: userId, card_type: cardType }); setUserId(''); load(); }
    catch (e: any) { setError(e.message || 'Failed to issue card'); }
  };

  const setStatus = async (id: string, status: string) => {
    try { await whiteLabelAdminApi.updateWLCardStatus(id, status); load(); }
    catch (e: any) { setError(e.message || 'Failed to update card status'); }
  };

  const block = async (id: string) => {
    if (!confirm('Block this card?')) return;
    try { await whiteLabelAdminApi.blockWLCard(id); load(); }
    catch (e: any) { setError(e.message || 'Failed to block card'); }
  };

  const activate = async (id: string) => {
    try { await whiteLabelAdminApi.activateWLCard(id); load(); }
    catch (e: any) { setError(e.message || 'Failed to activate card'); }
  };

  const setCardLimit = async () => {
    if (!limitId || !limit) { setError('Card and limit required'); return; }
    try { await whiteLabelAdminApi.setWLCardLimit(limitId, limit); setLimitId(null); setLimit(''); load(); }
    catch (e: any) { setError(e.message || 'Failed to set limit'); }
  };

  const cardBg = isDark ? 'bg-gray-800' : 'bg-white';
  const cardText = isDark ? 'text-white' : 'text-gray-900';
  const border = isDark ? 'border-gray-700' : 'border-gray-200';
  const muted = isDark ? 'text-gray-400' : 'text-gray-500';
  const thBg = isDark ? 'bg-gray-700' : 'bg-gray-50';
  const inputCls = isDark ? 'bg-gray-700 text-white border-gray-600' : 'bg-white text-gray-900 border-gray-300';

  const badge = (s: string) => {
    const b = 'px-2 py-0.5 text-xs rounded';
    if (s === 'active' || s === 'activated' || s === 'confirmed') return `${b} ${isDark ? 'bg-green-900 text-green-200' : 'bg-green-100 text-green-800'}`;
    if (s === 'pending') return `${b} ${isDark ? 'bg-yellow-900 text-yellow-200' : 'bg-yellow-100 text-yellow-800'}`;
    if (s === 'blocked') return `${b} ${isDark ? 'bg-red-900 text-red-200' : 'bg-red-100 text-red-800'}`;
    return `${b} ${isDark ? 'bg-gray-700 text-gray-300' : 'bg-gray-200 text-gray-700'}`;
  };

  const stat = (l: string, v: string) => (
    <div className={`${cardBg} p-4 rounded-lg shadow border ${border}`}><p className={`text-xs ${muted}`}>{l}</p><p className={`text-xl font-bold mt-1 ${cardText}`}>{v}</p></div>
  );

  const short = (v: string) => (v ? `${v.substring(0, 8)}…` : '—');

  return (
    <div className="p-6">
      <h1 className={`text-2xl font-bold mb-2 ${cardText}`}>Crypto Card</h1>
      <p className={`mb-4 ${muted}`}>WL-branded card issuance, status, limits and card transactions.</p>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-6">
        {stat('Total Cards', String(cards.length))}
        {stat('Active Cards', String(cards.filter((c) => c.status === 'active' || c.status === 'activated').length))}
        {stat('Card Tx', String(txs.length))}
        {stat('Card Volume', stats ? String(stats.total_volume ?? stats.volume ?? cardVolume.toFixed(2)) : cardVolume.toFixed(2))}
      </div>

      <div className={`${cardBg} p-4 rounded-lg shadow border ${border} mb-6`}>
        <h2 className={`text-sm font-semibold mb-3 ${cardText}`}>Issue Card</h2>
        <div className="flex flex-wrap gap-2">
          <input value={userId} onChange={(e) => setUserId(e.target.value)} placeholder="User ID" className={`px-3 py-2 rounded border ${inputCls}`} />
          <select value={cardType} onChange={(e) => setCardType(e.target.value)} className={`px-3 py-2 rounded border ${inputCls}`}>
            <option value="virtual">Virtual</option><option value="physical">Physical</option>
          </select>
          <button onClick={issue} className="px-4 py-2 rounded bg-blue-600 text-white">Issue</button>
        </div>
      </div>

      {limitId && (
        <div className={`${cardBg} p-4 rounded-lg shadow border ${border} mb-4`}>
          <h2 className={`text-sm font-semibold mb-2 ${cardText}`}>Set limit for card {limitId}</h2>
          <input value={limit} onChange={(e) => setLimit(e.target.value)} placeholder="New limit" className={`w-full px-3 py-2 rounded border ${inputCls} mb-2`} />
          <div className="flex gap-2">
            <button onClick={setCardLimit} className="px-4 py-2 rounded bg-blue-600 text-white">Set Limit</button>
            <button onClick={() => { setLimitId(null); setLimit(''); }} className={`px-4 py-2 rounded border ${inputCls}`}>Cancel</button>
          </div>
        </div>
      )}

      <div className="flex gap-2 mb-4">
        <button onClick={() => setTab('cards')} className={`px-3 py-1 rounded text-sm ${tab === 'cards' ? 'bg-blue-600 text-white' : (isDark ? 'bg-gray-700 text-gray-200' : 'bg-gray-200 text-gray-700')}`}>Cards</button>
        <button onClick={() => setTab('transactions')} className={`px-3 py-1 rounded text-sm ${tab === 'transactions' ? 'bg-blue-600 text-white' : (isDark ? 'bg-gray-700 text-gray-200' : 'bg-gray-200 text-gray-700')}`}>Card Transactions</button>
      </div>

      {error && <div className={`mb-3 p-2 rounded text-sm ${isDark ? 'bg-red-900/50 text-red-200' : 'bg-red-50 text-red-700'}`}>{error}</div>}
      {loading && <div className={muted}>Loading…</div>}

      {!loading && tab === 'cards' && (
        <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
          <table className="w-full">
            <thead className={thBg}><tr>
              {['Card ID', 'User', 'Type', 'Status', 'Limit', 'Actions'].map((h) => (
                <th key={h} className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>{h}</th>
              ))}
            </tr></thead>
            <tbody className={`divide-y ${border}`}>
              {cards.length === 0 && <tr><td colSpan={6} className={`px-6 py-8 text-center ${muted}`}>No cards.</td></tr>}
              {cards.map((c) => (
                <tr key={c.id}>
                  <td className={`px-6 py-4 font-mono text-xs ${cardText}`}>{short(c.id)}</td>
                  <td className={`px-6 py-4 font-mono text-xs ${muted}`}>{c.user_id ? short(c.user_id) : '—'}</td>
                  <td className={`px-6 py-4 ${cardText}`}>{c.card_type || '—'}</td>
                  <td className="px-6 py-4">
                    <select value={c.status || 'pending'} onChange={(e) => setStatus(c.id, e.target.value)} className={`px-2 py-1 rounded border text-xs ${inputCls}`}>
                      <option value="pending">pending</option><option value="active">active</option><option value="blocked">blocked</option><option value="disabled">disabled</option>
                    </select>
                  </td>
                  <td className={`px-6 py-4 ${muted}`}>{c.limit || '—'}</td>
                  <td className="px-6 py-4 flex gap-2">
                    <button onClick={() => activate(c.id)} className="text-green-500 text-xs">Activate</button>
                    <button onClick={() => block(c.id)} className="text-red-500 text-xs">Block</button>
                    <button onClick={() => { setLimitId(c.id); setLimit(''); }} className="text-blue-500 text-xs">Set Limit</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {!loading && tab === 'transactions' && (
        <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
          <table className="w-full">
            <thead className={thBg}><tr>
              {['Card', 'Amount', 'Currency', 'Status', 'Time'].map((h) => (
                <th key={h} className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>{h}</th>
              ))}
            </tr></thead>
            <tbody className={`divide-y ${border}`}>
              {txs.length === 0 && <tr><td colSpan={5} className={`px-6 py-8 text-center ${muted}`}>No card transactions.</td></tr>}
              {txs.map((t) => (
                <tr key={t.id}>
                  <td className={`px-6 py-4 font-mono text-xs ${muted}`}>{t.card_id ? short(t.card_id) : '—'}</td>
                  <td className={`px-6 py-4 ${cardText}`}>{t.amount}</td>
                  <td className={`px-6 py-4 ${cardText}`}>{t.currency}</td>
                  <td className="px-6 py-4"><span className={badge(t.status)}>{t.status}</span></td>
                  <td className={`px-6 py-4 text-xs ${muted}`}>{t.timestamp ? new Date(t.timestamp).toLocaleString() : '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
