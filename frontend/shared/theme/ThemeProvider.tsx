/**
 * TigerWallet Theme System
 * Complete Light/Dark theme with system preference support
 * Works across all pages and components
 * 
 * Features:
 * - System preference detection
 * - Manual toggle
 * - Persistence in localStorage
 * - CSS variables for easy theming
 * - Smooth transitions
 * - Component-specific overrides
 */

import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';

// ============================================================================
// Theme Types
// ============================================================================

export type ThemeMode = 'light' | 'dark' | 'system';
export type ThemeColors = {
  primary: string;
  primaryHover: string;
  secondary: string;
  accent: string;
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

export type Theme = {
  mode: ThemeMode;
  colors: ThemeColors;
  isDark: boolean;
};

// ============================================================================
// Theme Colors - Light
// ============================================================================

const lightColors: ThemeColors = {
  primary: '#FF6B35',
  primaryHover: '#E55A2B',
  secondary: '#1A1A2E',
  accent: '#00D9FF',
  background: '#FFFFFF',
  surface: '#F8F9FA',
  surfaceHover: '#F0F1F2',
  text: '#1A1A2E',
  textSecondary: '#6B7280',
  border: '#E5E7EB',
  success: '#10B981',
  warning: '#F59E0B',
  error: '#EF4444',
  info: '#3B82F6',
};

// ============================================================================
// Theme Colors - Dark
// ============================================================================

const darkColors: ThemeColors = {
  primary: '#FF6B35',
  primaryHover: '#FF8A5C',
  secondary: '#F8F9FA',
  accent: '#00D9FF',
  background: '#0D0D12',
  surface: '#16161D',
  surfaceHover: '#1E1E28',
  text: '#F8F9FA',
  textSecondary: '#9CA3AF',
  border: '#2D2D3A',
  success: '#34D399',
  warning: '#FBBF24',
  error: '#F87171',
  info: '#60A5FA',
};

// ============================================================================
// CSS Variables Injection
// ============================================================================

const injectCSSVariables = (colors: ThemeColors, isDark: boolean) => {
  if (typeof document === 'undefined') return;
  
  const root = document.documentElement;
  
  // Set CSS custom properties
  root.style.setProperty('--color-primary', colors.primary);
  root.style.setProperty('--color-primary-hover', colors.primaryHover);
  root.style.setProperty('--color-secondary', colors.secondary);
  root.style.setProperty('--color-accent', colors.accent);
  root.style.setProperty('--color-background', colors.background);
  root.style.setProperty('--color-surface', colors.surface);
  root.style.setProperty('--color-surface-hover', colors.surfaceHover);
  root.style.setProperty('--color-text', colors.text);
  root.style.setProperty('--color-text-secondary', colors.textSecondary);
  root.style.setProperty('--color-border', colors.border);
  root.style.setProperty('--color-success', colors.success);
  root.style.setProperty('--color-warning', colors.warning);
  root.style.setProperty('--color-error', colors.error);
  root.style.setProperty('--color-info', colors.info);
  
  // Set data attribute for styling
  root.setAttribute('data-theme', isDark ? 'dark' : 'light');
  
  // Add transition for smooth theme switching
  root.style.setProperty('--transition-duration', '0.3s');
};

// ============================================================================
// Theme Context
// ============================================================================

interface ThemeContextType {
  theme: Theme;
  setThemeMode: (mode: ThemeMode) => void;
  toggleTheme: () => void;
  isDark: boolean;
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

// ============================================================================
// Local Storage Keys
// ============================================================================

const THEME_STORAGE_KEY = 'tigerwallet-theme-mode';

// ============================================================================
// Theme Provider Component
// ============================================================================

interface ThemeProviderProps {
  children: React.ReactNode;
  defaultMode?: ThemeMode;
}

export const ThemeProvider: React.FC<ThemeProviderProps> = ({ 
  children, 
  defaultMode = 'system' 
}) => {
  const [themeMode, setThemeModeState] = useState<ThemeMode>(() => {
    if (typeof window !== 'undefined') {
      const stored = localStorage.getItem(THEME_STORAGE_KEY);
      if (stored && ['light', 'dark', 'system'].includes(stored)) {
        return stored as ThemeMode;
      }
    }
    return defaultMode;
  });

  const [systemPreference, setSystemPreference] = useState<'light' | 'dark'>(() => {
    if (typeof window !== 'undefined') {
      return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    }
    return 'light';
  });

  const isDark = themeMode === 'system' ? systemPreference === 'dark' : themeMode === 'dark';
  const colors = isDark ? darkColors : lightColors;

  const theme: Theme = {
    mode: themeMode,
    colors,
    isDark,
  };

  useEffect(() => {
    injectCSSVariables(colors, isDark);
    
    if (typeof window !== 'undefined') {
      const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
      const handleChange = (e: MediaQueryListEvent) => {
        setSystemPreference(e.matches ? 'dark' : 'light');
      };
      
      mediaQuery.addEventListener('change', handleChange);
      return () => mediaQuery.removeEventListener('change', handleChange);
    }
  }, [colors, isDark]);

  const setThemeMode = useCallback((mode: ThemeMode) => {
    setThemeModeState(mode);
    if (typeof window !== 'undefined') {
      localStorage.setItem(THEME_STORAGE_KEY, mode);
    }
  }, []);

  const toggleTheme = useCallback(() => {
    const newMode = isDark ? 'light' : 'dark';
    setThemeMode(newMode);
  }, [isDark, setThemeMode]);

  return (
    <ThemeContext.Provider value={{ theme, setThemeMode, toggleTheme, isDark }}>
      {children}
    </ThemeContext.Provider>
  );
};

// ============================================================================
// Hook for using theme
// ============================================================================

export const useTheme = (): ThemeContextType => {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
};

// ============================================================================
// Theme Toggle Button Component
// ============================================================================

interface ThemeToggleProps {
  size?: 'sm' | 'md' | 'lg';
  showLabel?: boolean;
  className?: string;
}

export const ThemeToggle: React.FC<ThemeToggleProps> = ({ 
  size = 'md', 
  showLabel = false,
  className = '' 
}) => {
  const { toggleTheme, isDark } = useTheme();

  const sizeClasses = {
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
    <button
      onClick={toggleTheme}
      className={`${sizeClasses[size]} flex items-center justify-center rounded-full bg-[var(--color-surface)] hover:bg-[var(--color-surface-hover)] border border-[var(--color-border)] transition-all duration-300 ${className}`}
      aria-label={isDark ? 'Switch to light mode' : 'Switch to dark mode'}
    >
      <svg className={`${iconSizes[size]} text-[var(--color-text)] transition-transform duration-300 ${isDark ? 'rotate-90 scale-0 opacity-0' : 'rotate-0 scale-100 opacity-100'} absolute`} fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" />
      </svg>
      <svg className={`${iconSizes[size]} text-[var(--color-text)] transition-transform duration-300 ${isDark ? 'rotate-0 scale-100 opacity-100' : '-rotate-90 scale-0 opacity-0'}`} fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />
      </svg>
      {showLabel && <span className="ml-2 text-[var(--color-text)] text-sm">{isDark ? 'Dark' : 'Light'}</span>}
    </button>
  );
};

// ============================================================================
// Styled Component Helpers
// ============================================================================

interface ThemedButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'outline' | 'ghost';
  size?: 'sm' | 'md' | 'lg';
}

export const ThemedButton: React.FC<ThemedButtonProps> = ({
  variant = 'primary',
  size = 'md',
  className = '',
  children,
  ...props
}) => {
  const variantClasses = {
    primary: 'bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white',
    secondary: 'bg-[var(--color-secondary)] text-white',
    outline: 'border-2 border-[var(--color-primary)] text-[var(--color-primary)] hover:bg-[var(--color-primary)] hover:text-white',
    ghost: 'bg-transparent hover:bg-[var(--color-surface)] text-[var(--color-text)]',
  };

  const sizeClasses = {
    sm: 'px-3 py-1.5 text-sm',
    md: 'px-4 py-2 text-base',
    lg: 'px-6 py-3 text-lg',
  };

  return (
    <button className={`${variantClasses[variant]} ${sizeClasses[size]} rounded-lg font-medium transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed ${className}`} {...props}>
      {children}
    </button>
  );
};

interface ThemedCardProps {
  children: React.ReactNode;
  className?: string;
  hover?: boolean;
}

export const ThemedCard: React.FC<ThemedCardProps> = ({ children, className = '', hover = false }) => (
  <div className={`bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl transition-all duration-300 ${hover ? 'hover:shadow-lg hover:border-[var(--color-primary)]' : ''} ${className}`}>
    {children}
  </div>
);

interface ThemedInputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
}

export const ThemedInput: React.FC<ThemedInputProps> = ({ label, error, className = '', ...props }) => (
  <div className="w-full">
    {label && <label className="block text-sm font-medium text-[var(--color-text-secondary)] mb-1">{label}</label>}
    <input className={`w-full px-4 py-2 bg-[var(--color-background)] border border-[var(--color-border)] rounded-lg text-[var(--color-text)] placeholder-[var(--color-text-secondary)] focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] focus:border-transparent transition-all duration-300 ${error ? 'border-[var(--color-error)]' : ''} ${className}`} {...props} />
    {error && <p className="mt-1 text-sm text-[var(--color-error)]">{error}</p>}
  </div>
);

interface BackgroundProps {
  children: React.ReactNode;
  className?: string;
}

export const ThemedBackground: React.FC<BackgroundProps> = ({ children, className = '' }) => (
  <div className={`min-h-screen bg-[var(--color-background)] text-[var(--color-text)] transition-colors duration-300 ${className}`}>
    {children}
  </div>
);

// ============================================================================
// Export
// ============================================================================

export default {
  ThemeProvider,
  useTheme,
  ThemeToggle,
  ThemedButton,
  ThemedCard,
  ThemedInput,
  ThemedBackground,
};
