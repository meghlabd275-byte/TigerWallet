'use client';

import React, { useState, useEffect } from 'react';

interface ApiResponse<T> {
  success: boolean;
  data: T;
  error?: string;
}

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'https://api.tigerwallet.io';

const fetchAPI = async <T,>(endpoint: string, options?: RequestInit): Promise<T> => {
  const token = typeof window !== 'undefined' ? localStorage.getItem('tigerwallet-token') : null;
  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options?.headers,
    },
  });
  if (!response.ok) throw new Error(`API Error: ${response.statusText}`);
  const data: ApiResponse<T> = await response.json();
  return data.data;
};

const WIDGET_TYPES = [
  { id: 'portfolio', name: 'Portfolio Value', icon: '💰', description: 'Show total balance' },
  { id: 'price', name: 'Price Ticker', icon: '📈', description: 'Live price display' },
  { id: 'quick', name: 'Quick Send', icon: '📤', description: 'Send tokens fast' },
  { id: 'chart', name: 'Price Chart', icon: '📊', description: 'Interactive chart' },
];

export default function Widgets() {
  const [activeWidgets, setActiveWidgets] = useState<string[]>(['portfolio', 'price']);
  const [theme, setTheme] = useState('light');
  const [loading, setLoading] = useState(false);
  const [saved, setSaved] = useState(false);

  // Load widget preferences
  useEffect(() => {
    const loadPreferences = async () => {
      try {
        const prefs = await fetchAPI<{ widgets: string[]; theme: string }>('/user/preferences');
        if (prefs?.widgets) setActiveWidgets(prefs.widgets);
        if (prefs?.theme) setTheme(prefs.theme);
      } catch (err) {
        const savedWidgets = localStorage.getItem('tigerwallet-widgets');
        if (savedWidgets) setActiveWidgets(JSON.parse(savedWidgets));
      }
    };
    loadPreferences();
  }, []);

  const toggleWidget = (id: string) => {
    setActiveWidgets(prev => prev.includes(id) ? prev.filter(w => w !== id) : [...prev, id]);
    setSaved(false);
  };

  const saveWidgets = async () => {
    setLoading(true);
    try {
      await fetchAPI('/user/preferences', {
        method: 'PUT',
        body: JSON.stringify({ widgets: activeWidgets, theme }),
      });
    } catch (err) {
      localStorage.setItem('tigerwallet-widgets', JSON.stringify(activeWidgets));
    }
    setLoading(false);
    setSaved(true);
    setTimeout(() => setSaved(false), 2000);
  };

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900 text-slate-900 dark:text-white">
      <header className="bg-white dark:bg-slate-800 border-b p-4"><div className="flex items-center gap-4"><a href="/wallet" className="text-2xl">🐯</a><h1 className="text-xl font-bold">Widgets</h1></div></header>
      <div className="max-w-2xl mx-auto p-8">
        <div className="bg-white dark:bg-slate-800 rounded-lg p-6 mb-6">
          <h2 className="font-semibold mb-4">Widget Type</h2>
          <div className="grid grid-cols-2 gap-4 mb-6">{WIDGET_TYPES.map(w => <button key={w.id} onClick={() => toggleWidget(w.id)} className={`p-4 rounded-lg border-2 text-left ${activeWidgets.includes(w.id) ? 'border-orange-500 bg-orange-50 dark:bg-orange-900/20' : 'border-slate-200 dark:border-slate-700'}`}><div className="text-2xl mb-2">{w.icon}</div><div className="font-semibold">{w.name}</div><div className="text-xs text-slate-500">{w.description}</div></button>)}</div>
        </div>
        <div className="bg-white dark:bg-slate-800 rounded-lg p-6 mb-6">
          <h2 className="font-semibold mb-4">Preview</h2>
          <div className="bg-slate-100 dark:bg-slate-700 rounded-lg p-4 min-h-[200px]">
            {activeWidgets.includes('portfolio') && <div className="bg-white dark:bg-slate-800 rounded-lg p-4 mb-2"><div className="text-sm text-slate-500">Total Balance</div><div className="text-2xl font-bold">$12,450.00</div></div>}
            {activeWidgets.includes('price') && <div className="bg-white dark:bg-slate-800 rounded-lg p-4 mb-2 flex justify-between"><span>ETH</span><span className="text-green-500">$3,524.50</span></div>}
            {activeWidgets.length === 0 && <div className="text-center text-slate-500 py-8">Select widgets to preview</div>}
          </div>
        </div>
        {saved && (
          <div className="bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400 px-4 py-2 rounded-lg mb-4 text-center">
            ✓ Widget preferences saved!
          </div>
        )}
        <button 
          onClick={saveWidgets}
          disabled={loading}
          className="w-full bg-orange-500 hover:bg-orange-600 disabled:bg-slate-400 text-white py-4 rounded-lg font-semibold"
        >
          {loading ? 'Saving...' : 'Save Widgets'}
        </button>
      </div>
    </div>
  );
}
