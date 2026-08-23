/**
 * TigerWallet Admin - Theme Hook
 * Dark/Light theme switching for all components.
 *
 * Delegates to the shared ThemeContext so every page and the header toggle
 * stay in sync (single source of truth, single localStorage key).
 */

import { useTheme as useThemeContext } from '../contexts/ThemeContext';

type Theme = 'light' | 'dark';

export const useTheme = () => {
  const { resolvedTheme, toggleTheme, setTheme } = useThemeContext();

  return {
    theme: resolvedTheme as Theme,
    isDark: resolvedTheme === 'dark',
    isLight: resolvedTheme === 'light',
    toggleTheme,
    setThemeMode: (mode: Theme) => setTheme(mode),
  };
};

export default useTheme;
