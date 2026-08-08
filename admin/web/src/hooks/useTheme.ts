/**
 * TigerWallet Admin - Theme Hook
 * Dark/Light theme switching for all components
 */

import { useState, useEffect, useCallback } from 'react';

type Theme = 'light' | 'dark';

export const useTheme = () => {
  const [theme, setTheme] = useState<Theme>(() => {
    const saved = localStorage.getItem('admin_theme');
    return (saved as Theme) || 'light';
  });

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('admin_theme', theme);
  }, [theme]);

  const toggleTheme = useCallback(() => {
    setTheme((prev) => (prev === 'light' ? 'dark' : 'light'));
  }, []);

  const setThemeMode = useCallback((mode: Theme) => {
    setTheme(mode);
  }, []);

  return {
    theme,
    isDark: theme === 'dark',
    isLight: theme === 'light',
    toggleTheme,
    setThemeMode,
  };
};

export default useTheme;
