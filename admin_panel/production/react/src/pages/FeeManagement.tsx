/**
 * Fee Management
 * Connected to backend APIs
 */

import React, { useState, useEffect, useCallback } from 'react';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';

interface FeeConfig {
  trading: {
    maker: number;
    taker: number;
    volumeDiscount: boolean;
  };
  withdrawal: Record<string, string>;
  deposit: {
    enabled: boolean;
    fee: string;
  };
  swap: {
    fee: number;
    router: string;
  };
  bridge: {
    fee: number;
    enabled: boolean;
  };
}

function FeeManagement() {
  const [activeTab, setActiveTab] = useState('trading');
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [fees, setFees] = useState<FeeConfig>({
    trading: { maker: 0.1, taker: 0.2, volumeDiscount: true },
    withdrawal: { eth: '0.005', btc: '0.0005', usdt: '1' },
    deposit: { enabled: false, fee: '0' },
    swap: { fee: 0.3, router: 'Uniswap V3' },
    bridge: { fee: 0.5, enabled: true },
  });

  // Fetch fees from backend
  const fetchFees = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const token = localStorage.getItem('superadmin_token');
      const response = await fetch(`${API_BASE_URL}/super-admin/fees`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      });
      
      if (!response.ok) {
        throw new Error('Failed to fetch fees');
      }
      
      const data = await response.json();
      if (data.fees) {
        setFees(data.fees);
      }
    } catch (err) {
      console.error('Error fetching fees:', err);
      setError(err instanceof Error ? err.message : 'Failed to load fees');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchFees();
  }, [fetchFees]);

  // Save fees to backend
  const handleSaveFees = async () => {
    setSaving(true);
    setError(null);
    setSuccess(null);
    try {
      const token = localStorage.getItem('superadmin_token');
      const response = await fetch(`${API_BASE_URL}/super-admin/fees`, {
        method: 'PUT',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ fees }),
      });
      
      if (!response.ok) {
        throw new Error('Failed to save fees');
      }
      
      setSuccess('Fees saved successfully');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save fees');
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-amber-500"></div>
      </div>
    );
  }

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Fee Management</h1>

      {error && (
        <div className="bg-red-500/20 border border-red-500 text-red-500 px-4 py-3 rounded-lg mb-4">
          {error}
        </div>
      )}

      {success && (
        <div className="bg-green-500/20 border border-green-500 text-green-500 px-4 py-3 rounded-lg mb-4">
          {success}
        </div>
      )}

      <div className="flex gap-2 mb-6">
        {['Trading', 'Withdrawal', 'Deposit', 'Swap', 'Bridge'].map(tab => (
          <button key={tab} onClick={() => setActiveTab(tab.toLowerCase())} className={`px-4 py-2 rounded-lg ${activeTab === tab.toLowerCase() ? 'bg-amber-500 text-black' : 'bg-slate-800'}`}>
            {tab}
          </button>
        ))}
      </div>

      {activeTab === 'trading' && (
        <div className="space-y-6">
          <div className="bg-slate-800 p-6 rounded-lg">
            <h3 className="font-semibold mb-4">Trading Fees</h3>
            <div className="grid grid-cols-2 gap-4 max-w-md">
              <div>
                <label className="label">Maker Fee (%)</label>
                <input 
                  type="number" 
                  className="input" 
                  value={fees.trading.maker} 
                  onChange={(e) => setFees({...fees, trading: {...fees.trading, maker: parseFloat(e.target.value)}})}
                  step="0.01" 
                />
              </div>
              <div>
                <label className="label">Taker Fee (%)</label>
                <input 
                  type="number" 
                  className="input" 
                  value={fees.trading.taker} 
                  onChange={(e) => setFees({...fees, trading: {...fees.trading, taker: parseFloat(e.target.value)}})}
                  step="0.01" 
                />
              </div>
            </div>
            <div className="flex items-center gap-2 mt-4">
              <input 
                type="checkbox" 
                checked={fees.trading.volumeDiscount} 
                onChange={(e) => setFees({...fees, trading: {...fees.trading, volumeDiscount: e.target.checked}})}
                id="volumeDiscount" 
              />
              <label htmlFor="volumeDiscount">Enable Volume Discounts</label>
            </div>
            <button onClick={handleSaveFees} disabled={saving} className="btn btn-primary mt-4">
              {saving ? 'Saving...' : 'Save'}
            </button>
          </div>
        </div>
      )}

      {activeTab === 'withdrawal' && (
        <div className="space-y-6">
          <div className="bg-slate-800 p-6 rounded-lg">
            <h3 className="font-semibold mb-4">Withdrawal Fees</h3>
            <div className="space-y-4 max-w-md">
              <div>
                <label className="label">Ethereum (ETH)</label>
                <input type="text" className="input" defaultValue={fees.withdrawal.eth} />
              </div>
              <div>
                <label className="label">Bitcoin (BTC)</label>
                <input type="text" className="input" defaultValue={fees.withdrawal.btc} />
              </div>
              <div>
                <label className="label">Tether (USDT)</label>
                <input type="text" className="input" defaultValue={fees.withdrawal.usdt} />
              </div>
            </div>
            <button className="btn btn-primary mt-4">Save</button>
          </div>
        </div>
      )}

      {activeTab === 'deposit' && (
        <div className="space-y-6">
          <div className="bg-slate-800 p-6 rounded-lg">
            <h3 className="font-semibold mb-4">Deposit Fees</h3>
            <div className="flex items-center gap-2 mb-4">
              <input type="checkbox" defaultChecked={fees.deposit.enabled} id="depositFee" />
              <label htmlFor="depositFee">Enable Deposit Fee</label>
            </div>
            <div className="max-w-md">
              <label className="label">Fee Amount (%)</label>
              <input type="number" className="input" defaultValue={fees.deposit.fee} step="0.01" />
            </div>
            <button className="btn btn-primary mt-4">Save</button>
          </div>
        </div>
      )}

      {activeTab === 'swap' && (
        <div className="space-y-6">
          <div className="bg-slate-800 p-6 rounded-lg">
            <h3 className="font-semibold mb-4">Swap Fees</h3>
            <div className="space-y-4 max-w-md">
              <div>
                <label className="label">Swap Fee (%)</label>
                <input type="number" className="input" defaultValue={fees.swap.fee.replace('%', '')} step="0.01" />
              </div>
              <div>
                <label className="label">Router</label>
                <select className="input" defaultValue={fees.swap.router}>
                  <option>Uniswap V3</option>
                  <option>Curve</option>
                  <option>Balancer</option>
                  <option>1inch</option>
                </select>
              </div>
            </div>
            <button className="btn btn-primary mt-4">Save</button>
          </div>
        </div>
      )}

      {activeTab === 'bridge' && (
        <div className="space-y-6">
          <div className="bg-slate-800 p-6 rounded-lg">
            <h3 className="font-semibold mb-4">Bridge Fees</h3>
            <div className="flex items-center gap-2 mb-4">
              <input type="checkbox" defaultChecked={fees.bridge.enabled} id="bridgeFee" />
              <label htmlFor="bridgeFee">Enable Bridge</label>
            </div>
            <div className="max-w-md">
              <label className="label">Bridge Fee (%)</label>
              <input type="number" className="input" defaultValue={fees.bridge.fee.replace('%', '')} step="0.01" />
            </div>
            <button className="btn btn-primary mt-4">Save</button>
          </div>
        </div>
      )}
    </div>
  );
}

export default FeeManagement;
