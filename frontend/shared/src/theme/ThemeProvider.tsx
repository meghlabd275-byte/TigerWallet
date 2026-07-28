/**
 * TigerWallet Theme Provider
 * Production-ready light/dark theme with system preference support
 * Works across all pages and components
 */

import React, { createContext, useContext, useState, useEffect, useCallback, ReactNode } from 'react';

// ============================================================================
// THEME TYPES
// ============================================================================

export type ThemeMode = 'light' | 'dark' | 'system';
export type ThemeColor = 'default' | 'blue' | 'green' | 'purple' | 'orange' | 'red';

export interface ThemeColors {
  primary: string;
  primaryLight: string;
  primaryDark: string;
  secondary: string;
  secondaryLight: string;
  secondaryDark: string;
  background: string;
  backgroundPaper: string;
  backgroundDefault: string;
  text: string;
  textSecondary: string;
  textDisabled: string;
  divider: string;
  error: string;
  warning: string;
  success: string;
  info: string;
}

export interface Theme {
  mode: ThemeMode;
  color: ThemeColor;
  colors: ThemeColors;
  isDark: boolean;
  spacing: number;
  borderRadius: number;
  fontFamily: string;
}

export interface ThemeContextValue {
  theme: Theme;
  setThemeMode: (mode: ThemeMode) => void;
  setThemeColor: (color: ThemeColor) => void;
  toggleTheme: () => void;
  getEffectiveTheme: () => 'light' | 'dark';
}

// ============================================================================
// THEME COLORS
// ============================================================================

const lightColors: ThemeColors = {
  primary: '#1976d2',
  primaryLight: '#42a5f5',
  primaryDark: '#1565c0',
  secondary: '#9c27b0',
  secondaryLight: '#ba68c8',
  secondaryDark: '#7b1fa2',
  background: '#f5f5f5',
  backgroundPaper: '#ffffff',
  backgroundDefault: '#ffffff',
  text: '#212121',
  textSecondary: '#757575',
  textDisabled: '#bdbdbd',
  divider: '#e0e0e0',
  error: '#d32f2f',
  warning: '#ed6c02',
  success: '#2e7d32',
  info: '#0288d1',
};

const darkColors: ThemeColors = {
  primary: '#90caf9',
  primaryLight: '#bbdefb',
  primaryDark: '#42a5f5',
  secondary: '#ce93d8',
  secondaryLight: '#e1bee7',
  secondaryDark: '#ab47bc',
  background: '#121212',
  backgroundPaper: '#1e1e1e',
  backgroundDefault: '#000000',
  text: '#ffffff',
  textSecondary: '#b0b0b0',
  textDisabled: '#666666',
  divider: '#2d2d2d',
  error: '#ef5350',
  warning: '#ffa726',
  success: '#66bb6a',
  info: '#29b6f6',
};

const colorPalettes: Record<ThemeColor, Partial<ThemeColors>> = {
  default: {},
  blue: {
    primary: '#1976d2',
    primaryLight: '#42a5f5',
    primaryDark: '#1565c0',
  },
  green: {
    primary: '#2e7d32',
    primaryLight: '#4caf50',
    primaryDark: '#1b5e20',
  },
  purple: {
    primary: '#7b1fa2',
    primaryLight: '#9c27b0',
    primaryDark: '#6a1b9a',
  },
  orange: {
    primary: '#f57c00',
    primaryLight: '#ff9800',
    primaryDark: '#e65100',
  },
  red: {
    primary: '#c62828',
    primaryLight: '#ef5350',
    primaryDark: '#b71c1c',
  },
};

// ============================================================================
// THEME STORAGE
// ============================================================================

const THEME_MODE_KEY = 'tigerwallet_theme_mode';
const THEME_COLOR_KEY = 'tigerwallet_theme_color';

const getStoredThemeMode = (): ThemeMode => {
  if (typeof window === 'undefined') return 'system';
  const stored = localStorage.getItem(THEME_MODE_KEY);
  if (stored === 'light' || stored === 'dark' || stored === 'system') {
    return stored;
  }
  return 'system';
};

const getStoredThemeColor = (): ThemeColor => {
  if (typeof window === 'undefined') return 'default';
  const stored = localStorage.getItem(THEME_COLOR_KEY);
  if (stored === 'default' || stored === 'blue' || stored === 'green' || 
      stored === 'purple' || stored === 'orange' || stored === 'red') {
    return stored;
  }
  return 'default';
};

// ============================================================================
// THEME CONTEXT
// ============================================================================

const ThemeContext = createContext<ThemeContextValue | null>(null);

// ============================================================================
// THEME PROVIDER COMPONENT
// ============================================================================

interface ThemeProviderProps {
  children: ReactNode;
  defaultMode?: ThemeMode;
  defaultColor?: ThemeColor;
}

export const ThemeProvider: React.FC<ThemeProviderProps> = ({
  children,
  defaultMode = 'system',
  defaultColor = 'default',
}) => {
  const [themeMode, setThemeModeState] = useState<ThemeMode>(() => getStoredThemeMode());
  const [themeColor, setThemeColorState] = useState<ThemeColor>(() => getStoredThemeColor());
  const [systemPreference, setSystemPreference] = useState<'light' | 'dark'>('light');

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

  // Get effective theme
  const getEffectiveTheme = useCallback((): 'light' | 'dark' => {
    if (themeMode === 'system') {
      return systemPreference;
    }
    return themeMode;
  }, [themeMode, systemPreference]);

  // Get theme colors
  const getColors = useCallback((): ThemeColors => {
    const baseColors = getEffectiveTheme() === 'dark' ? darkColors : lightColors;
    const palette = colorPalettes[themeColor];
    return { ...baseColors, ...palette };
  }, [themeColor, getEffectiveTheme]);

  // Build theme object
  const theme: Theme = {
    mode: themeMode,
    color: themeColor,
    colors: getColors(),
    isDark: getEffectiveTheme() === 'dark',
    spacing: 8,
    borderRadius: 8,
    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif',
  };

  // Set theme mode
  const setThemeMode = useCallback((mode: ThemeMode) => {
    setThemeModeState(mode);
    localStorage.setItem(THEME_MODE_KEY, mode);
    
    // Apply to document
    if (typeof document !== 'undefined') {
      document.documentElement.setAttribute('data-theme', mode === 'system' ? systemPreference : mode);
      document.documentElement.classList.remove('light', 'dark');
      document.documentElement.classList.add(mode === 'system' ? systemPreference : mode);
    }
  }, [systemPreference]);

  // Set theme color
  const setThemeColor = useCallback((color: ThemeColor) => {
    setThemeColorState(color);
    localStorage.setItem(THEME_COLOR_KEY, color);
    
    // Apply color to CSS variables
    if (typeof document !== 'undefined') {
      const root = document.documentElement;
      const palette = colorPalettes[color];
      Object.entries(palette).forEach(([key, value]) => {
        root.style.setProperty(`--${key}`, value);
      });
    }
  }, []);

  // Toggle theme
  const toggleTheme = useCallback(() => {
    const newMode = getEffectiveTheme() === 'dark' ? 'light' : 'dark';
    setThemeMode(newMode);
  }, [getEffectiveTheme, setThemeMode]);

  // Apply theme on mount
  useEffect(() => {
    if (typeof document === 'undefined') return;
    
    const effectiveTheme = getEffectiveTheme();
    document.documentElement.setAttribute('data-theme', effectiveTheme);
    document.documentElement.classList.add(effectiveTheme);
    
    // Set CSS variables
    const colors = getColors();
    const root = document.documentElement;
    Object.entries(colors).forEach(([key, value]) => {
      root.style.setProperty(`--color-${key}`, value);
    });
  }, [themeMode, themeColor, getEffectiveTheme, getColors]);

  const contextValue: ThemeContextValue = {
    theme,
    setThemeMode,
    setThemeColor,
    toggleTheme,
    getEffectiveTheme,
  };

  return (
    <ThemeContext.Provider value={contextValue}>
      {children}
    </ThemeContext.Provider>
  );
};

// ============================================================================
// HOOK
// ============================================================================

export const useTheme = (): ThemeContextValue => {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
};

// ============================================================================
// THEME STYLES
// ============================================================================

export const getThemeStyles = (theme: Theme): string => {
  const colors = theme.colors;
  
  return `
    :root {
      --color-primary: ${colors.primary};
      --color-primary-light: ${colors.primaryLight};
      --color-primary-dark: ${colors.primaryDark};
      --color-secondary: ${colors.secondary};
      --color-secondary-light: ${colors.secondaryLight};
      --color-secondary-dark: ${colors.secondaryDark};
      --color-background: ${colors.background};
      --color-background-paper: ${colors.backgroundPaper};
      --color-background-default: ${colors.backgroundDefault};
      --color-text: ${colors.text};
      --color-text-secondary: ${colors.textSecondary};
      --color-text-disabled: ${colors.textDisabled};
      --color-divider: ${colors.divider};
      --color-error: ${colors.error};
      --color-warning: ${colors.warning};
      --color-success: ${colors.success};
      --color-info: ${colors.info};
      
      --spacing-unit: ${theme.spacing}px;
      --border-radius: ${theme.borderRadius}px;
      --font-family: ${theme.fontFamily};
    }
    
    .light {
      --color-primary: ${colors.primary};
      --color-background: ${colors.background};
      --color-background-paper: ${colors.backgroundPaper};
      --color-text: ${colors.text};
    }
    
    .dark {
      --color-primary: ${colors.primary};
      --color-background: ${colors.background};
      --color-background-paper: ${colors.backgroundPaper};
      --color-text: ${colors.text};
    }
  `;
};

// ============================================================================
// EXPORTS
// ============================================================================

export default ThemeProvider;
export { lightColors, darkColors, colorPalettes };
