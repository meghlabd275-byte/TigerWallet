'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../components/ThemeProvider';

interface ApiResponse<T> {
  success: boolean;
  data: T;
  error?: string;
}

const API_BASE_URL = typeof window !== 'undefined' ? '' : (process.env.BACKEND_URL || 'http://localhost:8443');

const fetchAPI = async <T,>(endpoint: string, options?: RequestInit): Promise<T> => {
  const token = typeof window !== 'undefined' ? localStorage.getItem('tigerwallet-token') : null;
  const response = await fetch(`${API_BASE_URL}/api/v1${endpoint}`, {
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
  const { isDark } = useTheme();
  const [currentLang, setCurrentLang] = useState('en');
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(false);
  const [saved, setSaved] = useState(false);

  // Load user preferences on mount
  useEffect(() => {
    const loadPreferences = async () => {
      try {
        const prefs = await fetchAPI<{ language: string }>('/user/preferences');
        if (prefs?.language) {
          setCurrentLang(prefs.language);
        }
      } catch (err) {
        // Use localStorage fallback
        const saved = localStorage.getItem('tigerwallet-language');
        if (saved) setCurrentLang(saved);
      }
    };
    loadPreferences();
  }, []);

  const handleLanguageChange = async (langCode: string) => {
    setLoading(true);
    setSaved(false);

    try {
      // Save to backend
      await fetchAPI('/user/preferences', {
        method: 'PUT',
        body: JSON.stringify({ language: langCode }),
      });
    } catch (err) {
      // Fallback to localStorage
      localStorage.setItem('tigerwallet-language', langCode);
    }

    setCurrentLang(langCode);
    setLoading(false);
    setSaved(true);
    setTimeout(() => setSaved(false), 2000);
  };

  const filteredLanguages = search 
    ? LANGUAGES.filter(l => l.name.toLowerCase().includes(search.toLowerCase()) || l.native.includes(search)) 
    : LANGUAGES;

  return (
    <div className={`min-h-screen ${isDark ? 'bg-slate-900 text-white' : 'bg-slate-50 text-slate-900'}`}>
      <header className={`${isDark ? 'bg-slate-800' : 'bg-white'} border-b p-4`}><div className="flex items-center gap-4"><a href="/" className="text-2xl">🐯</a><h1 className="text-xl font-bold">Language Settings</h1></div></header>
      <div className="max-w-3xl mx-auto p-8">
        <div className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-lg p-6 mb-6`}>
          <div className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'} mb-2`}>Current Language</div>
          <div className={`flex items-center gap-3 p-3 ${isDark ? 'bg-slate-700' : 'bg-slate-100'} rounded-lg`}>
            <span className="text-2xl">{LANGUAGES.find(l => l.code === currentLang)?.flag}</span>
            <span className="font-semibold">{LANGUAGES.find(l => l.code === currentLang)?.native}</span>
          </div>
        </div>
        {saved && (
          <div className={`${isDark ? 'bg-green-900/30 text-green-400' : 'bg-green-100 text-green-700'} px-4 py-2 rounded-lg mb-4`}>
            ✓ Language preference saved!
          </div>
        )}
        
        <div className="mb-6"><input type="text" placeholder="Search languages..." value={search} onChange={(e) => setSearch(e.target.value)} className={`w-full ${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} border rounded-lg px-4 py-3`} /></div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          {filteredLanguages.map(lang => (
            <button 
              key={lang.code} 
              onClick={() => handleLanguageChange(lang.code)}
              disabled={loading}
              className={`flex items-center gap-3 p-4 rounded-lg border-2 transition-all ${currentLang === lang.code ? `border-orange-500 ${isDark ? 'bg-orange-900/20' : 'bg-orange-50'}` : `${isDark ? 'border-slate-700 bg-slate-800' : 'border-transparent bg-white hover:border-slate-300'}`} ${loading ? 'opacity-50' : ''}`}
            >
              <span className="text-2xl">{lang.flag}</span>
              <div className="text-left"><div className="font-semibold">{lang.native}</div><div className={`text-xs ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>{lang.name}</div></div>
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}
