/**
 * TigerWallet Admin Platform - Theme Context
 * Provides light/dark theme switching that works everywhere
 */

import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';

type Theme = 'light' | 'dark';

interface ThemeContextType {
  theme: Theme;
  toggleTheme: () => void;
  setTheme: (theme: Theme) => void;
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

interface ThemeProviderProps {
  children: ReactNode;
}

export const ThemeProvider: React.FC<ThemeProviderProps> = ({ children }) => {
  const [theme, setThemeState] = useState<Theme>(() => {
    // Check localStorage first
    const stored = localStorage.getItem('tigerwallet_theme') as Theme;
    if (stored) return stored;
    
    // Check system preference
    if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
      return 'dark';
    }
    return 'light';
  });

  useEffect(() => {
    // Update document class and localStorage when theme changes
    const root = document.documentElement;
    
    if (theme === 'dark') {
      root.classList.add('dark');
      root.classList.remove('light');
    } else {
      root.classList.add('light');
      root.classList.remove('dark');
    }
    
    localStorage.setItem('tigerwallet_theme', theme);
    
    // Update meta theme-color
    const metaTheme = document.querySelector('meta[name="theme-color"]');
    if (metaTheme) {
      metaTheme.setAttribute('content', theme === 'dark' ? '#1a1a2e' : '#ffffff');
    }
  }, [theme]);

  // Listen for system theme changes
  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    
    const handleChange = (e: MediaQueryListEvent) => {
      const stored = localStorage.getItem('tigerwallet_theme');
      if (!stored) {
        setThemeState(e.matches ? 'dark' : 'light');
      }
    };
    
    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, []);

  const toggleTheme = () => {
    setThemeState(prev => prev === 'light' ? 'dark' : 'light');
  };

  const setTheme = (newTheme: Theme) => {
    setThemeState(newTheme);
  };

  return (
    <ThemeContext.Provider value={{ theme, toggleTheme, setTheme }}>
      {children}
    </ThemeContext.Provider>
  );
};

export const useTheme = (): ThemeContextType => {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
};

// Export theme colors for use in components
export const themeColors = {
  light: {
    background: '#ffffff',
    backgroundSecondary: '#f8fafc',
    backgroundTertiary: '#f1f5f9',
    surface: '#ffffff',
    border: '#e2e8f0',
    borderHover: '#cbd5e1',
    text: '#0f172a',
    textSecondary: '#475569',
    textMuted: '#94a3b8',
    primary: '#0ea5e9',
    primaryHover: '#0284c7',
    primaryLight: '#e0f2fe',
    secondary: '#8b5cf6',
    secondaryHover: '#7c3aed',
    success: '#22c55e',
    successLight: '#dcfce7',
    warning: '#f59e0b',
    warningLight: '#fef3c7',
    error: '#ef4444',
    errorLight: '#fee2e2',
    info: '#3b82f6',
    infoLight: '#dbeafe',
  },
  dark: {
    background: '#0f0f23',
    backgroundSecondary: '#1a1a2e',
    backgroundTertiary: '#16213e',
    surface: '#1a1a2e',
    border: '#2d3748',
    borderHover: '#4a5568',
    text: '#f1f5f9',
    textSecondary: '#cbd5e1',
    textMuted: '#94a3b8',
    primary: '#38bdf8',
    primaryHover: '#0ea5e9',
    primaryLight: '#1e3a5f',
    secondary: '#a78bfa',
    secondaryHover: '#8b5cf6',
    success: '#4ade80',
    successLight: '#14532d',
    warning: '#fbbf24',
    warningLight: '#451a03',
    error: '#f87171',
    errorLight: '#450a0a',
    info: '#60a5fa',
    infoLight: '#1e3a5f',
  },
};

export type ThemeColors = typeof themeColors.light;
