/**
 * P2P & Fiat (p2p_admin) — P2P adverts, on-ramp/off-ramp orders, merchants.
 * Backend: GET /api/v1/admin/transactions (filter p2p/onramp/offramp),
 * GET /api/v1/admin/users (merchant list).
 */

import React, { useEffect, useMemo, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';
import { useTheme } from '../context/ThemeContext';

const P2P_TYPES = ['p2p', 'on_ramp', 'off_ramp', 'onramp', 'offramp'];

export default function P2PFiat() {
  const { isDark } = useTheme();
  const [txs, setTxs] = useState<any[]>([]);
  const [users, setUsers] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [tab, setTab] = useState<'orders' | 'merchants'>('orders');

  useEffect(() => { load(); }, []);

  const load = async () => {
    setLoading(true); setError('');
    try {
      const [txResp, userResp] = await Promise.all([
        whiteLabelAdminApi.getTransactions(),
        whiteLabelAdminApi.getUsers(),
      ]);
      const all = txResp.transactions || [];
      setTxs(all.filter((t) => P2P_TYPES.includes(String(t.type).toLowerCase())));
      setUsers(userResp.users || []);
    } catch (e: any) { setError(e.message || 'Failed to load P2P data'); }
    finally { setLoading(false); }
  };

  const volume = useMemo(() => {
    const sum: Record<string, number> = {};
    txs.forEach((t) => {
      const k = String(t.type).toLowerCase();
      const amt = parseFloat(t.amount) || 0;
      sum[k] = (sum[k] || 0) + amt;
    });
    return sum;
  }, [txs]);

  const cardBg = isDark ? 'bg-gray-800' : 'bg-white';
  const cardText = isDark ? 'text-white' : 'text-gray-900';
  const border = isDark ? 'border-gray-700' : 'border-gray-200';
  const muted = isDark ? 'text-gray-400' : 'text-gray-500';
  const thBg = isDark ? 'bg-gray-700' : 'bg-gray-50';

  const badge = (s: string) => {
    const b = 'px-2 py-0.5 text-xs rounded';
    if (s === 'confirmed' || s === 'completed') return `${b} ${isDark ? 'bg-green-900 text-green-200' : 'bg-green-100 text-green-800'}`;
    if (s === 'pending') return `${b} ${isDark ? 'bg-yellow-900 text-yellow-200' : 'bg-yellow-100 text-yellow-800'}`;
    return `${b} ${isDark ? 'bg-gray-700 text-gray-300' : 'bg-gray-200 text-gray-700'}`;
  };

  const statCard = (label: string, value: string) => (
    <div className={`${cardBg} p-4 rounded-lg shadow border ${border}`}>
      <p className={`text-xs ${muted}`}>{label}</p>
      <p className={`text-xl font-bold mt-1 ${cardText}`}>{value}</p>
    </div>
  );

  return (
    <div className="p-6">
      <h1 className={`text-2xl font-bold mb-2 ${cardText}`}>P2P & Fiat</h1>
      <p className={`mb-4 ${muted}`}>Adverts, on-ramp / off-ramp orders and merchants.</p>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-6">
        {statCard('Total Orders', String(txs.length))}
        {statCard('On-Ramp Volume', (volume['on_ramp'] || volume['onramp'] || 0).toFixed(2))}
        {statCard('Off-Ramp Volume', (volume['off_ramp'] || volume['offramp'] || 0).toFixed(2))}
        {statCard('Merchants', String(users.length))}
      </div>

      <div className="flex gap-2 mb-4">
        <button onClick={() => setTab('orders')} className={`px-3 py-1 rounded text-sm ${tab === 'orders' ? 'bg-blue-600 text-white' : (isDark ? 'bg-gray-700 text-gray-200' : 'bg-gray-200 text-gray-700')}`}>Orders</button>
        <button onClick={() => setTab('merchants')} className={`px-3 py-1 rounded text-sm ${tab === 'merchants' ? 'bg-blue-600 text-white' : (isDark ? 'bg-gray-700 text-gray-200' : 'bg-gray-200 text-gray-700')}`}>Merchants</button>
      </div>

      {error && <div className={`mb-3 p-2 rounded text-sm ${isDark ? 'bg-red-900/50 text-red-200' : 'bg-red-50 text-red-700'}`}>{error}</div>}
      {loading && <div className={muted}>Loading…</div>}

      {!loading && tab === 'orders' && (
        <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
          <table className="w-full">
            <thead className={thBg}>
              <tr>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Type</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>User</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Amount</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Asset</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Status</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Time</th>
              </tr>
            </thead>
            <tbody className={`divide-y ${border}`}>
              {txs.length === 0 && <tr><td colSpan={6} className={`px-6 py-8 text-center ${muted}`}>No P2P / fiat orders.</td></tr>}
              {txs.map((t) => (
                <tr key={t.id}>
                  <td className={`px-6 py-4 ${cardText} capitalize`}>{t.type}</td>
                  <td className={`px-6 py-4 font-mono text-xs ${muted}`}>{t.user_id ? String(t.user_id).substring(0, 8) + '…' : '—'}</td>
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

      {!loading && tab === 'merchants' && (
        <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
          <table className="w-full">
            <thead className={thBg}>
              <tr>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Email</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Username</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>KYC</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Status</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Country</th>
              </tr>
            </thead>
            <tbody className={`divide-y ${border}`}>
              {users.length === 0 && <tr><td colSpan={5} className={`px-6 py-8 text-center ${muted}`}>No merchants.</td></tr>}
              {users.map((u) => (
                <tr key={u.id}>
                  <td className={`px-6 py-4 ${cardText}`}>{u.email}</td>
                  <td className={`px-6 py-4 ${cardText}`}>{u.username}</td>
                  <td className="px-6 py-4"><span className={badge(u.kyc_status)}>{u.kyc_status || '—'}</span></td>
                  <td className="px-6 py-4"><span className={badge(u.status)}>{u.status}</span></td>
                  <td className={`px-6 py-4 ${muted}`}>{u.country || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
