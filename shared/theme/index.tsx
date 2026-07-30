/**
 * TigerWallet Global Theme System
 * Universal dark/light theme that works across all pages
 */

import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';

// ============================================================================
// Theme Types
// ============================================================================

export type Theme = 'light' | 'dark';

export interface ThemeColors {
  // Background colors
  background: string;
  backgroundSecondary: string;
  backgroundTertiary: string;
  surface: string;
  surfaceHover: string;
  surfaceActive: string;
  
  // Text colors
  text: string;
  textSecondary: string;
  textTertiary: string;
  textInverse: string;
  
  // Border colors
  border: string;
  borderLight: string;
  borderFocus: string;
  
  // Accent colors
  primary: string;
  primaryHover: string;
  primaryActive: string;
  secondary: string;
  secondaryHover: string;
  
  // Status colors
  success: string;
  successBg: string;
  warning: string;
  warningBg: string;
  error: string;
  errorBg: string;
  info: string;
  infoBg: string;
  
  // Special
  glass: string;
  overlay: string;
}

export interface ThemeContextValue {
  theme: Theme;
  colors: ThemeColors;
  setTheme: (theme: Theme) => void;
  toggleTheme: () => void;
}

// ============================================================================
// Light Theme Colors
// ============================================================================

const lightColors: ThemeColors = {
  background: '#FFFFFF',
  backgroundSecondary: '#F8FAFC',
  backgroundTertiary: '#F1F5F9',
  surface: '#FFFFFF',
  surfaceHover: '#F1F5F9',
  surfaceActive: '#E2E8F0',
  
  text: '#0F172A',
  textSecondary: '#475569',
  textTertiary: '#94A3B8',
  textInverse: '#FFFFFF',
  
  border: '#E2E8F0',
  borderLight: '#F1F5F9',
  borderFocus: '#3B82F6',
  
  primary: '#3B82F6',
  primaryHover: '#2563EB',
  primaryActive: '#1D4ED8',
  secondary: '#6366F1',
  secondaryHover: '#4F46E5',
  
  success: '#10B981',
  successBg: '#D1FAE5',
  warning: '#F59E0B',
  warningBg: '#FEF3C7',
  error: '#EF4444',
  errorBg: '#FEE2E2',
  info: '#3B82F6',
  infoBg: '#DBEAFE',
  
  glass: 'rgba(255, 255, 255, 0.7)',
  overlay: 'rgba(0, 0, 0, 0.5)',
};

// ============================================================================
// Dark Theme Colors
// ============================================================================

const darkColors: ThemeColors = {
  background: '#0B0E14',
  backgroundSecondary: '#111827',
  backgroundTertiary: '#1F2937',
  surface: '#1F2937',
  surfaceHover: '#374151',
  surfaceActive: '#4B5563',
  
  text: '#F9FAFB',
  textSecondary: '#D1D5DB',
  textTertiary: '#9CA3AF',
  textInverse: '#0B0E14',
  
  border: '#374151',
  borderLight: '#4B5563',
  borderFocus: '#60A5FA',
  
  primary: '#3B82F6',
  primaryHover: '#60A5FA',
  primaryActive: '#2563EB',
  secondary: '#6366F1',
  secondaryHover: '#818CF8',
  
  success: '#10B981',
  successBg: '#064E3B',
  warning: '#F59E0B',
  warningBg: '#78350F',
  error: '#EF4444',
  errorBg: '#7F1D1D',
  info: '#3B82F6',
  infoBg: '#1E3A8A',
  
  glass: 'rgba(17, 24, 39, 0.8)',
  overlay: 'rgba(0, 0, 0, 0.7)',
};

// ============================================================================
// Theme Storage Keys
// ============================================================================

const THEME_STORAGE_KEY = 'tiger_wallet_theme';

// ============================================================================
// Context
// ============================================================================

const ThemeContext = createContext<ThemeContextValue | null>(null);

export function useTheme() {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within ThemeProvider');
  }
  return context;
}

// ============================================================================
// Provider
// ============================================================================

interface ThemeProviderProps {
  children: React.ReactNode;
  defaultTheme?: Theme;
  storageKey?: string;
}

export function ThemeProvider({ 
  children, 
  defaultTheme = 'dark',
  storageKey = THEME_STORAGE_KEY 
}: ThemeProviderProps) {
  const [theme, setThemeState] = useState<Theme>(() => {
    // Check localStorage first
    if (typeof window !== 'undefined') {
      const stored = localStorage.getItem(storageKey);
      if (stored === 'light' || stored === 'dark') {
        return stored;
      }
      // Check system preference
      if (window.matchMedia && window.matchMedia('(prefers-color-scheme: light)').matches) {
        return 'light';
      }
    }
    return defaultTheme;
  });

  const colors = theme === 'dark' ? darkColors : lightColors;

  // Apply theme to document
  useEffect(() => {
    const root = document.documentElement;
    
    // Set data attribute
    root.setAttribute('data-theme', theme);
    root.setAttribute('data-color-scheme', theme);
    
    // Set CSS custom properties
    Object.entries(colors).forEach(([key, value]) => {
      const cssVar = `--color-${key.replace(/([A-Z])/g, '-$1').toLowerCase()}`;
      root.style.setProperty(cssVar, value);
    });
    
    // Add/remove dark class
    if (theme === 'dark') {
      root.classList.add('dark');
      root.classList.remove('light');
    } else {
      root.classList.add('light');
      root.classList.remove('dark');
    }
    
    // Save to localStorage
    localStorage.setItem(storageKey, theme);
    
    // Update meta theme-color
    let metaTheme = document.querySelector('meta[name="theme-color"]');
    if (!metaTheme) {
      metaTheme = document.createElement('meta');
      metaTheme.setAttribute('name', 'theme-color');
      document.head.appendChild(metaTheme);
    }
    metaTheme.setAttribute('content', colors.background);
  }, [theme, colors, storageKey]);

  const setTheme = useCallback((newTheme: Theme) => {
    setThemeState(newTheme);
  }, []);

  const toggleTheme = useCallback(() => {
    setThemeState(prev => prev === 'dark' ? 'light' : 'dark');
  }, []);

  return (
    <ThemeContext.Provider value={{ theme, colors, setTheme, toggleTheme }}>
      {children}
    </ThemeContext.Provider>
  );
}

// ============================================================================
// CSS Styles (to be included in global stylesheet)
// ============================================================================

export const globalStyles = `
:root {
  /* Default to dark */
  --color-background: ${darkColors.background};
  --color-background-secondary: ${darkColors.backgroundSecondary};
  --color-background-tertiary: ${darkColors.backgroundTertiary};
  --color-surface: ${darkColors.surface};
  --color-surface-hover: ${darkColors.surfaceHover};
  --color-surface-active: ${darkColors.surfaceActive};
  
  --color-text: ${darkColors.text};
  --color-text-secondary: ${darkColors.textSecondary};
  --color-text-tertiary: ${darkColors.textTertiary};
  --color-text-inverse: ${darkColors.textInverse};
  
  --color-border: ${darkColors.border};
  --color-border-light: ${darkColors.borderLight};
  --color-border-focus: ${darkColors.borderFocus};
  
  --color-primary: ${darkColors.primary};
  --color-primary-hover: ${darkColors.primaryHover};
  --color-primary-active: ${darkColors.primaryActive};
  --color-secondary: ${darkColors.secondary};
  --color-secondary-hover: ${darkColors.secondaryHover};
  
  --color-success: ${darkColors.success};
  --color-success-bg: ${darkColors.successBg};
  --color-warning: ${darkColors.warning};
  --color-warning-bg: ${darkColors.warningBg};
  --color-error: ${darkColors.error};
  --color-error-bg: ${darkColors.errorBg};
  --color-info: ${darkColors.info};
  --color-info-bg: ${darkColors.infoBg};
  
  --color-glass: ${darkColors.glass};
  --color-overlay: ${darkColors.overlay};
  
  /* Transitions */
  --transition-fast: 150ms ease;
  --transition-normal: 250ms ease;
  --transition-slow: 350ms ease;
  
  /* Shadows */
  --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
  --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
  --shadow-lg: 0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05);
  --shadow-xl: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04);
  
  /* Border radius */
  --radius-sm: 0.25rem;
  --radius-md: 0.375rem;
  --radius-lg: 0.5rem;
  --radius-xl: 0.75rem;
  --radius-full: 9999px;
  
  /* Spacing */
  --spacing-1: 0.25rem;
  --spacing-2: 0.5rem;
  --spacing-3: 0.75rem;
  --spacing-4: 1rem;
  --spacing-6: 1.5rem;
  --spacing-8: 2rem;
  --spacing-10: 2.5rem;
  --spacing-12: 3rem;
  
  /* Font sizes */
  --font-xs: 0.75rem;
  --font-sm: 0.875rem;
  --font-base: 1rem;
  --font-lg: 1.125rem;
  --font-xl: 1.25rem;
  --font-2xl: 1.5rem;
  --font-3xl: 1.875rem;
  --font-4xl: 2.25rem;
}

[data-theme="light"] {
  --color-background: ${lightColors.background};
  --color-background-secondary: ${lightColors.backgroundSecondary};
  --color-background-tertiary: ${lightColors.backgroundTertiary};
  --color-surface: ${lightColors.surface};
  --color-surface-hover: ${lightColors.surfaceHover};
  --color-surface-active: ${lightColors.surfaceActive};
  
  --color-text: ${lightColors.text};
  --color-text-secondary: ${lightColors.textSecondary};
  --color-text-tertiary: ${lightColors.textTertiary};
  --color-text-inverse: ${lightColors.textInverse};
  
  --color-border: ${lightColors.border};
  --color-border-light: ${lightColors.borderLight};
  --color-border-focus: ${lightColors.borderFocus};
  
  --color-primary: ${lightColors.primary};
  --color-primary-hover: ${lightColors.primaryHover};
  --color-primary-active: ${lightColors.primaryActive};
  --color-secondary: ${lightColors.secondary};
  --color-secondary-hover: ${lightColors.secondaryHover};
  
  --color-success: ${lightColors.success};
  --color-success-bg: ${lightColors.successBg};
  --color-warning: ${lightColors.warning};
  --color-warning-bg: ${lightColors.warningBg};
  --color-error: ${lightColors.error};
  --color-error-bg: ${lightColors.errorBg};
  --color-info: ${lightColors.info};
  --color-info-bg: ${lightColors.infoBg};
  
  --color-glass: ${lightColors.glass};
  --color-overlay: ${lightColors.overlay};
  
  --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
  --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
  --shadow-lg: 0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05);
  --shadow-xl: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04);
}

/* Base styles */
* {
  box-sizing: border-box;
}

html, body {
  margin: 0;
  padding: 0;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
  background-color: var(--color-background);
  color: var(--color-text);
  line-height: 1.5;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  transition: background-color var(--transition-normal), color var(--transition-normal);
}

/* Scrollbar styling */
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track {
  background: var(--color-background-secondary);
}

::-webkit-scrollbar-thumb {
  background: var(--color-border);
  border-radius: var(--radius-full);
}

::-webkit-scrollbar-thumb:hover {
  background: var(--color-text-tertiary);
}

/* Focus styles */
:focus-visible {
  outline: 2px solid var(--color-border-focus);
  outline-offset: 2px;
}

/* Selection */
::selection {
  background-color: var(--color-primary);
  color: var(--color-text-inverse);
}

/* Common component classes */
.tiger-card {
  background-color: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--spacing-4);
  transition: all var(--transition-normal);
}

.tiger-card:hover {
  border-color: var(--color-border-light);
}

.tiger-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: var(--spacing-2) var(--spacing-4);
  font-size: var(--font-sm);
  font-weight: 500;
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--transition-fast);
  border: none;
  outline: none;
}

.tiger-button-primary {
  background-color: var(--color-primary);
  color: var(--color-text-inverse);
}

.tiger-button-primary:hover {
  background-color: var(--color-primary-hover);
}

.tiger-button-primary:active {
  background-color: var(--color-primary-active);
}

.tiger-button-secondary {
  background-color: var(--color-secondary);
  color: var(--color-text-inverse);
}

.tiger-button-secondary:hover {
  background-color: var(--color-secondary-hover);
}

.tiger-input {
  width: 100%;
  padding: var(--spacing-2) var(--spacing-3);
  font-size: var(--font-sm);
  background-color: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  color: var(--color-text);
  transition: all var(--transition-fast);
}

.tiger-input:focus {
  border-color: var(--color-border-focus);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.tiger-input::placeholder {
  color: var(--color-text-tertiary);
}

/* Glass effect */
.glass {
  background: var(--color-glass);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
}

/* Status colors */
.status-success { color: var(--color-success); }
.status-warning { color: var(--color-warning); }
.status-error { color: var(--color-error); }
.status-info { color: var(--color-info); }

.bg-success { background-color: var(--color-success-bg); }
.bg-warning { background-color: var(--color-warning-bg); }
.bg-error { background-color: var(--color-error-bg); }
.bg-info { background-color: var(--color-info-bg); }
`;

// ============================================================================
// Theme Toggle Button Component
// ============================================================================

export function ThemeToggle({ className = '' }: { className?: string }) {
  const { theme, toggleTheme } = useTheme();
  
  return (
    <button
      onClick={toggleTheme}
      className={`theme-toggle-button ${className}`}
      aria-label={`Switch to ${theme === 'dark' ? 'light' : 'dark'} mode`}
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        width: '40px',
        height: '40px',
        borderRadius: 'var(--radius-lg)',
        border: '1px solid var(--color-border)',
        backgroundColor: 'var(--color-surface)',
        cursor: 'pointer',
        fontSize: '1.25rem',
        transition: 'all var(--transition-fast)',
      }}
    >
      {theme === 'dark' ? '☀️' : '🌙'}
    </button>
  );
}

// ============================================================================
// Export
// ============================================================================

export default {
  ThemeProvider,
  useTheme,
  ThemeToggle,
  globalStyles,
};
