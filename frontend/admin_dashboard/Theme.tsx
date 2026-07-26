/**
 * TigerWallet Theme System
 * Light/Dark theme with complete system-wide support
 */

import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';

// ==================== Theme Types ====================

export type ThemeMode = 'light' | 'dark' | 'system';

export interface ThemeColors {
  // Primary colors
  primary: string;
  primaryLight: string;
  primaryDark: string;
  primaryGradient: string[];
  
  // Background colors
  background: string;
  backgroundSecondary: string;
  surface: string;
  card: string;
  cardBackground: string;
  
  // Text colors
  textPrimary: string;
  textSecondary: string;
  textMuted: string;
  textInverse: string;
  
  // Border colors
  border: string;
  borderLight: string;
  
  // Status colors
  success: string;
  successLight: string;
  error: string;
  errorLight: string;
  warning: string;
  warningLight: string;
  info: string;
  infoLight: string;
  
  // Special
  overlay: string;
  shadow: string;
}

export interface Theme {
  mode: ThemeMode;
  colors: ThemeColors;
  isDark: boolean;
}

// ==================== Light Theme ====================

export const lightTheme: Theme = {
  mode: 'light',
  isDark: false,
  colors: {
    primary: '#f39c12',
    primaryLight: '#f5b041',
    primaryDark: '#d68910',
    primaryGradient: ['#f39c12', '#e67e22'],
    
    background: '#f8f9fa',
    backgroundSecondary: '#ffffff',
    surface: '#ffffff',
    card: '#ffffff',
    cardBackground: '#ffffff',
    
    textPrimary: '#1a1a2e',
    textSecondary: '#6c757d',
    textMuted: '#adb5bd',
    textInverse: '#ffffff',
    
    border: '#dee2e6',
    borderLight: '#e9ecef',
    
    success: '#27ae60',
    successLight: '#d4edda',
    error: '#e74c3c',
    errorLight: '#f8d7da',
    warning: '#f39c12',
    warningLight: '#fff3cd',
    info: '#3498db',
    infoLight: '#d1ecf1',
    
    overlay: 'rgba(0, 0, 0, 0.5)',
    shadow: 'rgba(0, 0, 0, 0.1)',
  },
};

// ==================== Dark Theme ====================

export const darkTheme: Theme = {
  mode: 'dark',
  isDark: true,
  colors: {
    primary: '#f39c12',
    primaryLight: '#f5b041',
    primaryDark: '#d68910',
    primaryGradient: ['#f39c12', '#e67e22'],
    
    background: '#0a0a0f',
    backgroundSecondary: '#12121a',
    surface: '#1a1a24',
    card: '#1e1e2e',
    cardBackground: '#252535',
    
    textPrimary: '#ffffff',
    textSecondary: '#a0a0b0',
    textMuted: '#6c6c7c',
    textInverse: '#1a1a2e',
    
    border: '#2a2a3a',
    borderLight: '#333345',
    
    success: '#2ecc71',
    successLight: '#1e3a2a',
    error: '#e74c3c',
    errorLight: '#3a1e1e',
    warning: '#f39c12',
    warningLight: '#3a2e1e',
    info: '#3498db',
    infoLight: '#1e2a3a',
    
    overlay: 'rgba(0, 0, 0, 0.7)',
    shadow: 'rgba(0, 0, 0, 0.5)',
  },
};

// ==================== Theme Context ====================

interface ThemeContextType {
  theme: Theme;
  setThemeMode: (mode: ThemeMode) => void;
  toggleTheme: () => void;
  colors: ThemeColors;
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

// ==================== Theme Provider ====================

interface ThemeProviderProps {
  children: ReactNode;
  defaultMode?: ThemeMode;
}

export const ThemeProvider: React.FC<ThemeProviderProps> = ({ 
  children, 
  defaultMode = 'dark' 
}) => {
  const [themeMode, setThemeModeState] = useState<ThemeMode>(() => {
    // Check localStorage first
    const stored = localStorage.getItem('theme-mode') as ThemeMode;
    if (stored && ['light', 'dark', 'system'].includes(stored)) {
      return stored;
    }
    return defaultMode;
  });

  const [systemPreference, setSystemPreference] = useState<'light' | 'dark'>(() => {
    if (typeof window !== 'undefined') {
      return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    }
    return 'dark';
  });

  // Listen for system preference changes
  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handler = (e: MediaQueryListEvent) => {
      setSystemPreference(e.matches ? 'dark' : 'light');
    };
    
    mediaQuery.addEventListener('change', handler);
    return () => mediaQuery.removeEventListener('change', handler);
  }, []);

  // Determine actual theme based on mode
  const theme = themeMode === 'system' 
    ? (systemPreference === 'dark' ? darkTheme : lightTheme)
    : (themeMode === 'dark' ? darkTheme : lightTheme);

  const setThemeMode = (mode: ThemeMode) => {
    setThemeModeState(mode);
    localStorage.setItem('theme-mode', mode);
  };

  const toggleTheme = () => {
    const newMode = theme.isDark ? 'light' : 'dark';
    setThemeMode(newMode);
  };

  // Apply theme to document
  useEffect(() => {
    const root = document.documentElement;
    
    // Set CSS variables
    Object.entries(theme.colors).forEach(([key, value]) => {
      const cssKey = key.replace(/([A-Z])/g, '-$1').toLowerCase();
      root.style.setProperty(`--color-${cssKey}`, value);
    });
    
    // Set data attribute for global styling
    root.setAttribute('data-theme', theme.mode);
    
    // Update body class
    document.body.classList.remove('light-theme', 'dark-theme');
    document.body.classList.add(`${theme.mode}-theme`);
    
    // Update meta theme-color
    let metaTheme = document.querySelector('meta[name="theme-color"]');
    if (!metaTheme) {
      metaTheme = document.createElement('meta');
      metaTheme.setAttribute('name', 'theme-color');
      document.head.appendChild(metaTheme);
    }
    metaTheme.setAttribute('content', theme.colors.background);
  }, [theme]);

  return (
    <ThemeContext.Provider value={{ theme, setThemeMode, toggleTheme, colors: theme.colors }}>
      {children}
    </ThemeContext.Provider>
  );
};

// ==================== Hook ====================

export const useTheme = (): ThemeContextType => {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
};

// ==================== Theme Toggle Button ====================

export const ThemeToggle: React.FC = () => {
  const { theme, toggleTheme, colors } = useTheme();

  return (
    <button
      onClick={toggleTheme}
      className="theme-toggle-btn"
      style={{
        background: colors.card,
        border: `1px solid ${colors.border}`,
        borderRadius: '8px',
        padding: '8px 12px',
        cursor: 'pointer',
        display: 'flex',
        alignItems: 'center',
        gap: '8px',
        color: colors.textPrimary,
        transition: 'all 0.2s ease',
      }}
      aria-label={`Switch to ${theme.isDark ? 'light' : 'dark'} mode`}
    >
      {theme.isDark ? (
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
          <circle cx="12" cy="12" r="5"/>
          <path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/>
        </svg>
      ) : (
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
          <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>
        </svg>
      )}
      <span style={{ fontSize: '14px', fontWeight: 500 }}>
        {theme.isDark ? 'Light' : 'Dark'}
      </span>
    </button>
  );
};

// ==================== Global Styles ====================

export const getGlobalStyles = (theme: Theme): string => `
  :root {
    --color-primary: ${theme.colors.primary};
    --color-primary-light: ${theme.colors.primaryLight};
    --color-primary-dark: ${theme.colors.primaryDark};
    --color-background: ${theme.colors.background};
    --color-surface: ${theme.colors.surface};
    --color-card: ${theme.colors.card};
    --color-card-background: ${theme.colors.cardBackground};
    --color-text-primary: ${theme.colors.textPrimary};
    --color-text-secondary: ${theme.colors.textSecondary};
    --color-text-muted: ${theme.colors.textMuted};
    --color-border: ${theme.colors.border};
    --color-border-light: ${theme.colors.borderLight};
    --color-success: ${theme.colors.success};
    --color-error: ${theme.colors.error};
    --color-warning: ${theme.colors.warning};
    --color-info: ${theme.colors.info};
  }

  .light-theme {
    background: ${lightTheme.colors.background};
    color: ${lightTheme.colors.textPrimary};
  }

  .dark-theme {
    background: ${darkTheme.colors.background};
    color: ${darkTheme.colors.textPrimary};
  }

  * {
    transition: background-color 0.2s ease, color 0.2s ease, border-color 0.2s ease;
  }

  body {
    background: var(--color-background);
    color: var(--color-text-primary);
  }

  .card, .surface {
    background: var(--color-card);
    border-color: var(--color-border);
  }

  .text-muted {
    color: var(--color-text-secondary);
  }
`;

// ==================== Styled Components Helper ====================

export interface StyledProps {
  theme?: Theme;
  [key: string]: any;
}

export const styled = (component: any, styles: (props: StyledProps) => string) => {
  return (props: StyledProps) => {
    const theme = props.theme || darkTheme;
    return component;
  };
};

export default { ThemeProvider, useTheme, ThemeToggle, lightTheme, darkTheme };
