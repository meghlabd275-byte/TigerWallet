/**
 * Fees Page - White Label Admin
 * Lists fee_structures rows and allows creating/toggling fees.
 */

import React, { useEffect, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';
import { useTheme } from '../context/ThemeContext';

export default function Fees() {
  const { isDark } = useTheme();
  const [fees, setFees] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showAdd, setShowAdd] = useState(false);
  const [form, setForm] = useState({ fee_type: 'trading', asset: '', fee_percent: '0.1', fee_fixed: '0', tier: '' });

  useEffect(() => { loadFees(); }, []);

  const loadFees = async () => {
    setLoading(true); setError('');
    try {
      const data = await whiteLabelAdminApi.getFees();
      setFees(data.fees || []);
    } catch (e: any) { setError(e.message || 'Failed to load fees'); }
    finally { setLoading(false); }
  };

  const handleCreate = async () => {
    try {
      await whiteLabelAdminApi.createFee({ ...form, tier: form.tier || undefined, asset: form.asset || undefined });
      setShowAdd(false);
      setForm({ fee_type: 'trading', asset: '', fee_percent: '0.1', fee_fixed: '0', tier: '' });
      loadFees();
    } catch (e: any) { setError(e.message || 'Failed to create fee'); }
  };

  const toggle = async (f: any) => {
    try { await whiteLabelAdminApi.updateFees(f.id, { is_active: !f.is_active }); loadFees(); }
    catch (e: any) { setError(e.message || 'Failed to update fee'); }
  };

  const cardBg = isDark ? 'bg-gray-800' : 'bg-white';
  const cardText = isDark ? 'text-white' : 'text-gray-900';
  const border = isDark ? 'border-gray-700' : 'border-gray-200';
  const muted = isDark ? 'text-gray-400' : 'text-gray-500';
  const thBg = isDark ? 'bg-gray-700' : 'bg-gray-50';
  const inputCls = `w-full px-3 py-2 rounded border ${border} ${cardBg} ${cardText}`;

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className={`text-2xl font-bold ${cardText}`}>Fee Configuration</h1>
        <button onClick={() => setShowAdd(true)} className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">+ Add Fee</button>
      </div>
      {error && <div className={`mb-4 p-3 rounded ${isDark ? 'bg-red-900/50 text-red-200' : 'bg-red-50 text-red-700'}`}>{error}</div>}
      {loading && <div className={muted}>Loading...</div>}
      {!loading && (
        <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
          <table className="w-full">
            <thead className={thBg}>
              <tr>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Type</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Asset</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Percent</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Fixed</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Tier</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Status</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Actions</th>
              </tr>
            </thead>
            <tbody className={`divide-y ${border}`}>
              {fees.length === 0 && (
                <tr><td colSpan={7} className={`px-6 py-8 text-center ${muted}`}>No fee structures configured.</td></tr>
              )}
              {fees.map((f) => (
                <tr key={f.id}>
                  <td className={`px-6 py-4 ${cardText}`}>{f.fee_type}</td>
                  <td className={`px-6 py-4 ${muted}`}>{f.asset || '—'}</td>
                  <td className={`px-6 py-4 ${cardText}`}>{f.fee_percent}</td>
                  <td className={`px-6 py-4 ${cardText}`}>{f.fee_fixed}</td>
                  <td className={`px-6 py-4 ${muted}`}>{f.tier || '—'}</td>
                  <td className="px-6 py-4"><span className={`px-2 py-0.5 text-xs rounded ${f.is_active ? (isDark ? 'bg-green-900 text-green-200' : 'bg-green-100 text-green-800') : (isDark ? 'bg-gray-700 text-gray-300' : 'bg-gray-200 text-gray-600')}`}>{f.is_active ? 'Active' : 'Inactive'}</span></td>
                  <td className="px-6 py-4"><button onClick={() => toggle(f)} className="text-yellow-600 hover:underline text-sm">{f.is_active ? 'Disable' : 'Enable'}</button></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {showAdd && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className={`${cardBg} rounded-lg p-6 w-full max-w-md`}>
            <h2 className={`text-lg font-bold mb-4 ${cardText}`}>Add Fee Structure</h2>
            <div className="space-y-3">
              <select value={form.fee_type} onChange={e => setForm({ ...form, fee_type: e.target.value })} className={inputCls}>
                <option value="trading">Trading</option>
                <option value="withdrawal">Withdrawal</option>
                <option value="deposit">Deposit</option>
                <option value="swap">Swap</option>
              </select>
              <input placeholder="Asset (optional)" value={form.asset} onChange={e => setForm({ ...form, asset: e.target.value })} className={inputCls} />
              <input placeholder="Fee %" value={form.fee_percent} onChange={e => setForm({ ...form, fee_percent: e.target.value })} className={inputCls} />
              <input placeholder="Fixed fee" value={form.fee_fixed} onChange={e => setForm({ ...form, fee_fixed: e.target.value })} className={inputCls} />
              <input placeholder="Tier (optional)" value={form.tier} onChange={e => setForm({ ...form, tier: e.target.value })} className={inputCls} />
            </div>
            <div className="flex justify-end gap-2 mt-4">
              <button onClick={() => setShowAdd(false)} className={`px-4 py-2 rounded ${isDark ? 'bg-gray-700 text-white' : 'bg-gray-200 text-gray-800'}`}>Cancel</button>
              <button onClick={handleCreate} className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">Create</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
