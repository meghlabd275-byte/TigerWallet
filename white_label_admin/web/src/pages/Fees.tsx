/**
 * Fees Page - White Label Admin
 */

import React, { useEffect, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';

export default function Fees() {
  const [fees, setFees] = useState<any>({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => { loadFees(); }, []);

  const loadFees = async () => {
    try {
      const data = await whiteLabelAdminApi.getFees();
      setFees(data);
    } catch (error) { console.error('Failed:', error); }
    finally { setLoading(false); }
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      await whiteLabelAdminApi.updateFees(fees);
      alert('Fees saved');
    } catch (error) { console.error('Failed:', error); }
    finally { setSaving(false); }
  };

  if (loading) return <div className="p-8">Loading...</div>;

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Fee Configuration</h1>
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6 space-y-4 max-w-lg">
        <div>
          <label className="block text-sm font-medium mb-1">Trading Fee (%)</label>
          <input type="number" step="0.01" value={fees.tradingFee || 0} onChange={(e) => setFees({...fees, tradingFee: parseFloat(e.target.value)})} className="w-full px-3 py-2 border rounded dark:bg-gray-700" />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Withdrawal Fee (%)</label>
          <input type="number" step="0.01" value={fees.withdrawalFee || 0} onChange={(e) => setFees({...fees, withdrawalFee: parseFloat(e.target.value)})} className="w-full px-3 py-2 border rounded dark:bg-gray-700" />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Deposit Fee (%)</label>
          <input type="number" step="0.01" value={fees.depositFee || 0} onChange={(e) => setFees({...fees, depositFee: parseFloat(e.target.value)})} className="w-full px-3 py-2 border rounded dark:bg-gray-700" />
        </div>
        <button onClick={handleSave} disabled={saving} className="bg-blue-600 text-white px-6 py-2 rounded hover:bg-blue-700">{saving ? 'Saving...' : 'Save'}</button>
      </div>
    </div>
  );
}
