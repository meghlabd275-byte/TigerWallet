'use client';

import React, { useState } from 'react';

const THEMES = [
  { id: 'light', name: 'Light', bg: '#ffffff', accent: '#f97316', text: '#1f2937' },
  { id: 'dark', name: 'Dark', bg: '#0f172a', accent: '#f97316', text: '#ffffff' },
  { id: 'orange', name: 'Tiger', bg: '#fff7ed', accent: '#ea580c', text: '#1f2937' },
  { id: 'ocean', name: 'Ocean', bg: '#0c1929', accent: '#06b6d4', text: '#e0f2fe' },
  { id: 'forest', name: 'Forest', bg: '#052e16', accent: '#22c55e', text: '#dcfce7' },
  { id: 'purple', name: 'Royal', bg: '#1e1b4b', accent: '#8b5cf6', text: '#ede9fe' },
];

export default function Themes() {
  const [currentTheme, setCurrentTheme] = useState('dark');
  const [accentColor, setAccentColor] = useState('#f97316');

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900 text-slate-900 dark:text-white">
      <header className="bg-white dark:bg-slate-800 border-b p-4"><div className="flex items-center gap-4"><a href="/wallet" className="text-2xl">🐯</a><h1 className="text-xl font-bold">Theme Customization</h1></div></header>
      <div className="max-w-4xl mx-auto p-8">
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4 mb-8">{THEMES.map(t => <button key={t.id} onClick={() => setCurrentTheme(t.id)} className={`p-4 rounded-lg border-2 ${currentTheme === t.id ? 'border-orange-500' : 'border-slate-200 dark:border-slate-700'}`}><div className="h-20 rounded-lg mb-2 flex items-center justify-center" style={{ background: t.bg }}><span style={{ color: t.accent, fontSize: 24 }}>🐯</span></div><div className="font-semibold">{t.name}</div></button>)}</div>
        <div className="bg-white dark:bg-slate-800 rounded-lg p-6 mb-6">
          <h2 className="font-semibold mb-4">Accent Color</h2>
          <div className="flex gap-3 mb-6">{['#f97316', '#3b82f6', '#22c55e', '#8b5cf6', '#ec4899', '#eab308'].map(c => <button key={c} onClick={() => setAccentColor(c)} className={`w-10 h-10 rounded-full ${accentColor === c ? 'ring-2 ring-offset-2 ring-orange-500' : ''}`} style={{ background: c }}></button>)}
          </div>
        </div>
        <div className="bg-white dark:bg-slate-800 rounded-lg p-6 mb-6">
          <h2 className="font-semibold mb-4">Preview</h2>
          <div className="rounded-lg p-4" style={{ background: THEMES.find(t => t.id === currentTheme)?.bg }}>
            <div className="flex items-center gap-2 mb-4"><span style={{ color: THEMES.find(t => t.id === currentTheme)?.accent, fontSize: 24 }}>🐯</span><span className="font-bold" style={{ color: THEMES.find(t => t.id === currentTheme)?.text }}>TigerWallet</span></div>
            <div className="text-sm mb-2" style={{ color: THEMES.find(t => t.id === currentTheme)?.text }}>Balance: $12,450.00</div>
            <button className="px-4 py-2 rounded-lg text-white font-medium" style={{ background: accentColor }}>Send</button>
          </div>
        </div>
        <button className="w-full bg-orange-500 hover:bg-orange-600 text-white py-4 rounded-lg font-semibold">Apply Theme</button>
      </div>
    </div>
  );
}
