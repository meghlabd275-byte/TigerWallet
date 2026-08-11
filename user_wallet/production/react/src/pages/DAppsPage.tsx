/**
 * DApps Page - DApp Browser
 *
 * Lists dapps from the canonical backend (GET /dapps, /dapps/categories).
 * No hardcoded dapp list — everything comes from wallet_api.
 */

import React, { useEffect, useState } from 'react';
import { useTheme } from '../contexts/ThemeContext';
import { WalletService } from '../services/WalletService';

interface Dapp {
  id?: string | number;
  name: string;
  category?: string;
  url?: string;
  description?: string;
}

function DAppsPage() {
  const { theme } = useTheme();
  const [walletService] = useState(() => new WalletService());
  const [url, setUrl] = useState('');
  const [activeTab, setActiveTab] = useState('all');
  const [dapps, setDapps] = useState<Dapp[]>([]);
  const [categories, setCategories] = useState<string[]>(['All']);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setLoading(true);
      setError(null);
      try {
        const [list, cats] = await Promise.all([
          walletService.getDapps() as Promise<Dapp[]>,
          walletService.getDappCategories(),
        ]);
        if (cancelled) return;
        setDapps(list);
        setCategories(['All', ...cats]);
      } catch (e) {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : 'Failed to load dapps');
          setDapps([]);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, [walletService]);

  const filtered =
    activeTab === 'all'
      ? dapps
      : dapps.filter((d) => (d.category ?? '').toLowerCase() === activeTab.toLowerCase());

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">DApps</h1>

      {/* URL Bar */}
      <div className={`card mb-6 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
        <div className="flex gap-2">
          <input
            type="text"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="Enter DApp URL..."
            className="input flex-1"
          />
          <button className="btn btn-primary">Go</button>
        </div>
      </div>

      {/* Categories */}
      <div className="flex gap-2 mb-6 overflow-x-auto">
        {categories.map((cat) => (
          <button
            key={cat}
            onClick={() => setActiveTab(cat.toLowerCase())}
            className={`px-4 py-2 rounded-lg text-sm whitespace-nowrap ${
              activeTab === cat.toLowerCase()
                ? 'bg-amber-500 text-black'
                : theme === 'dark' ? 'bg-slate-800' : 'bg-gray-200'
            }`}
          >
            {cat}
          </button>
        ))}
      </div>

      {loading && <p className="opacity-60">Loading dapps…</p>}
      {error && (
        <div className={`card mb-6 ${theme === 'dark' ? 'bg-red-900/30' : 'bg-red-50'}`}>
          <p className="text-sm">{error}</p>
        </div>
      )}

      {/* DApps Grid */}
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
        {filtered.map((dapp) => (
          <div
            key={dapp.id ?? dapp.name}
            className={`card cursor-pointer hover:border-amber-500 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}
          >
            <div className="w-12 h-12 bg-amber-500/20 rounded-lg mb-3 flex items-center justify-center text-2xl">
              🔗
            </div>
            <h3 className="font-semibold">{dapp.name}</h3>
            <p className="text-sm opacity-60">{dapp.category}</p>
            {dapp.url && (
              <p className="text-xs opacity-40 truncate">{dapp.url}</p>
            )}
          </div>
        ))}
      </div>

      {!loading && !error && filtered.length === 0 && (
        <p className="opacity-60">No dapps available.</p>
      )}
    </div>
  );
}

export default DAppsPage;
