/**
 * TigerWallet Super Admin - Theme Context
 * Light/Dark theme switching for all pages
 */

import React, { createContext, useContext, useEffect, useState, useCallback } from 'react';

type Theme = 'light' | 'dark' | 'system';

interface ThemeContextType {
  theme: Theme;
  setTheme: (theme: Theme) => void;
  resolvedTheme: 'light' | 'dark';
  toggleTheme: () => void;
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setTheme] = useState<Theme>(() => {
    if (typeof window !== 'undefined') {
      const saved = localStorage.getItem('super_admin_theme') as Theme;
      if (saved) return saved;
      return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    }
    return 'system';
  });

  const [resolvedTheme, setResolvedTheme] = useState<'light' | 'dark'>(() => {
    if (typeof window !== 'undefined') {
      if (theme === 'system') {
        return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
      }
      return theme;
    }
    return 'light';
  });

  useEffect(() => {
    const root = document.documentElement;
    
    const updateTheme = (newTheme: 'light' | 'dark') => {
      root.setAttribute('data-theme', newTheme);
      root.classList.remove('light', 'dark');
      root.classList.add(newTheme);
      
      // Set CSS variables for the theme
      if (newTheme === 'dark') {
        root.style.setProperty('--bg-primary', '#0f172a');
        root.style.setProperty('--bg-secondary', '#1e293b');
        root.style.setProperty('--bg-tertiary', '#334155');
        root.style.setProperty('--text-primary', '#f1f5f9');
        root.style.setProperty('--text-secondary', '#94a3b8');
        root.style.setProperty('--text-tertiary', '#64748b');
        root.style.setProperty('--border-primary', '#334155');
        root.style.setProperty('--border-secondary', '#475569');
        root.style.setProperty('--accent-primary', '#3b82f6');
        root.style.setProperty('--accent-secondary', '#60a5fa');
        root.style.setProperty('--success', '#22c55e');
        root.style.setProperty('--warning', '#f59e0b');
        root.style.setProperty('--error', '#ef4444');
        root.style.setProperty('--info', '#06b6d4');
      } else {
        root.style.setProperty('--bg-primary', '#ffffff');
        root.style.setProperty('--bg-secondary', '#f8fafc');
        root.style.setProperty('--bg-tertiary', '#f1f5f9');
        root.style.setProperty('--text-primary', '#0f172a');
        root.style.setProperty('--text-secondary', '#475569');
        root.style.setProperty('--text-tertiary', '#64748b');
        root.style.setProperty('--border-primary', '#e2e8f0');
        root.style.setProperty('--border-secondary', '#cbd5e1');
        root.style.setProperty('--accent-primary', '#2563eb');
        root.style.setProperty('--accent-secondary', '#3b82f6');
        root.style.setProperty('--success', '#16a34a');
        root.style.setProperty('--warning', '#d97706');
        root.style.setProperty('--error', '#dc2626');
        root.style.setProperty('--info', '#0891b2');
      }
    };

    if (theme === 'system') {
      const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
      const handleChange = (e: MediaQueryListEvent) => {
        const newTheme = e.matches ? 'dark' : 'light';
        setResolvedTheme(newTheme);
        updateTheme(newTheme);
      };
      mediaQuery.addEventListener('change', handleChange);
      updateTheme(mediaQuery.matches ? 'dark' : 'light');
      return () => mediaQuery.removeEventListener('change', handleChange);
    } else {
      setResolvedTheme(theme);
      updateTheme(theme);
    }
  }, [theme]);

  useEffect(() => {
    if (typeof window !== 'undefined') {
      localStorage.setItem('super_admin_theme', theme);
    }
  }, [theme]);

  const toggleTheme = useCallback(() => {
    setTheme((current) => {
      if (current === 'light') return 'dark';
      if (current === 'dark') return 'system';
      return 'light';
    });
  }, []);

  return (
    <ThemeContext.Provider value={{ theme, setTheme, resolvedTheme, toggleTheme }}>
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
