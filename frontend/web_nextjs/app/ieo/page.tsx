'use client';

import React, { useState, useEffect, useCallback } from 'react';
import api, { IEOProject } from '../../src/lib/api/client';
import { useTheme } from '../components/ThemeProvider';

export default function IEOPage() {
  const [activeTab, setActiveTab] = useState<'upcoming' | 'sale' | 'ended'>('sale');
  const [projects, setProjects] = useState<IEOProject[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedProject, setSelectedProject] = useState<IEOProject | null>(null);
  const [buyAmount, setBuyAmount] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [success, setSuccess] = useState<string | null>(null);
  const { isDark } = useTheme();

  const fetchProjects = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.getIEOProjects();
      if (res.success && res.data) {
        setProjects(res.data);
      } else {
        setProjects([]);
        if (res.error) setError(res.error);
      }
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || 'Failed to load IEO projects');
      setProjects([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchProjects();
  }, [fetchProjects]);

  const filteredProjects = projects.filter(p => p.status === activeTab);

  const handleBuy = async () => {
    if (!selectedProject || !buyAmount) return;
    setSubmitting(true);
    setError(null);
    setSuccess(null);
    try {
      const res = await api.participateInIEO(selectedProject.id, buyAmount);
      if (res.success) {
        const tokens = (parseFloat(buyAmount) / selectedProject.price).toFixed(2);
        setSuccess(`Successfully purchased ${tokens} ${selectedProject.symbol}!`);
        setBuyAmount('');
        setSelectedProject(null);
        fetchProjects();
      } else {
        setError(res.error || 'Failed to participate in IEO');
      }
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || 'Failed to participate in IEO');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className={`min-h-screen ${isDark ? 'bg-gray-900 text-white' : 'bg-gray-50 text-gray-900'}`}>
      <header className="bg-gradient-to-r from-orange-600 to-red-600 text-white">
        <div className="max-w-7xl mx-auto px-4 py-6">
          <div className="flex items-center gap-4">
            <a href="/" className="text-3xl">🐯</a>
            <div>
              <h1 className="text-2xl font-bold">IEO / IDO</h1>
              <p className="text-orange-200">Invest in early stage projects</p>
            </div>
          </div>
        </div>
      </header>

      <div className="max-w-7xl mx-auto px-4 py-6">
        <div className="flex gap-2 mb-6">
          {(['upcoming', 'sale', 'ended'] as const).map(tab => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`px-6 py-2 rounded-lg font-medium ${activeTab === tab ? 'bg-orange-600 text-white' : isDark ? 'bg-gray-800' : 'bg-white border border-gray-200'}`}
            >
              {tab.charAt(0).toUpperCase() + tab.slice(1)}
            </button>
          ))}
        </div>

        {error && (
          <div className={`mb-4 p-3 rounded-lg ${isDark ? 'bg-red-900 text-red-200' : 'bg-red-100 text-red-800'}`}>
            {error}
          </div>
        )}
        {success && (
          <div className={`mb-4 p-3 rounded-lg ${isDark ? 'bg-green-900 text-green-200' : 'bg-green-100 text-green-800'}`}>
            {success}
          </div>
        )}

        {loading ? (
          <div className={`text-center py-12 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Loading IEO projects...</div>
        ) : filteredProjects.length === 0 ? (
          <div className={`text-center py-12 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>No {activeTab} IEO projects available.</div>
        ) : (
          <div className="space-y-4">
            {filteredProjects.map(project => (
              <div key={project.id} className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-xl p-6 border ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-4">
                    <span className="text-5xl">{project.logo || '🚀'}</span>
                    <div>
                      <h3 className="text-xl font-bold">{project.name}</h3>
                      <p className={isDark ? 'text-gray-400' : 'text-gray-500'}>{project.symbol} • {project.chain}</p>
                    </div>
                  </div>
                  <span className={`px-3 py-1 rounded-full text-sm ${
                    project.status === 'sale' ? (isDark ? 'bg-green-900/30 text-green-300' : 'bg-green-100 text-green-800') :
                    project.status === 'upcoming' ? (isDark ? 'bg-blue-900/30 text-blue-300' : 'bg-blue-100 text-blue-800') :
                    (isDark ? 'bg-gray-700 text-gray-200' : 'bg-gray-100 text-gray-800')
                  }`}>
                    {project.status.toUpperCase()}
                  </span>
                </div>

                <p className={isDark ? 'text-gray-400' : 'text-gray-500'} style={{ marginTop: '1rem' }}>{project.description}</p>

                <div className="grid grid-cols-4 gap-4 mt-6">
                  <div>
                    <p className={`text-xs ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Price</p>
                    <p className="font-bold">${project.price}</p>
                  </div>
                  <div>
                    <p className={`text-xs ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Raised</p>
                    <p className="font-bold">${project.raised.toLocaleString()}</p>
                  </div>
                  <div>
                    <p className={`text-xs ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Hard Cap</p>
                    <p className="font-bold">${project.hardCap.toLocaleString()}</p>
                  </div>
                  <div>
                    <p className={`text-xs ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Participants</p>
                    <p className="font-bold">{project.participants.toLocaleString()}</p>
                  </div>
                </div>

                {project.status === 'sale' && (
                  <div className="mt-6">
                    <div className={`h-3 ${isDark ? 'bg-gray-700' : 'bg-gray-200'} rounded-full overflow-hidden`}>
                      <div
                        className="h-full bg-gradient-to-r from-orange-500 to-red-500"
                        style={{ width: `${(project.raised / project.hardCap) * 100}%` }}
                      />
                    </div>
                    <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'} mt-2`}>{((project.raised / project.hardCap) * 100).toFixed(1)}% filled</p>
                    <button
                      onClick={() => setSelectedProject(project)}
                      className="w-full mt-4 py-3 bg-gradient-to-r from-orange-600 to-red-600 text-white rounded-lg font-semibold"
                    >
                      Buy Now
                    </button>
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>

      {selectedProject && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className={`${isDark ? 'bg-gray-800' : 'bg-white border border-gray-200'} rounded-xl p-6 max-w-md`}>
            <h3 className="text-xl font-bold mb-4">Buy {selectedProject.symbol}</h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm mb-2">Amount (USD)</label>
                <input
                  type="number"
                  value={buyAmount}
                  onChange={(e) => setBuyAmount(e.target.value)}
                  placeholder={`Min: $${selectedProject.minBuy}, Max: $${selectedProject.maxBuy}`}
                  className={`w-full p-3 border rounded-lg ${isDark ? 'bg-gray-700 border-gray-600' : 'bg-white border-gray-300'}`}
                />
              </div>
              <div className={`p-3 ${isDark ? 'bg-gray-700' : 'bg-slate-50'} rounded-lg`}>
                <p className="text-sm">You will receive: {buyAmount ? (parseFloat(buyAmount) / selectedProject.price).toFixed(2) : '0'} {selectedProject.symbol}</p>
              </div>
              <div className="flex gap-4">
                <button onClick={() => setSelectedProject(null)} className={`flex-1 py-3 ${isDark ? 'bg-gray-700' : 'bg-slate-200'} rounded-lg`}>Cancel</button>
                <button onClick={handleBuy} disabled={submitting} className="flex-1 py-3 bg-orange-600 text-white rounded-lg">
                  {submitting ? 'Processing...' : 'Confirm Purchase'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
