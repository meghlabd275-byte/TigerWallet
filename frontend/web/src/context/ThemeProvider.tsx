'use client';

import React, { createContext, useContext, useState, useEffect, useCallback, ReactNode } from 'react';

// ============================================================================
// THEME TYPES
// ============================================================================

export type Theme = 'light' | 'dark' | 'system';
export type ThemeColors = {
  primary: string;
  primaryHover: string;
  secondary: string;
  background: string;
  surface: string;
  surfaceHover: string;
  text: string;
  textSecondary: string;
  border: string;
  success: string;
  warning: string;
  error: string;
  info: string;
};

export interface ThemeContextType {
  theme: Theme;
  resolvedTheme: 'light' | 'dark';
  colors: ThemeColors;
  setTheme: (theme: Theme) => void;
  toggleTheme: () => void;
  isDark: boolean;
}

const darkColors: ThemeColors = {
  primary: '#3B82F6',
  primaryHover: '#2563EB',
  secondary: '#10B981',
  background: '#0F172A',
  surface: '#1E293B',
  surfaceHover: '#334155',
  text: '#F8FAFC',
  textSecondary: '#94A3B8',
  border: '#334155',
  success: '#22C55E',
  warning: '#F59E0B',
  error: '#EF4444',
  info: '#3B82F6',
};

const lightColors: ThemeColors = {
  primary: '#3B82F6',
  primaryHover: '#2563EB',
  secondary: '#10B981',
  background: '#F8FAFC',
  surface: '#FFFFFF',
  surfaceHover: '#F1F5F9',
  text: '#0F172A',
  textSecondary: '#64748B',
  border: '#E2E8F0',
  success: '#22C55E',
  warning: '#F59E0B',
  error: '#EF4444',
  info: '#3B82F6',
};

// ============================================================================
// THEME CONTEXT
// ============================================================================

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

// ============================================================================
// THEME STORAGE
// ============================================================================

const THEME_MODE_KEY = 'tigerwallet_theme_mode';

function getSystemTheme(): 'light' | 'dark' {
  if (typeof window === 'undefined') return 'dark';
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function getStoredTheme(): Theme {
  if (typeof window === 'undefined') return 'dark';
  const stored = localStorage.getItem(THEME_MODE_KEY);
  if (stored === 'light' || stored === 'dark' || stored === 'system') {
    return stored;
  }
  return 'dark';
}

function setStoredTheme(theme: Theme): void {
  if (typeof window === 'undefined') return;
  localStorage.setItem(THEME_MODE_KEY, theme);
}

// ============================================================================
// THEME PROVIDER COMPONENT
// ============================================================================

interface ThemeProviderProps {
  children: ReactNode;
  defaultTheme?: Theme;
  enableSystem?: boolean;
}

export function ThemeProvider({ 
  children, 
  defaultTheme = 'dark',
  enableSystem = true 
}: ThemeProviderProps) {
  const [theme, setThemeState] = useState<Theme>(defaultTheme);
  const [resolvedTheme, setResolvedTheme] = useState<'light' | 'dark'>('dark');
  const [colors, setColors] = useState<ThemeColors>(darkColors);
  const [mounted, setMounted] = useState(false);

  // Initialize theme on mount
  useEffect(() => {
    const storedTheme = getStoredTheme();
    setThemeState(storedTheme);
    setMounted(true);
  }, []);

  // Update resolved theme and colors when theme changes
  useEffect(() => {
    let resolved: 'light' | 'dark';

    if (theme === 'system' && enableSystem) {
      resolved = getSystemTheme();
    } else {
      resolved = theme === 'light' ? 'light' : 'dark';
    }

    setResolvedTheme(resolved);
    setColors(resolved === 'dark' ? darkColors : lightColors);

    // Apply to document
    if (typeof document !== 'undefined') {
      const root = document.documentElement;
      
      // Set data attribute for CSS selectors
      root.setAttribute('data-theme', resolved);
      root.setAttribute('data-color-mode', resolved);

      // Set CSS custom properties
      const colorSet = resolved === 'dark' ? darkColors : lightColors;
      Object.entries(colorSet).forEach(([key, value]) => {
        const cssKey = `--color-${key}`;
        root.style.setProperty(cssKey, value);
      });

      // Add/remove dark class for backward compatibility
      if (resolved === 'dark') {
        root.classList.add('dark');
        root.classList.remove('light');
      } else {
        root.classList.add('light');
        root.classList.remove('dark');
      }
    }
  }, [theme, enableSystem]);

  // Listen for system theme changes
  useEffect(() => {
    if (!enableSystem || theme !== 'system') return;

    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handler = (e: MediaQueryListEvent) => {
      const resolved = e.matches ? 'dark' : 'light';
      setResolvedTheme(resolved);
      setColors(resolved === 'dark' ? darkColors : lightColors);
    };

    mediaQuery.addEventListener('change', handler);
    return () => mediaQuery.removeEventListener('change', handler);
  }, [theme, enableSystem]);

  // Set theme function
  const setTheme = useCallback((newTheme: Theme) => {
    setThemeState(newTheme);
    setStoredTheme(newTheme);
  }, []);

  // Toggle theme function
  const toggleTheme = useCallback(() => {
    const newTheme = resolvedTheme === 'dark' ? 'light' : 'dark';
    setTheme(newTheme);
  }, [resolvedTheme, setTheme]);

  const value: ThemeContextType = {
    theme,
    resolvedTheme,
    colors,
    setTheme,
    toggleTheme,
    isDark: resolvedTheme === 'dark',
  };

  return (
    <ThemeContext.Provider value={value}>
      {children}
    </ThemeContext.Provider>
  );
}

// ============================================================================
// HOOKS
// ============================================================================

export function useTheme(): ThemeContextType {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
}

export function useColorMode() {
  const { resolvedTheme, setTheme, toggleTheme, isDark } = useTheme();
  return {
    colorMode: resolvedTheme,
    setColorMode: setTheme,
    toggleColorMode: toggleTheme,
    isDark,
  };
}

// ============================================================================
// THEME COLORS HOOK
// ============================================================================

export function useColors(): ThemeColors {
  const { colors } = useTheme();
  return colors;
}

// ============================================================================
// THEME CSS VARIABLES
// ============================================================================

export const themeCSSVariables = `
  :root,
  [data-theme="dark"] {
    --color-primary: #3B82F6;
    --color-primary-hover: #2563EB;
    --color-secondary: #10B981;
    --color-background: #0F172A;
    --color-surface: #1E293B;
    --color-surface-hover: #334155;
    --color-text: #F8FAFC;
    --color-text-secondary: #94A3B8;
    --color-border: #334155;
    --color-success: #22C55E;
    --color-warning: #F59E0B;
    --color-error: #EF4444;
    --color-info: #3B82F6;
  }

  [data-theme="light"] {
    --color-primary: #3B82F6;
    --color-primary-hover: #2563EB;
    --color-secondary: #10B981;
    --color-background: #F8FAFC;
    --color-surface: #FFFFFF;
    --color-surface-hover: #F1F5F9;
    --color-text: #0F172A;
    --color-text-secondary: #64748B;
    --color-border: #E2E8F0;
    --color-success: #22C55E;
    --color-warning: #F59E0B;
    --color-error: #EF4444;
    --color-info: #3B82F6;
  }
`;

// ============================================================================
// THEME SWITCHER COMPONENT
// ============================================================================

interface ThemeSwitcherProps {
  size?: 'sm' | 'md' | 'lg';
  showLabel?: boolean;
}

export function ThemeSwitcher({ size = 'md', showLabel = false }: ThemeSwitcherProps) {
  const { theme, setTheme, resolvedTheme, toggleTheme } = useTheme();

  const sizes = {
    sm: 'w-8 h-8',
    md: 'w-10 h-10',
    lg: 'w-12 h-12',
  };

  const iconSizes = {
    sm: 'w-4 h-4',
    md: 'w-5 h-5',
    lg: 'w-6 h-6',
  };

  return (
    <div className="flex items-center gap-2">
      {showLabel && (
        <span className="text-sm text-[var(--color-text-secondary)]">
          {resolvedTheme === 'dark' ? 'Dark' : 'Light'}
        </span>
      )}
      
      <button
        onClick={toggleTheme}
        className={`
          ${sizes[size]} 
          flex items-center justify-center 
          rounded-lg 
          bg-[var(--color-surface)] 
          border border-[var(--color-border)]
          hover:bg-[var(--color-surface-hover)]
          transition-all duration-200
          cursor-pointer
        `}
        aria-label={`Switch to ${resolvedTheme === 'dark' ? 'light' : 'dark'} mode`}
      >
        {resolvedTheme === 'dark' ? (
          <svg 
            className={`${iconSizes[size]} text-[var(--color-warning)]`} 
            fill="none" 
            viewBox="0 0 24 24" 
            stroke="currentColor"
          >
            <path 
              strokeLinecap="round" 
              strokeLinejoin="round" 
              strokeWidth={2} 
              d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" 
            />
          </svg>
        ) : (
          <svg 
            className={`${iconSizes[size]} text-[var(--color-primary)]`} 
            fill="none" 
            viewBox="0 0 24 24" 
            stroke="currentColor"
          >
            <path 
              strokeLinecap="round" 
              strokeLinejoin="round" 
              strokeWidth={2} 
              d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" 
            />
          </svg>
        )}
      </button>
    </div>
  );
}

// ============================================================================
// THEME PRESETS
// ============================================================================

export const themePresets = {
  default: {
    dark: darkColors,
    light: lightColors,
  },
  ocean: {
    dark: { ...darkColors, primary: '#0EA5E9', secondary: '#06B6D4' },
    light: { ...lightColors, primary: '#0284C7', secondary: '#0891B2' },
  },
  forest: {
    dark: { ...darkColors, primary: '#22C55E', secondary: '#14B8A6' },
    light: { ...lightColors, primary: '#16A34A', secondary: '#0D9488' },
  },
  sunset: {
    dark: { ...darkColors, primary: '#F97316', secondary: '#EF4444' },
    light: { ...lightColors, primary: '#EA580C', secondary: '#DC2626' },
  },
  purple: {
    dark: { ...darkColors, primary: '#A855F7', secondary: '#8B5CF6' },
    light: { ...lightColors, primary: '#9333EA', secondary: '#7C3AED' },
  },
};

export type ThemePreset = keyof typeof themePresets;

// ============================================================================
// EXPORTS
// ============================================================================

export default ThemeProvider;
