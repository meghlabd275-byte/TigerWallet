// Theme Store - Light/Dark Theme Management
// Works on every page throughout the admin panel
// Consolidated with shared theme provider - no duplicates

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
const THEME_KEY = 'tigerwallet_admin_theme';

export const ThemeProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
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
};

export const useTheme = () => {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
};

export default ThemeContext;
