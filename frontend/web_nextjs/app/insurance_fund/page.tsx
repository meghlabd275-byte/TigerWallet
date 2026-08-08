'use client';

import React, { useState, useEffect, useCallback } from 'react';
import api, { InsuranceStats, InsurancePosition, InsuranceClaim, InsuranceProduct } from '../../src/lib/api/client';

export default function InsuranceFundPage() {
  const [activeTab, setActiveTab] = useState<'coverage' | 'claims' | 'positions'>('coverage');
  const [stats, setStats] = useState<InsuranceStats | null>(null);
  const [products, setProducts] = useState<InsuranceProduct[]>([]);
  const [positions, setPositions] = useState<InsurancePosition[]>([]);
  const [claims, setClaims] = useState<InsuranceClaim[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [showBuyModal, setShowBuyModal] = useState(false);
  const [selectedPool, setSelectedPool] = useState('');
  const [coverageAmount, setCoverageAmount] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [success, setSuccess] = useState<string | null>(null);

  const fetchAll = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [statsRes, productsRes, positionsRes, claimsRes] = await Promise.all([
        api.getInsuranceStats(),
        api.getInsuranceProducts(),
        api.getInsurancePositions(),
        api.getInsuranceClaims(),
      ]);
      if (statsRes.success && statsRes.data) setStats(statsRes.data);
      if (productsRes.success && productsRes.data) setProducts(productsRes.data);
      if (positionsRes.success && positionsRes.data) setPositions(positionsRes.data);
      if (claimsRes.success && claimsRes.data) setClaims(claimsRes.data);

      const failed = [statsRes, productsRes, positionsRes, claimsRes].find(r => !r.success);
      if (failed?.error) setError(failed.error);
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || 'Failed to load insurance data');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchAll();
  }, [fetchAll]);

  const handleBuyCoverage = async () => {
    if (!coverageAmount || !selectedPool) return;
    setSubmitting(true);
    setError(null);
    setSuccess(null);
    try {
      const res = await api.buyInsuranceCoverage(selectedPool, coverageAmount);
      if (res.success) {
        setSuccess(`Successfully purchased $${coverageAmount} coverage for ${selectedPool}!`);
        setShowBuyModal(false);
        setCoverageAmount('');
        setSelectedPool('');
        fetchAll();
      } else {
        setError(res.error || 'Failed to purchase coverage');
      }
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || 'Failed to purchase coverage');
    } finally {
      setSubmitting(false);
    }
  };

  const totalCoverage = positions.reduce((sum, p) => sum + p.coverage, 0);
  const totalPremium = positions.reduce((sum, p) => sum + p.premium, 0);

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900">
      <header className="bg-gradient-to-r from-emerald-600 to-teal-600 text-white">
        <div className="max-w-7xl mx-auto px-4 py-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <a href="/" className="text-3xl">🐯</a>
              <div>
                <h1 className="text-2xl font-bold">Insurance Fund</h1>
                <p className="text-emerald-200">Protect your assets with coverage</p>
              </div>
            </div>
            <button
              onClick={() => setShowBuyModal(true)}
              className="px-6 py-3 bg-white text-emerald-600 rounded-lg font-semibold hover:bg-emerald-50"
            >
              Buy Coverage
            </button>
          </div>
        </div>
      </header>

      <div className="max-w-7xl mx-auto px-4 py-6">
        {error && (
          <div className="mb-4 p-3 rounded-lg bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200">
            {error}
          </div>
        )}
        {success && (
          <div className="mb-4 p-3 rounded-lg bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200">
            {success}
          </div>
        )}

        {/* Hero Stats */}
        <div className="bg-gradient-to-r from-emerald-600 to-teal-600 rounded-2xl p-8 mb-6 text-white">
          {loading && !stats ? (
            <div className="text-center py-6 text-emerald-100">Loading fund stats...</div>
          ) : stats ? (
            <div className="grid grid-cols-5 gap-4">
              <div>
                <p className="text-emerald-200 text-sm">Total Pool</p>
                <p className="text-3xl font-bold">${(stats.totalPool / 1000000).toFixed(1)}M</p>
              </div>
              <div>
                <p className="text-emerald-200 text-sm">Reserves</p>
                <p className="text-3xl font-bold">${(stats.reserves / 1000000).toFixed(1)}M</p>
              </div>
              <div>
                <p className="text-emerald-200 text-sm">Coverage Active</p>
                <p className="text-3xl font-bold">${(stats.coverage / 1000000).toFixed(1)}M</p>
              </div>
              <div>
                <p className="text-emerald-200 text-sm">Claims Paid</p>
                <p className="text-3xl font-bold">${(stats.claimsPaid / 1000).toFixed(0)}K</p>
              </div>
              <div>
                <p className="text-emerald-200 text-sm">APY</p>
                <p className="text-3xl font-bold">{stats.apy}%</p>
              </div>
            </div>
          ) : (
            <div className="text-center py-6 text-emerald-100">Fund stats unavailable.</div>
          )}
        </div>

        {/* Your Coverage */}
        <div className="bg-white dark:bg-slate-800 rounded-xl p-6 mb-6 border">
          <h3 className="text-lg font-bold mb-4">Your Active Coverage</h3>
          {positions.length === 0 ? (
            <p className="text-slate-500 text-center py-8">No active coverage</p>
          ) : (
            <div className="space-y-3">
              {positions.map(pos => (
                <div key={pos.id} className="flex items-center justify-between p-4 bg-slate-50 dark:bg-slate-700 rounded-lg">
                  <div>
                    <p className="font-semibold">{pos.pool}</p>
                    <p className="text-sm text-slate-500">Premium: ${pos.premium}/year</p>
                  </div>
                  <div className="text-right">
                    <p className="font-bold text-green-600">${pos.coverage.toLocaleString()} covered</p>
                    <p className="text-xs text-slate-500">Expires: {new Date(pos.expiryTime).toLocaleDateString()}</p>
                  </div>
                </div>
              ))}
            </div>
          )}
          <div className="mt-4 p-4 bg-emerald-50 dark:bg-emerald-900/20 rounded-lg flex justify-between">
            <div>
              <p className="text-sm text-emerald-600">Total Coverage</p>
              <p className="text-xl font-bold">${totalCoverage.toLocaleString()}</p>
            </div>
            <div>
              <p className="text-sm text-emerald-600">Total Premium</p>
              <p className="text-xl font-bold">${totalPremium}/year</p>
            </div>
          </div>
        </div>

        {/* Tabs */}
        <div className="bg-white dark:bg-slate-800 rounded-xl border">
          <div className="flex border-b">
            {(['coverage', 'claims', 'positions'] as const).map(tab => (
              <button
                key={tab}
                onClick={() => setActiveTab(tab)}
                className={`px-6 py-4 font-medium ${activeTab === tab ? 'text-emerald-600 border-b-2 border-emerald-600' : 'text-slate-500'}`}
              >
                {tab.charAt(0).toUpperCase() + tab.slice(1)}
              </button>
            ))}
          </div>

          <div className="p-6">
            {activeTab === 'coverage' && (
              <div className="grid grid-cols-3 gap-4">
                {loading && products.length === 0 ? (
                  <div className="col-span-3 text-center py-8 text-slate-500">Loading coverage products...</div>
                ) : products.length === 0 ? (
                  <div className="col-span-3 text-center py-8 text-slate-500">No coverage products available.</div>
                ) : (
                  products.map(item => (
                    <div key={item.name} className="p-4 border rounded-lg hover:border-emerald-500 transition-colors">
                      <h4 className="font-bold mb-2">{item.name}</h4>
                      <p className="text-emerald-600 font-semibold">{item.coverage}</p>
                      <p className="text-sm text-slate-500">{item.premium} premium</p>
                      <p className="text-xs text-slate-400 mt-2">{item.desc}</p>
                      <button
                        onClick={() => { setSelectedPool(item.name); setShowBuyModal(true); }}
                        className="w-full mt-4 py-2 bg-emerald-600 text-white rounded-lg text-sm"
                      >
                        Get Coverage
                      </button>
                    </div>
                  ))
                )}
              </div>
            )}

            {activeTab === 'claims' && (
              <div className="space-y-4">
                {loading && claims.length === 0 ? (
                  <div className="text-center py-8 text-slate-500">Loading claims...</div>
                ) : claims.length === 0 ? (
                  <div className="text-center py-8 text-slate-500">No claims filed.</div>
                ) : (
                  claims.map(claim => (
                    <div key={claim.id} className="flex items-center justify-between p-4 bg-slate-50 dark:bg-slate-700 rounded-lg">
                      <div>
                        <p className="font-medium">{claim.pool}</p>
                        <p className="text-sm text-slate-500">{claim.reason}</p>
                      </div>
                      <div className="flex items-center gap-4">
                        <span className={`px-3 py-1 rounded-full text-xs ${
                          claim.status === 'paid' ? 'bg-green-100 text-green-800' :
                          claim.status === 'approved' ? 'bg-blue-100 text-blue-800' :
                          claim.status === 'pending' ? 'bg-yellow-100 text-yellow-800' :
                          'bg-red-100 text-red-800'
                        }`}>
                          {claim.status.toUpperCase()}
                        </span>
                        <p className="font-bold">${claim.amount.toLocaleString()}</p>
                      </div>
                    </div>
                  ))
                )}
              </div>
            )}

            {activeTab === 'positions' && (
              <div className="space-y-4">
                {loading && positions.length === 0 ? (
                  <div className="text-center py-8 text-slate-500">Loading positions...</div>
                ) : positions.length === 0 ? (
                  <div className="text-center py-8 text-slate-500">No insurance positions.</div>
                ) : (
                  positions.map(pos => (
                    <div key={pos.id} className="p-4 bg-slate-50 dark:bg-slate-700 rounded-lg">
                      <div className="flex justify-between mb-2">
                        <span className="font-medium">{pos.pool}</span>
                        <span className={`px-2 py-1 rounded text-xs ${
                          pos.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
                        }`}>
                          {pos.status.toUpperCase()}
                        </span>
                      </div>
                      <div className="grid grid-cols-3 gap-4 text-sm">
                        <div>
                          <p className="text-slate-500">Covered Amount</p>
                          <p className="font-bold">${pos.coverage.toLocaleString()}</p>
                        </div>
                        <div>
                          <p className="text-slate-500">Premium</p>
                          <p className="font-bold">${pos.premium}/year</p>
                        </div>
                        <div>
                          <p className="text-slate-500">Expires</p>
                          <p className="font-bold">{new Date(pos.expiryTime).toLocaleDateString()}</p>
                        </div>
                      </div>
                    </div>
                  ))
                )}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Buy Coverage Modal */}
      {showBuyModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-slate-800 rounded-xl p-6 max-w-md">
            <h3 className="text-xl font-bold mb-4">Buy Coverage</h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-2">Pool Type</label>
                <select
                  value={selectedPool}
                  onChange={(e) => setSelectedPool(e.target.value)}
                  className="w-full p-3 border rounded-lg"
                >
                  <option value="">Select pool</option>
                  <option value="Liquidity Pool">Liquidity Pool</option>
                  <option value="Staking Pool">Staking Pool</option>
                  <option value="Bridge">Bridge</option>
                  <option value="Smart Contract">Smart Contract</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">Coverage Amount ($)</label>
                <input
                  type="number"
                  value={coverageAmount}
                  onChange={(e) => setCoverageAmount(e.target.value)}
                  placeholder="Enter amount"
                  className="w-full p-3 border rounded-lg"
                />
              </div>
              <div className="p-3 bg-slate-50 rounded-lg">
                <p className="text-sm">Estimated Premium: ${coverageAmount ? (parseFloat(coverageAmount) * 0.001).toFixed(2) : '0'}/year</p>
              </div>
              <div className="flex gap-4">
                <button onClick={() => setShowBuyModal(false)} className="flex-1 py-3 bg-slate-200 rounded-lg">Cancel</button>
                <button onClick={handleBuyCoverage} disabled={!coverageAmount || !selectedPool || submitting} className="flex-1 py-3 bg-emerald-600 text-white rounded-lg disabled:opacity-50">
                  {submitting ? 'Processing...' : 'Buy Coverage'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
