'use client';

import React, { useState } from 'react';

const LANGUAGES = [
  { code: 'en', name: 'English', native: 'English', flag: '🇺🇸' },
  { code: 'es', name: 'Spanish', native: 'Español', flag: '🇪🇸' },
  { code: 'fr', name: 'French', native: 'Français', flag: '🇫🇷' },
  { code: 'de', name: 'German', native: 'Deutsch', flag: '🇩🇪' },
  { code: 'it', name: 'Italian', native: 'Italiano', flag: '🇮🇹' },
  { code: 'pt', name: 'Portuguese', native: 'Português', flag: '🇧🇷' },
  { code: 'ru', name: 'Russian', native: 'Русский', flag: '🇷🇺' },
  { code: 'zh', name: 'Chinese', native: '中文', flag: '🇨🇳' },
  { code: 'ja', name: 'Japanese', native: '日本語', flag: '🇯🇵' },
  { code: 'ko', name: 'Korean', native: '한국어', flag: '🇰🇷' },
  { code: 'ar', name: 'Arabic', native: 'العربية', flag: '🇸🇦' },
  { code: 'hi', name: 'Hindi', native: 'हिन्दी', flag: '🇮🇳' },
  { code: 'tr', name: 'Turkish', native: 'Türkçe', flag: '🇹🇷' },
  { code: 'vi', name: 'Vietnamese', native: 'Tiếng Việt', flag: '🇻🇳' },
  { code: 'th', name: 'Thai', native: 'ไทย', flag: '🇹🇭' },
  { code: 'id', name: 'Indonesian', native: 'Indonesia', flag: '🇮🇩' },
  { code: 'ms', name: 'Malay', native: 'Melayu', flag: '🇲🇾' },
  { code: 'pl', name: 'Polish', native: 'Polski', flag: '🇵🇱' },
  { code: 'nl', name: 'Dutch', native: 'Nederlands', flag: '🇳🇱' },
  { code: 'uk', name: 'Ukrainian', native: 'Українська', flag: '🇺🇦' },
];

export default function I18n() {
  const [currentLang, setCurrentLang] = useState('en');
  const [search, setSearch] = useState('');

  const filteredLanguages = search ? LANGUAGES.filter(l => l.name.toLowerCase().includes(search.toLowerCase()) || l.native.includes(search)) : LANGUAGES;

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900 text-slate-900 dark:text-white">
      <header className="bg-white dark:bg-slate-800 border-b p-4"><div className="flex items-center gap-4"><a href="/" className="text-2xl">🐯</a><h1 className="text-xl font-bold">Language Settings</h1></div></header>
      <div className="max-w-3xl mx-auto p-8">
        <div className="bg-white dark:bg-slate-800 rounded-lg p-6 mb-6">
          <div className="text-sm text-slate-500 mb-2">Current Language</div>
          <div className="flex items-center gap-3 p-3 bg-slate-100 dark:bg-slate-700 rounded-lg">
            <span className="text-2xl">{LANGUAGES.find(l => l.code === currentLang)?.flag}</span>
            <span className="font-semibold">{LANGUAGES.find(l => l.code === currentLang)?.native}</span>
          </div>
        </div>
        <div className="mb-6"><input type="text" placeholder="Search languages..." value={search} onChange={(e) => setSearch(e.target.value)} className="w-full bg-white dark:bg-slate-800 border rounded-lg px-4 py-3" /></div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          {filteredLanguages.map(lang => (
            <button key={lang.code} onClick={() => setCurrentLang(lang.code)} className={`flex items-center gap-3 p-4 rounded-lg border-2 ${currentLang === lang.code ? 'border-orange-500 bg-orange-50 dark:bg-orange-900/20' : 'border-transparent bg-white dark:bg-slate-800 hover:border-slate-300'}`}>
              <span className="text-2xl">{lang.flag}</span>
              <div className="text-left"><div className="font-semibold">{lang.native}</div><div className="text-xs text-slate-500">{lang.name}</div></div>
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}
