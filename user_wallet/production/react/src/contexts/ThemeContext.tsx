/**
 * Theme Context - Dark/Light Mode Support
 * Works everywhere across all pages
 * Consolidated theme - uses same storage keys as other TigerWallet apps
 */

import React, { createContext, useContext, useState, useEffect, useCallback, ReactNode } from 'react';

type ThemeMode = 'light' | 'dark' | 'system';

interface ThemeContextType {
  theme: 'light' | 'dark';
  themeMode: ThemeMode;
  toggleTheme: () => void;
  setTheme: (theme: 'light' | 'dark') => void;
  setThemeMode: (mode: ThemeMode) => void;
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

const THEME_MODE_KEY = 'tigerwallet_theme_mode';
const THEME_KEY = 'tigerwallet_user_theme';

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [themeMode, setThemeModeState] = useState<ThemeMode>(() => {
    const stored = localStorage.getItem(THEME_MODE_KEY);
    return (stored as ThemeMode) || 'system';
  });
  
  const [systemPreference, setSystemPreference] = useState<'light' | 'dark'>('dark');
  const [theme, setThemeState] = useState<'light' | 'dark'>('dark');

  // Get system preference
  useEffect(() => {
    if (typeof window === 'undefined') return;
    
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    setSystemPreference(mediaQuery.matches ? 'dark' : 'light');

    const handler = (e: MediaQueryListEvent) => {
      setSystemPreference(e.matches ? 'dark' : 'light');
    };

    mediaQuery.addEventListener('change', handler);
    return () => mediaQuery.removeEventListener('change', handler);
  }, []);

  // Calculate effective theme
  useEffect(() => {
    let effectiveTheme: 'light' | 'dark';
    if (themeMode === 'system') {
      effectiveTheme = systemPreference;
    } else {
      effectiveTheme = themeMode;
    }
    setThemeState(effectiveTheme);
    
    // Apply to document
    document.documentElement.setAttribute('data-theme', effectiveTheme);
    document.documentElement.classList.remove('light', 'dark');
    document.documentElement.classList.add(effectiveTheme);
    localStorage.setItem(THEME_KEY, effectiveTheme);
  }, [themeMode, systemPreference]);

  const setThemeMode = useCallback((mode: ThemeMode) => {
    setThemeModeState(mode);
    localStorage.setItem(THEME_MODE_KEY, mode);
  }, []);

  const toggleTheme = useCallback(() => {
    const newTheme = theme === 'dark' ? 'light' : 'dark';
    setThemeModeState(newTheme);
    localStorage.setItem(THEME_MODE_KEY, newTheme);
  }, [theme]);

  const setTheme = useCallback((newTheme: 'light' | 'dark') => {
    setThemeModeState(newTheme);
    localStorage.setItem(THEME_MODE_KEY, newTheme);
  }, []);

  return (
    <ThemeContext.Provider value={{ theme, themeMode, toggleTheme, setTheme, setThemeMode }}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme() {
  const context = useContext(ThemeContext);
  if (context === undefined) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
}

export default ThemeContext;
