/**
 * TigerWallet Admin Platform - Complete Theme System
 * Dark/Light theme with system preference support
 * Works everywhere across all admin apps
 */

import React, { createContext, useContext, useState, useEffect, useCallback, ReactNode } from 'react';
import { adminApi, ThemeMode, Language } from './api_complete';

// ============================================================================
// Theme Types
// ============================================================================

type ThemeMode = 'light' | 'dark' | 'system';

interface ThemeColors {
  // Primary colors
  primary: string;
  primaryHover: string;
  primaryLight: string;
  primaryDark: string;
  
  // Background colors
  background: string;
  backgroundSecondary: string;
  backgroundTertiary: string;
  backgroundElevated: string;
  
  // Surface colors
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
  
  // Status colors
  success: string;
  successLight: string;
  successDark: string;
  
  warning: string;
  warningLight: string;
  warningDark: string;
  
  error: string;
  errorLight: string;
  errorDark: string;
  
  info: string;
  infoLight: string;
  infoDark: string;
  
  // Special colors
  accent: string;
  accentHover: string;
  overlay: string;
  shadow: string;
  divider: string;
}

interface Theme {
  mode: 'light' | 'dark';
  colors: ThemeColors;
  spacing: ThemeSpacing;
  typography: ThemeTypography;
  borderRadius: ThemeBorderRadius;
  shadows: ThemeShadows;
  transitions: ThemeTransitions;
}

interface ThemeSpacing {
  xs: string;
  sm: string;
  md: string;
  lg: string;
  xl: string;
  xxl: string;
}

interface ThemeTypography {
  fontFamily: string;
  fontFamilyMono: string;
  fontSize: ThemeFontSize;
  fontWeight: ThemeFontWeight;
  lineHeight: ThemeLineHeight;
}

interface ThemeFontSize {
  xs: string;
  sm: string;
  md: string;
  lg: string;
  xl: string;
  xxl: string;
  xxxl: string;
}

interface ThemeFontWeight {
  normal: number;
  medium: number;
  semibold: number;
  bold: number;
}

interface ThemeLineHeight {
  tight: number;
  normal: number;
  relaxed: number;
}

interface ThemeBorderRadius {
  sm: string;
  md: string;
  lg: string;
  xl: string;
  full: string;
}

interface ThemeShadows {
  sm: string;
  md: string;
  lg: string;
  xl: string;
  xxl: string;
}

interface ThemeTransitions {
  fast: string;
  normal: string;
  slow: string;
}

// ============================================================================
// Light Theme
// ============================================================================

const lightColors: ThemeColors = {
  // Primary colors
  primary: '#2563EB',
  primaryHover: '#1D4ED8',
  primaryLight: '#DBEAFE',
  primaryDark: '#1E40AF',
  
  // Background colors
  background: '#FFFFFF',
  backgroundSecondary: '#F9FAFB',
  backgroundTertiary: '#F3F4F6',
  backgroundElevated: '#FFFFFF',
  
  // Surface colors
  surface: '#FFFFFF',
  surfaceHover: '#F9FAFB',
  surfaceActive: '#F3F4F6',
  
  // Text colors
  text: '#111827',
  textSecondary: '#6B7280',
  textTertiary: '#9CA3AF',
  textInverse: '#FFFFFF',
  
  // Border colors
  border: '#E5E7EB',
  borderLight: '#F3F4F6',
  borderFocus: '#2563EB',
  
  // Status colors
  success: '#10B981',
  successLight: '#D1FAE5',
  successDark: '#059669',
  
  warning: '#F59E0B',
  warningLight: '#FEF3C7',
  warningDark: '#D97706',
  
  error: '#EF4444',
  errorLight: '#FEE2E2',
  errorDark: '#DC2626',
  
  info: '#3B82F6',
  infoLight: '#DBEAFE',
  infoDark: '#2563EB',
  
  // Special colors
  accent: '#8B5CF6',
  accentHover: '#7C3AED',
  overlay: 'rgba(0, 0, 0, 0.5)',
  shadow: 'rgba(0, 0, 0, 0.1)',
  divider: '#E5E7EB',
};

const lightTheme: Theme = {
  mode: 'light',
  colors: lightColors,
  spacing: {
    xs: '0.25rem',
    sm: '0.5rem',
    md: '1rem',
    lg: '1.5rem',
    xl: '2rem',
    xxl: '3rem',
  },
  typography: {
    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif',
    fontFamilyMono: '"SF Mono", "Monaco", "Inconsolata", "Fira Mono", "Droid Sans Mono", monospace',
    fontSize: {
      xs: '0.75rem',
      sm: '0.875rem',
      md: '1rem',
      lg: '1.125rem',
      xl: '1.25rem',
      xxl: '1.5rem',
      xxxl: '2rem',
    },
    fontWeight: {
      normal: 400,
      medium: 500,
      semibold: 600,
      bold: 700,
    },
    lineHeight: {
      tight: 1.25,
      normal: 1.5,
      relaxed: 1.75,
    },
  },
  borderRadius: {
    sm: '0.25rem',
    md: '0.375rem',
    lg: '0.5rem',
    xl: '0.75rem',
    full: '9999px',
  },
  shadows: {
    sm: '0 1px 2px 0 rgba(0, 0, 0, 0.05)',
    md: '0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06)',
    lg: '0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05)',
    xl: '0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04)',
    xxl: '0 25px 50px -12px rgba(0, 0, 0, 0.25)',
  },
  transitions: {
    fast: '150ms ease',
    normal: '250ms ease',
    slow: '350ms ease',
  },
};

// ============================================================================
// Dark Theme
// ============================================================================

const darkColors: ThemeColors = {
  // Primary colors
  primary: '#3B82F6',
  primaryHover: '#60A5FA',
  primaryLight: '#1E3A8A',
  primaryDark: '#2563EB',
  
  // Background colors
  background: '#111827',
  backgroundSecondary: '#1F2937',
  backgroundTertiary: '#374151',
  backgroundElevated: '#1F2937',
  
  // Surface colors
  surface: '#1F2937',
  surfaceHover: '#374151',
  surfaceActive: '#4B5563',
  
  // Text colors
  text: '#F9FAFB',
  textSecondary: '#D1D5DB',
  textTertiary: '#9CA3AF',
  textInverse: '#111827',
  
  // Border colors
  border: '#374151',
  borderLight: '#4B5563',
  borderFocus: '#3B82F6',
  
  // Status colors
  success: '#34D399',
  successLight: '#064E3B',
  successDark: '#10B981',
  
  warning: '#FBBF24',
  warningLight: '#78350F',
  warningDark: '#F59E0B',
  
  error: '#F87171',
  errorLight: '#7F1D1D',
  errorDark: '#EF4444',
  
  info: '#60A5FA',
  infoLight: '#1E3A8A',
  infoDark: '#3B82F6',
  
  // Special colors
  accent: '#A78BFA',
  accentHover: '#C4B5FD',
  overlay: 'rgba(0, 0, 0, 0.7)',
  shadow: 'rgba(0, 0, 0, 0.3)',
  divider: '#374151',
};

const darkTheme: Theme = {
  mode: 'dark',
  colors: darkColors,
  spacing: lightTheme.spacing,
  typography: {
    ...lightTheme.typography,
    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif',
    fontFamilyMono: lightTheme.typography.fontFamilyMono,
  },
  borderRadius: lightTheme.borderRadius,
  shadows: {
    sm: '0 1px 2px 0 rgba(0, 0, 0, 0.3)',
    md: '0 4px 6px -1px rgba(0, 0, 0, 0.4), 0 2px 4px -1px rgba(0, 0, 0, 0.3)',
    lg: '0 10px 15px -3px rgba(0, 0, 0, 0.4), 0 4px 6px -2px rgba(0, 0, 0, 0.3)',
    xl: '0 20px 25px -5px rgba(0, 0, 0, 0.5), 0 10px 10px -5px rgba(0, 0, 0, 0.3)',
    xxl: '0 25px 50px -12px rgba(0, 0, 0, 0.5)',
  },
  transitions: lightTheme.transitions,
};

// ============================================================================
// Theme Context
// ============================================================================

interface ThemeContextType {
  theme: Theme;
  themeMode: ThemeMode;
  setThemeMode: (mode: ThemeMode) => Promise<void>;
  isDark: boolean;
  colors: ThemeColors;
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

// ============================================================================
// Theme Provider
// ============================================================================

interface ThemeProviderProps {
  children: ReactNode;
  defaultMode?: ThemeMode;
}

export function ThemeProvider({ children, defaultMode = 'system' }: ThemeProviderProps) {
  const [themeMode, setThemeModeState] = useState<ThemeMode>(defaultMode);
  const [isDark, setIsDark] = useState(false);

  // Initialize theme from API or localStorage
  useEffect(() => {
    const initTheme = async () => {
      try {
        // Try to get theme from API
        const prefs = await adminApi.getThemePreference();
        if (prefs?.theme_mode) {
          setThemeModeState(prefs.theme_mode as ThemeMode);
        }
      } catch (error) {
        // Fall back to localStorage
        const stored = localStorage.getItem('admin_theme');
        if (stored && ['light', 'dark', 'system'].includes(stored)) {
          setThemeModeState(stored as ThemeMode);
        }
      }
    };

    initTheme();
  }, []);

  // Update isDark when themeMode changes
  useEffect(() => {
    const updateIsDark = () => {
      if (themeMode === 'system') {
        setIsDark(window.matchMedia('(prefers-color-scheme: dark)').matches);
      } else {
        setIsDark(themeMode === 'dark');
      }
    };

    updateIsDark();

    // Listen for system theme changes
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handler = () => {
      if (themeMode === 'system') {
        updateIsDark();
      }
    };

    mediaQuery.addEventListener('change', handler);
    return () => mediaQuery.removeEventListener('change', handler);
  }, [themeMode]);

  // Apply theme to document
  useEffect(() => {
    const root = document.documentElement;
    root.setAttribute('data-theme', isDark ? 'dark' : 'light');
    
    // Set CSS variables
    const colors = isDark ? darkColors : lightColors;
    Object.entries(colors).forEach(([key, value]) => {
      root.style.setProperty(`--color-${key}`, value);
    });

    // Set body background
    document.body.style.backgroundColor = colors.background;
    document.body.style.color = colors.text;
  }, [isDark]);

  // Set theme mode
  const setThemeMode = useCallback(async (mode: ThemeMode) => {
    setThemeModeState(mode);
    
    try {
      // Save to API
      await adminApi.setThemePreference(mode);
    } catch (error) {
      // Fall back to localStorage
      localStorage.setItem('admin_theme', mode);
    }
  }, []);

  const theme = isDark ? darkTheme : lightTheme;

  return (
    <ThemeContext.Provider value={{ theme, themeMode, setThemeMode, isDark, colors: theme.colors }}>
      {children}
    </ThemeContext.Provider>
  );
}

// ============================================================================
// Hook
// ============================================================================

export function useTheme(): ThemeContextType {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
}

// ============================================================================
// Language Context
// ============================================================================

interface LanguageContextType {
  language: Language;
  setLanguage: (lang: Language) => Promise<void>;
  translations: Record<string, string>;
}

const translations: Record<Language, Record<string, string>> = {
  en: {
    // Common
    'app.name': 'TigerWallet Admin',
    'app.loading': 'Loading...',
    'app.save': 'Save',
    'app.cancel': 'Cancel',
    'app.delete': 'Delete',
    'app.edit': 'Edit',
    'app.create': 'Create',
    'app.search': 'Search',
    'app.filter': 'Filter',
    'app.export': 'Export',
    'app.import': 'Import',
    
    // Auth
    'auth.login': 'Login',
    'auth.logout': 'Logout',
    'auth.email': 'Email',
    'auth.password': 'Password',
    'auth.twoFactor': 'Two-Factor Code',
    
    // Navigation
    'nav.dashboard': 'Dashboard',
    'nav.users': 'Users',
    'nav.kyc': 'KYC',
    'nav.tokens': 'Tokens',
    'nav.pairs': 'Pairs',
    'nav.chains': 'Chains',
    'nav.fees': 'Fees',
    'nav.withdrawals': 'Withdrawals',
    'nav.transactions': 'Transactions',
    'nav.whitelabels': 'White Labels',
    'nav.admins': 'Admins',
    'nav.settings': 'Settings',
    'nav.analytics': 'Analytics',
    'nav.reports': 'Reports',
    
    // Dashboard
    'dashboard.title': 'Dashboard',
    'dashboard.totalUsers': 'Total Users',
    'dashboard.activeUsers': 'Active Users',
    'dashboard.pendingKYC': 'Pending KYC',
    'dashboard.totalTransactions': 'Total Transactions',
    'dashboard.volume24h': '24h Volume',
    'dashboard.revenue24h': '24h Revenue',
    
    // Settings
    'settings.title': 'Settings',
    'settings.theme': 'Theme',
    'settings.theme.light': 'Light',
    'settings.theme.dark': 'Dark',
    'settings.theme.system': 'System',
    'settings.language': 'Language',
    'settings.notifications': 'Notifications',
    'settings.security': 'Security',
    'settings.profile': 'Profile',
    
    // Status
    'status.success': 'Success',
    'status.error': 'Error',
    'status.warning': 'Warning',
    'status.info': 'Info',
    'status.pending': 'Pending',
    'status.active': 'Active',
    'status.inactive': 'Inactive',
    'status.suspended': 'Suspended',
  },
  es: {
    'app.name': 'TigerWallet Admin',
    'app.loading': 'Cargando...',
    'app.save': 'Guardar',
    'app.cancel': 'Cancelar',
    'auth.login': 'Iniciar sesión',
    'nav.dashboard': 'Panel',
    'settings.theme.light': 'Claro',
    'settings.theme.dark': 'Oscuro',
    'settings.theme.system': 'Sistema',
  },
  fr: {
    'app.name': 'TigerWallet Admin',
    'app.loading': 'Chargement...',
    'app.save': 'Enregistrer',
    'app.cancel': 'Annuler',
    'auth.login': 'Connexion',
    'nav.dashboard': 'Tableau de bord',
    'settings.theme.light': 'Clair',
    'settings.theme.dark': 'Sombre',
    'settings.theme.system': 'Système',
  },
  de: {
    'app.name': 'TigerWallet Admin',
    'app.loading': 'Laden...',
    'app.save': 'Speichern',
    'app.cancel': 'Abbrechen',
    'auth.login': 'Anmelden',
    'nav.dashboard': 'Dashboard',
    'settings.theme.light': 'Hell',
    'settings.theme.dark': 'Dunkel',
    'settings.theme.system': 'System',
  },
  zh: {
    'app.name': 'TigerWallet 管理后台',
    'app.loading': '加载中...',
    'app.save': '保存',
    'app.cancel': '取消',
    'auth.login': '登录',
    'nav.dashboard': '仪表板',
    'settings.theme.light': '浅色',
    'settings.theme.dark': '深色',
    'settings.theme.system': '跟随系统',
  },
  ja: {
    'app.name': 'TigerWallet 管理者',
    'app.loading': '読み込み中...',
    'app.save': '保存',
    'app.cancel': 'キャンセル',
    'auth.login': 'ログイン',
    'nav.dashboard': 'ダッシュボード',
    'settings.theme.light': 'ライト',
    'settings.theme.dark': 'ダーク',
    'settings.theme.system': 'システム',
  },
  ko: {
    'app.name': 'TigerWallet 관리자',
    'app.loading': '로딩 중...',
    'app.save': '저장',
    'app.cancel': '취소',
    'auth.login': '로그인',
    'nav.dashboard': '대시보드',
    'settings.theme.light': '라이트',
    'settings.theme.dark': '다크',
    'settings.theme.system': '시스템',
  },
  ar: {
    'app.name': 'TigerWallet المدير',
    'app.loading': 'جار التحميل...',
    'app.save': 'حفظ',
    'app.cancel': 'إلغاء',
    'auth.login': 'تسجيل الدخول',
    'nav.dashboard': 'لوحة القيادة',
    'settings.theme.light': 'فاتح',
    'settings.theme.dark': 'داكن',
    'settings.theme.system': 'النظام',
  },
};

const LanguageContext = createContext<LanguageContextType | undefined>(undefined);

// ============================================================================
// Language Provider
// ============================================================================

interface LanguageProviderProps {
  children: ReactNode;
}

export function LanguageProvider({ children }: LanguageProviderProps) {
  const [language, setLanguageState] = useState<Language>('en');

  // Initialize language from API or localStorage
  useEffect(() => {
    const initLanguage = async () => {
      try {
        // Try to get language from API
        const stored = localStorage.getItem('admin_language');
        if (stored && ['en', 'es', 'fr', 'de', 'zh', 'ja', 'ko', 'ar'].includes(stored)) {
          setLanguageState(stored as Language);
        }
      } catch (error) {
        // Fall back to localStorage
        const stored = localStorage.getItem('admin_language');
        if (stored && ['en', 'es', 'fr', 'de', 'zh', 'ja', 'ko', 'ar'].includes(stored)) {
          setLanguageState(stored as Language);
        }
      }
    };

    initLanguage();
  }, []);

  const setLanguage = useCallback(async (lang: Language) => {
    setLanguageState(lang);
    localStorage.setItem('admin_language', lang);
    
    try {
      await adminApi.setLanguagePreference(lang);
    } catch (error) {
      console.error('Failed to save language preference:', error);
    }
  }, []);

  return (
    <LanguageContext.Provider value={{ language, setLanguage, translations: translations[language] }}>
      {children}
    </LanguageContext.Provider>
  );
}

// ============================================================================
// Hook
// ============================================================================

export function useLanguage(): LanguageContextType {
  const context = useContext(LanguageContext);
  if (!context) {
    throw new Error('useLanguage must be used within a LanguageProvider');
  }
  return context;
}

// ============================================================================
// Combined Provider
// ============================================================================

interface AdminAppProviderProps {
  children: ReactNode;
}

export function AdminAppProvider({ children }: AdminAppProviderProps) {
  return (
    <ThemeProvider>
      <LanguageProvider>
        {children}
      </LanguageProvider>
    </ThemeProvider>
  );
}

// ============================================================================
// CSS Variables Injector
// ============================================================================

export function injectThemeCSS(): void {
  const style = document.createElement('style');
  style.textContent = `
    :root {
      /* Default to light theme - will be overridden by JS */
      --color-primary: #2563EB;
      --color-primary-hover: #1D4ED8;
      --color-background: #FFFFFF;
      --color-background-secondary: #F9FAFB;
      --color-surface: #FFFFFF;
      --color-text: #111827;
      --color-text-secondary: #6B7280;
      --color-border: #E5E7EB;
      --color-success: #10B981;
      --color-warning: #F59E0B;
      --color-error: #EF4444;
      --color-info: #3B82F6;
    }

    [data-theme="dark"] {
      --color-primary: #3B82F6;
      --color-primary-hover: #60A5FA;
      --color-background: #111827;
      --color-background-secondary: #1F2937;
      --color-surface: #1F2937;
      --color-text: #F9FAFB;
      --color-text-secondary: #D1D5DB;
      --color-border: #374151;
      --color-success: #34D399;
      --color-warning: #FBBF24;
      --color-error: #F87171;
      --color-info: #60A5FA;
    }

    * {
      margin: 0;
      padding: 0;
      box-sizing: border-box;
    }

    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
      background-color: var(--color-background);
      color: var(--color-text);
      transition: background-color 0.25s ease, color 0.25s ease;
    }

    /* Utility classes */
    .bg-primary { background-color: var(--color-primary); }
    .bg-background { background-color: var(--color-background); }
    .bg-surface { background-color: var(--color-surface); }
    .text-primary { color: var(--color-primary); }
    .text-text { color: var(--color-text); }
    .text-secondary { color: var(--color-text-secondary); }
    .border { border-color: var(--color-border); }

    /* Theme-aware components */
    .card {
      background-color: var(--color-surface);
      border: 1px solid var(--color-border);
      border-radius: 0.5rem;
      padding: 1rem;
      box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
    }

    .button {
      padding: 0.5rem 1rem;
      border-radius: 0.375rem;
      font-weight: 500;
      cursor: pointer;
      transition: all 0.25s ease;
    }

    .button-primary {
      background-color: var(--color-primary);
      color: white;
      border: none;
    }

    .button-primary:hover {
      background-color: var(--color-primary-hover);
    }

    .input {
      padding: 0.5rem 0.75rem;
      border: 1px solid var(--color-border);
      border-radius: 0.375rem;
      background-color: var(--color-surface);
      color: var(--color-text);
    }

    .input:focus {
      outline: none;
      border-color: var(--color-primary);
      box-shadow: 0 0 0 2px var(--color-primary-light, rgba(37, 99, 235, 0.2));
    }
  `;
  document.head.appendChild(style);
}

export default {
  ThemeProvider,
  LanguageProvider,
  AdminAppProvider,
  useTheme,
  useLanguage,
  injectThemeCSS,
  lightTheme,
  darkTheme,
};
