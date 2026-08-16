/**
 * Tokens Page - White Label Admin
 * Base token listing view. The expanded Listings page adds pairs + blockchains.
 */

import React, { useEffect, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';
import { useTheme } from '../context/ThemeContext';

export default function Tokens() {
  const { isDark } = useTheme();
  const [tokens, setTokens] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => { loadTokens(); }, []);

  const loadTokens = async () => {
    setLoading(true); setError('');
    try {
      const data = await whiteLabelAdminApi.getTokens();
      setTokens(data.tokens || []);
    } catch (e: any) { setError(e.message || 'Failed to load tokens'); }
    finally { setLoading(false); }
  };

  const toggleActive = async (t: any) => {
    try { await whiteLabelAdminApi.updateToken(t.id, { is_active: !t.is_active }); loadTokens(); }
    catch (e: any) { setError(e.message || 'Failed to update token'); }
  };

  const verify = async (t: any) => {
    try { await whiteLabelAdminApi.updateToken(t.id, { is_verified: true }); loadTokens(); }
    catch (e: any) { setError(e.message || 'Failed to verify token'); }
  };

  const remove = async (id: string) => {
    if (!confirm('Delete this token?')) return;
    try { await whiteLabelAdminApi.deleteToken(id); loadTokens(); }
    catch (e: any) { setError(e.message || 'Failed to delete token'); }
  };

  const cardBg = isDark ? 'bg-gray-800' : 'bg-white';
  const cardText = isDark ? 'text-white' : 'text-gray-900';
  const border = isDark ? 'border-gray-700' : 'border-gray-200';
  const muted = isDark ? 'text-gray-400' : 'text-gray-500';
  const thBg = isDark ? 'bg-gray-700' : 'bg-gray-50';

  const short = (v: string) => (v ? `${v.substring(0, 10)}…` : '—');

  return (
    <div className="p-6">
      <h1 className={`text-2xl font-bold mb-6 ${cardText}`}>Token Management</h1>
      {error && <div className={`mb-4 p-3 rounded ${isDark ? 'bg-red-900/50 text-red-200' : 'bg-red-50 text-red-700'}`}>{error}</div>}
      {loading && <div className={muted}>Loading...</div>}
      {!loading && (
        <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
          <table className="w-full">
            <thead className={thBg}>
              <tr>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Name</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Symbol</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Contract</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Chain</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Status</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Actions</th>
              </tr>
            </thead>
            <tbody className={`divide-y ${border}`}>
              {tokens.length === 0 && (
                <tr><td colSpan={6} className={`px-6 py-8 text-center ${muted}`}>No tokens listed yet.</td></tr>
              )}
              {tokens.map((t) => (
                <tr key={t.id}>
                  <td className={`px-6 py-4 ${cardText}`}>{t.name}</td>
                  <td className={`px-6 py-4 ${cardText}`}>{t.symbol}</td>
                  <td className={`px-6 py-4 font-mono text-xs ${muted}`}>{short(t.contract_address)}</td>
                  <td className={`px-6 py-4 ${muted}`}>{t.chain_id}</td>
                  <td className="px-6 py-4 space-x-1">
                    <span className={`px-2 py-0.5 text-xs rounded ${t.is_active ? (isDark ? 'bg-green-900 text-green-200' : 'bg-green-100 text-green-800') : (isDark ? 'bg-gray-700 text-gray-300' : 'bg-gray-200 text-gray-600')}`}>{t.is_active ? 'Active' : 'Inactive'}</span>
                    {t.is_verified && <span className={`px-2 py-0.5 text-xs rounded ${isDark ? 'bg-blue-900 text-blue-200' : 'bg-blue-100 text-blue-800'}`}>Verified</span>}
                  </td>
                  <td className="px-6 py-4 space-x-2">
                    {!t.is_verified && <button onClick={() => verify(t)} className="text-blue-600 hover:underline text-sm">Verify</button>}
                    <button onClick={() => toggleActive(t)} className="text-yellow-600 hover:underline text-sm">{t.is_active ? 'Disable' : 'Enable'}</button>
                    <button onClick={() => remove(t.id)} className="text-red-600 hover:underline text-sm">Delete</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
