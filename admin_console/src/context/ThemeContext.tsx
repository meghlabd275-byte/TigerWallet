/**
 * TigerWallet Theme Context - Admin Console
 */

import React, { createContext, useContext, useEffect, useState } from 'react';

type Theme = 'light' | 'dark';
interface ThemeContextType { theme: Theme; toggleTheme: () => void; setTheme: (t: Theme) => void; }

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setThemeState] = useState<Theme>('light');

  useEffect(() => {
    const stored = localStorage.getItem('tigerwallet_theme') as Theme;
    if (stored) { setThemeState(stored); document.documentElement.setAttribute('data-theme', stored); }
    else { const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches; const t = prefersDark ? 'dark' : 'light'; setThemeState(t); document.documentElement.setAttribute('data-theme', t); }
  }, []);

  const toggleTheme = () => {
    const t = theme === 'light' ? 'dark' : 'light';
    setThemeState(t); localStorage.setItem('tigerwallet_theme', t); document.documentElement.setAttribute('data-theme', t);
  };

  const setTheme = (t: Theme) => { setThemeState(t); localStorage.setItem('tigerwallet_theme', t); document.documentElement.setAttribute('data-theme', t); };

  return <ThemeContext.Provider value={{ theme, toggleTheme, setTheme }}>{children}</ThemeContext.Provider>;
}

export function useTheme() { const c = useContext(ThemeContext); if (!c) throw new Error('useTheme must be used within ThemeProvider'); return c; }
export default ThemeContext;
