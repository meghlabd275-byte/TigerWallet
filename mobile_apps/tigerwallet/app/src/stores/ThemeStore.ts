// ============================================================================
// TigerWallet - Theme Store
// Complete Light/Dark Theme System
// ============================================================================

import { create } from 'zustand';
import AsyncStorage from '@react-native-async-storage/async-storage';
import { Theme, ThemeColors, ThemeSpacing, ThemeTypography, ThemeBorderRadius } from '../types/wallet';

// ============================================================================
// Light Theme
// ============================================================================

const lightColors: ThemeColors = {
  primary: '#FF6B35',
  secondary: '#0047AB',
  background: '#FFFFFF',
  surface: '#F8F9FA',
  surfaceVariant: '#E9ECEF',
  text: '#1A1A2E',
  textSecondary: '#6C757D',
  textTertiary: '#ADB5BD',
  border: '#DEE2E6',
  borderLight: '#E9ECEF',
  success: '#28A745',
  warning: '#FFC107',
  error: '#DC3545',
  info: '#17A2B8',
  positive: '#28A745',
  negative: '#DC3545',
};

const lightTheme: Theme = {
  mode: 'light',
  colors: lightColors,
  spacing: {
    xs: 4,
    sm: 8,
    md: 16,
    lg: 24,
    xl: 32,
    xxl: 48,
  },
  typography: {
    fontFamily: 'System',
    fontSize: {
      xs: 10,
      sm: 12,
      md: 14,
      lg: 16,
      xl: 18,
      xxl: 24,
      xxxl: 32,
    },
    fontWeight: {
      regular: 400,
      medium: 500,
      semibold: 600,
      bold: 700,
    },
  },
  borderRadius: {
    sm: 4,
    md: 8,
    lg: 12,
    xl: 16,
    full: 9999,
  },
};

// ============================================================================
// Dark Theme
// ============================================================================

const darkColors: ThemeColors = {
  primary: '#FF6B35',
  secondary: '#4A90D9',
  background: '#0D0D0D',
  surface: '#1A1A1A',
  surfaceVariant: '#2D2D2D',
  text: '#FFFFFF',
  textSecondary: '#B0B0B0',
  textTertiary: '#707070',
  border: '#3D3D3D',
  borderLight: '#2D2D2D',
  success: '#4ADE80',
  warning: '#FBBF24',
  error: '#F87171',
  info: '#38BDF8',
  positive: '#4ADE80',
  negative: '#F87171',
};

const darkTheme: Theme = {
  mode: 'dark',
  colors: darkColors,
  spacing: lightTheme.spacing,
  typography: lightTheme.typography,
  borderRadius: lightTheme.borderRadius,
};

// ============================================================================
// Theme Store
// ============================================================================

interface ThemeState {
  theme: Theme;
  isDark: boolean;
  systemPreference: boolean;
  setTheme: (mode: 'light' | 'dark') => Promise<void>;
  toggleTheme: () => Promise<void>;
  setSystemPreference: (enabled: boolean) => void;
  initialize: () => Promise<void>;
}

// ============================================================================
// Color Helpers (for dynamic theming)
// ============================================================================

export const getPrimaryColor = (isDark: boolean): string => 
  isDark ? '#FF6B35' : '#FF6B35';

export const getBackgroundColor = (isDark: boolean): string => 
  isDark ? '#0D0D0D' : '#FFFFFF';

export const getSurfaceColor = (isDark: boolean): string => 
  isDark ? '#1A1A1A' : '#F8F9FA';

export const getTextColor = (isDark: boolean): string => 
  isDark ? '#FFFFFF' : '#1A1A2E';

export const getTextSecondaryColor = (isDark: boolean): string => 
  isDark ? '#B0B0B0' : '#6C757D';

export const getBorderColor = (isDark: boolean): string => 
  isDark ? '#3D3D3D' : '#DEE2E6';

// ============================================================================
// Gradient Definitions
// ============================================================================

export const lightGradients = {
  primary: ['#FF6B35', '#FF8C5A'],
  secondary: ['#0047AB', '#1E5AAF'],
  success: ['#28A745', '#34CE57'],
  warning: ['#FFC107', '#FFD43B'],
  error: ['#DC3545', '#E4606D'],
  background: ['#FFFFFF', '#F8F9FA'],
  card: ['#FFFFFF', '#F8F9FA'],
  header: ['#FF6B35', '#E55A28'],
};

export const darkGradients = {
  primary: ['#FF6B35', '#FF8C5A'],
  secondary: ['#4A90D9', '#6BA3E0'],
  success: ['#4ADE80', '#6EE7A0'],
  warning: ['#FBBF24', '#FCD34D'],
  error: ['#F87171', '#F98B8B'],
  background: ['#0D0D0D', '#1A1A1A'],
  card: ['#1A1A1A', '#2D2D2D'],
  header: ['#1A1A1A', '#0D0D0D'],
};

// ============================================================================
// Create Theme Store
// ============================================================================

export const useThemeStore = create<ThemeState>((set, get) => ({
  theme: lightTheme,
  isDark: false,
  systemPreference: false,

  setTheme: async (mode: 'light' | 'dark') => {
    const theme = mode === 'dark' ? darkTheme : lightTheme;
    set({ theme, isDark: mode === 'dark' });
    await AsyncStorage.setItem('theme_mode', mode);
  },

  toggleTheme: async () => {
    const { isDark } = get();
    const newMode = isDark ? 'light' : 'dark';
    await get().setTheme(newMode);
  },

  setSystemPreference: (enabled: boolean) => {
    set({ systemPreference: enabled });
    AsyncStorage.setItem('system_preference', enabled ? 'true' : 'false');
  },

  initialize: async () => {
    try {
      // Get saved theme
      const savedMode = await AsyncStorage.getItem('theme_mode');
      const systemPref = await AsyncStorage.getItem('system_preference');
      
      if (savedMode) {
        const theme = savedMode === 'dark' ? darkTheme : lightTheme;
        set({ theme, isDark: savedMode === 'dark' });
      } else if (systemPref === 'true') {
        // Would use Appearance API in real app
        const isSystemDark = false; // Would detect from system
        const theme = isSystemDark ? darkTheme : lightTheme;
        set({ theme, isDark: isSystemDark, systemPreference: true });
      }
    } catch (error) {
      console.error('Failed to initialize theme:', error);
    }
  },
}));

// ============================================================================
// Export
// ============================================================================

export default useThemeStore;
